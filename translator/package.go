package translator

import (
	"bytes"
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

var ReservedKeywords = make(map[string]bool)

func init() {
	for _, keyword := range []string{"abstract", "arguments", "boolean", "break", "byte", "case", "catch", "char", "class", "const", "continue", "debugger", "default", "delete", "do", "double", "else", "enum", "eval", "export", "extends", "false", "final", "finally", "float", "for", "function", "goto", "if", "implements", "import", "in", "instanceof", "int", "interface", "let", "long", "native", "new", "package", "private", "protected", "public", "return", "short", "static", "super", "switch", "synchronized", "this", "throw", "throws", "transient", "true", "try", "typeof", "var", "void", "volatile", "while", "with", "yield"} {
		ReservedKeywords[keyword] = true
	}
}

type ErrorList []error

func (err ErrorList) Error() string {
	return err[0].Error()
}

type PkgContext struct {
	pkg           *types.Package
	info          *types.Info
	pkgVars       map[string]string
	objectVars    map[types.Object]string
	allVarNames   map[string]int
	funcVarNames  []string
	functionSig   *types.Signature
	resultNames   []ast.Expr
	postLoopStmt  map[string]ast.Stmt
	escapingVars  []string
	output        []byte
	delayedOutput []byte
	indentation   int
	positions     map[int]token.Pos
}

func (c *PkgContext) Write(b []byte) (int, error) {
	c.output = append(c.output, b...)
	return len(b), nil
}

func (c *PkgContext) Printf(format string, values ...interface{}) {
	c.Write([]byte(strings.Repeat("\t", c.indentation)))
	fmt.Fprintf(c, format, values...)
	c.Write([]byte{'\n'})
	c.Write(c.delayedOutput)
	c.delayedOutput = nil
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
	c.delayedOutput = c.CatchOutput(f)
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
		allVarNames:  make(map[string]int),
		postLoopStmt: make(map[string]ast.Stmt),
		positions:    make(map[int]token.Pos),
	}
	for name := range ReservedKeywords {
		c.allVarNames[name] = 1
	}

	var functions []*ast.FuncDecl
	functionsByObject := make(map[types.Object]*ast.FuncDecl)
	var initStmts []ast.Stmt
	var toplevelTypes []*types.TypeName
	var constants []*types.Const
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
				if isBlank(d.Name) {
					continue
				}
				if sig.Recv() == nil && d.Name.Name == "init" {
					initStmts = append(initStmts, d.Body.List...)
					continue
				}
				functions = append(functions, d)
				o := c.info.Objects[d.Name]
				functionsByObject[o] = d
				if sig.Recv() == nil {
					c.objectName(o) // register toplevel name
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					for _, spec := range d.Specs {
						o := c.info.Objects[spec.(*ast.TypeSpec).Name].(*types.TypeName)
						toplevelTypes = append(toplevelTypes, o)
						c.objectName(o) // register toplevel name
					}
				case token.CONST:
					for _, spec := range d.Specs {
						s := spec.(*ast.ValueSpec)
						for _, name := range s.Names {
							if !isBlank(name) {
								o := c.info.Objects[name].(*types.Const)
								constants = append(constants, o)
								c.objectName(o) // register toplevel name
							}
						}
					}
				case token.VAR:
					for _, spec := range d.Specs {
						s := spec.(*ast.ValueSpec)
						varSpecs = append(varSpecs, s)
						for _, name := range s.Names {
							if !isBlank(name) {
								c.objectName(c.info.Objects[name]) // register toplevel name
							}
						}
					}
				}
			}
		}
	}

	// resolve var dependencies
	var unorderedSingleVarSpecs []*ast.ValueSpec
	pendingObjects := make(map[types.Object]bool)
	for _, spec := range varSpecs {
		for _, singleSpec := range c.splitValueSpec(spec) {
			if singleSpec.Values[0] == nil {
				continue
			}
			unorderedSingleVarSpecs = append(unorderedSingleVarSpecs, singleSpec)
			for _, name := range singleSpec.Names {
				pendingObjects[c.info.Objects[name]] = true
			}
		}
	}
	complete := false
	var initVarStmts []ast.Stmt
	for !complete {
		complete = true
		for i, spec := range unorderedSingleVarSpecs {
			if spec == nil {
				continue
			}
			v := DependencyCollector{info: c.info, functions: functionsByObject}
			ast.Walk(&v, spec.Values[0])
			currentObjs := make(map[types.Object]bool)
			for _, name := range spec.Names {
				currentObjs[c.info.Objects[name]] = true
			}
			ready := true
			for _, dep := range v.dependencies {
				if currentObjs[dep] {
					return nil, fmt.Errorf("%s: initialization loop", fileSet.Position(dep.Pos()).String())
				}
				ready = ready && !pendingObjects[dep]
			}
			if !ready {
				complete = false
				continue
			}
			lhs := make([]ast.Expr, len(spec.Names))
			for i, name := range spec.Names {
				lhs[i] = name
				delete(pendingObjects, c.info.Objects[name])
			}
			initVarStmts = append(initVarStmts, &ast.AssignStmt{
				Lhs: lhs,
				Tok: token.DEFINE,
				Rhs: spec.Values,
			})
			unorderedSingleVarSpecs[i] = nil
		}
	}

	c.Indent(func() {
		for _, importedPkg := range typesPkg.Imports() {
			varName := c.newVariable(importedPkg.Name())
			c.Printf(`var %s = go$packages["%s"];`, varName, importedPkg.Path())
			c.pkgVars[importedPkg.Path()] = varName
		}

		// types
		for _, o := range toplevelTypes {
			typeName := c.objectName(o)
			c.Printf("var %s;", typeName)
			c.translateType(o)
			c.Printf("go$pkg.%s = %s;", typeName, typeName)
		}
		for _, o := range toplevelTypes {
			c.initType(o)
			if _, isInterface := o.Type().Underlying().(*types.Interface); !isInterface {
				writeMethodSet := func(t types.Type) {
					methodSet := t.MethodSet()
					if methodSet.Len() == 0 {
						return
					}
					methods := make([]string, methodSet.Len())
					for i := range methods {
						method := methodSet.At(i).Obj()
						pkgPath := ""
						if !method.IsExported() {
							pkgPath = method.Pkg().Path()
						}
						t := method.Type().(*types.Signature)
						methods[i] = fmt.Sprintf(`["%s", "%s", %s, %s, %t]`, method.Name(), pkgPath, c.typeArray(t.Params()), c.typeArray(t.Results()), t.IsVariadic())
					}
					c.Printf("%s.methods = [%s];", c.typeName(t), strings.Join(methods, ", "))
				}
				writeMethodSet(o.Type())
				writeMethodSet(types.NewPointer(o.Type()))
			}
		}

		// functions
		for _, fun := range functions {
			c.translateFunction(fun)
		}

		// constants
		for _, o := range constants {
			varPrefix := ""
			if !o.IsExported() {
				varPrefix = "var "
			}
			v := c.newIdent(o.Name(), o.Type())
			c.info.Values[v] = o.Val()
			c.Printf("%s%s = %s;", varPrefix, c.objectName(o), c.translateExpr(v))
		}

		// variables
		for _, spec := range varSpecs {
			for _, name := range spec.Names {
				o := c.info.Objects[name].(*types.Var)
				varPrefix := ""
				if !o.IsExported() {
					varPrefix = "var "
				}
				c.Printf("%s%s = %s;", varPrefix, c.objectName(o), c.zeroValue(o.Type()))
			}
		}

		// builtin native implementations
		if native, hasNative := pkgNatives[importPath]; hasNative {
			c.Write([]byte(strings.TrimSpace(native)))
			c.Write([]byte{'\n'})
		}

		// exports for package functions
		for _, fun := range functions {
			name := fun.Name.Name
			if fun.Recv == nil && (fun.Name.IsExported() || name == "main") {
				c.Printf("go$pkg.%s = %s;", name, name)
			}
		}

		// init function
		c.Printf("go$pkg.init = function() {")
		c.Indent(func() {
			c.translateFunctionBody(append(initVarStmts, initStmts...), nil)
		})
		c.Printf("};")
	})

	return c.output, nil
}

