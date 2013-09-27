package main

import (
	"bytes"
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"path"
	"sort"
	"strings"
)

type Translator struct {
	packages map[string]*PkgContext
	writer   io.Writer
}

type PkgContext struct {
	pkg          *types.Package
	info         *types.Info
	pkgVars      map[string]string
	objectVars   map[types.Object]string
	usedVarNames []string
	resultNames  []ast.Expr
	writer       io.Writer
	indentation  int
	delayedLines *bytes.Buffer
}

func (c *PkgContext) newVarName(prefix string) string {
	n := 0
	for {
		name := prefix
		if n != 0 {
			name += fmt.Sprintf("%d", n)
		}
		used := false
		for _, usedName := range c.usedVarNames {
			if usedName == name {
				used = true
				break
			}
		}
		if !used {
			c.usedVarNames = append(c.usedVarNames, name)
			return name
		}
		n += 1
	}
}

func (c *PkgContext) Write(b []byte) (int, error) {
	return c.writer.Write(b)
}

func (c *PkgContext) Printf(format string, values ...interface{}) {
	c.Write([]byte(strings.Repeat("\t", c.indentation)))
	fmt.Fprintf(c, format, values...)
	c.Write([]byte{'\n'})
	c.delayedLines.WriteTo(c.writer)
	c.delayedLines.Reset()
}

func (c *PkgContext) PrintfDelayed(format string, values ...interface{}) {
	c.delayedLines.Write([]byte(strings.Repeat("\t", c.indentation)))
	fmt.Fprintf(c.delayedLines, format, values...)
	c.delayedLines.Write([]byte{'\n'})
}

func (c *PkgContext) Indent(f func()) {
	c.indentation += 1
	f()
	c.indentation -= 1
}

func (c *PkgContext) CatchOutput(f func()) string {
	origWriter := c.writer
	b := bytes.NewBuffer(nil)
	c.writer = b
	f()
	c.writer = origWriter
	return b.String()
}

func main() {
	fi, err := os.Stat(os.Args[1])
	if err != nil {
		panic(err)
	}

	var pkg *build.Package
	if !fi.IsDir() {
		pkg = &build.Package{
			Name:       "main",
			ImportPath: "main",
			Dir:        path.Dir(os.Args[1]),
			GoFiles:    []string{path.Base(os.Args[1])},
		}
	}
	if fi.IsDir() {
		var err error
		pkg, err = build.ImportDir(os.Args[1], 0)
		if err != nil {
			panic(err)
		}
	}

	fileSet := token.NewFileSet()
	out := os.Stdout

	t := &Translator{
		writer:   out,
		packages: make(map[string]*PkgContext),
	}
	t.packages["reflect"] = nil
	t.packages["go/doc"] = nil
	out.WriteString(strings.TrimSpace(prelude))
	out.WriteString("\n")
	t.translatePackage(fileSet, pkg)
}

