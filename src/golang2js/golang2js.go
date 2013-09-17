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
	namedResults []string
	writer       io.Writer
	indentation  int
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
	c.Write([]byte(strings.Repeat("  ", c.indentation)))
	fmt.Fprintf(c, format, values...)
	c.Write([]byte{'\n'})
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

	prelude, err := os.Open("prelude.js")
	if err != nil {
		panic(err)
	}
	io.Copy(out, prelude)
	prelude.Close()

	t := &Translator{
		writer:   out,
		packages: make(map[string]*PkgContext),
	}
	t.packages["math"] = nil
	t.packages["reflect"] = nil
	t.packages["runtime"] = nil
	t.packages["syscall"] = nil
	t.packages["sync"] = nil
	t.packages["sync/atomic"] = nil
	t.packages["time"] = nil
	t.translatePackage(fileSet, pkg)
}

type This struct {
	ast.Ident
}

func (t *Translator) translatePackage(fileSet *token.FileSet, pkg *build.Package) {
	// os.Stderr.WriteString(pkg.Name + "\n")

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
				fmt.Println(entry)
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

	// for _, importedPkg := range typesPkg.Imports() {
	// 	if _, found := t.packages[importedPkg.Path()]; found {
	// 		continue
	// 	}

	// 	otherPkg, err := build.Import(importedPkg.Path(), pkg.Dir, 0)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	t.translatePackage(fileSet, otherPkg)
	// }

	c := &PkgContext{
		pkg:          typesPkg,
		info:         info,
		pkgVars:      make(map[string]string),
		objectVars:   make(map[types.Object]string),
		usedVarNames: []string{"delete", "false", "new", "true", "try", "packages", "Array", "Boolean", "Channel", "Float", "Integer", "Map", "Slice", "String"},
		writer:       t.writer,
	}
	t.packages[pkg.ImportPath] = c

	functions := make(map[types.Type][]*ast.FuncDecl)
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
				functions[recvType] = append(functions[recvType], fun)
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
							c.Printf("%s._Pointer = function(getter, setter) { this.get = getter; this.set = setter; };", recvType.Obj().Name())
						}
						for _, fun := range functions[recvType] {
							c.translateFunction(fun, hasPtrType)
						}
					}
				}
			}
		}

		// package functions
		hasInit := false
		for _, fun := range functions[nil] {
			if fun.Name.Name == "init" {
				hasInit = true
			}
			if fun.Body == nil {
				c.Printf(`var %s = function() { throw new Error("Native function not implemented: %s"); };`, fun.Name, fun.Name)
				continue
			}
			c.translateStmt(&ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{fun.Name},
				Rhs: []ast.Expr{&ast.FuncLit{
					Type: fun.Type,
					Body: &ast.BlockStmt{
						List: fun.Body.List,
					},
				}},
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
							if len(s.Values) != 0 {
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
					v := IsReadyVisitor{info: c.info, pendingObjects: pendingObjects, isReady: true}
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
		for i, name := range s.Names {
			fieldType := c.info.Objects[name].Type()
			var value string
			switch {
			case i < len(s.Values):
				c.info.Types[s.Values[i]] = fieldType
				value = c.translateExpr(s.Values[i])
			default:
				value = c.zeroValue(fieldType)
			}
			if isUnderscore(name) {
				continue
			}
			c.Printf("var %s = %s;", c.translateExpr(name), value)
		}

	case *ast.TypeSpec:
		nt := c.info.Objects[s.Name].Type().(*types.Named)
		switch t := nt.Underlying().(type) {
		case *types.Basic:
			c.Printf("var %s = function(v) { this.v = v; };", nt.Obj().Name())
			if t.Info()&types.IsString != 0 {
				c.Printf("%s.prototype.len = function() { return this.v.length; };", nt.Obj().Name())
			}
		case *types.Struct:
			params := make([]string, t.NumFields())
			for i := 0; i < t.NumFields(); i++ {
				params[i] = t.Field(i).Name() + "_"
			}
			c.Printf("var %s = function(%s) {", nt.Obj().Name(), strings.Join(params, ", "))
			c.Indent(func() {
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
						c.Printf("%s.prototype.%s = function(%s) { return this.%s.%s(%s); };", nt.Obj().Name(), name, strings.Join(params, ", "), field.Name(), name, strings.Join(params, ", "))
					}
				}
			}
		case *types.Slice:
			c.Printf("var %s = function() { Slice.apply(this, arguments); };", nt.Obj().Name())
			c.Printf("%s.prototype = Slice.prototype;", nt.Obj().Name())
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
	recv := fun.Recv.List[0]
	recvType := c.info.Objects[recv.Names[0]].Type()
	_, recvIsPtr := recvType.(*types.Pointer)

	typeTarget := recv.Type
	if recvIsPtr && hasPtrType {
		typeTarget = &ast.SelectorExpr{
			X:   recv.Type,
			Sel: ast.NewIdent("_Pointer"),
		}
	}

	var this ast.Expr = &This{}
	if _, isUnderlyingStruct := recvType.Underlying().(*types.Struct); isUnderlyingStruct {
		this = &ast.StarExpr{X: this}
	}
	c.info.Types[this] = recvType

	c.translateStmt(&ast.AssignStmt{
		Tok: token.ASSIGN,
		Lhs: []ast.Expr{&ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   typeTarget,
				Sel: ast.NewIdent("prototype"),
			},
			Sel: fun.Name,
		}},
		Rhs: []ast.Expr{&ast.FuncLit{
			Type: fun.Type,
			Body: &ast.BlockStmt{
				List: append([]ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{recv.Names[0]},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{this},
					},
				}, fun.Body.List...),
			},
		}},
	}, "")

	if hasPtrType {
		typeName := c.translateExpr(recv.Type)
		params := c.translateParams(fun.Type)
		if !recvIsPtr {
			c.Printf("%s._Pointer.prototype.%s = function(%s) { return this.get().%s(%s); };", typeName, fun.Name.Name, params, fun.Name.Name, params)
		}
		if recvIsPtr {
			c.Printf("%s.prototype.%s = function(%s) { var obj = this; return (new %s._Pointer(function() { return obj; }, null)).%s(%s); };", typeName, fun.Name.Name, params, typeName, fun.Name.Name, params)
		}
	}
}

