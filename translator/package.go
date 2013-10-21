package translator

import (
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

var ReservedKeywords = []string{"arguments", "class", "delete", "eval", "export", "false", "function", "implements", "interface", "in", "let", "new", "package", "private", "protected", "public", "static", "this", "true", "try", "yield"}

type ErrorList []error

func (err ErrorList) Error() string {
	return err[0].Error()
}

type PkgContext struct {
	pkg          *types.Package
	info         *types.Info
	pkgVars      map[string]string
	objectVars   map[types.Object]string
	usedVarNames []string
	functionSig  *types.Signature
	resultNames  []ast.Expr
	postLoopStmt ast.Stmt
	output       []byte
	indentation  int
	delayedLines []byte
}

func (c *PkgContext) Write(b []byte) (int, error) {
	c.output = append(c.output, b...)
	return len(b), nil
}

func (c *PkgContext) Printf(format string, values ...interface{}) {
	c.Write([]byte(strings.Repeat("\t", c.indentation)))
	fmt.Fprintf(c, format, values...)
	c.Write([]byte{'\n'})
	c.Write(c.delayedLines)
	c.delayedLines = nil
}

func (c *PkgContext) Indent(f func()) {
	c.indentation += 1
	f()
	c.indentation -= 1
}

func (c *PkgContext) CatchOutput(f func()) []byte {
	origoutput := c.output
	c.output = nil
	f()
	catched := c.output
	c.output = origoutput
	return catched
}

func (c *PkgContext) Delayed(f func()) {
	c.delayedLines = c.CatchOutput(f)
}

func TranslatePackage(importPath string, files []*ast.File, fileSet *token.FileSet, config *types.Config) ([]byte, error) {
	info := &types.Info{
		Types:      make(map[ast.Expr]types.Type),
		Values:     make(map[ast.Expr]exact.Value),
		Objects:    make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	var errList ErrorList
	var previousErr error
	config.Error = func(err error) {
		if previousErr != nil && previousErr.Error() == err.Error() {
			return
		}
		errList = append(errList, err)
		previousErr = err
	}
	config.Sizes = sizes32
	typesPkg, err := config.Check(importPath, fileSet, files, info)
	if errList != nil {
		return nil, errList
	}
	if err != nil {
		return nil, err
	}
	config.Packages[importPath] = typesPkg

	c := &PkgContext{
		pkg:          typesPkg,
		info:         info,
		pkgVars:      make(map[string]string),
		objectVars:   make(map[types.Object]string),
		usedVarNames: ReservedKeywords,
	}

	functionsByType := make(map[types.Type][]*ast.FuncDecl)
	functionsByObject := make(map[types.Object]*ast.FuncDecl)
	var initStmts []ast.Stmt
	var typeSpecs []*ast.TypeSpec
	var constSpecs []*ast.ValueSpec
	var varSpecs []*ast.ValueSpec
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
				if sig.Recv() == nil && d.Name.Name == "init" {
					initStmts = append(initStmts, d.Body.List...)
					continue
				}
				functionsByType[recvType] = append(functionsByType[recvType], d)
				o := c.info.Objects[d.Name]
				functionsByObject[o] = d
				if sig.Recv() == nil {
					c.objectName(o) // register toplevel name
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					for _, spec := range d.Specs {
						s := spec.(*ast.TypeSpec)
						typeSpecs = append(typeSpecs, s)
						c.objectName(c.info.Objects[s.Name]) // register toplevel name
					}
				case token.CONST:
					for _, spec := range d.Specs {
						s := spec.(*ast.ValueSpec)
						constSpecs = append(constSpecs, s)
						for _, name := range s.Names {
							if !isUnderscore(name) {
								c.objectName(c.info.Objects[name]) // register toplevel name
							}
						}
					}
				case token.VAR:
					for _, spec := range d.Specs {
						s := spec.(*ast.ValueSpec)
						varSpecs = append(varSpecs, s)
						for _, name := range s.Names {
							if !isUnderscore(name) {
								c.objectName(c.info.Objects[name]) // register toplevel name
							}
						}
					}
				}
			}
		}
	}

	return c.CatchOutput(func() {
		c.Indent(func() {
			c.Printf("var Go$pkg = {};")

			for _, importedPkg := range typesPkg.Imports() {
				varName := c.newVariable(importedPkg.Name())
				c.Printf(`var %s = Go$packages["%s"];`, varName, importedPkg.Path())
				c.pkgVars[importedPkg.Path()] = varName
			}

			// types and their functions
			for _, spec := range typeSpecs {
				obj := c.info.Objects[spec.Name]
				typeName := c.objectName(obj)
				c.Printf("var %s;", typeName)
				c.translateSpec(spec)
				for _, fun := range functionsByType[obj.Type()] {
					funName := fun.Name.Name
					jsCode, _ := typesPkg.Scope().Lookup("js_" + typeName + "_" + funName).(*types.Const)
					if jsCode != nil {
						c.Printf("%s.prototype.%s = function(%s) {\n%s\n};", typeName, funName, strings.Join(c.translateParams(fun.Type), ", "), exact.StringVal(jsCode.Val()))
						continue
					}
					_, isStruct := obj.Type().Underlying().(*types.Struct)
					c.translateFunction(typeName, isStruct, fun)
				}
				c.Printf("Go$pkg.%s = %s;", typeName, typeName)
			}

			// package functions
			for _, fun := range functionsByType[nil] {
				name := fun.Name.Name
				jsCode, _ := typesPkg.Scope().Lookup("js_" + name).(*types.Const)
				if jsCode != nil {
					c.Printf("var %s = function(%s) {\n%s\n};", name, strings.Join(c.translateParams(fun.Type), ", "), exact.StringVal(jsCode.Val()))
					continue
				}
				if fun.Body == nil {
					c.Printf(`var %s = function() { throw new Go$Panic("Native function not implemented: %s"); };`, name, name)
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
				c.Printf("var %s = %s;", c.translateExpr(fun.Name), c.translateExpr(funcLit))
			}

			// constants
			for _, spec := range constSpecs {
				for _, name := range spec.Names {
					if isUnderscore(name) || strings.HasPrefix(name.Name, "js_") {
						continue
					}
					o := c.info.Objects[name].(*types.Const)
					c.info.Types[name] = o.Type()
					c.info.Values[name] = o.Val()
					c.Printf("%s = %s;", c.objectName(o), c.translateExpr(name))
				}
			}

			// variables
			var unorderedSingleVarSpecs []*ast.ValueSpec
			pendingObjects := make(map[types.Object]bool)
			for _, spec := range varSpecs {
				for i, name := range spec.Names {
					o := c.info.Objects[name].(*types.Var)
					c.Printf("%s = %s;", c.objectName(o), c.zeroValue(o.Type()))
					if i < len(spec.Values) {
						unorderedSingleVarSpecs = append(unorderedSingleVarSpecs, &ast.ValueSpec{
							Names:  []*ast.Ident{name},
							Type:   spec.Type,
							Values: []ast.Expr{spec.Values[i]},
						})
						pendingObjects[c.info.Objects[name]] = true
					}
				}
			}

			var orderedVarStmts []ast.Stmt
			complete := false
			for !complete {
				complete = true
				for i, spec := range unorderedSingleVarSpecs {
					if spec == nil {
						continue
					}
					v := IsReadyVisitor{info: c.info, functions: functionsByObject, pendingObjects: pendingObjects, isReady: true}
					ast.Walk(&v, spec.Values[0])
					if !v.isReady {
						complete = false
						continue
					}
					orderedVarStmts = append(orderedVarStmts, &ast.AssignStmt{
						Lhs: []ast.Expr{spec.Names[0]},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{spec.Values[0]},
					})
					delete(pendingObjects, c.info.Objects[spec.Names[0]])
					unorderedSingleVarSpecs[i] = nil
				}
			}

			// native implementations
			if native, hasNative := natives[importPath]; hasNative {
				c.Write([]byte(strings.TrimSpace(native)))
				c.Write([]byte{'\n'})
			}

			// exports for package functions
			for _, fun := range functionsByType[nil] {
				name := fun.Name.Name
				if fun.Name.IsExported() || name == "main" {
					c.Printf("Go$pkg.%s = %s;", name, name)
				}
			}

			// init function
			funcLit := &ast.FuncLit{
				Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{}},
				Body: &ast.BlockStmt{
					List: append(orderedVarStmts, initStmts...),
				},
			}
			c.info.Types[funcLit] = types.NewSignature(c.pkg.Scope(), nil, types.NewTuple(), types.NewTuple(), false)
			c.Printf("Go$pkg.init = %s;", c.translateExpr(funcLit))

			c.Printf("return Go$pkg;")
		})
	}), nil
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
		obj := c.info.Objects[s.Name]
		typeName := c.objectName(obj)
		if isWrapped(obj.Type()) {
			c.Printf(`var %s = function(v) { this.v = v; };`, typeName)
			c.Printf(`%s.prototype.Go$key = function() { return "%s$" + this.v; };`, typeName, typeName)
			c.Printf("%s.Go$Pointer = function(getter, setter) { this.Go$get = getter; this.Go$set = setter; };", typeName)
			return
		}
		switch t := obj.Type().Underlying().(type) {
		case *types.Struct:
			params := make([]string, t.NumFields())
			for i := 0; i < t.NumFields(); i++ {
				params[i] = t.Field(i).Name() + "_"
			}
			c.Printf("%s = function(%s) {", typeName, strings.Join(params, ", "))
			c.Indent(func() {
				c.Printf("this.Go$id = Go$idCounter++;")
				for i := 0; i < t.NumFields(); i++ {
					field := t.Field(i)
					c.Printf("this.%s = %s_;", field.Name(), field.Name())
				}
			})
			c.Printf("};")
			c.Printf(`%s.prototype.Go$key = function() { return this.Go$id; };`, typeName)
			c.Printf("%s.Go$NonPointer = function(v) { this.v = v; };", typeName)
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
						paramList := strings.Join(params, ", ")
						c.Printf("%s.prototype.%s = function(%s) { return %s.%s(%s); };", typeName, name, paramList, value, name, paramList)
						c.Printf("%s.Go$NonPointer.prototype.%s = function(%s) { return this.v.%s(%s); };", typeName, name, paramList, name, paramList)
					}
				}
			}
		case *types.Interface:
			c.Printf("%s = { Go$implementedBy: [] };", typeName)
		default:
			underlyingTypeName := c.typeName(t)
			c.Printf("%s = function() { %s.apply(this, arguments); };", typeName, underlyingTypeName)
			c.Printf("%s.prototype.Go$key = function() { return \"%s$\" + %s.prototype.Go$key.apply(this); };", typeName, typeName, underlyingTypeName)
			c.Printf("%s.Go$Pointer = function(getter, setter) { this.Go$get = getter; this.Go$set = setter; };", typeName)
			if _, isSlice := t.(*types.Slice); isSlice {
				c.Printf("%s.Go$nil = new %s({ isNil: true, length: 0 });", typeName, typeName)
			}
		}

	case *ast.ImportSpec:
		// ignored

	default:
		panic(fmt.Sprintf("Unhandled spec: %T\n", s))

	}
}