func (c *PkgContext) translateType(o *types.TypeName) {
	typeName := c.objectName(o)
	switch t := o.Type().Underlying().(type) {
	case *types.Struct:
		params := make([]string, t.NumFields())
		for i := 0; i < t.NumFields(); i++ {
			params[i] = fieldName(t, i) + "_"
		}
		c.Printf(`%s = go$newType(0, "Struct", "%s.%s", "%s", "%s", function(%s) {`, typeName, o.Pkg().Name(), o.Name(), o.Name(), o.Pkg().Path(), strings.Join(params, ", "))
		c.Indent(func() {
			c.Printf("this.go$val = this;")
			for i := 0; i < t.NumFields(); i++ {
				name := fieldName(t, i)
				c.Printf("this.%s = %s_ !== undefined ? %s_ : %s;", name, name, name, c.zeroValue(t.Field(i).Type()))
			}
		})
		c.Printf("});")
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
					value := "this." + fieldName(t, i)
					if isWrapped(field.Type()) {
						value = fmt.Sprintf("new %s(%s)", c.typeName(field.Type()), value)
					}
					paramList := strings.Join(params, ", ")
					c.Printf("%s.prototype.%s = function(%s) { return this.go$val.%s(%s); };", typeName, name, paramList, name, paramList)
					c.Printf("%s.Ptr.prototype.%s = function(%s) { return %s.%s(%s); };", typeName, name, paramList, value, name, paramList)
				}
			}
		}
	default:
		c.Printf(`%s = go$newType(0, "%s", "%s.%s", "%s", "%s", null);`, typeName, typeKind(t), o.Pkg().Name(), o.Name(), o.Name(), o.Pkg().Path())
	}
}

