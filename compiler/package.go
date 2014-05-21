package translator

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"encoding/json"
	"fmt"
	"github.com/gopherjs/gopherjs/gcexporter"
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

type funcContext struct {
	p             *pkgContext
	sig           *types.Signature
	allVars       map[string]int
	localVars     []string
	resultNames   []ast.Expr
	flowDatas     map[string]*flowData
	escapingVars  []string
	flattened     bool
	caseCounter   int
	labelCases    map[string]int
	hasGoto       map[ast.Node]bool
	output        []byte
	delayedOutput []byte
}

type pkgContext struct {
	pkg          *types.Package
	info         *types.Info
	pkgVars      map[string]string
	objectVars   map[types.Object]string
	indentation  int
	dependencies map[types.Object]bool
}

type flowData struct {
	postStmt  func()
	beginCase int
	endCase   int
}

func (t *Translator) Compile(importPath string, files []*ast.File, fileSet *token.FileSet, importPkg func(string) (*Archive, error)) (*Archive, error) {
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	patchedFiles := make([]*ast.File, len(files))
	for i, file := range files {
		patchedFiles[i] = applyPatches(file, fileSet, importPath)
	}

	var errList ErrorList
	var previousErr error
	config := &types.Config{
		Packages: t.typesPackages,
		Import: func(_ map[string]*types.Package, path string) (*types.Package, error) {
			if _, err := importPkg(path); err != nil {
				return nil, err
			}
			return t.typesPackages[path], nil
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
	typesPkg, err := config.Check(importPath, fileSet, patchedFiles, info)
	if errList != nil {
		return nil, errList
	}
	if err != nil {
		return nil, err
	}
	t.typesPackages[importPath] = typesPkg

	c := &funcContext{
		p: &pkgContext{
			pkg:          typesPkg,
			info:         info,
			pkgVars:      make(map[string]string),
			objectVars:   make(map[types.Object]string),
			indentation:  1,
			dependencies: make(map[types.Object]bool),
		},
		allVars:     make(map[string]int),
		flowDatas:   map[string]*flowData{"": &flowData{}},
		caseCounter: 1,
		labelCases:  make(map[string]int),
		hasGoto:     make(map[ast.Node]bool),
	}
	for name := range reservedKeywords {
		c.allVars[name] = 1
	}

	var functions []*ast.FuncDecl
	var initStmts []ast.Stmt
	var toplevelTypes []*types.TypeName
	var vars []*types.Var
	for _, file := range patchedFiles {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				sig := c.p.info.Defs[d.Name].(*types.Func).Type().(*types.Signature)
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
				if sig.Recv() == nil {
					c.objectName(c.p.info.Defs[d.Name]) // register toplevel name
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					for _, spec := range d.Specs {
						o := c.p.info.Defs[spec.(*ast.TypeSpec).Name].(*types.TypeName)
						toplevelTypes = append(toplevelTypes, o)
						c.objectName(o) // register toplevel name
					}
				case token.VAR:
					for _, spec := range d.Specs {
						for _, name := range spec.(*ast.ValueSpec).Names {
							if !isBlank(name) {
								o := c.p.info.Defs[name].(*types.Var)
								vars = append(vars, o)
								c.objectName(o) // register toplevel name
							}
						}
					}
				case token.CONST:
					// skip, constants are inlined
				}
			}
		}
	}

	collectDependencies := func(self types.Object, f func()) []DepId {
		c.p.dependencies = make(map[types.Object]bool)
		f()
		var deps []string
		for dep := range c.p.dependencies {
			if dep != self {
				deps = append(deps, dep.Pkg().Path()+":"+dep.Name())
			}
		}
		sort.Strings(deps)
		depIds := make([]DepId, len(deps))
		for i, dep := range deps {
			depIds[i] = DepId(dep)
		}
		return depIds
	}

	gcData := bytes.NewBuffer(nil)
	gcexporter.Write(typesPkg, gcData, sizes32)
	encodedFileSet := bytes.NewBuffer(nil)
	if err := fileSet.Write(json.NewEncoder(encodedFileSet).Encode); err != nil {
		return nil, err
	}
	archive := &Archive{
		ImportPath:   importPath,
		GcData:       gcData.Bytes(),
		Dependencies: []string{"github.com/gopherjs/gopherjs/js", "runtime"}, // all packages depend on those
		FileSet:      encodedFileSet.Bytes(),
	}

	// imports
	for _, importedPkg := range typesPkg.Imports() {
		varName := c.newVariable(importedPkg.Name())
		c.p.pkgVars[importedPkg.Path()] = varName
		archive.Imports = append(archive.Imports, Import{Path: importedPkg.Path(), VarName: varName})
	}

	// types
	for _, o := range toplevelTypes {
		typeName := c.objectName(o)
		var d Decl
		d.Var = typeName
		d.DceFilters = []DepId{DepId(o.Name())}
		d.DceDeps = collectDependencies(o, func() {
			d.BodyCode = c.CatchOutput(0, func() { c.translateType(o, true) })
			d.InitCode = c.CatchOutput(1, func() { c.initType(o) })
		})
		archive.Declarations = append(archive.Declarations, d)
	}

	// functions
	for _, fun := range functions {
		var d Decl
		o := c.p.info.Defs[fun.Name].(*types.Func)
		funName := fun.Name.Name
		if fun.Recv == nil {
			d.Var = c.objectName(o)
			if o.Name() != "main" {
				d.DceFilters = []DepId{DepId(o.Name())}
			}
		}
		if fun.Recv != nil {
			recvType := o.Type().(*types.Signature).Recv().Type()
			ptr, isPointer := recvType.(*types.Pointer)
			namedRecvType, _ := recvType.(*types.Named)
			if isPointer {
				namedRecvType = ptr.Elem().(*types.Named)
			}
			funName = namedRecvType.Obj().Name() + "." + funName
			d.DceFilters = []DepId{DepId(namedRecvType.Obj().Name())}
			if !fun.Name.IsExported() {
				d.DceFilters = append(d.DceFilters, DepId(fun.Name.Name))
			}
		}

		d.DceDeps = collectDependencies(o, func() {
			d.BodyCode = c.translateToplevelFunction(fun)
		})
		archive.Declarations = append(archive.Declarations, d)
		if strings.HasPrefix(fun.Name.String(), "Test") {
			archive.Tests = append(archive.Tests, fun.Name.String())
		}
	}

	// variables
	initOrder := c.p.info.InitOrder

	// workaround for https://code.google.com/p/go/issues/detail?id=6703#c6
	if importPath == "math/rand" {
		findInit := func(name string) int {
			for i, init := range initOrder {
				if init.Lhs[0].Name() == name {
					return i
				}
			}
			panic("init not found")
		}
		i := findInit("rng_cooked")
		j := findInit("globalRand")
		if i > j {
			initOrder[i], initOrder[j] = initOrder[j], initOrder[i]
		}
	}

	varsWithInit := make(map[*types.Var]bool)
	for _, init := range initOrder {
		for _, o := range init.Lhs {
			varsWithInit[o] = true
		}
	}
	for _, o := range vars {
		var d Decl
		if !o.Exported() {
			d.Var = c.objectName(o)
		}
		if _, ok := varsWithInit[o]; !ok {
			d.DceDeps = collectDependencies(nil, func() {
				value := c.zeroValue(o.Type())
				if importPath == "runtime" && o.Name() == "sizeof_C_MStats" {
					value = "3712"
				}
				d.InitCode = []byte(fmt.Sprintf("\t\t%s = %s;\n", c.objectName(o), value))
			})
		}
		d.DceFilters = []DepId{DepId(o.Name())}
		archive.Declarations = append(archive.Declarations, d)
	}
	for _, init := range initOrder {
		lhs := make([]ast.Expr, len(init.Lhs))
		for i, o := range init.Lhs {
			ident := ast.NewIdent(o.Name())
			c.p.info.Types[ident] = types.TypeAndValue{Type: o.Type()}
			c.p.info.Defs[ident] = o
			lhs[i] = ident
			varsWithInit[o] = true
		}
		var d Decl
		d.DceDeps = collectDependencies(nil, func() {
			d.InitCode = c.translateFunctionBody(1, []ast.Stmt{
				&ast.AssignStmt{
					Lhs: lhs,
					Tok: token.DEFINE,
					Rhs: []ast.Expr{init.Rhs},
				},
			})
		})
		if len(init.Lhs) == 1 {
			v := hasCallVisitor{c.p.info, false}
			ast.Walk(&v, init.Rhs)
			if !v.hasCall {
				d.DceFilters = []DepId{DepId(init.Lhs[0].Name())}
			}
		}
		archive.Declarations = append(archive.Declarations, d)
	}

	// init functions
	var init Decl
	init.DceDeps = collectDependencies(nil, func() {
		init.InitCode = c.translateFunctionBody(1, initStmts)
	})
	archive.Declarations = append(archive.Declarations, init)

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
		archive.AddDependenciesOf(impOutput)
	}
	archive.AddDependency(importPath)

	return archive, nil
}

