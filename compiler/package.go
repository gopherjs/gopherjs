package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/gcexporter"
	"golang.org/x/tools/go/types"
)

type funcContext struct {
	p             *pkgContext
	name          string
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
	pkg           *types.Package
	info          *types.Info
	importContext *ImportContext
	comments      ast.CommentMap
	funcContexts  map[*types.Func]*funcContext
	pkgVars       map[string]string
	objectVars    map[types.Object]string
	escapingVars  map[types.Object]bool
	indentation   int
	dependencies  map[types.Object]bool
	minify        bool
}

type flowData struct {
	postStmt  func()
	beginCase int
	endCase   int
}

type ImportContext struct {
	Packages map[string]*types.Package
	Import   func(string) (*Archive, error)
}

func NewImportContext(importFunc func(string) (*Archive, error)) *ImportContext {
	return &ImportContext{
		Packages: map[string]*types.Package{"unsafe": types.Unsafe},
		Import:   importFunc,
	}
}

func Compile(importPath string, files []*ast.File, fileSet *token.FileSet, importContext *ImportContext, minify bool) (*Archive, error) {
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	var importError error
	var errList ErrorList
	var previousErr error
	config := &types.Config{
		Packages: importContext.Packages,
		Import: func(_ map[string]*types.Package, path string) (*types.Package, error) {
			if _, err := importContext.Import(path); err != nil {
				if importError == nil {
					importError = err
				}
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
	if importError != nil {
		return nil, importError
	}
	if errList != nil {
		if len(errList) > 10 {
			pos := token.NoPos
			if last, ok := errList[9].(types.Error); ok {
				pos = last.Pos
			}
			errList = append(errList[:10], types.Error{Fset: fileSet, Pos: pos, Msg: "too many errors"})
		}
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
		ImportPath: importPath,
		GcData:     gcData.Bytes(),
		FileSet:    encodedFileSet.Bytes(),
		Minified:   minify,
	}

	c := &funcContext{
		p: &pkgContext{
			pkg:           typesPkg,
			info:          info,
			importContext: importContext,
			comments:      make(ast.CommentMap),
			funcContexts:  make(map[*types.Func]*funcContext),
			pkgVars:       make(map[string]string),
			objectVars:    make(map[types.Object]string),
			escapingVars:  make(map[types.Object]bool),
			indentation:   1,
			dependencies:  make(map[types.Object]bool),
			minify:        minify,
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
	var importedPaths []string
	for _, importedPkg := range typesPkg.Imports() {
		varName := c.newVariableWithLevel(importedPkg.Name(), true, "")
		c.p.pkgVars[importedPkg.Path()] = varName
		archive.Imports = append(archive.Imports, &PkgImport{Path: importedPkg.Path(), VarName: varName})
		importedPaths = append(importedPaths, importedPkg.Path())
	}
	sort.Strings(importedPaths)
	for _, impPath := range importedPaths {
		id := c.newIdent(fmt.Sprintf(`%s.$init`, c.p.pkgVars[impPath]), types.NewSignature(nil, nil, nil, nil, false))
		call := &ast.CallExpr{Fun: id}
		c.blocking[call] = true
		c.flattened[call] = true
		archive.Declarations = append(archive.Declarations, &Decl{
			InitCode: removeWhitespace(c.CatchOutput(1, func() { c.translateStmt(&ast.ExprStmt{X: call}, "") }), minify),
		})
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
				context := c.p.analyzeFunction(sig, d.Body)
				context.name = d.Name.Name
				c.p.funcContexts[o] = context
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

	collectDependencies := func(self types.Object, f func()) []string {
		c.p.dependencies = make(map[types.Object]bool)
		f()
		var deps []string
		for dep := range c.p.dependencies {
			if dep != self {
				deps = append(deps, dep.Pkg().Path()+":"+dep.Name())
			}
		}
		sort.Strings(deps)
		return deps
	}

	// types
	for _, o := range toplevelTypes {
		typeName := c.objectName(o)
		var d Decl
		d.Vars = []string{typeName}
		d.DceFilters = []string{o.Name()}
		d.DceDeps = collectDependencies(o, func() {
			d.BodyCode = removeWhitespace(c.CatchOutput(0, func() { c.translateType(o, true) }), minify)
			d.InitCode = removeWhitespace(c.CatchOutput(1, func() { c.initType(o) }), minify)
		})
		archive.Declarations = append(archive.Declarations, &d)
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
			d.Vars = []string{c.objectName(o)}
		}
		if _, ok := varsWithInit[o]; !ok {
			d.DceDeps = collectDependencies(nil, func() {
				d.InitCode = removeWhitespace([]byte(fmt.Sprintf("\t\t%s = %s;\n", c.objectName(o), c.zeroValue(o.Type()))), minify)
			})
		}
		d.DceFilters = []string{o.Name()}
		archive.Declarations = append(archive.Declarations, &d)
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
			c.localVars = nil
			d.InitCode = removeWhitespace(c.CatchOutput(1, func() {
				ast.Walk(c, init.Rhs)
				c.translateStmt(&ast.AssignStmt{
					Lhs: lhs,
					Tok: token.DEFINE,
					Rhs: []ast.Expr{init.Rhs},
				}, "")
			}), minify)
			d.Vars = append(d.Vars, c.localVars...)
		})
		if len(init.Lhs) == 1 {
			v := hasCallVisitor{c.p.info, false}
			ast.Walk(&v, init.Rhs)
			if !v.hasCall {
				d.DceFilters = []string{init.Lhs[0].Name()}
			}
		}
		archive.Declarations = append(archive.Declarations, &d)
	}

	// functions
	var mainFunc *types.Func
	for _, fun := range functions {
		o := c.p.info.Defs[fun.Name].(*types.Func)
		context := c.p.funcContexts[o]
		d := Decl{
			FullName: o.FullName(),
			Blocking: len(context.blocking) != 0,
		}
		if fun.Recv == nil {
			d.Vars = []string{c.objectName(o)}
			switch o.Name() {
			case "main":
				mainFunc = o
			case "init":
				d.InitCode = removeWhitespace(c.CatchOutput(1, func() {
					id := c.newIdent("", types.NewSignature(nil, nil, nil, nil, false))
					c.p.info.Uses[id] = o
					call := &ast.CallExpr{Fun: id}
					c.Visit(call)
					c.translateStmt(&ast.ExprStmt{X: call}, "")
				}), minify)
			default:
				d.DceFilters = []string{o.Name()}
			}
		}
		if fun.Recv != nil {
			recvType := o.Type().(*types.Signature).Recv().Type()
			ptr, isPointer := recvType.(*types.Pointer)
			namedRecvType, _ := recvType.(*types.Named)
			if isPointer {
				namedRecvType = ptr.Elem().(*types.Named)
			}
			d.DceFilters = []string{namedRecvType.Obj().Name()}
			if !fun.Name.IsExported() {
				d.DceFilters = append(d.DceFilters, fun.Name.Name)
			}
		}

		d.DceDeps = collectDependencies(o, func() {
			d.BodyCode = removeWhitespace(c.translateToplevelFunction(fun, context), minify)
		})
		archive.Declarations = append(archive.Declarations, &d)
	}

	if typesPkg.Name() == "main" {
		if mainFunc == nil {
			return nil, fmt.Errorf("missing main function")
		}
		id := c.newIdent("", types.NewSignature(nil, nil, nil, nil, false))
		c.p.info.Uses[id] = mainFunc
		call := &ast.CallExpr{Fun: id}
		c.Visit(call)
		archive.Declarations = append(archive.Declarations, &Decl{
			InitCode: removeWhitespace(c.CatchOutput(1, func() { c.translateStmt(&ast.ExprStmt{X: call}, "") }), minify),
		})
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
	c.Printf(`%s = $newType(%d, %s, "%s.%s", "%s", "%s", %s);`, lhs, size, typeKind(o.Type()), o.Pkg().Name(), o.Name(), o.Name(), o.Pkg().Path(), constructor)
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
				methods[i] = fmt.Sprintf(`["%s", "%s", "%s", $funcType(%s), %d]`, name, method.Obj().Name(), pkgPath, c.initArgs(t), embeddedIndex)
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
			methods[i] = fmt.Sprintf(`["%s", "%s", "%s", $funcType(%s)]`, method.Name(), method.Name(), pkgPath, c.initArgs(method.Type()))
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
	primaryFunction := func(lhs string) []byte {
		if fun.Body == nil {
			return []byte(fmt.Sprintf("\t%s = function() {\n\t\t$panic(\"Native function not implemented: %s\");\n\t};\n", lhs, o.FullName()))
		}

		stmts := fun.Body.List
		if recv != nil && !isBlank(recv) {
			stmts = append([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{recv},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{c.setType(&this{}, sig.Recv().Type())},
				},
			}, stmts...)
		}
		params, body := context.translateFunction(fun.Type, stmts, c.allVars)
		joinedParams = strings.Join(params, ", ")
		return []byte(fmt.Sprintf("\t%s = function(%s) {\n%s\t};\n", lhs, joinedParams, string(body)))
	}

	if fun.Recv == nil {
		lhs := c.objectName(o)
		if fun.Name.IsExported() {
			lhs += " = $pkg." + fun.Name.Name
		}
		return primaryFunction(lhs)
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
		code.Write(primaryFunction(typeName + ".Ptr.prototype." + funName))
		fmt.Fprintf(code, "\t%s.prototype.%s = function(%s) { return this.$val.%s(%s); };\n", typeName, funName, joinedParams, funName, joinedParams)
		return code.Bytes()
	}

	if isPointer {
		if _, isArray := ptr.Elem().Underlying().(*types.Array); isArray {
			code.Write(primaryFunction(typeName + ".prototype." + funName))
			fmt.Fprintf(code, "\t$ptrType(%s).prototype.%s = function(%s) { return (new %s(this.$get())).%s(%s); };\n", typeName, funName, joinedParams, typeName, funName, joinedParams)
			return code.Bytes()
		}
		return primaryFunction(fmt.Sprintf("$ptrType(%s).prototype.%s", typeName, funName))
	}

	value := "this.$get()"
	if isWrapped(recvType) {
		value = fmt.Sprintf("new %s(%s)", typeName, value)
	}
	code.Write(primaryFunction(typeName + ".prototype." + funName))
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
		lookForComment := func() {
			for i := len(c.analyzeStack) - 1; i >= 0; i-- {
				n2 := c.analyzeStack[i]
				for _, group := range c.p.comments[n2] {
					for _, comment := range group.List {
						if comment.Text == "//gopherjs:blocking" {
							c.markBlocking(c.analyzeStack)
							return
						}
					}
				}
				if _, ok := n2.(ast.Stmt); ok {
					break
				}
			}
		}
		callTo := func(obj types.Object) {
			switch o := obj.(type) {
			case *types.Func:
				if recv := o.Type().(*types.Signature).Recv(); recv != nil {
					if _, ok := recv.Type().Underlying().(*types.Interface); ok {
						lookForComment()
						return
					}
				}
				if o.Pkg() != c.p.pkg {
					fullName := o.FullName()
					archive, err := c.p.importContext.Import(o.Pkg().Path())
					if err != nil {
						panic(err)
					}
					for _, d := range archive.Declarations {
						if string(d.FullName) == fullName {
							if d.Blocking {
								c.markBlocking(c.analyzeStack)
							}
							return
						}
					}
					return
				}
				if context, ok := c.p.funcContexts[o]; ok && len(context.blocking) != 0 {
					c.markBlocking(c.analyzeStack)
					return
				}
				stack := make([]ast.Node, len(c.analyzeStack))
				copy(stack, c.analyzeStack)
				c.localCalls[o] = append(c.localCalls[o], stack)
			case *types.Var:
				lookForComment()
			}
		}
		switch f := n.Fun.(type) {
		case *ast.Ident:
			callTo(c.p.info.Uses[f])
		case *ast.SelectorExpr:
			callTo(c.p.info.Uses[f.Sel])
		default:
			lookForComment()
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
		for _, s := range n.Body.List {
			if s.(*ast.CommClause).Comm == nil { // default clause
				return c
			}
		}
		c.markBlocking(c.analyzeStack)
	case *ast.CommClause:
		for _, s := range n.Body {
			ast.Walk(c, s)
		}
		return nil
	case *ast.DeferStmt:
		c.hasDefer = true
		if funcLit, ok := n.Call.Fun.(*ast.FuncLit); ok {
			ast.Walk(c, funcLit.Body)
		}
	case *ast.FuncLit, *ast.GoStmt:
		return nil
	}
	return c
}

func (c *funcContext) markBlocking(stack []ast.Node) {
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
		if len(param.Names) == 0 {
			params = append(params, c.newVariable("param"))
			continue
		}
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
				c.p.objectVars[result] = c.newVariableWithLevel(name, false, c.zeroValue(result.Type()))
				id := ast.NewIdent(name)
				c.p.info.Uses[id] = result
				c.resultNames[i] = c.setType(id, result.Type())
			}
		}

		var prefix, suffix string

		if len(c.blocking) != 0 {
			c.localVars = append(c.localVars, "$r")
			f := "$f"
			if c.name != "" && !c.p.minify {
				f = "$blocking_" + c.name
			}
			prefix = prefix + fmt.Sprintf(" if(!$b) { $nonblockingCall(); }; var %s = function() {", f)
			suffix = fmt.Sprintf(" }; %s.$blocking = true; return %s;", f, f) + suffix
		}

		if c.hasDefer {
			c.localVars = append(c.localVars, "$deferred = []", "$err = null")
			prefix = prefix + " try { $deferFrames.push($deferred);"
			deferSuffix := " } catch(err) { $err = err;"
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
			deferSuffix += " } finally { $deferFrames.pop();"
			if len(c.blocking) != 0 {
				deferSuffix += " if ($curGoroutine.asleep && !$jumpToDefer) { throw null; } $s = -1;"
			}
			deferSuffix += " $callDeferred($deferred, $err);"
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
			prefix = prefix + " s: while (true) { switch ($s) { case 0:"
			suffix = " case -1: } return; }" + suffix
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