func (c *PkgContext) initType(obj types.Object) {
	switch t := obj.Type().Underlying().(type) {
	case *types.Array, *types.Chan, *types.Interface, *types.Map, *types.Pointer, *types.Slice, *types.Signature, *types.Struct:
		c.Printf("%s.init(%s);", c.objectName(obj), c.initArgs(t))
	}
}

func (c *PkgContext) initArgs(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Array:
		return fmt.Sprintf("%s, %d", c.typeName(t.Elem()), t.Len())
	case *types.Chan:
		return fmt.Sprintf("%s, %t, %t", c.typeName(t.Elem()), t.Dir()&types.SendOnly != 0, t.Dir()&types.RecvOnly != 0)
	case *types.Interface:
		methods := make([]string, t.NumMethods())
		for i := range methods {
			method := t.Method(i)
			pkgPath := ""
			if !method.IsExported() {
				pkgPath = method.Pkg().Path()
			}
			methods[i] = fmt.Sprintf(`["%s", "%s", %s]`, method.Name(), pkgPath, c.typeName(method.Type()))
		}
		return fmt.Sprintf("[%s]", strings.Join(methods, ", "))
	case *types.Map:
		return fmt.Sprintf("%s, %s", c.typeName(t.Key()), c.typeName(t.Elem()))
	case *types.Pointer:
		return fmt.Sprintf("%s", c.typeName(t.Elem()))
	case *types.Slice:
		return fmt.Sprintf("%s", c.typeName(t.Elem()))
	case *types.Signature:
		return fmt.Sprintf("%s, %s, %t", c.typeArray(t.Params()), c.typeArray(t.Results()), t.IsVariadic())
	case *types.Struct:
		fields := make([]string, t.NumFields())
		for i := range fields {
			field := t.Field(i)
			name := ""
			if !field.Anonymous() {
				name = field.Name()
			}
			pkgPath := ""
			if !field.IsExported() {
				pkgPath = field.Pkg().Path()
			}
			fields[i] = fmt.Sprintf(`["%s", "%s", %s, %s]`, name, pkgPath, c.typeName(field.Type()), encodeString(t.Tag(i)))
		}
		return fmt.Sprintf("[%s]", strings.Join(fields, ", "))
	default:
		panic("invalid type")
	}
}

