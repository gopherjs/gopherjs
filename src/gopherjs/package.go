package main

import (
	"bytes"
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"sort"
	"strings"
)

var ReservedKeywords = []string{"arguments", "class", "delete", "eval", "export", "false", "implements", "interface", "in", "let", "new", "package", "private", "protected", "public", "static", "this", "true", "try", "yield", "packages"}

type PkgContext struct {
	pkg          *types.Package
	info         *types.Info
	pkgVars      map[string]string
	objectVars   map[types.Object]string
	usedVarNames []string
	functionSig  *types.Signature
	resultNames  []ast.Expr
	postLoopStmt ast.Stmt
	buffer       *bytes.Buffer
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

func (c *PkgContext) nameForObject(o types.Object) string {
	name, found := c.objectVars[o]
	if !found {
		name = c.newVarName(o.Name())
		c.objectVars[o] = name
	}
	return name
}

func (c *PkgContext) Write(b []byte) (int, error) {
	return c.buffer.Write(b)
}

func (c *PkgContext) Printf(format string, values ...interface{}) {
	c.Write([]byte(strings.Repeat("\t", c.indentation)))
	fmt.Fprintf(c, format, values...)
	c.Write([]byte{'\n'})
	if c.delayedLines != nil {
		c.delayedLines.WriteTo(c.buffer)
		c.delayedLines = nil
	}
}

func (c *PkgContext) Indent(f func()) {
	c.indentation += 1
	f()
	c.indentation -= 1
}

func (c *PkgContext) CatchOutput(f func()) *bytes.Buffer {
	origbuffer := c.buffer
	b := bytes.NewBuffer(nil)
	c.buffer = b
	f()
	c.buffer = origbuffer
	return b
}

func (c *PkgContext) Delayed(f func()) {
	c.delayedLines = c.CatchOutput(f)
}

func (pkg *GopherPackage) translate(fileSet *token.FileSet) (buffer *bytes.Buffer, translateErr error) {
	var previousErr string
	config := &types.Config{
		Error: func(err error) {
			if err.Error() != previousErr {
				fmt.Println(err.Error())
			}
			previousErr = err.Error()
		},
	}

	files := make([]*ast.File, 0)
	for _, name := range pkg.GoFiles {
		fullName := pkg.Dir + "/" + name
		file, err := parser.ParseFile(fileSet, fullName, nil, 0)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
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
		return nil, err
	}

	c := &PkgContext{
		pkg:        typesPkg,
		info:       info,
		pkgVars:    make(map[string]string),
		objectVars: make(map[types.Object]string),

		usedVarNames: ReservedKeywords,
	}

	functionsByType := make(map[types.Type][]*ast.FuncDecl)
	functionsByObject := make(map[types.Object]*ast.FuncDecl)
	var typeSpecs []*ast.TypeSpec
	var valueSpecs []*ast.ValueSpec
	for _, file := range files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				sig := c.info.Objects[d.Name].(*types.Func).Type().(*types.Signature)
				var recvType types.Type
				if sig.Recv() != nil {
					recvType = sig.Recv().Type()
					if ptr, isPtr := recvType.(*types.Pointer); isPtr {
						recvType = ptr.Elem()
					}
				}
				functionsByType[recvType] = append(functionsByType[recvType], d)
				o := c.info.Objects[d.Name]
				functionsByObject[o] = d
				if sig.Recv() == nil {
					c.nameForObject(o) // register toplevel name
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					for _, spec := range d.Specs {
						s := spec.(*ast.TypeSpec)
						typeSpecs = append(typeSpecs, s)
						c.nameForObject(c.info.Objects[s.Name]) // register toplevel name
					}
				case token.CONST, token.VAR:
					for _, spec := range d.Specs {
						s := spec.(*ast.ValueSpec)
						valueSpecs = append(valueSpecs, s)
						for _, name := range s.Names {
							if !isUnderscore(name) {
								c.nameForObject(c.info.Objects[name]) // register toplevel name
							}
						}
					}
				}
			}
		}
	}

	buffer = c.CatchOutput(func() {
		if pkg.IsCommand() {
			c.Write([]byte(strings.TrimSpace(prelude)))
			c.Write([]byte("\n"))

			loaded := make(map[*GopherPackage]bool)
			var loadImports func(*GopherPackage) error
			loadImports = func(pkg *GopherPackage) error {
				for _, imp := range pkg.importedPackages {
					if _, alreadyLoaded := loaded[imp]; alreadyLoaded {
						continue
					}
					loaded[imp] = true

					if err := loadImports(imp); err != nil {
						return err
					}
					if imp.archiveFile == "" {
						continue
					}

					depFile, err := os.Open(imp.archiveFile)
					if err != nil {
						return err
					}
					io.Copy(c, depFile)
					depFile.Close()
				}
				return nil
			}
			if err := loadImports(pkg); err != nil {
				translateErr = err
				return
			}
		}

		c.Printf(`packages["%s"] = (function() {`, pkg.ImportPath)
		c.Indent(func() {
			c.Printf("var %s;", strings.Join(c.usedVarNames[len(ReservedKeywords):], ", "))

			for _, importedPkg := range typesPkg.Imports() {
				varName := c.newVarName(importedPkg.Name())
				c.Printf(`var %s = packages["%s"];`, varName, importedPkg.Path())
				c.pkgVars[importedPkg.Path()] = varName
			}

			// types and their functions
			for _, spec := range typeSpecs {
				recvType := c.info.Objects[spec.Name].Type().(*types.Named)
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

			// package functions
			hasInit := false
			for _, fun := range functionsByType[nil] {
				if fun.Name.Name == "init" {
					hasInit = true
				}
				if fun.Body == nil {
					c.Printf(`var %s = function() { throw new Go$Panic("Native function not implemented: %s"); };`, fun.Name.Name, fun.Name.Name)
					continue
				}
				funcLit := &ast.FuncLit{
					Type: fun.Type,
					Body: &ast.BlockStmt{
						List: fun.Body.List,
					},
				}
				funType := c.info.Objects[fun.Name].Type()
				c.info.Types[fun.Name] = funType
				c.info.Types[funcLit] = funType
				c.translateStmt(&ast.AssignStmt{
					Tok: token.DEFINE,
					Lhs: []ast.Expr{fun.Name},
					Rhs: []ast.Expr{funcLit},
				}, "")
			}

			// constants and variables in dependency aware order
			var specs []*ast.ValueSpec
			pendingObjects := make(map[types.Object]bool)
			for _, spec := range valueSpecs {
				for i, name := range spec.Names {
					var values []ast.Expr
					switch o := c.info.Objects[name].(type) {
					case *types.Var:
						if i < len(spec.Values) {
							values = []ast.Expr{spec.Values[i]}
						}
					case *types.Const:
						id := ast.NewIdent("")
						c.info.Types[id] = o.Type()
						c.info.Values[id] = o.Val()
						values = []ast.Expr{id}
					default:
						panic("")
					}
					specs = append(specs, &ast.ValueSpec{
						Names:  []*ast.Ident{name},
						Type:   spec.Type,
						Values: values,
					})
					pendingObjects[c.info.Objects[spec.Names[0]]] = true
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
	})
	return
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
		if isWrapped(nt) {
			c.Printf(`var %s = function(v) { this.v = v; };`, c.typeName(nt))
			c.Printf(`%s.prototype.Go$key = function() { return "%s$" + this.v; };`, c.typeName(nt), c.typeName(nt))
			return
		}
		switch t := nt.Underlying().(type) {
		case *types.Struct:
			params := make([]string, t.NumFields())
			for i := 0; i < t.NumFields(); i++ {
				params[i] = t.Field(i).Name() + "_"
			}
			c.Printf("%s = function(%s) {", c.typeName(nt), strings.Join(params, ", "))
			c.Indent(func() {
				c.Printf("this.Go$id = Go$idCounter++;")
				for i := 0; i < t.NumFields(); i++ {
					field := t.Field(i)
					c.Printf("this.%s = %s_ || %s;", field.Name(), field.Name(), c.zeroValue(field.Type()))
				}
			})
			c.Printf("};")
			c.Printf(`%s.name = "%s";`, c.typeName(nt), c.typeName(nt))
			c.Printf(`%s.prototype.Go$key = function() { return this.Go$id; };`, c.typeName(nt))
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
						value := "this." + field.Name()
						if isWrapped(field.Type()) {
							value = fmt.Sprintf("new %s(%s)", field.Name(), value)
						}
						c.Printf("%s.prototype.%s = function(%s) { return %s.%s(%s); };", c.typeName(nt), name, strings.Join(params, ", "), value, name, strings.Join(params, ", "))
					}
				}
			}
		case *types.Interface:
			if t.MethodSet().Len() == 0 {
				c.Printf("%s = function(t) { return true };", c.typeName(nt))
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
			c.Printf("%s = function(t) { return %s };", c.typeName(nt), strings.Join(conditions, " || "))
		default:
			typeName := c.typeName(t)
			c.Printf("%s = function() { %s.apply(this, arguments); };", c.typeName(nt), typeName)
			c.Printf("Go$copyFields(%s.prototype, %s.prototype);", typeName, c.typeName(nt))
		}

	case *ast.ImportSpec:
		// ignored

	default:
		panic(fmt.Sprintf("Unhandled spec: %T\n", s))

	}
}

func (c *PkgContext) translateFunction(fun *ast.FuncDecl, hasPtrType bool) {
	sig := c.info.Objects[fun.Name].(*types.Func).Type().(*types.Signature)

	recvType := sig.Recv().Type()
	ptr, isPointer := recvType.(*types.Pointer)

	body := fun.Body.List
	if fun.Recv.List[0].Names != nil {
		recv := fun.Recv.List[0].Names[0]
		var this ast.Expr = ast.NewIdent("this")
		if isWrapped(recvType) {
			this = ast.NewIdent("this.v")
		}
		if _, isUnderlyingStruct := recvType.Underlying().(*types.Struct); isUnderlyingStruct {
			this = &ast.StarExpr{X: this}
		}
		c.info.Types[this] = recvType
		body = append([]ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{recv},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{this},
			},
		}, body...)
	}

	lhs := ast.NewIdent(c.typeName(recvType) + ".prototype." + fun.Name.Name)
	c.info.Types[lhs] = c.info.Objects[fun.Name].Type()
	funcLit := &ast.FuncLit{
		Type: fun.Type,
		Body: &ast.BlockStmt{
			List: body,
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
			if isWrapped(recvType) {
				value = fmt.Sprintf("new %s(%s)", typeName, value)
			}
			c.Printf("%s.Go$Pointer.prototype.%s = function(%s) { return %s.%s(%s); };", typeName, fun.Name.Name, params, value, fun.Name.Name, params)
		}
		if isPointer {
			typeName := c.typeName(ptr.Elem())
			value := "this"
			if isWrapped(ptr.Elem()) {
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
		if funType.IsVariadic() && i == len(args)-1 && !call.Ellipsis.IsValid() {
			varargType := funType.Params().At(i).Type().(*types.Slice).Elem()
			varargs := make([]string, len(call.Args)-i)
			for i, vararg := range call.Args[i:] {
				varargs[i] = c.translateExprToType(vararg, varargType)
			}
			args[i] = fmt.Sprintf("new Go$Slice(%s)", createListComposite(varargType, varargs))
			break
		}
		args[i] = c.translateExprToType(call.Args[i], funType.Params().At(i).Type())
	}
	return args
}

func (c *PkgContext) zeroValue(ty types.Type) string {
	named, isNamed := ty.(*types.Named)
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		if is64Bit(t) {
			return fmt.Sprintf("new %s(0, 0)", c.typeName(ty))
		}
		if t.Info()&types.IsBoolean != 0 {
			return "false"
		}
		if t.Info()&types.IsNumeric != 0 {
			return "0"
		}
		if t.Info()&types.IsString != 0 {
			return `""`
		}
	case *types.Array:
		return fmt.Sprintf("Go$clear(new %s(%d))", toArrayType(t.Elem()), t.Len())
	case *types.Slice:
		if isNamed {
			return fmt.Sprintf("new %s(%s)", c.typeName(named), c.zeroValue(types.NewArray(t.Elem(), 0)))
		}
	case *types.Struct:
		if isNamed {
			return fmt.Sprintf("new %s()", c.typeName(named))
		}
		fields := make([]string, t.NumFields())
		for i := range fields {
			field := t.Field(i)
			fields[i] = field.Name() + ": " + c.zeroValue(field.Type())
		}
		return fmt.Sprintf("{%s}", strings.Join(fields, ", "))
	}
	return "null"
}

