package translator

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"github.com/gopherjs/gopherjs/gcexporter"
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

var reservedKeywords = make(map[string]bool)

func init() {
	for _, keyword := range []string{"abstract", "arguments", "boolean", "break", "byte", "case", "catch", "char", "class", "const", "continue", "debugger", "default", "delete", "do", "double", "else", "enum", "eval", "export", "extends", "false", "final", "finally", "float", "for", "function", "goto", "if", "implements", "import", "in", "instanceof", "int", "interface", "let", "long", "native", "new", "package", "private", "protected", "public", "return", "short", "static", "super", "switch", "synchronized", "this", "throw", "throws", "transient", "true", "try", "typeof", "var", "void", "volatile", "while", "with", "yield"} {
		reservedKeywords[keyword] = true
	}
}

type ErrorList []error

func (err ErrorList) Error() string {
	return err[0].Error()
}

type pkgContext struct {
	pkg           *types.Package
	info          *types.Info
	pkgVars       map[string]string
	objectVars    map[types.Object]string
	output        []byte
	delayedOutput []byte
	indentation   int
	f             *funcContext
}

type funcContext struct {
	sig          *types.Signature
	allVars      map[string]int
	localVars    []string
	resultNames  []ast.Expr
	flowDatas    map[string]*flowData
	escapingVars []string
	flattened    bool
	caseCounter  int
	labelCases   map[string]int
	hasGoto      map[ast.Node]bool
}

type flowData struct {
	postStmt  string
	beginCase int
	endCase   int
}

func (c *pkgContext) Write(b []byte) (int, error) {
	c.output = append(c.output, b...)
	return len(b), nil
}

func (c *pkgContext) Printf(format string, values ...interface{}) {
	c.Write([]byte(strings.Repeat("\t", c.indentation)))
	fmt.Fprintf(c, format, values...)
	c.Write([]byte{'\n'})
	c.Write(c.delayedOutput)
	c.delayedOutput = nil
}

func (c *pkgContext) PrintCond(cond bool, onTrue, onFalse string) {
	if !cond {
		c.Printf("/* %s */ %s", strings.Replace(onTrue, "*/", "<star>/", -1), onFalse)
		return
	}
	c.Printf("%s", onTrue)
}

func (c *pkgContext) Indent(f func()) {
	c.indentation++
	f()
	c.indentation--
}

func (c *pkgContext) CatchOutput(indent int, f func()) []byte {
	origoutput := c.output
	c.output = nil
	c.indentation += indent
	f()
	catched := c.output
	c.output = origoutput
	c.indentation -= indent
	return catched
}

func (c *pkgContext) Delayed(f func()) {
	c.delayedOutput = c.CatchOutput(0, f)
}