func (t *Translator) translatePackage(fileSet *token.FileSet, pkg *build.Package) {
	files := make([]*ast.File, 0)
	for _, name := range pkg.GoFiles {
		fullName := pkg.Dir + "/" + name
		file, err := parser.ParseFile(fileSet, fullName, nil, 0)
		if err != nil {
			list, isList := err.(scanner.ErrorList)
			if !isList {
				panic(err)
			}
			for _, entry := range list {
				fmt.Println(entry.Error())
			}
			return
		}
		files = append(files, file)
	}

	var previousErr string
	config := &types.Config{
		Error: func(err error) {
			if err.Error() != previousErr {
				fmt.Println(err.Error())
			}
			previousErr = err.Error()
		},
	}

	info := &types.Info{
		Types:      make(map[ast.Expr]types.Type),
		Values:     make(map[ast.Expr]exact.Value),
		Objects:    make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	typesPkg, err := config.Check(files[0].Name.Name, fileSet, files, info)
	if err != nil {
		return
	}

	for _, importedPkg := range typesPkg.Imports() {
		if _, found := t.packages[importedPkg.Path()]; found {
			continue
		}

		otherPkg, err := build.Import(importedPkg.Path(), pkg.Dir, 0)
		if err != nil {
			panic(err)
		}
		t.translatePackage(fileSet, otherPkg)
	}

	c := &PkgContext{
		pkg:          typesPkg,
		info:         info,
		pkgVars:      make(map[string]string),
		objectVars:   make(map[types.Object]string),
		usedVarNames: []string{"class", "delete", "eval", "export", "false", "implements", "in", "new", "static", "this", "true", "try", "packages", "Array", "Boolean", "Float", "Integer", "String"},
		writer:       t.writer,
		delayedLines: bytes.NewBuffer(nil),
	}
	t.packages[pkg.ImportPath] = c

	functionsByType := make(map[types.Type][]*ast.FuncDecl)
	functionsByObject := make(map[types.Object]*ast.FuncDecl)
	for _, file := range files {
		for _, decl := range file.Decls {
			if fun, isFunction := decl.(*ast.FuncDecl); isFunction {
				var recvType types.Type
				if fun.Recv != nil && len(fun.Recv.List[0].Names) == 1 {
					recvType = c.info.Objects[fun.Recv.List[0].Names[0]].Type()
					if ptr, isPtr := recvType.(*types.Pointer); isPtr {
						recvType = ptr.Elem()
					}
				}
				functionsByType[recvType] = append(functionsByType[recvType], fun)
				functionsByObject[c.info.Objects[fun.Name]] = fun
			}
		}
	}

	c.Printf(`packages["%s"] = (function() {`, pkg.ImportPath)
	c.Indent(func() {
		for _, importedPkg := range c.pkg.Imports() {
			varName := c.newVarName(importedPkg.Name())
			c.Printf(`var %s = packages["%s"];`, varName, importedPkg.Path())
			c.pkgVars[importedPkg.Path()] = varName
		}

		// types and their functions
		for _, file := range files {
			for _, decl := range file.Decls {
				if genDecl, isGenDecl := decl.(*ast.GenDecl); isGenDecl && genDecl.Tok == token.TYPE {
					for _, spec := range genDecl.Specs {
						recvType := c.info.Objects[spec.(*ast.TypeSpec).Name].Type().(*types.Named)
						_, isStruct := recvType.Underlying().(*types.Struct)
						hasPtrType := !isStruct
						c.translateSpec(spec)
						if hasPtrType {
							c.Printf("%s.Go$Pointer = function(getter, setter) { this.Go$get = getter; this.Go$set = setter; };", recvType.Obj().Name())
						}
						for _, fun := range functionsByType[recvType] {
							c.translateFunction(fun, hasPtrType)
						}
					}
				}
			}
		}

		// package functions
		hasInit := false
		for _, fun := range functionsByType[nil] {
			if fun.Name.Name == "init" {
				hasInit = true
			}
			if fun.Body == nil {
				c.Printf(`var %s = function() { throw new GoError("Native function not implemented: %s"); };`, fun.Name, fun.Name)
				continue
			}
			funcLit := &ast.FuncLit{
				Type: fun.Type,
				Body: &ast.BlockStmt{
					List: fun.Body.List,
				},
			}
			c.info.Types[funcLit] = c.info.Objects[fun.Name].Type()
			c.translateStmt(&ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{fun.Name},
				Rhs: []ast.Expr{funcLit},
			}, "")
		}

		// constants and variables in dependency aware order
		var specs []*ast.ValueSpec
		pendingObjects := make(map[types.Object]bool)
		for _, file := range files {
			for _, decl := range file.Decls {
				if genDecl, isGenDecl := decl.(*ast.GenDecl); isGenDecl && (genDecl.Tok == token.CONST || genDecl.Tok == token.VAR) {
					for _, spec := range genDecl.Specs {
						s := spec.(*ast.ValueSpec)
						for i, name := range s.Names {
							var values []ast.Expr
							if genDecl.Tok == token.CONST {
								id := ast.NewIdent("")
								o := c.info.Objects[name].(*types.Const)
								c.info.Types[id] = o.Type()
								c.info.Values[id] = o.Val()
								values = []ast.Expr{id}
							}
							if genDecl.Tok == token.VAR && i < len(s.Values) {
								values = []ast.Expr{s.Values[i]}
							}
							specs = append(specs, &ast.ValueSpec{
								Names:  []*ast.Ident{name},
								Type:   s.Type,
								Values: values,
							})
							pendingObjects[c.info.Objects[s.Names[0]]] = true
						}
					}
				}
			}
		}
		complete := false
		for !complete {
			complete = true
			for i, spec := range specs {
				if spec == nil {
					continue
				}
				if spec.Values != nil {
					v := IsReadyVisitor{info: c.info, functions: functionsByObject, pendingObjects: pendingObjects, isReady: true}
					ast.Walk(&v, spec.Values[0])
					if !v.isReady {
						complete = false
						continue
					}
				}
				c.translateSpec(spec)
				delete(pendingObjects, c.info.Objects[spec.Names[0]])
				specs[i] = nil
			}
		}

		c.Write([]byte(natives[pkg.ImportPath]))

		if hasInit {
			c.Printf("init();")
		}
		if pkg.IsCommand() {
			c.Printf("main();")
		}
		exports := make([]string, 0)
		for _, name := range c.pkg.Scope().Names() {
			if ast.IsExported(name) {
				exports = append(exports, fmt.Sprintf("%s: %s", name, name))
			}
		}
		c.Printf("return { %s };", strings.Join(exports, ", "))
	})
	c.Printf("})()")
}