func (c *PkgContext) splitValueSpec(s *ast.ValueSpec) []*ast.ValueSpec {
	if len(s.Values) == 1 {
		if _, isTuple := c.info.Types[s.Values[0]].(*types.Tuple); isTuple {
			return []*ast.ValueSpec{s}
		}
	}

	list := make([]*ast.ValueSpec, len(s.Names))
	for i, name := range s.Names {
		var value ast.Expr
		if i < len(s.Values) {
			value = s.Values[i]
		}
		list[i] = &ast.ValueSpec{
			Names:  []*ast.Ident{name},
			Values: []ast.Expr{value},
		}
	}
	return list
}

func (c *PkgContext) translateFunction(fun *ast.FuncDecl) {
	c.newScope(func() {
		sig := c.info.Objects[fun.Name].(*types.Func).Type().(*types.Signature)
		var recv *ast.Ident
		if fun.Recv != nil && fun.Recv.List[0].Names != nil {
			recv = fun.Recv.List[0].Names[0]
		}
		params := c.translateParams(fun.Type)
		joinedParams := strings.Join(params, ", ")

		printPrimaryFunction := func(lhs string, fullName string) {
			c.Printf("%s = function(%s) {", lhs, joinedParams)
			c.Indent(func() {
				if fun.Body == nil {
					c.Printf(`throw go$panic("Native function not implemented: %s");`, fullName)
					return
				}

				body := fun.Body.List
				if recv != nil {
					recvType := sig.Recv().Type()
					c.info.Types[recv] = recvType
					this := c.newIdent("this", recvType)
					if isWrapped(recvType) {
						this = c.newIdent("this.go$val", recvType)
					}
					body = append([]ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{recv},
							Tok: token.DEFINE,
							Rhs: []ast.Expr{this},
						},
					}, body...)
				}
				c.translateFunctionBody(body, sig)
			})
			c.Printf("};")
		}

		if fun.Recv == nil {
			funName := c.objectName(c.info.Objects[fun.Name])
			printPrimaryFunction("var "+funName, funName)
			return
		}

		recvType := sig.Recv().Type()
		ptr, isPointer := recvType.(*types.Pointer)
		namedRecvType, _ := recvType.(*types.Named)
		if isPointer {
			namedRecvType = ptr.Elem().(*types.Named)
		}
		typeName := c.objectName(namedRecvType.Obj())
		funName := fun.Name.Name
		if ReservedKeywords[funName] {
			funName += "$"
		}

		if _, isStruct := namedRecvType.Underlying().(*types.Struct); isStruct {
			c.Printf("%s.prototype.%s = function(%s) { return this.go$val.%s(%s); };", typeName, funName, joinedParams, funName, joinedParams)
			printPrimaryFunction(typeName+".Ptr.prototype."+funName, typeName+"."+funName)
			return
		}

		if isPointer {
			if _, isArray := ptr.Elem().Underlying().(*types.Array); isArray {
				printPrimaryFunction(typeName+".prototype."+funName, typeName+"."+funName)
				c.Printf("go$ptrType(%s).prototype.%s = function(%s) { return (new %s(this.go$get())).%s(%s); };", typeName, funName, joinedParams, typeName, funName, joinedParams)
				return
			}
			value := "this"
			if isWrapped(ptr.Elem()) {
				value = "this.go$val"
			}
			c.Printf("%s.prototype.%s = function(%s) { var obj = %s; return (new (go$ptrType(%s))(function() { return obj; }, null)).%s(%s); };", typeName, funName, joinedParams, value, typeName, funName, joinedParams)
			printPrimaryFunction(fmt.Sprintf("go$ptrType(%s).prototype.%s", typeName, funName), typeName+"."+funName)
			return
		}

		value := "this.go$get()"
		if isWrapped(recvType) {
			value = fmt.Sprintf("new %s(%s)", typeName, value)
		}
		printPrimaryFunction(typeName+".prototype."+funName, typeName+"."+funName)
		c.Printf("go$ptrType(%s).prototype.%s = function(%s) { return %s.%s(%s); };", typeName, funName, joinedParams, value, funName, joinedParams)
	})
}