func TranslatePackage(importPath string, files []*ast.File, fileSet *token.FileSet, importPkg func(string) (*Archive, error)) (*Archive, error) {
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
			if _, err := importPkg(path); err != nil {
				return nil, err
			}
			return typesPackages[path], nil
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

	c := &pkgContext{
		pkg:        typesPkg,
		info:       info,
		pkgVars:    make(map[string]string),
		objectVars: make(map[types.Object]string),
		f: &funcContext{
			allVars:     make(map[string]int),
			flowDatas:   map[string]*flowData{"": &flowData{}},
			caseCounter: 1,
			labelCases:  make(map[string]int),
			hasGoto:     make(map[ast.Node]bool),
		},
	}
	for name := range reservedKeywords {
		c.f.allVars[name] = 1
	}

	var functions []*ast.FuncDecl
	var initStmts []ast.Stmt
	var toplevelTypes []*types.TypeName
	var vars []*types.Var
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
				if sig.Recv() == nil {
					c.objectName(c.info.Objects[d.Name]) // register toplevel name
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					for _, spec := range d.Specs {
						o := c.info.Objects[spec.(*ast.TypeSpec).Name].(*types.TypeName)
						toplevelTypes = append(toplevelTypes, o)
						c.objectName(o) // register toplevel name
					}
				case token.VAR:
					for _, spec := range d.Specs {
						for _, name := range spec.(*ast.ValueSpec).Names {
							if !isBlank(name) {
								o := c.info.Objects[name].(*types.Var)
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

	gcData := bytes.NewBuffer(nil)
	gcexporter.Write(typesPkg, gcData, sizes32)
	archive := &Archive{
		ImportPath:   importPath,
		GcData:       gcData.Bytes(),
		Dependencies: []string{"runtime"}, // all packages depend on runtime
	}

	// imports
	for _, importedPkg := range typesPkg.Imports() {
		varName := c.newVariable(importedPkg.Name())
		c.pkgVars[importedPkg.Path()] = varName
		archive.Imports = append(archive.Imports, Import{Path: importedPkg.Path(), VarName: varName})
	}

	// types
	for _, o := range toplevelTypes {
		typeName := c.objectName(o)
		archive.Declarations = append(archive.Declarations, Decl{
			Var:      typeName,
			BodyCode: c.CatchOutput(1, func() { c.translateType(o, true) }),
			InitCode: c.CatchOutput(2, func() { c.initType(o) }),
		})
	}

	// functions
	natives := pkgNatives[importPath]
	for _, fun := range functions {
		var d Decl
		o := c.info.Objects[fun.Name].(*types.Func)
		funName := fun.Name.Name
		if fun.Recv == nil {
			d.Var = c.objectName(o)
		}
		if fun.Recv != nil {
			recvType := o.Type().(*types.Signature).Recv().Type()
			ptr, isPointer := recvType.(*types.Pointer)
			namedRecvType, _ := recvType.(*types.Named)
			if isPointer {
				namedRecvType = ptr.Elem().(*types.Named)
			}
			funName = namedRecvType.Obj().Name() + "." + funName
		}

		native := natives[funName]
		delete(natives, funName)

		c.Indent(func() {
			d.BodyCode = c.translateToplevelFunction(fun, native)
		})
		archive.Declarations = append(archive.Declarations, d)
		if strings.HasPrefix(fun.Name.String(), "Test") {
			archive.Tests = append(archive.Tests, fun.Name.String())
		}
	}

	// variables
	initOrder := c.info.InitOrder

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
			value := c.zeroValue(o.Type())
			if native, ok := natives[o.Name()]; ok {
				value = native
				delete(natives, o.Name())
			}
			d.InitCode = []byte(fmt.Sprintf("\t\t%s = %s;\n", c.objectName(o), value))
		}
		archive.Declarations = append(archive.Declarations, d)
	}
	for _, init := range initOrder {
		lhs := make([]ast.Expr, len(init.Lhs))
		for i, o := range init.Lhs {
			ident := ast.NewIdent(o.Name())
			c.info.Types[ident] = types.TypeAndValue{Type: o.Type()}
			c.info.Objects[ident] = o
			lhs[i] = ident
			varsWithInit[o] = true
		}
		var d Decl
		d.InitCode = c.translateFunctionBody(2, []ast.Stmt{
			&ast.AssignStmt{
				Lhs: lhs,
				Tok: token.DEFINE,
				Rhs: []ast.Expr{init.Rhs},
			},
		})
		archive.Declarations = append(archive.Declarations, d)
	}

	// natives
	archive.Declarations = append(archive.Declarations, Decl{
		BodyCode: []byte(natives["toplevel"]),
	})
	delete(natives, "toplevel")

	// init functions
	archive.Declarations = append(archive.Declarations, Decl{
		InitCode: c.translateFunctionBody(2, initStmts),
	})

	if len(natives) != 0 {
		panic("not all natives used: " + importPath)
	}

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

func (c *pkgContext) translateType(o *types.TypeName, toplevel bool) {
	typeName := c.objectName(o)
	lhs := typeName
	if toplevel {
		lhs += " = go$pkg." + typeName
	}
	size := int64(0)
	switch t := o.Type().Underlying().(type) {
	case *types.Struct:
		params := make([]string, t.NumFields())
		for i := 0; i < t.NumFields(); i++ {
			params[i] = fieldName(t, i) + "_"
		}
		c.Printf(`%s = go$newType(0, "Struct", "%s.%s", "%s", "%s", function(%s) {`, lhs, o.Pkg().Name(), o.Name(), o.Name(), o.Pkg().Path(), strings.Join(params, ", "))
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
				methods := types.NewMethodSet(fieldType)
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
	c.Printf(`%s = go$newType(%d, "%s", "%s.%s", "%s", "%s", null);`, lhs, size, typeKind(o.Type()), o.Pkg().Name(), o.Name(), o.Name(), o.Pkg().Path())
}

func (c *pkgContext) initType(o types.Object) {
	switch t := o.Type().Underlying().(type) {
	case *types.Array, *types.Chan, *types.Interface, *types.Map, *types.Pointer, *types.Slice, *types.Signature, *types.Struct:
		c.Printf("%s.init(%s);", c.objectName(o), c.initArgs(t))
	}
	if _, isInterface := o.Type().Underlying().(*types.Interface); !isInterface {
		writeMethodSet := func(t types.Type) {
			methodSet := types.NewMethodSet(t)
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

func (c *pkgContext) initArgs(ty types.Type) string {
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

func (c *pkgContext) translateToplevelFunction(fun *ast.FuncDecl, native string) []byte {
	sig := c.info.Objects[fun.Name].(*types.Func).Type().(*types.Signature)
	var recv *ast.Ident
	if fun.Recv != nil && fun.Recv.List[0].Names != nil {
		recv = fun.Recv.List[0].Names[0]
	}

	var joinedParams string
	primaryFunction := func(lhs string, fullName string) []byte {
		if native != "" {
			return []byte(fmt.Sprintf("\t%s = %s;\n", lhs, native))
		}

		if fun.Body == nil {
			return []byte(fmt.Sprintf("\t%s = function() {\n\t\tthrow go$panic(\"Native function not implemented: %s\");\n\t};\n", lhs, fullName))
		}

		stmts := fun.Body.List
		if recv != nil {
			this := &This{}
			c.info.Types[this] = types.TypeAndValue{Type: sig.Recv().Type()}
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
		funName := c.objectName(c.info.Objects[fun.Name])
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

func (c *pkgContext) translateFunction(t *ast.FuncType, sig *types.Signature, stmts []ast.Stmt) (params []string, body []byte) {
	outerFuncContext := c.f
	vars := make(map[string]int, len(c.f.allVars))
	for k, v := range c.f.allVars {
		vars[k] = v
	}
	c.f = &funcContext{
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
				params = append(params, c.newVariable("param"))
				continue
			}
			params = append(params, c.objectName(c.info.Objects[ident]))
		}
	}

	body = c.translateFunctionBody(1, stmts)

	c.f = outerFuncContext
	return
}

func (c *pkgContext) translateFunctionBody(indent int, stmts []ast.Stmt) []byte {
	v := gotoVisitor{f: c.f}
	for _, stmt := range stmts {
		ast.Walk(&v, stmt)
	}
	c.f.localVars = nil
	if c.f.flattened {
		c.f.localVars = append(c.f.localVars, "go$this = this")
	}

	body := c.CatchOutput(indent, func() {
		if c.f.sig != nil && c.f.sig.Results().Len() != 0 && c.f.sig.Results().At(0).Name() != "" {
			c.f.resultNames = make([]ast.Expr, c.f.sig.Results().Len())
			for i := 0; i < c.f.sig.Results().Len(); i++ {
				result := c.f.sig.Results().At(i)
				name := result.Name()
				if result.Name() == "_" {
					name = "result"
				}
				id := ast.NewIdent(name)
				c.info.Types[id] = types.TypeAndValue{Type: result.Type()}
				c.info.Objects[id] = result
				c.Printf("%s = %s;", c.translateExpr(id), c.zeroValue(result.Type()))
				c.f.resultNames[i] = id
			}
		}

		printBody := func() {
			if c.f.flattened {
				c.Printf("/* */ var go$s = 0, go$f = function() { while (true) { switch (go$s) { case 0:")
				c.translateStmtList(stmts)
				c.Printf("/* */ } break; } }; return go$f();")
				return
			}
			c.translateStmtList(stmts)
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
				if c.f.sig != nil && c.f.resultNames == nil {
					switch c.f.sig.Results().Len() {
					case 0:
						// nothing
					case 1:
						c.Printf("return %s;", c.zeroValue(c.f.sig.Results().At(0).Type()))
					default:
						zeros := make([]string, c.f.sig.Results().Len())
						for i := range zeros {
							zeros[i] = c.zeroValue(c.f.sig.Results().At(i).Type())
						}
						c.Printf("return [%s];", strings.Join(zeros, ", "))
					}
				}
			})
			c.Printf("} finally {")
			c.Indent(func() {
				c.Printf("go$callDeferred(go$deferred);")
				if c.f.resultNames != nil {
					c.translateStmt(&ast.ReturnStmt{}, "")
				}
			})
			c.Printf("}")
			return
		}
		printBody()
	})

	if len(c.f.localVars) != 0 {
		body = append([]byte(fmt.Sprintf("%svar %s;\n", strings.Repeat("\t", c.indentation+indent), strings.Join(c.f.localVars, ", "))), body...)
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
	f     *funcContext
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
			v.f.flattened = true
			for _, n2 := range v.stack {
				v.f.hasGoto[n2] = true
			}
			if _, ok := v.f.labelCases[n.Label.String()]; !ok {
				v.f.labelCases[n.Label.String()] = v.f.caseCounter
				v.f.caseCounter++
			}
			return nil
		}
	case ast.Expr:
		return nil
	}
	v.stack = append(v.stack, node)
	return v
}