func (c *PkgContext) typeName(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		if t.Kind() == types.UntypedNil {
			return "null"
		}
		return "Go$" + toJavaScriptType(t)
	case *types.Named:
		objPkg := t.Obj().Pkg()
		if objPkg != nil && objPkg != c.pkg {
			return c.pkgVars[objPkg.Path()] + "." + t.Obj().Name()
		}
		return c.nameForObject(t.Obj())
	case *types.Pointer:
		if named, isNamed := t.Elem().(*types.Named); isNamed && named.Obj().Name() != "error" {
			if _, isStruct := t.Elem().Underlying().(*types.Struct); !isStruct {
				return c.typeName(t.Elem()) + ".Go$Pointer"
			}
			return c.typeName(t.Elem())
		}
		return "Go$Pointer"
	case *types.Array:
		return "Go$Array"
	case *types.Slice:
		return "Go$Slice"
	case *types.Map:
		return "Go$Map"
	case *types.Interface:
		return "Go$Interface"
	case *types.Chan:
		return "Go$Channel"
	case *types.Signature:
		return "Go$Func"
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
}

func toJavaScriptType(t *types.Basic) string {
	switch t.Kind() {
	case types.UntypedInt:
		return "Int"
	default:
		name := t.String()
		return strings.ToUpper(name[:1]) + name[1:]
	}
}