func (c *PkgContext) translateFunction(typeName string, isStruct bool, fun *ast.FuncDecl) {
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
		c.info.Types[recv] = recvType
		c.info.Types[this] = recvType
		body = append([]ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{recv},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{this},
			},
		}, body...)
	}

	funcLit := &ast.FuncLit{
		Type: fun.Type,
		Body: &ast.BlockStmt{
			List: body,
		},
	}
	c.info.Types[funcLit] = c.info.Objects[fun.Name].Type()

	params := strings.Join(c.translateParams(fun.Type), ", ")
	switch {
	case isStruct:
		c.Printf("%s.prototype.%s = %s;", typeName, fun.Name.Name, c.translateExpr(funcLit))
		c.Printf("%s.Go$NonPointer.prototype.%s = function(%s) { return this.v.%s(%s); };", typeName, fun.Name.Name, params, fun.Name.Name, params)
	case !isStruct && !isPointer:
		value := "this.Go$get()"
		if isWrapped(recvType) {
			value = fmt.Sprintf("new %s(%s)", typeName, value)
		}
		c.Printf("%s.prototype.%s = %s;", typeName, fun.Name.Name, c.translateExpr(funcLit))
		c.Printf("%s.Go$Pointer.prototype.%s = function(%s) { return %s.%s(%s); };", typeName, fun.Name.Name, params, value, fun.Name.Name, params)
	case !isStruct && isPointer:
		value := "this"
		if isWrapped(ptr.Elem()) {
			value = "this.v"
		}
		c.Printf("%s.prototype.%s = function(%s) { var obj = %s; return (new %s.Go$Pointer(function() { return obj; }, null)).%s(%s); };", typeName, fun.Name.Name, params, value, typeName, fun.Name.Name, params)
		c.Printf("%s.Go$Pointer.prototype.%s = %s;", typeName, fun.Name.Name, c.translateExpr(funcLit))
	}
}

