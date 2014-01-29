package translator

import (
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/token"
	"sort"
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

func TranslatePackage(importPath string, files []*ast.File, fileSet *token.FileSet, importPkg func(string) (*Output, error)) (*Output, error) {
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Objects:    make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	var errList ErrorList
	var previousErr error
	config := &types.Config{
		Packages: typesPackages,
		Import: func(_ map[string]*types.Package, path string) (*types.Package, error) {
			output, err := importPkg(path)
			if err != nil {
				return nil, err
			}
			return output.Types, nil
		},
		Sizes: sizes32,
		Error: func(err error) {
			if previousErr != nil && previousErr.Error() == err.Error() {
				return
			}
			errList = append(errList, err)
			previousErr = err
		},
	}
	typesPkg, err := config.Check(importPath, fileSet, files, info)
	if errList != nil {
		return nil, errList
	}
	if err != nil {
		return nil, err
	}
	typesPackages[importPath] = typesPkg

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
			v := VarDependencyCollector{info: c.info, functions: functionsByObject}
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
						if !method.Exported() {
							pkgPath = method.Pkg().Path()
						}
						t := method.Type().(*types.Signature)
						methods[i] = fmt.Sprintf(`["%s", "%s", %s, %s, %t]`, method.Name(), pkgPath, c.typeArray(t.Params()), c.typeArray(t.Results()), t.Variadic())
					}
					c.Printf("%s.methods = [%s];", c.typeName(t), strings.Join(methods, ", "))
				}
				writeMethodSet(o.Type())
				writeMethodSet(types.NewPointer(o.Type()))
			}
		}

		// functions
		natives := pkgNatives[importPath]
		nativeInit := natives["init"]
		delete(natives, "init")
		for _, fun := range functions {
			c.translateFunction(fun, natives, false)
		}
		for _, fun := range functions {
			c.translateFunction(fun, natives, true)
		}
		if len(natives) != 0 {
			panic("not all natives used: " + importPath)
		}

		// constants
		for _, o := range constants {
			varPrefix := ""
			if !o.Exported() {
				varPrefix = "var "
			}
			v := c.newIdent(o.Name(), o.Type())
			c.info.Types[v] = types.TypeAndValue{Type: o.Type(), Value: o.Val()}
			c.Printf("%s%s = %s;", varPrefix, c.objectName(o), c.translateExpr(v))
		}

		// variables
		for _, spec := range varSpecs {
			for _, name := range spec.Names {
				o := c.info.Objects[name].(*types.Var)
				varPrefix := ""
				if !o.Exported() {
					varPrefix = "var "
				}
				c.Printf("%s%s = %s;", varPrefix, c.objectName(o), c.zeroValue(o.Type()))
			}
		}

		// builtin native implementations
		c.Write([]byte(nativeInit))

		// init function
		c.Printf("go$pkg.init = function() {")
		c.Indent(func() {
			c.translateFunctionBody(append(initVarStmts, initStmts...), nil)
		})
		c.Printf("};")
	})

	output := &Output{typesPkg, []string{"runtime"}, c.output} // all packages depend on runtime

	var importedPaths []string
	for _, imp := range typesPkg.Imports() {
		importedPaths = append(importedPaths, imp.Path())
	}
	sort.Strings(importedPaths)
	for _, impPath := range importedPaths {
		impOutput, err := importPkg(impPath)
		if err != nil {
			return nil, err
		}
		output.AddDependenciesOf(impOutput)
	}
	output.AddDependency(importPath)

	return output, nil
}