func is64Bit(t *types.Basic) bool {
	return t.Kind() == types.Int64 || t.Kind() == types.Uint64
}

func isComplex(t *types.Basic) bool {
	return t.Kind() == types.Complex64 || t.Kind() == types.Complex128
}

func isTypedArray(t types.Type) bool {
	basic, isBasic := t.(*types.Basic)
	return isBasic && basic.Info()&types.IsNumeric != 0 && !is64Bit(basic) && !isComplex(basic)
}

func toArrayType(t types.Type) string {
	if isTypedArray(t) {
		return "Go$" + toJavaScriptType(t.(*types.Basic)) + "Array"
	}
	return "Go$Array"
}

func createListComposite(elementType types.Type, elements []string) string {
	if isTypedArray(elementType) {
		return fmt.Sprintf("new %s([%s])", toArrayType(elementType), strings.Join(elements, ", "))
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

func isUnderscore(expr ast.Expr) bool {
	if id, isIdent := expr.(*ast.Ident); isIdent {
		return id.Name == "_"
	}
	return false
}

func hasId(ty types.Type) bool {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		return is64Bit(t)
	case *types.Pointer, *types.Interface:
		return true
	}
	return false
}

func isWrapped(ty types.Type) bool {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		return !is64Bit(t) && t.Kind() != types.UntypedNil
	case *types.Array, *types.Signature:
		return true
	}
	return false
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