func (c *funcContext) translateType(o *types.TypeName, toplevel bool) {
	typeName := c.objectName(o)
	lhs := typeName
	if toplevel {
		lhs += " = go$pkg." + typeName
	}
	size := int64(0)
	constructor := "null"
	switch t := o.Type().Underlying().(type) {
	case *types.Struct:
		params := make([]string, t.NumFields())
		for i := 0; i < t.NumFields(); i++ {
			params[i] = fieldName(t, i) + "_"
		}
		constructor = fmt.Sprintf("function(%s) {\n%sthis.go$val = this;\n", strings.Join(params, ", "), strings.Repeat("\t", c.p.indentation+1))
		for i := 0; i < t.NumFields(); i++ {
			name := fieldName(t, i)
			constructor += fmt.Sprintf("%sthis.%s = %s_ !== undefined ? %s_ : %s;\n", strings.Repeat("\t", c.p.indentation+1), name, name, name, c.zeroValue(t.Field(i).Type()))
		}
		constructor += strings.Repeat("\t", c.p.indentation) + "}"
	case *types.Basic, *types.Array, *types.Slice, *types.Chan, *types.Signature, *types.Interface, *types.Pointer, *types.Map:
		size = sizes32.Sizeof(t)
	}
	c.Printf(`%s = go$newType(%d, "%s", "%s.%s", "%s", "%s", %s);`, lhs, size, typeKind(o.Type()), o.Pkg().Name(), o.Name(), o.Name(), o.Pkg().Path(), constructor)
}