func (c *PkgContext) translateFunctionBody(stmts []ast.Stmt, sig *types.Signature) {
	c.funcVarNames = nil

	body := c.CatchOutput(func() {
		var resultNames []ast.Expr
		if sig != nil && sig.Results().Len() != 0 && sig.Results().At(0).Name() != "" {
			resultNames = make([]ast.Expr, sig.Results().Len())
			for i := 0; i < sig.Results().Len(); i++ {
				result := sig.Results().At(i)
				name := result.Name()
				if result.Name() == "_" {
					name = "result"
				}
				id := ast.NewIdent(name)
				c.info.Types[id] = result.Type()
				c.info.Objects[id] = result
				c.Printf("%s = %s;", c.translateExpr(id), c.zeroValue(result.Type()))
				resultNames[i] = id
			}
		}

		if sig != nil {
			s := c.functionSig
			defer func() { c.functionSig = s }()
			c.functionSig = sig
		}
		r := c.resultNames
		defer func() { c.resultNames = r }()
		c.resultNames = resultNames
		p := c.postLoopStmt
		defer func() { c.postLoopStmt = p }()
		c.postLoopStmt = make(map[string]ast.Stmt)

		v := HasDeferVisitor{}
		ast.Walk(&v, &ast.BlockStmt{List: stmts})
		switch v.hasDefer {
		case true:
			c.Printf("var go$deferred = [];")
			c.Printf("try {")
			c.Indent(func() {
				c.translateStmtList(stmts)
			})
			c.Printf("} catch(go$err) {")
			c.Indent(func() {
				c.Printf("go$pushErr(go$err);")
				if sig != nil && resultNames == nil {
					switch sig.Results().Len() {
					case 0:
						// nothing
					case 1:
						c.Printf("return %s;", c.zeroValue(sig.Results().At(0).Type()))
					default:
						zeros := make([]string, sig.Results().Len())
						for i := range zeros {
							zeros[i] = c.zeroValue(sig.Results().At(i).Type())
						}
						c.Printf("return [%s];", strings.Join(zeros, ", "))
					}
				}
			})
			c.Printf("} finally {")
			c.Indent(func() {
				c.Printf("go$callDeferred(go$deferred);")
				if resultNames != nil {
					c.translateStmt(&ast.ReturnStmt{}, "")
				}
			})
			c.Printf("}")
		case false:
			c.translateStmtList(stmts)
		}
	})

	if len(c.funcVarNames) != 0 {
		c.Printf("var %s;", strings.Join(c.funcVarNames, ", "))
	}
	c.Write(body)
}

func (c *PkgContext) translateParams(t *ast.FuncType) []string {
	params := make([]string, 0)
	for _, param := range t.Params.List {
		for _, ident := range param.Names {
			if isBlank(ident) {
				params = append(params, c.newVariable("param"))
				continue
			}
			params = append(params, c.objectName(c.info.Objects[ident]))
		}
	}
	return params
}

func (c *PkgContext) translateArgs(sig *types.Signature, args []ast.Expr, ellipsis bool) string {
	params := make([]string, sig.Params().Len())
	for i := range params {
		if sig.IsVariadic() && i == len(params)-1 && !ellipsis {
			varargType := sig.Params().At(i).Type().(*types.Slice)
			varargs := make([]string, len(args)-i)
			for j, arg := range args[i:] {
				varargs[j] = c.translateExprToType(arg, varargType.Elem())
			}
			params[i] = fmt.Sprintf("new %s([%s])", c.typeName(varargType), strings.Join(varargs, ", "))
			break
		}
		argType := sig.Params().At(i).Type()
		params[i] = c.translateExprToType(args[i], argType)
	}
	return strings.Join(params, ", ")
}

func (c *PkgContext) zeroValue(ty types.Type) string {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		switch {
		case is64Bit(t) || t.Info()&types.IsComplex != 0:
			return fmt.Sprintf("new %s(0, 0)", c.typeName(ty))
		case t.Info()&types.IsBoolean != 0:
			return "false"
		case t.Info()&types.IsNumeric != 0, t.Kind() == types.UnsafePointer:
			return "0"
		case t.Info()&types.IsString != 0:
			return `""`
		case t.Kind() == types.UntypedNil:
			panic("Zero value for untyped nil.")
		default:
			panic("Unhandled type")
		}
	case *types.Array:
		return fmt.Sprintf(`go$makeNativeArray("%s", %d, function() { return %s; })`, typeKind(t.Elem()), t.Len(), c.zeroValue(t.Elem()))
	case *types.Signature:
		return "go$throwNilPointerError"
	case *types.Slice:
		return fmt.Sprintf("%s.nil", c.typeName(ty))
	case *types.Struct:
		if named, isNamed := ty.(*types.Named); isNamed {
			return fmt.Sprintf("new %s.Ptr()", c.objectName(named.Obj()))
		}
		fields := make([]string, t.NumFields())
		for i := range fields {
			fields[i] = c.zeroValue(t.Field(i).Type())
		}
		return fmt.Sprintf("new %s.Ptr(%s)", c.typeName(ty), strings.Join(fields, ", "))
	case *types.Map:
		return "false"
	case *types.Interface:
		return "null"
	}
	return fmt.Sprintf("%s.nil", c.typeName(ty))
}

