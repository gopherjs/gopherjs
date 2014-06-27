package compiler

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
	hasDefer      bool
	flattened     map[ast.Node]bool
	blocking      map[ast.Node]bool
	caseCounter   int
	labelCases    map[string]int
	output        []byte
	delayedOutput []byte
	analyzeStack  []ast.Node
	localCalls    map[*types.Func][][]ast.Node
}

type pkgContext struct {
	pkg          *types.Package
	info         *types.Info
	comments     ast.CommentMap
	funcContexts map[*types.Func]*funcContext
	pkgVars      map[string]string
	objectVars   map[types.Object]string
	escapingVars map[types.Object]bool
	indentation  int
	dependencies map[types.Object]bool
	minify       bool
}

type flowData struct {
	postStmt  func()
	beginCase int
	endCase   int
}

func Compile(importPath string, files []*ast.File, fileSet *token.FileSet, importContext *ImportContext, minify bool) (*Archive, error) {
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	var errList ErrorList
	var previousErr error
	config := &types.Config{
		Packages: importContext.Packages,
		Import: func(_ map[string]*types.Package, path string) (*types.Package, error) {
			if _, err := importContext.Import(path); err != nil {
				return nil, err
			}
			return importContext.Packages[path], nil
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
	importContext.Packages[importPath] = typesPkg

	gcData := bytes.NewBuffer(nil)
	gcexporter.Write(typesPkg, gcData, sizes32)
	encodedFileSet := bytes.NewBuffer(nil)
	if err := fileSet.Write(json.NewEncoder(encodedFileSet).Encode); err != nil {
		return nil, err
	}
	archive := &Archive{
		ImportPath:   PkgPath(importPath),
		GcData:       gcData.Bytes(),
		Dependencies: []PkgPath{PkgPath("github.com/gopherjs/gopherjs/js"), PkgPath("runtime")}, // all packages depend on those
		FileSet:      encodedFileSet.Bytes(),
		Minified:     minify,
	}

	c := &funcContext{
		p: &pkgContext{
			pkg:          typesPkg,
			info:         info,
			comments:     make(ast.CommentMap),
			funcContexts: make(map[*types.Func]*funcContext),
			pkgVars:      make(map[string]string),
			objectVars:   make(map[types.Object]string),
			escapingVars: make(map[types.Object]bool),
			indentation:  1,
			dependencies: make(map[types.Object]bool),
			minify:       minify,
		},
		allVars:     make(map[string]int),
		flowDatas:   map[string]*flowData{"": &flowData{}},
		flattened:   make(map[ast.Node]bool),
		blocking:    make(map[ast.Node]bool),
		caseCounter: 1,
		labelCases:  make(map[string]int),
		localCalls:  make(map[*types.Func][][]ast.Node),
	}
	for name := range reservedKeywords {
		c.allVars[name] = 1
	}

	// imports
	for _, importedPkg := range typesPkg.Imports() {
		varName := c.newVariableWithLevel(importedPkg.Name(), true)
		c.p.pkgVars[importedPkg.Path()] = varName
		archive.Imports = append(archive.Imports, PkgImport{Path: PkgPath(importedPkg.Path()), VarName: varName})
	}

	var functions []*ast.FuncDecl
	var toplevelTypes []*types.TypeName
	var vars []*types.Var
	for _, file := range files {
		for k, v := range ast.NewCommentMap(fileSet, file, file.Comments) {
			c.p.comments[k] = v
		}

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
				o := c.p.info.Defs[d.Name].(*types.Func)
				c.p.funcContexts[o] = c.p.analyzeFunction(sig, d.Body)
				if sig.Recv() == nil {
					c.objectName(o) // register toplevel name
				}
				if !isBlank(d.Name) {
					functions = append(functions, d)
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

	for {
		done := true
		for _, context := range c.p.funcContexts {
			for obj, calls := range context.localCalls {
				if len(c.p.funcContexts[obj].blocking) != 0 {
					for _, call := range calls {
						context.markBlocking(call)
					}
					delete(context.localCalls, obj)
					done = false
				}
			}
		}
		if done {
			break
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

	// types
	for _, o := range toplevelTypes {
		typeName := c.objectName(o)
		var d Decl
		d.Var = typeName
		d.DceFilters = []DepId{DepId(o.Name())}
		d.DceDeps = collectDependencies(o, func() {
			d.BodyCode = removeWhitespace(c.CatchOutput(0, func() { c.translateType(o, true) }), minify)
			d.InitCode = removeWhitespace(c.CatchOutput(1, func() { c.initType(o) }), minify)
		})
		archive.Declarations = append(archive.Declarations, d)
	}

	// variables
	varsWithInit := make(map[*types.Var]bool)
	for _, init := range c.p.info.InitOrder {
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
				d.InitCode = removeWhitespace([]byte(fmt.Sprintf("\t\t%s = %s;\n", c.objectName(o), value)), minify)
			})
		}
		d.DceFilters = []DepId{DepId(o.Name())}
		archive.Declarations = append(archive.Declarations, d)
	}
	for _, init := range c.p.info.InitOrder {
		lhs := make([]ast.Expr, len(init.Lhs))
		for i, o := range init.Lhs {
			ident := ast.NewIdent(o.Name())
			c.p.info.Defs[ident] = o
			lhs[i] = c.setType(ident, o.Type())
			varsWithInit[o] = true
		}
		var d Decl
		d.DceDeps = collectDependencies(nil, func() {
			ast.Walk(c, init.Rhs)
			d.InitCode = removeWhitespace(c.translateFunctionBody([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: lhs,
					Tok: token.DEFINE,
					Rhs: []ast.Expr{init.Rhs},
				},
			}), minify)
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

	// functions
	for _, fun := range functions {
		var d Decl
		o := c.p.info.Defs[fun.Name].(*types.Func)
		funName := fun.Name.Name
		if fun.Recv == nil {
			d.Var = c.objectName(o)
			if o.Name() != "main" && o.Name() != "init" {
				d.DceFilters = []DepId{DepId(o.Name())}
			}
			if o.Name() == "init" {
				d.InitCode = removeWhitespace([]byte(fmt.Sprintf("\t\t%s();\n", d.Var)), minify)
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
			d.BodyCode = removeWhitespace(c.translateToplevelFunction(fun, c.p.funcContexts[o]), minify)
		})
		archive.Declarations = append(archive.Declarations, d)
		if strings.HasPrefix(fun.Name.String(), "Test") {
			archive.Tests = append(archive.Tests, fun.Name.String())
		}
	}

	var importedPaths []string
	for _, imp := range typesPkg.Imports() {
		importedPaths = append(importedPaths, imp.Path())
	}
	sort.Strings(importedPaths)
	for _, impPath := range importedPaths {
		impOutput, err := importContext.Import(impPath)
		if err != nil {
			return nil, err
		}
		archive.AddDependenciesOf(impOutput)
	}

	return archive, nil
}

func (c *funcContext) translateType(o *types.TypeName, toplevel bool) {
	typeName := c.objectName(o)
	lhs := typeName
	if toplevel {
		lhs += " = $pkg." + o.Name()
	}
	size := int64(0)
	constructor := "null"
	switch t := o.Type().Underlying().(type) {
	case *types.Struct:
		params := make([]string, t.NumFields())
		for i := 0; i < t.NumFields(); i++ {
			params[i] = fieldName(t, i) + "_"
		}
		constructor = fmt.Sprintf("function(%s) {\n%sthis.$val = this;\n", strings.Join(params, ", "), strings.Repeat("\t", c.p.indentation+1))
		for i := 0; i < t.NumFields(); i++ {
			name := fieldName(t, i)
			constructor += fmt.Sprintf("%sthis.%s = %s_ !== undefined ? %s_ : %s;\n", strings.Repeat("\t", c.p.indentation+1), name, name, name, c.zeroValue(t.Field(i).Type()))
		}
		constructor += strings.Repeat("\t", c.p.indentation) + "}"
	case *types.Basic, *types.Array, *types.Slice, *types.Chan, *types.Signature, *types.Interface, *types.Pointer, *types.Map:
		size = sizes32.Sizeof(t)
	}
	c.Printf(`%s = $newType(%d, "%s", "%s.%s", "%s", "%s", %s);`, lhs, size, typeKind(o.Type()), o.Pkg().Name(), o.Name(), o.Name(), o.Pkg().Path(), constructor)
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
				methods[i] = fmt.Sprintf(`["%s", "%s", "%s", %s, %d]`, name, method.Obj().Name(), pkgPath, c.initArgs(t), embeddedIndex)
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
			methods[i] = fmt.Sprintf(`["%s", "%s", "%s", %s]`, method.Name(), method.Name(), pkgPath, c.initArgs(method.Type()))
		}
		return fmt.Sprintf("[%s]", strings.Join(methods, ", "))
	case *types.Map:
		return fmt.Sprintf("%s, %s", c.typeName(t.Key()), c.typeName(t.Elem()))
	case *types.Pointer:
		return fmt.Sprintf("%s", c.typeName(t.Elem()))
	case *types.Slice:
		return fmt.Sprintf("%s", c.typeName(t.Elem()))
	case *types.Signature:
		params := make([]string, t.Params().Len())
		for i := range params {
			params[i] = c.typeName(t.Params().At(i).Type())
		}
		results := make([]string, t.Results().Len())
		for i := range results {
			results[i] = c.typeName(t.Results().At(i).Type())
		}
		return fmt.Sprintf("[%s], [%s], %t", strings.Join(params, ", "), strings.Join(results, ", "), t.Variadic())
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

func (c *funcContext) translateToplevelFunction(fun *ast.FuncDecl, context *funcContext) []byte {
	o := c.p.info.Defs[fun.Name].(*types.Func)
	sig := o.Type().(*types.Signature)
	var recv *ast.Ident
	if fun.Recv != nil && fun.Recv.List[0].Names != nil {
		recv = fun.Recv.List[0].Names[0]
	}

	var joinedParams string
	primaryFunction := func(lhs string, fullName string) []byte {
		if fun.Body == nil {
			return []byte(fmt.Sprintf("\t%s = function() {\n\t\tthrow $panic(\"Native function not implemented: %s\");\n\t};\n", lhs, fullName))
		}

		stmts := fun.Body.List
		if recv != nil && !isBlank(recv) {
			stmts = append([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{recv},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{c.setType(&This{}, sig.Recv().Type())},
				},
			}, stmts...)
		}
		params, body := context.translateFunction(fun.Type, stmts, c.allVars)
		joinedParams = strings.Join(params, ", ")
		return []byte(fmt.Sprintf("\t%s = function(%s) {\n%s\t};\n", lhs, joinedParams, string(body)))
	}

	if fun.Recv == nil {
		funName := c.objectName(o)
		lhs := funName
		if fun.Name.IsExported() || fun.Name.Name == "main" {
			lhs += " = $pkg." + fun.Name.Name
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
		fmt.Fprintf(code, "\t%s.prototype.%s = function(%s) { return this.$val.%s(%s); };\n", typeName, funName, joinedParams, funName, joinedParams)
		return code.Bytes()
	}

	if isPointer {
		if _, isArray := ptr.Elem().Underlying().(*types.Array); isArray {
			code.Write(primaryFunction(typeName+".prototype."+funName, typeName+"."+funName))
			fmt.Fprintf(code, "\t$ptrType(%s).prototype.%s = function(%s) { return (new %s(this.$get())).%s(%s); };\n", typeName, funName, joinedParams, typeName, funName, joinedParams)
			return code.Bytes()
		}
		return primaryFunction(fmt.Sprintf("$ptrType(%s).prototype.%s", typeName, funName), typeName+"."+funName)
	}

	value := "this.$get()"
	if isWrapped(recvType) {
		value = fmt.Sprintf("new %s(%s)", typeName, value)
	}
	code.Write(primaryFunction(typeName+".prototype."+funName, typeName+"."+funName))
	fmt.Fprintf(code, "\t$ptrType(%s).prototype.%s = function(%s) { return %s.%s(%s); };\n", typeName, funName, joinedParams, value, funName, joinedParams)
	return code.Bytes()
}

func (c *pkgContext) analyzeFunction(sig *types.Signature, body *ast.BlockStmt) *funcContext {
	newFuncContext := &funcContext{
		p:           c,
		sig:         sig,
		flowDatas:   map[string]*flowData{"": &flowData{}},
		flattened:   make(map[ast.Node]bool),
		blocking:    make(map[ast.Node]bool),
		caseCounter: 1,
		labelCases:  make(map[string]int),
		localCalls:  make(map[*types.Func][][]ast.Node),
	}
	if body != nil {
		ast.Walk(newFuncContext, body)
	}
	return newFuncContext
}

func (c *funcContext) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		c.analyzeStack = c.analyzeStack[:len(c.analyzeStack)-1]
		return nil
	}
	c.analyzeStack = append(c.analyzeStack, node)

	switch n := node.(type) {
	case *ast.BranchStmt:
		if n.Tok == token.GOTO {
			for _, n2 := range c.analyzeStack {
				c.flattened[n2] = true
			}
			if _, ok := c.labelCases[n.Label.String()]; !ok {
				c.labelCases[n.Label.String()] = c.caseCounter
				c.caseCounter++
			}
		}
	case *ast.CallExpr:
		callTo := func(o *types.Func) {
			if recv := o.Type().(*types.Signature).Recv(); recv != nil {
				if _, ok := recv.Type().Underlying().(*types.Interface); ok {
					for i := len(c.analyzeStack) - 1; i >= 0; i-- {
						n2 := c.analyzeStack[i]
						for _, group := range c.p.comments[n2] {
							for _, comment := range group.List {
								if comment.Text == "//go:blocking" {
									c.markBlocking(c.analyzeStack)
									return
								}
							}
						}
						if _, ok := n2.(ast.Stmt); ok {
							break
						}
					}
					return
				}
			}
			if o.Pkg() != c.p.pkg {
				return
			}
			if context, ok := c.p.funcContexts[o]; ok && len(context.blocking) != 0 {
				c.markBlocking(c.analyzeStack)
				return
			}
			stack := make([]ast.Node, len(c.analyzeStack))
			copy(stack, c.analyzeStack)
			c.localCalls[o] = append(c.localCalls[o], stack)
		}
		switch f := n.Fun.(type) {
		case *ast.Ident:
			if o, ok := c.p.info.Uses[f].(*types.Func); ok {
				callTo(o)
			}
		case *ast.SelectorExpr:
			if o, ok := c.p.info.Uses[f.Sel].(*types.Func); ok {
				if isJsPackage(o.Pkg()) && o.Name() == "BlockAfter" {
					c.markBlocking(c.analyzeStack)
				}
				callTo(o)
			}
		}
	case *ast.SendStmt:
		c.markBlocking(c.analyzeStack)
	case *ast.UnaryExpr:
		if n.Op == token.ARROW {
			c.markBlocking(c.analyzeStack)
		}
	case *ast.RangeStmt:
		if _, ok := c.p.info.Types[n.X].Type.Underlying().(*types.Chan); ok {
			c.markBlocking(c.analyzeStack)
		}
	case *ast.SelectStmt:
		c.markBlocking(c.analyzeStack)
	case *ast.CommClause:
		for _, s := range n.Body {
			ast.Walk(c, s)
		}
		return nil
	case *ast.DeferStmt:
		c.hasDefer = true
	case *ast.FuncLit:
		return nil
	}
	return c
}

func (c *funcContext) markBlocking(stack []ast.Node) {
	if !GoroutinesSupport {
		return
	}
	c.blocking[stack[len(stack)-1]] = true
	for _, n := range stack {
		c.flattened[n] = true
	}
}

func (c *funcContext) translateFunction(typ *ast.FuncType, stmts []ast.Stmt, outerVars map[string]int) ([]string, []byte) {
	c.allVars = make(map[string]int, len(outerVars))
	for k, v := range outerVars {
		c.allVars[k] = v
	}

	var params []string
	for _, param := range typ.Params.List {
		for _, ident := range param.Names {
			if isBlank(ident) {
				params = append(params, c.newVariable("param"))
				continue
			}
			params = append(params, c.objectName(c.p.info.Defs[ident]))
		}
	}
	if len(c.blocking) != 0 {
		params = append(params, "$b")
	}

	return params, c.translateFunctionBody(stmts)
}

func (c *funcContext) translateFunctionBody(stmts []ast.Stmt) []byte {
	c.localVars = nil
	if len(c.flattened) != 0 {
		c.localVars = append(c.localVars, "$this = this", "$args = arguments")
	}

	body := c.CatchOutput(1, func() {
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
				c.p.info.Uses[id] = result
				c.resultNames[i] = c.setType(id, result.Type())
			}
		}

		var prefix, suffix string

		if len(c.blocking) != 0 {
			c.localVars = append(c.localVars, "$r")
			prefix = prefix + "if(!$b) { $notSupported($nonblockingCall); }; return function() {"
			suffix = " };" + suffix
		}

		if c.hasDefer {
			c.localVars = append(c.localVars, "$deferred = []")
			prefix = prefix + " try {"
			deferSuffix := " } catch($err) { $pushErr($err);"
			if c.sig != nil && c.resultNames == nil {
				switch c.sig.Results().Len() {
				case 0:
					// nothing
				case 1:
					deferSuffix += fmt.Sprintf(" return %s;", c.zeroValue(c.sig.Results().At(0).Type()))
				default:
					zeros := make([]string, c.sig.Results().Len())
					for i := range zeros {
						zeros[i] = c.zeroValue(c.sig.Results().At(i).Type())
					}
					deferSuffix += fmt.Sprintf(" return [%s];", strings.Join(zeros, ", "))
				}
			}
			deferSuffix += " } finally { $callDeferred($deferred);"
			if c.resultNames != nil {
				switch len(c.resultNames) {
				case 1:
					deferSuffix += fmt.Sprintf(" return %s;", c.translateExpr(c.resultNames[0]))
				default:
					values := make([]string, len(c.resultNames))
					for i, result := range c.resultNames {
						values[i] = c.translateExpr(result).String()
					}
					deferSuffix += fmt.Sprintf(" return [%s];", strings.Join(values, ", "))
				}
			}
			deferSuffix += " }"
			suffix = deferSuffix + suffix
		}

		if len(c.flattened) != 0 {
			c.localVars = append(c.localVars, "$s = 0")
			prefix = prefix + " while (true) { switch ($s) { case 0:"
			suffix = " } return; }" + suffix
		}

		if prefix != "" {
			c.Printf("/* */%s", prefix)
		}
		c.translateStmtList(stmts)
		if suffix != "" {
			c.Printf("/* */%s", suffix)
		}
	})

	if len(c.localVars) != 0 {
		body = append([]byte(fmt.Sprintf("%svar %s;\n", strings.Repeat("\t", c.p.indentation+1), strings.Join(c.localVars, ", "))), body...)
	}
	return body
}