func (c *funcContext) initType(o types.Object) {
	if _, isInterface := o.Type().Underlying().(*types.Interface); !isInterface {
		writeMethodSet := func(t types.Type) {
			methodSet := types.NewMethodSet(t)
			if methodSet.Len() == 0 {
				return
			}
			methods := make([]string, methodSet.Len())
			for i := range methods {
				method := methodSet.At(i)
				pkgPath := ""
				if !method.Obj().Exported() {
					pkgPath = method.Obj().Pkg().Path()
				}
				t := method.Type().(*types.Signature)
				embeddedIndex := -1
				if len(method.Index()) > 1 {
					embeddedIndex = method.Index()[0]
				}
				name := method.Obj().Name()
				if reservedKeywords[name] {
					name += "$"
				}
				methods[i] = fmt.Sprintf(`["%s", "%s", "%s", %s, %s, %t, %d]`, name, method.Obj().Name(), pkgPath, c.typeArray(t.Params()), c.typeArray(t.Results()), t.Variadic(), embeddedIndex)
			}
			c.Printf("%s.methods = [%s];", c.typeName(t), strings.Join(methods, ", "))
		}
		writeMethodSet(o.Type())
		writeMethodSet(types.NewPointer(o.Type()))
	}
	switch t := o.Type().Underlying().(type) {
	case *types.Array, *types.Chan, *types.Interface, *types.Map, *types.Pointer, *types.Slice, *types.Signature, *types.Struct:
		c.Printf("%s.init(%s);", c.objectName(o), c.initArgs(t))
	}
}

func (c *funcContext) initArgs(ty types.Type) string {
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
			fields[i] = fmt.Sprintf(`["%s", "%s", "%s", %s, %s]`, fieldName(t, i), name, pkgPath, c.typeName(field.Type()), encodeString(t.Tag(i)))
		}
		return fmt.Sprintf("[%s]", strings.Join(fields, ", "))
	default:
		panic("invalid type")
	}
}