func (c *PkgContext) translateParams(t *ast.FuncType) []string {
	n := c.usedVarNames
	params := make([]string, 0)
	for _, param := range t.Params.List {
		for _, ident := range param.Names {
			if isUnderscore(ident) {
				params = append(params, c.newVariable("param"))
				continue
			}
			params = append(params, c.objectName(c.info.Objects[ident]))
		}
	}
	c.usedVarNames = n
	return params
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
		argType := funType.Params().At(i).Type()
		args[i] = c.translateExprToType(call.Args[i], argType)
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
		if t.Kind() == types.UntypedNil {
			panic("Zero value for untyped nil.")
		}
	case *types.Array:
		return fmt.Sprintf("Go$clear(new %s(%d), %s)", toArrayType(t.Elem()), t.Len(), c.zeroValue(t.Elem()))
	case *types.Slice:
		return fmt.Sprintf("%s.Go$nil", c.typeName(ty))
	case *types.Struct:
		fields := make([]string, t.NumFields())
		for i := range fields {
			field := t.Field(i)
			if !isNamed {
				fields[i] = field.Name() + ": "
			}
			fields[i] += c.zeroValue(field.Type())
		}
		if isNamed {
			return fmt.Sprintf("new %s(%s)", c.objectName(named.Obj()), strings.Join(fields, ", "))
		}
		return fmt.Sprintf("{%s}", strings.Join(fields, ", "))
	}
	return "null"
}