func (c *PkgContext) translateType(o *types.TypeName) {
	typeName := c.objectName(o)
	size := int64(0)
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
		return
	case *types.Basic:
		if t.Info()&types.IsInteger != 0 {
			size = sizes32.Sizeof(t)
		}
	}
	c.Printf(`%s = go$newType(%d, "%s", "%s.%s", "%s", "%s", null);`, typeName, size, typeKind(o.Type()), o.Pkg().Name(), o.Name(), o.Name(), o.Pkg().Path())
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
			if !method.Exported() {
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
		return fmt.Sprintf("%s, %s, %t", c.typeArray(t.Params()), c.typeArray(t.Results()), t.Variadic())
	case *types.Struct:
		fields := make([]string, t.NumFields())
		for i := range fields {
			field := t.Field(i)
			name := ""
			if !field.Anonymous() {
				name = field.Name()
			}
			pkgPath := ""
			if !field.Exported() {
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
		if _, isTuple := c.info.Types[s.Values[0]].Type.(*types.Tuple); isTuple {
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

func (c *PkgContext) translateFunction(fun *ast.FuncDecl, natives map[string]string, translateNatives bool) {
	c.newScope(func() {
		sig := c.info.Objects[fun.Name].(*types.Func).Type().(*types.Signature)
		var recv *ast.Ident
		if fun.Recv != nil && fun.Recv.List[0].Names != nil {
			recv = fun.Recv.List[0].Names[0]
		}
		params := c.translateParams(fun.Type)
		joinedParams := strings.Join(params, ", ")

		printPrimaryFunction := func(lhs string, fullName string) bool {
			native, hasNative := natives[fullName]
			if translateNatives != hasNative {
				return false
			}
			if hasNative {
				c.Printf("%s = %s;", lhs, native)
				delete(natives, fullName)
				return true
			}

			c.Printf("%s = function(%s) {", lhs, joinedParams)
			c.Indent(func() {
				if fun.Body == nil {
					c.Printf(`throw go$panic("Native function not implemented: %s");`, fullName)
					return
				}

				body := fun.Body.List
				if recv != nil {
					recvType := sig.Recv().Type()
					c.info.Types[recv] = types.TypeAndValue{Type: recvType}
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
			return true
		}

		if fun.Recv == nil {
			funName := c.objectName(c.info.Objects[fun.Name])
			lhs := "var " + funName
			if fun.Name.IsExported() || fun.Name.Name == "main" {
				lhs += " = go$pkg." + funName
			}
			printPrimaryFunction(lhs, funName)
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
			if printPrimaryFunction(typeName+".Ptr.prototype."+funName, typeName+"."+funName) {
				c.Printf("%s.prototype.%s = function(%s) { return this.go$val.%s(%s); };", typeName, funName, joinedParams, funName, joinedParams)
			}
			return
		}

		if isPointer {
			if _, isArray := ptr.Elem().Underlying().(*types.Array); isArray {
				if printPrimaryFunction(typeName+".prototype."+funName, typeName+"."+funName) {
					c.Printf("go$ptrType(%s).prototype.%s = function(%s) { return (new %s(this.go$get())).%s(%s); };", typeName, funName, joinedParams, typeName, funName, joinedParams)
				}
				return
			}
			value := "this"
			if isWrapped(ptr.Elem()) {
				value = "this.go$val"
			}
			if printPrimaryFunction(fmt.Sprintf("go$ptrType(%s).prototype.%s", typeName, funName), typeName+"."+funName) {
				c.Printf("%s.prototype.%s = function(%s) { var obj = %s; return (new (go$ptrType(%s))(function() { return obj; }, null)).%s(%s); };", typeName, funName, joinedParams, value, typeName, funName, joinedParams)
			}
			return
		}

		value := "this.go$get()"
		if isWrapped(recvType) {
			value = fmt.Sprintf("new %s(%s)", typeName, value)
		}
		if printPrimaryFunction(typeName+".prototype."+funName, typeName+"."+funName) {
			c.Printf("go$ptrType(%s).prototype.%s = function(%s) { return %s.%s(%s); };", typeName, funName, joinedParams, value, funName, joinedParams)
		}
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
				c.info.Types[id] = types.TypeAndValue{Type: result.Type()}
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

type VarDependencyCollector struct {
	info         *types.Info
	functions    map[types.Object]*ast.FuncDecl
	dependencies []types.Object
}

func (v *VarDependencyCollector) Visit(node ast.Node) (w ast.Visitor) {
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

// type DependencyAnalysis struct {
// }

// func (v *DependencyAnalysis) Visit(node ast.Node) (w ast.Visitor) {
// 	switch n := node.(type) {

// 	}
// 	return v
// }