func (c *funcContext) translateToplevelFunction(fun *ast.FuncDecl) []byte {
	o := c.p.info.Defs[fun.Name].(*types.Func)
	sig := o.Type().(*types.Signature)
	var recv *ast.Ident
	if fun.Recv != nil && fun.Recv.List[0].Names != nil {
		recv = fun.Recv.List[0].Names[0]
	}

	var joinedParams string
	primaryFunction := func(lhs string, fullName string) []byte {
		if fun.Body == nil {
			return []byte(fmt.Sprintf("\t%s = function() {\n\t\tthrow go$panic(\"Native function not implemented: %s\");\n\t};\n", lhs, fullName))
		}

		stmts := fun.Body.List
		if recv != nil {
			this := &This{}
			c.p.info.Types[this] = types.TypeAndValue{Type: sig.Recv().Type()}
			stmts = append([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{recv},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{this},
				},
			}, stmts...)
		}
		params, body := c.translateFunction(fun.Type, sig, stmts)
		joinedParams = strings.Join(params, ", ")
		return []byte(fmt.Sprintf("\t%s = function(%s) {\n%s\t};\n", lhs, joinedParams, string(body)))
	}

	if fun.Recv == nil {
		funName := c.objectName(o)
		lhs := funName
		if fun.Name.IsExported() || fun.Name.Name == "main" {
			lhs += " = go$pkg." + funName
		}
		return primaryFunction(lhs, funName)
	}

	recvType := sig.Recv().Type()
	ptr, isPointer := recvType.(*types.Pointer)
	namedRecvType, _ := recvType.(*types.Named)
	if isPointer {
		namedRecvType = ptr.Elem().(*types.Named)
	}
	typeName := c.objectName(namedRecvType.Obj())
	funName := fun.Name.Name
	if reservedKeywords[funName] {
		funName += "$"
	}

	code := bytes.NewBuffer(nil)

	if _, isStruct := namedRecvType.Underlying().(*types.Struct); isStruct {
		code.Write(primaryFunction(typeName+".Ptr.prototype."+funName, typeName+"."+funName))
		fmt.Fprintf(code, "\t%s.prototype.%s = function(%s) { return this.go$val.%s(%s); };\n", typeName, funName, joinedParams, funName, joinedParams)
		return code.Bytes()
	}

	if isPointer {
		if _, isArray := ptr.Elem().Underlying().(*types.Array); isArray {
			code.Write(primaryFunction(typeName+".prototype."+funName, typeName+"."+funName))
			fmt.Fprintf(code, "\tgo$ptrType(%s).prototype.%s = function(%s) { return (new %s(this.go$get())).%s(%s); };\n", typeName, funName, joinedParams, typeName, funName, joinedParams)
			return code.Bytes()
		}
		value := "this"
		if isWrapped(ptr.Elem()) {
			value = "this.go$val"
		}
		code.Write(primaryFunction(fmt.Sprintf("go$ptrType(%s).prototype.%s", typeName, funName), typeName+"."+funName))
		fmt.Fprintf(code, "\t%s.prototype.%s = function(%s) { var obj = %s; return (new (go$ptrType(%s))(function() { return obj; }, null)).%s(%s); };\n", typeName, funName, joinedParams, value, typeName, funName, joinedParams)
		return code.Bytes()
	}

	value := "this.go$get()"
	if isWrapped(recvType) {
		value = fmt.Sprintf("new %s(%s)", typeName, value)
	}
	code.Write(primaryFunction(typeName+".prototype."+funName, typeName+"."+funName))
	fmt.Fprintf(code, "\tgo$ptrType(%s).prototype.%s = function(%s) { return %s.%s(%s); };\n", typeName, funName, joinedParams, value, funName, joinedParams)
	return code.Bytes()
}

func (c *funcContext) translateFunction(t *ast.FuncType, sig *types.Signature, stmts []ast.Stmt) (params []string, body []byte) {
	vars := make(map[string]int, len(c.allVars))
	for k, v := range c.allVars {
		vars[k] = v
	}
	newFuncContext := &funcContext{
		p:           c.p,
		sig:         sig,
		allVars:     vars,
		flowDatas:   map[string]*flowData{"": &flowData{}},
		caseCounter: 1,
		labelCases:  make(map[string]int),
		hasGoto:     make(map[ast.Node]bool),
	}

	for _, param := range t.Params.List {
		for _, ident := range param.Names {
			if isBlank(ident) {
				params = append(params, newFuncContext.newVariable("param"))
				continue
			}
			params = append(params, newFuncContext.objectName(newFuncContext.p.info.Defs[ident]))
		}
	}

	body = newFuncContext.translateFunctionBody(1, stmts)
	return
}