func (c *PkgContext) newVariable(name string) string {
	if name == "" {
		panic("newVariable: empty name")
	}
	for _, b := range []byte(name) {
		if b < '0' || b > 'z' {
			name = "nonAsciiName"
			break
		}
	}
	if strings.HasPrefix(name, "dollar_") {
		name = "$" + name[7:]
	}
	n := c.allVarNames[name]
	c.allVarNames[name] = n + 1
	if n > 0 {
		name = fmt.Sprintf("%s$%d", name, n)
	}
	c.funcVarNames = append(c.funcVarNames, name)
	return name
}

func (c *PkgContext) newScope(f func()) {
	outerVarNames := make(map[string]int, len(c.allVarNames))
	for k, v := range c.allVarNames {
		outerVarNames[k] = v
	}
	outerFuncVarNames := c.funcVarNames
	f()
	c.allVarNames = outerVarNames
	c.funcVarNames = outerFuncVarNames
}

func (c *PkgContext) newIdent(name string, t types.Type) *ast.Ident {
	ident := ast.NewIdent(name)
	c.info.Types[ident] = t
	obj := types.NewVar(0, c.pkg, name, t)
	c.info.Objects[ident] = obj
	c.objectVars[obj] = name
	return ident
}

func (c *PkgContext) objectName(o types.Object) string {
	if o.Pkg() != nil && o.Pkg() != c.pkg {
		pkgVar, found := c.pkgVars[o.Pkg().Path()]
		if !found {
			pkgVar = fmt.Sprintf(`go$packages["%s"]`, o.Pkg().Path())
		}
		return pkgVar + "." + o.Name()
	}

	name, found := c.objectVars[o]
	if !found {
		name = c.newVariable(o.Name())
		c.objectVars[o] = name
	}

	switch o.(type) {
	case *types.Var, *types.Const:
		if o.IsExported() && o.Parent() == c.pkg.Scope() {
			return "go$pkg." + name
		}
	}
	return name
}

func (c *PkgContext) typeName(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.UntypedNil:
			return "null"
		case types.UnsafePointer:
			return "Go$UnsafePointer"
		default:
			return "Go$" + toJavaScriptType(t)
		}
	case *types.Named:
		if t.Obj().Name() == "error" {
			return "go$error"
		}
		return c.objectName(t.Obj())
	case *types.Pointer:
		return fmt.Sprintf("(go$ptrType(%s))", c.initArgs(t))
	case *types.Array, *types.Chan, *types.Slice, *types.Map, *types.Signature, *types.Interface, *types.Struct:
		return fmt.Sprintf("(go$%sType(%s))", strings.ToLower(typeKind(t)), c.initArgs(t))
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
}

func (c *PkgContext) makeKey(expr ast.Expr, keyType types.Type) string {
	switch t := keyType.Underlying().(type) {
	case *types.Array, *types.Struct:
		return fmt.Sprintf("(new %s(%s)).go$key()", c.typeName(keyType), c.translateExpr(expr))
	case *types.Basic:
		if is64Bit(t) {
			return fmt.Sprintf("%s.go$key()", c.translateExprToType(expr, keyType))
		}
		return c.translateExprToType(expr, keyType)
	case *types.Chan, *types.Pointer:
		return fmt.Sprintf("%s.go$key()", c.translateExprToType(expr, keyType))
	case *types.Interface:
		return fmt.Sprintf("(%s || go$interfaceNil).go$key()", c.translateExprToType(expr, keyType))
	default:
		return c.translateExprToType(expr, keyType)
	}
}