func (c *PkgContext) translateSpec(spec ast.Spec) {
	switch s := spec.(type) {
	case *ast.ValueSpec:
		for _, name := range s.Names {
			c.info.Types[name] = c.info.Objects[name].Type()
		}
		i := 0
		for i < len(s.Names) {
			var rhs ast.Expr
			n := 1
			if i < len(s.Values) {
				rhs = s.Values[i]
				if tuple, isTuple := c.info.Types[rhs].(*types.Tuple); isTuple {
					n = tuple.Len()
				}
			}
			lhs := make([]ast.Expr, n)
			for j := range lhs {
				if j >= len(s.Names) {
					lhs[j] = ast.NewIdent("_")
					continue
				}
				lhs[j] = s.Names[i+j]
			}
			c.translateStmt(&ast.AssignStmt{
				Lhs: lhs,
				Tok: token.DEFINE,
				Rhs: []ast.Expr{rhs},
			}, "")
			i += n
		}

	case *ast.TypeSpec:
		nt := c.info.Objects[s.Name].Type().(*types.Named)
		switch t := nt.Underlying().(type) {
		case *types.Basic, *types.Array, *types.Signature:
			c.Printf("var %s = function(v) { this.v = v; };", nt.Obj().Name())
		case *types.Struct:
			params := make([]string, t.NumFields())
			for i := 0; i < t.NumFields(); i++ {
				params[i] = t.Field(i).Name() + "_"
			}
			c.Printf("var %s = function(%s) {", nt.Obj().Name(), strings.Join(params, ", "))
			c.Indent(func() {
				c.Printf("this.Go$id = Go$idCounter++;")
				for i := 0; i < t.NumFields(); i++ {
					field := t.Field(i)
					c.Printf("this.%s = %s_;", field.Name(), field.Name())
				}
			})
			c.Printf("};")
			for i := 0; i < t.NumFields(); i++ {
				field := t.Field(i)
				if field.Anonymous() {
					fieldType := field.Type()
					_, isPointer := fieldType.(*types.Pointer)
					_, isUnderlyingBasic := fieldType.Underlying().(*types.Basic)
					_, isUnderlyingInterface := fieldType.Underlying().(*types.Interface)
					if !isPointer && !isUnderlyingInterface {
						fieldType = types.NewPointer(fieldType) // strange, seems like a bug in go/types
					}
					methods := fieldType.MethodSet()
					for j := 0; j < methods.Len(); j++ {
						name := methods.At(j).Obj().Name()
						sig := methods.At(j).Type().(*types.Signature)
						params := make([]string, sig.Params().Len())
						for k := range params {
							params[k] = sig.Params().At(k).Name()
						}
						value := "this." + field.Name()
						if isUnderlyingBasic {
							value = fmt.Sprintf("new %s(%s)", field.Name(), value)
						}
						c.Printf("%s.prototype.%s = function(%s) { return %s.%s(%s); };", nt.Obj().Name(), name, strings.Join(params, ", "), value, name, strings.Join(params, ", "))
					}
				}
			}
		case *types.Slice:
			c.Printf("var %s = function() { Go$Slice.apply(this, arguments); };", nt.Obj().Name())
			c.Printf("Go$copyFields(Go$Slice.prototype, %s.prototype);", nt.Obj().Name())
		case *types.Map:
			c.Printf("var %s = function() { Go$Map.apply(this, arguments); };", nt.Obj().Name())
		case *types.Interface:
			if t.MethodSet().Len() == 0 {
				c.Printf("var %s = function(t) { return true };", nt.Obj().Name())
				return
			}
			implementedBy := make([]string, 0)
			for _, other := range c.info.Objects {
				if otherTypeName, isTypeName := other.(*types.TypeName); isTypeName {
					index := sort.SearchStrings(implementedBy, otherTypeName.Name())
					if (index == len(implementedBy) || implementedBy[index] != otherTypeName.Name()) && types.IsAssignableTo(otherTypeName.Type(), t) {
						implementedBy = append(implementedBy, otherTypeName.Name())
						sort.Strings(implementedBy)
					}
				}
			}
			conditions := make([]string, len(implementedBy))
			for i, other := range implementedBy {
				conditions[i] = "t === " + other
			}
			if len(conditions) == 0 {
				conditions = []string{"false"}
			}
			c.Printf("var %s = function(t) { return %s };", nt.Obj().Name(), strings.Join(conditions, " || "))
		default:
			panic(fmt.Sprintf("Unhandled type: %T\n", t))
		}

	case *ast.ImportSpec:
		// ignored

	default:
		panic(fmt.Sprintf("Unhandled spec: %T\n", s))

	}
}