func (c *funcContext) translateFunctionBody(indent int, stmts []ast.Stmt) []byte {
	v := gotoVisitor{c: c}
	for _, stmt := range stmts {
		ast.Walk(&v, stmt)
	}
	c.localVars = nil
	if c.flattened {
		c.localVars = append(c.localVars, "go$this = this", "go$args = arguments")
	}

	body := c.CatchOutput(indent, func() {
		if c.sig != nil && c.sig.Results().Len() != 0 && c.sig.Results().At(0).Name() != "" {
			c.resultNames = make([]ast.Expr, c.sig.Results().Len())
			for i := 0; i < c.sig.Results().Len(); i++ {
				result := c.sig.Results().At(i)
				name := result.Name()
				if result.Name() == "_" {
					name = "result"
				}
				c.Printf("%s = %s;", c.objectName(result), c.zeroValue(result.Type()))
				id := ast.NewIdent(name)
				c.p.info.Types[id] = types.TypeAndValue{Type: result.Type()}
				c.p.info.Uses[id] = result
				c.resultNames[i] = id
			}
		}

		printBody := func() {
			if c.flattened {
				c.Printf("/* */ var go$s = 0, go$f = function() { while (true) { switch (go$s) { case 0:")
				c.translateStmtList(stmts)
				c.WritePos(token.NoPos)
				c.Printf("/* */ } break; } }; return go$f();")
				return
			}
			c.translateStmtList(stmts)
			c.WritePos(token.NoPos)
		}

		v := hasDeferVisitor{}
		ast.Walk(&v, &ast.BlockStmt{List: stmts})
		if v.hasDefer {
			c.Printf("var go$deferred = [];")
			c.Printf("try {")
			c.Indent(func() {
				printBody()
			})
			c.Printf("} catch(go$err) {")
			c.Indent(func() {
				c.Printf("go$pushErr(go$err);")
				if c.sig != nil && c.resultNames == nil {
					switch c.sig.Results().Len() {
					case 0:
						// nothing
					case 1:
						c.Printf("return %s;", c.zeroValue(c.sig.Results().At(0).Type()))
					default:
						zeros := make([]string, c.sig.Results().Len())
						for i := range zeros {
							zeros[i] = c.zeroValue(c.sig.Results().At(i).Type())
						}
						c.Printf("return [%s];", strings.Join(zeros, ", "))
					}
				}
			})
			c.Printf("} finally {")
			c.Indent(func() {
				c.Printf("go$callDeferred(go$deferred);")
				if c.resultNames != nil {
					c.translateStmt(&ast.ReturnStmt{}, "")
				}
			})
			c.Printf("}")
			return
		}
		printBody()
	})

	if len(c.localVars) != 0 {
		body = append([]byte(fmt.Sprintf("%svar %s;\n", strings.Repeat("\t", c.p.indentation+indent), strings.Join(c.localVars, ", "))), body...)
	}
	return body
}

type hasDeferVisitor struct {
	hasDefer bool
}

func (v *hasDeferVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if v.hasDefer {
		return nil
	}
	switch node.(type) {
	case *ast.DeferStmt:
		v.hasDefer = true
		return nil
	case ast.Expr:
		return nil
	}
	return v
}

type gotoVisitor struct {
	c     *funcContext
	stack []ast.Node
}

func (v *gotoVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil {
		v.stack = v.stack[:len(v.stack)-1]
		return
	}
	switch n := node.(type) {
	case *ast.BranchStmt:
		if n.Tok == token.GOTO {
			v.c.flattened = true
			for _, n2 := range v.stack {
				v.c.hasGoto[n2] = true
			}
			if _, ok := v.c.labelCases[n.Label.String()]; !ok {
				v.c.labelCases[n.Label.String()] = v.c.caseCounter
				v.c.caseCounter++
			}
			return nil
		}
	case ast.Expr:
		return nil
	}
	v.stack = append(v.stack, node)
	return v
}