func (c *PkgContext) savedAsPointer(expr ast.Expr) bool {
	t := c.info.Types[expr].Underlying()
	if ptr, isPtr := t.(*types.Pointer); isPtr {
		_, isStruct := ptr.Elem().Underlying().(*types.Struct)
		return !isStruct
	}
	return false
}

func (c *PkgContext) translateParams(t *ast.FuncType) string {
	params := make([]string, 0)
	for _, param := range t.Params.List {
		for _, ident := range param.Names {
			params = append(params, c.translateExpr(ident))
		}
	}
	return strings.Join(params, ", ")
}

func (c *PkgContext) translateArgs(call *ast.CallExpr) []string {
	funType := c.info.Types[call.Fun]
	args := make([]string, len(call.Args))
	for i, arg := range call.Args {
		args[i] = c.translateExpr(arg)
		sig, isSig := funType.(*types.Signature)
		if isSig && c.savedAsPointer(arg) {
			if _, isInt := sig.Params().At(i).Type().Underlying().(*types.Interface); isInt {
				args[i] += ".get()"
			}
		}
	}
	isVariadic, numParams, variadicType := getVariadicInfo(funType)
	if isVariadic && !call.Ellipsis.IsValid() {
		args = append(args[:numParams-1], fmt.Sprintf("new Slice(%s)", createListComposite(variadicType, args[numParams-1:])))
	}
	if call.Ellipsis.IsValid() && len(call.Args) > 0 {
		l := len(call.Args)
		if t, isBasic := c.info.Types[call.Args[l-1]].(*types.Basic); isBasic && t.Info()&types.IsString != 0 {
			args[l-1] = fmt.Sprintf("%s.toSlice()", args[l-1])
		}
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
		switch t.Elem().(type) {
		case *types.Basic:
			return fmt.Sprintf("newNumericArray(%d)", t.Len())
			// return fmt.Sprintf("new %s(%d)", toTypedArray(elt), t.Len())
		default:
			return fmt.Sprintf("new Array(%d)", t.Len())
		}
	case *types.Named:
		if s, isStruct := t.Underlying().(*types.Struct); isStruct {
			zeros := make([]string, s.NumFields())
			for i := range zeros {
				zeros[i] = c.zeroValue(s.Field(i).Type())
			}
			return fmt.Sprintf("new %s(%s)", c.TypeName(t), strings.Join(zeros, ", "))
		}
		return fmt.Sprintf("new %s(%s)", c.TypeName(t), c.zeroValue(t.Underlying()))
	}
	return "null"
}

func (c *PkgContext) TypeName(t *types.Named) string {
	name := t.Obj().Name()
	objPkg := t.Obj().Pkg()
	if objPkg != nil && objPkg != c.pkg {
		name = c.pkgVars[objPkg.Path()] + "." + name
	}
	return name
}

// func toTypedArray(t *types.Basic) string {
// 	switch t.Kind() {
// 	case types.Int8:
// 		return "Int8Array"
// 	case types.Uint8:
// 		return "Uint8Array"
// 	case types.Int16:
// 		return "Int16Array"
// 	case types.Uint16:
// 		return "Uint16Array"
// 	case types.Int32, types.Int:
// 		return "Int32Array"
// 	case types.Uint32:
// 		return "Uint32Array"
// 	case types.Float32:
// 		return "Float32Array"
// 	case types.Float64, types.Complex64, types.Complex128:
// 		return "Float64Array"
// 	default:
// 		panic("Unhandled typed array: " + t.String())
// 	}
// 	return ""
// }

func createListComposite(elementType types.Type, elements []string) string {
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	// switch elt := elementType.(type) {
	// case *types.Basic:
	// 	switch elt.Kind() {
	// 	case types.Bool, types.String:
	// 		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	// 	default:
	// 		return fmt.Sprintf("new %s([%s])", toTypedArray(elt), strings.Join(elements, ", "))
	// 	}
	// default:
	// 	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	// 	// panic(fmt.Sprintf("Unhandled element type: %T\n", elt))
	// }
}

func getVariadicInfo(funType types.Type) (bool, int, types.Type) {
	switch t := funType.(type) {
	case *types.Signature:
		if t.IsVariadic() {
			return true, t.Params().Len(), t.Params().At(t.Params().Len() - 1).Type()
		}
	case *types.Builtin:
		switch t.Name() {
		case "append":
			return true, 2, types.NewInterface(nil)
		case "print", "println":
			return true, 1, types.NewInterface(nil)
		}
	}
	return false, 0, nil
}

func isUnderscore(expr ast.Expr) bool {
	if id, isIdent := expr.(*ast.Ident); isIdent {
		return id.Name == "_"
	}
	return false
}

type IsReadyVisitor struct {
	info           *types.Info
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
	case *This:
		return nil
	}
	return v
}