func (c *PkgContext) translateFunction(fun *ast.FuncDecl, hasPtrType bool) {
	recv := fun.Recv.List[0].Names[0]
	recvType := c.info.Objects[recv].Type()
	ptr, isPointer := recvType.(*types.Pointer)
	_, isUnderlyingBasic := recvType.Underlying().(*types.Basic)

	var this ast.Expr = ast.NewIdent("this")
	if isUnderlyingBasic {
		this = ast.NewIdent("this.v")
	}
	if _, isUnderlyingStruct := recvType.Underlying().(*types.Struct); isUnderlyingStruct {
		this = &ast.StarExpr{X: this}
	}
	c.info.Types[this] = recvType

	lhs := ast.NewIdent(c.typeName(recvType) + ".prototype." + fun.Name.Name)
	c.info.Types[lhs] = c.info.Objects[fun.Name].Type()
	funcLit := &ast.FuncLit{
		Type: fun.Type,
		Body: &ast.BlockStmt{
			List: append([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{recv},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{this},
				},
			}, fun.Body.List...),
		},
	}
	c.info.Types[funcLit] = c.info.Objects[fun.Name].Type()
	c.translateStmt(&ast.AssignStmt{
		Tok: token.ASSIGN,
		Lhs: []ast.Expr{lhs},
		Rhs: []ast.Expr{funcLit},
	}, "")

	if hasPtrType {
		params := c.translateParams(fun.Type)
		if !isPointer {
			typeName := c.typeName(recvType)
			value := "this.Go$get()"
			if isUnderlyingBasic {
				value = fmt.Sprintf("new %s(%s)", typeName, value)
			}
			c.Printf("%s.Go$Pointer.prototype.%s = function(%s) { return %s.%s(%s); };", typeName, fun.Name.Name, params, value, fun.Name.Name, params)
		}
		if isPointer {
			typeName := c.typeName(ptr.Elem())
			value := "this"
			if _, isUnderlyingBasic := ptr.Elem().Underlying().(*types.Basic); isUnderlyingBasic {
				value = "this.v"
			}
			c.Printf("%s.prototype.%s = function(%s) { var obj = %s; return (new %s.Go$Pointer(function() { return obj; }, null)).%s(%s); };", typeName, fun.Name.Name, params, value, typeName, fun.Name.Name, params)
		}
	}
}

func (c *PkgContext) translateParams(t *ast.FuncType) string {
	params := make([]string, 0)
	for _, param := range t.Params.List {
		for _, ident := range param.Names {
			if isUnderscore(ident) {
				params = append(params, c.newVarName("param"))
				continue
			}
			params = append(params, c.translateExpr(ident))
		}
	}
	return strings.Join(params, ", ")
}

func (c *PkgContext) translateArgs(call *ast.CallExpr) []string {
	funType := c.info.Types[call.Fun].Underlying().(*types.Signature)
	args := make([]string, funType.Params().Len())
	for i := range args {
		if funType.IsVariadic() && i == len(args)-1 {
			if call.Ellipsis.IsValid() {
				args[i] = c.translateExprToSlice(call.Args[i])
				break
			}
			args[i] = fmt.Sprintf("new Go$Slice(%s)", createListComposite(funType.Params().At(i).Type(), c.translateExprSlice(call.Args[i:])))
			break
		}
		args[i] = c.translateExprToType(call.Args[i], funType.Params().At(i).Type())
	}
	return args
}