func (c *PkgContext) newVariable(prefix string) string {
	n := 0
	for {
		name := prefix
		for _, b := range []byte(name) {
			if b < '0' || b > 'z' {
				name = "nonAasciiName"
				break
			}
		}
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

func (c *PkgContext) objectName(o types.Object) string {
	if o.Name() == "error" {
		return "Go$error"
	}
	if o.Pkg() != nil && o.Pkg() != c.pkg {
		return c.pkgVars[o.Pkg().Path()] + "." + o.Name()
	}

	name, found := c.objectVars[o]
	if !found {
		name = c.newVariable(o.Name())
		c.objectVars[o] = name
	}

	switch o.(type) {
	case *types.Var, *types.Const:
		if o.Parent() == c.pkg.Scope() {
			return "Go$pkg." + name
		}
	}
	return name
}

func (c *PkgContext) typeName(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		if t.Kind() == types.UntypedNil {
			return "null"
		}
		return "Go$" + toJavaScriptType(t)
	case *types.Named:
		if _, isStruct := t.Underlying().(*types.Struct); isStruct {
			return c.objectName(t.Obj()) + ".Go$NonPointer"
		}
		return c.objectName(t.Obj())
	case *types.Pointer:
		if named, isNamed := t.Elem().(*types.Named); isNamed && named.Obj().Name() != "error" {
			switch t.Elem().Underlying().(type) {
			case *types.Struct:
				return c.objectName(named.Obj())
			case *types.Interface:
				return "Go$Pointer"
			default:
				return c.objectName(named.Obj()) + ".Go$Pointer"
			}
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

func elemType(ty types.Type) types.Type {
	switch t := ty.Underlying().(type) {
	case *types.Array:
		return t.Elem()
	case *types.Slice:
		return t.Elem()
	default:
		panic("")
	}
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