func (c *PkgContext) typeArray(t *types.Tuple) string {
	s := make([]string, t.Len())
	for i := range s {
		s[i] = c.typeName(t.At(i).Type())
	}
	return "[" + strings.Join(s, ", ") + "]"
}

func fieldName(t *types.Struct, i int) string {
	name := t.Field(i).Name()
	if name == "_" || ReservedKeywords[name] {
		return fmt.Sprintf("%s$%d", name, i)
	}
	return name
}

func typeKind(ty types.Type) string {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		return toJavaScriptType(t)
	case *types.Array:
		return "Array"
	case *types.Chan:
		return "Chan"
	case *types.Interface:
		return "Interface"
	case *types.Map:
		return "Map"
	case *types.Signature:
		return "Func"
	case *types.Slice:
		return "Slice"
	case *types.Struct:
		return "Struct"
	case *types.Pointer:
		return "Ptr"
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
}

func toJavaScriptType(t *types.Basic) string {
	switch t.Kind() {
	case types.UntypedInt:
		return "Int"
	case types.Byte:
		return "Uint8"
	case types.Rune:
		return "Int32"
	case types.UnsafePointer:
		return "UnsafePointer"
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

func isBlank(expr ast.Expr) bool {
	if expr == nil {
		return true
	}
	if id, isIdent := expr.(*ast.Ident); isIdent {
		return id.Name == "_"
	}
	return false
}

func isWrapped(ty types.Type) bool {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		return !is64Bit(t) && t.Info()&types.IsComplex == 0 && t.Kind() != types.UntypedNil
	case *types.Array, *types.Map, *types.Signature:
		return true
	case *types.Pointer:
		_, isArray := t.Elem().Underlying().(*types.Array)
		return isArray
	}
	return false
}

func elemType(ty types.Type) types.Type {
	switch t := ty.Underlying().(type) {
	case *types.Array:
		return t.Elem()
	case *types.Slice:
		return t.Elem()
	case *types.Pointer:
		return t.Elem().Underlying().(*types.Array).Elem()
	default:
		panic("")
	}
}

func encodeString(s string) string {
	buffer := bytes.NewBuffer(nil)
	for _, r := range []byte(s) {
		switch r {
		case '\b':
			buffer.WriteString(`\b`)
		case '\f':
			buffer.WriteString(`\f`)
		case '\n':
			buffer.WriteString(`\n`)
		case '\r':
			buffer.WriteString(`\r`)
		case '\t':
			buffer.WriteString(`\t`)
		case '\v':
			buffer.WriteString(`\v`)
		case '"':
			buffer.WriteString(`\"`)
		case '\\':
			buffer.WriteString(`\\`)
		default:
			if r < 0x20 || r > 0x7E {
				fmt.Fprintf(buffer, `\x%02X`, r)
				continue
			}
			buffer.WriteByte(r)
		}
	}
	return `"` + buffer.String() + `"`
}

func isJsObject(t types.Type) bool {
	named, isNamed := t.(*types.Named)
	return isNamed && named.Obj().Pkg().Path() == "github.com/neelance/gopherjs/js" && named.Obj().Name() == "Object"
}

func getJsTag(tag string) string {
	for tag != "" {
		// skip leading space
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// scan to colon.
		// a space or a quote is a syntax error
		i = 0
		for i < len(tag) && tag[i] != ' ' && tag[i] != ':' && tag[i] != '"' {
			i++
		}
		if i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// scan quoted string to find value
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if name == "js" {
			value, _ := strconv.Unquote(qvalue)
			return value
		}
	}
	return ""
}

type DependencyCollector struct {
	info         *types.Info
	functions    map[types.Object]*ast.FuncDecl
	dependencies []types.Object
}

func (v *DependencyCollector) Visit(node ast.Node) (w ast.Visitor) {
	switch n := node.(type) {
	case *ast.Ident:
		o := v.info.Objects[n]
		if fun, found := v.functions[o]; found {
			delete(v.functions, o)
			ast.Walk(v, fun)
			v.functions[o] = fun
			return v
		}
		v.dependencies = append(v.dependencies, o)
	}
	return v
}