func (c *PkgContext) zeroValue(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		if t.Info()&types.IsNumeric != 0 {
			return "0"
		}
		if t.Info()&types.IsString != 0 {
			return `""`
		}
	case *types.Array:
		return fmt.Sprintf("Go$clear(new %s(%d))", toTypedArray(t.Elem()), t.Len())
	case *types.Named:
		switch ut := t.Underlying().(type) {
		case *types.Struct:
			zeros := make([]string, ut.NumFields())
			for i := range zeros {
				zeros[i] = c.zeroValue(ut.Field(i).Type())
			}
			return fmt.Sprintf("new %s(%s)", c.typeName(t), strings.Join(zeros, ", "))
		case *types.Slice:
			return fmt.Sprintf("new %s(%s)", c.typeName(t), c.zeroValue(types.NewArray(ut.Elem(), 0)))
		case *types.Basic:
			return c.zeroValue(ut)
		}
	}
	return "null"
}

func (c *PkgContext) typeName(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		switch {
		case t.Kind() == types.Uint64:
			return "Go$Uint64"
		case t.Info()&types.IsInteger != 0:
			return "Integer"
		case t.Info()&types.IsFloat != 0:
			return "Float"
		case t.Info()&types.IsComplex != 0:
			return "Complex"
		case t.Info()&types.IsBoolean != 0:
			return "Boolean"
		case t.Info()&types.IsString != 0:
			return "String"
		case t.Kind() == types.UntypedNil:
			return "null"
		default:
			panic(fmt.Sprintf("Unhandled basic type: %v\n", t))
		}
	case *types.Named:
		objPkg := t.Obj().Pkg()
		if objPkg != nil && objPkg != c.pkg {
			return c.pkgVars[objPkg.Path()] + "." + t.Obj().Name()
		}
		return t.Obj().Name()
	case *types.Pointer:
		if _, isNamed := t.Elem().(*types.Named); isNamed {
			if _, isStruct := t.Elem().Underlying().(*types.Struct); !isStruct {
				return c.typeName(t.Elem()) + ".Go$Pointer"
			}
			return c.typeName(t.Elem())
		}
		return "Go$Pointer"
	case *types.Array:
		return "Array"
	case *types.Slice:
		return "Go$Slice"
	case *types.Map:
		return "Go$Map"
	case *types.Interface:
		return "Go$Interface"
	case *types.Chan:
		return "Go$Channel"
	case *types.Signature:
		return "Function"
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
}

func toJavaScriptType(t *types.Basic) string {
	switch t.Kind() {
	case types.Int8:
		return "Int8"
	case types.Uint8:
		return "Uint8"
	case types.Int16:
		return "Int16"
	case types.Uint16:
		return "Uint16"
	case types.Int32, types.Int64, types.Int:
		return "Int32"
	case types.Uint32, types.Uint64, types.Uint, types.Uintptr:
		return "Uint32"
	case types.Float32:
		return "Float32"
	case types.Float64, types.Complex64, types.Complex128:
		return "Float64"
	}
	panic(fmt.Sprintf("Unhandled basic type: %v\n", t))
}

func toTypedArray(t types.Type) string {
	if basic, isBasic := t.(*types.Basic); isBasic && basic.Info()&types.IsNumeric != 0 {
		return toJavaScriptType(basic) + "Array"
	}
	return "Array"
}

func createListComposite(elementType types.Type, elements []string) string {
	switch elt := elementType.(type) {
	case *types.Basic:
		if elt.Info()&types.IsNumeric != 0 {
			return fmt.Sprintf("new %s([%s])", toTypedArray(elt), strings.Join(elements, ", "))
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

func isUnderscore(expr ast.Expr) bool {
	if id, isIdent := expr.(*ast.Ident); isIdent {
		return id.Name == "_"
	}
	return false
}

func hasId(t types.Type) bool {
	_, isPointer := t.Underlying().(*types.Pointer)
	_, isInterface := t.Underlying().(*types.Interface)
	return isPointer || isInterface
}

type IsReadyVisitor struct {
	info           *types.Info
	functions      map[types.Object]*ast.FuncDecl
	pendingObjects map[types.Object]bool
	isReady        bool
}

func (v *IsReadyVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if !v.isReady {
		return nil
	}
	switch n := node.(type) {
	case *ast.Ident:
		o := v.info.Objects[n]
		if v.pendingObjects[o] {
			v.isReady = false
			return nil
		}
		if fun, found := v.functions[o]; found {
			delete(v.functions, o)
			ast.Walk(v, fun)
			v.functions[o] = fun
		}
	}
	return v
}
