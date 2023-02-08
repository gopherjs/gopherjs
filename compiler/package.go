package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"time"

	"github.com/gopherjs/gopherjs/compiler/analysis"
	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
	"golang.org/x/tools/go/gcexportdata"
)

// pkgContext maintains compiler context for a specific package.
type pkgContext struct {
	*analysis.Info
	additionalSelections map[*ast.SelectorExpr]selection

	// List of type names declared in the package, including those defined inside
	// functions.
	typeNames []*types.TypeName
	// Mapping from package import paths to JS variables that were assigned to an
	// imported package and can be used to access it.
	pkgVars map[string]string
	// Mapping from a named Go object (e.g. type, func, var...) to a JS variable
	// name assigned to them.
	objectNames  map[types.Object]string
	varPtrNames  map[*types.Var]string
	anonTypes    typesutil.AnonymousTypes
	escapingVars map[*types.Var]bool
	indentation  int
	dependencies map[types.Object]bool
	minify       bool
	fileSet      *token.FileSet
	errList      ErrorList
}

// IsMain returns true if this is the main package of the program.
func (pc *pkgContext) IsMain() bool {
	return pc.Pkg.Name() == "main"
}

func (p *pkgContext) SelectionOf(e *ast.SelectorExpr) (selection, bool) {
	if sel, ok := p.Selections[e]; ok {
		return sel, true
	}
	if sel, ok := p.additionalSelections[e]; ok {
		return sel, true
	}
	return nil, false
}

// genericCtx contains compiler context for a generic function or type.
//
// It is used to accumulate information about types and objects that depend on
// type parameters and must be constructed in a generic factory function.
type genericCtx struct {
	anonTypes typesutil.AnonymousTypes
}

type selection interface {
	Kind() types.SelectionKind
	Recv() types.Type
	Index() []int
	Obj() types.Object
	Type() types.Type
}

type fakeSelection struct {
	kind  types.SelectionKind
	recv  types.Type
	index []int
	obj   types.Object
	typ   types.Type
}

func (sel *fakeSelection) Kind() types.SelectionKind { return sel.kind }
func (sel *fakeSelection) Recv() types.Type          { return sel.recv }
func (sel *fakeSelection) Index() []int              { return sel.index }
func (sel *fakeSelection) Obj() types.Object         { return sel.obj }
func (sel *fakeSelection) Type() types.Type          { return sel.typ }

// funcContext maintains compiler context for a specific function.
//
// An instance of this type roughly corresponds to a lexical scope for generated
// JavaScript code (as defined for `var` declarations).
type funcContext struct {
	*analysis.FuncInfo
	// Surrounding package context.
	pkgCtx *pkgContext
	// Surrounding generic function context. nil if non-generic code.
	genericCtx *genericCtx
	// Function context, surrounding this function definition. For package-level
	// functions or methods it is the package-level function context (even though
	// it technically doesn't correspond to a function). nil for the package-level
	// function context.
	parent *funcContext
	// Information about function signature types. nil for the package-level
	// function context.
	sigTypes *signatureTypes
	// All variable names available in the current function scope. The key is a Go
	// variable name and the value is the number of synonymous variable names
	// visible from this scope (e.g. due to shadowing). This number is used to
	// avoid conflicts when assigning JS variable names for Go variables.
	allVars map[string]int
	// Local JS variable names defined within this function context. This list
	// contains JS variable names assigned to Go variables, as well as other
	// auxiliary variables the compiler needs. It is used to generate `var`
	// declaration at the top of the function, as well as context save/restore.
	localVars []string
	// AST expressions representing function's named return values. nil if the
	// function has no return values or they are not named.
	resultNames []ast.Expr
	// Function's internal control flow graph used for generation of a "flattened"
	// version of the function when the function is blocking or uses goto.
	// TODO(nevkontakte): Describe the exact semantics of this map.
	flowDatas map[*types.Label]*flowData
	// Number of control flow blocks in a "flattened" function.
	caseCounter int
	// A mapping from Go labels statements (e.g. labelled loop) to the flow block
	// id corresponding to it.
	labelCases map[*types.Label]int
	// Generated code buffer for the current function.
	output []byte
	// Generated code that should be emitted at the end of the JS statement (?).
	delayedOutput []byte
	// Set to true if source position is available and should be emitted for the
	// source map.
	posAvailable bool
	// Current position in the Go source code.
	pos token.Pos
}

// newRootCtx creates a new package-level instance of a functionContext object.
func newRootCtx(srcs sources, typesInfo *types.Info, typesPkg *types.Package, isBlocking func(*types.Func) bool, minify bool) *funcContext {
	pkgInfo := analysis.AnalyzePkg(srcs.Files, srcs.FileSet, typesInfo, typesPkg, isBlocking)
	rootCtx := &funcContext{
		parent:   nil, // Package-level context has no parent.
		FuncInfo: pkgInfo.InitFuncInfo,
		pkgCtx: &pkgContext{
			Info:                 pkgInfo,
			additionalSelections: make(map[*ast.SelectorExpr]selection),

			pkgVars:      make(map[string]string),
			objectNames:  make(map[types.Object]string),
			varPtrNames:  make(map[*types.Var]string),
			escapingVars: make(map[*types.Var]bool),
			indentation:  1,
			minify:       minify,
			fileSet:      srcs.FileSet,
		},
		allVars:     make(map[string]int),
		flowDatas:   map[*types.Label]*flowData{nil: {}},
		caseCounter: 1,
		labelCases:  make(map[*types.Label]int),
	}
	for name := range reservedKeywords {
		rootCtx.allVars[name] = 1
	}
	return rootCtx
}

type flowData struct {
	postStmt  func()
	beginCase int
	endCase   int
}

// ImportContext provides access to information about imported packages.
type ImportContext struct {
	// Mapping for an absolute import path to the package type information.
	Packages map[string]*types.Package
	// Import returns a previously compiled Archive for a dependency package. If
	// the Import() call was successful, the corresponding entry must be added to
	// the Packages map.
	Import func(importPath string) (*Archive, error)
}

// isBlocking returns true if an _imported_ function is blocking. It will panic
// if the function decl is not found in the imported package or the package
// hasn't been compiled yet.
//
// Note: see analysis.FuncInfo.Blocking if you need to determine if a function
// in the _current_ package is blocking. Usually available via functionContext
// object.
func (ic *ImportContext) isBlocking(f *types.Func) bool {
	archive, err := ic.Import(f.Pkg().Path())
	if err != nil {
		panic(err)
	}
	fullName := f.FullName()
	for _, d := range archive.Declarations {
		if string(d.FullName) == fullName {
			return d.Blocking
		}
	}
	panic(bailout(fmt.Errorf("can't determine if function %s is blocking: decl not found in package archive", fullName)))
}

// Compile the provided Go sources as a single package.
//
// Import path must be the absolute import path for a package. Provided sources
// are always sorted by name to ensure reproducible JavaScript output.
func Compile(importPath string, files []*ast.File, fileSet *token.FileSet, importContext *ImportContext, minify bool) (_ *Archive, err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if fe, ok := bailingOut(e); ok {
			// Orderly bailout, return whatever clues we already have.
			err = fe
			return
		}
		// Some other unexpected panic, catch the stack trace and return as an error.
		err = bailout(fmt.Errorf("unexpected compiler panic while building package %q: %v", importPath, e))
	}()

	srcs := sources{
		ImportPath: importPath,
		Files:      files,
		FileSet:    fileSet,
	}.Sort()

	typesInfo, typesPkg, err := srcs.TypeCheck(importContext)
	if err != nil {
		return nil, err
	}
	importContext.Packages[srcs.ImportPath] = typesPkg

	// Extract all go:linkname compiler directives from the package source.
	goLinknames, err := srcs.ParseGoLinknames()
	if err != nil {
		return nil, err
	}

	srcs = srcs.Simplified(typesInfo)

	rootCtx := newRootCtx(srcs, typesInfo, typesPkg, importContext.isBlocking, minify)

	importedPaths, importDecls := rootCtx.importDecls()

	vars, functions, typeNames := rootCtx.topLevelObjects(srcs)

	// More named types may be added to the list when function bodies are processed.
	rootCtx.pkgCtx.typeNames = typeNames

	// Translate functions and variables.
	varDecls := rootCtx.varDecls(vars)
	funcDecls, err := rootCtx.funcDecls(functions)
	if err != nil {
		return nil, err
	}

	// It is important that we translate types *after* we've processed all
	// functions to make sure we've discovered all types declared inside function
	// bodies.
	typeDecls := rootCtx.namedTypeDecls(rootCtx.pkgCtx.typeNames)

	// Finally, anonymous types are translated the last, to make sure we've
	// discovered all of them referenced in functions, variable and type
	// declarations.
	typeDecls = append(typeDecls, rootCtx.anonTypeDecls(rootCtx.pkgCtx.anonTypes.Ordered())...)

	// Combine all decls in a single list in the order they must appear in the
	// final program.
	allDecls := append(append(append(importDecls, typeDecls...), varDecls...), funcDecls...)

	if minify {
		for _, d := range allDecls {
			*d = d.minify()
		}
	}

	if len(rootCtx.pkgCtx.errList) != 0 {
		return nil, rootCtx.pkgCtx.errList
	}

	exportData := new(bytes.Buffer)
	if err := gcexportdata.Write(exportData, nil, typesPkg); err != nil {
		return nil, fmt.Errorf("failed to write export data: %w", err)
	}
	encodedFileSet := new(bytes.Buffer)
	if err := srcs.FileSet.Write(json.NewEncoder(encodedFileSet).Encode); err != nil {
		return nil, err
	}

	return &Archive{
		ImportPath:   srcs.ImportPath,
		Name:         typesPkg.Name(),
		Imports:      importedPaths,
		ExportData:   exportData.Bytes(),
		Declarations: allDecls,
		FileSet:      encodedFileSet.Bytes(),
		Minified:     minify,
		GoLinknames:  goLinknames,
		BuildTime:    time.Now(),
	}, nil
}

func (fc *funcContext) initArgs(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Array:
		return fmt.Sprintf("%s, %d", fc.typeName(t.Elem()), t.Len())
	case *types.Chan:
		return fmt.Sprintf("%s, %t, %t", fc.typeName(t.Elem()), t.Dir()&types.SendOnly != 0, t.Dir()&types.RecvOnly != 0)
	case *types.Interface:
		methods := make([]string, t.NumMethods())
		for i := range methods {
			method := t.Method(i)
			pkgPath := ""
			if !method.Exported() {
				pkgPath = method.Pkg().Path()
			}
			methods[i] = fmt.Sprintf(`{prop: "%s", name: "%s", pkg: "%s", typ: $funcType(%s)}`, method.Name(), method.Name(), pkgPath, fc.initArgs(method.Type()))
		}
		return fmt.Sprintf("[%s]", strings.Join(methods, ", "))
	case *types.Map:
		return fmt.Sprintf("%s, %s", fc.typeName(t.Key()), fc.typeName(t.Elem()))
	case *types.Pointer:
		return fmt.Sprintf("%s", fc.typeName(t.Elem()))
	case *types.Slice:
		return fmt.Sprintf("%s", fc.typeName(t.Elem()))
	case *types.Signature:
		params := make([]string, t.Params().Len())
		for i := range params {
			params[i] = fc.typeName(t.Params().At(i).Type())
		}
		results := make([]string, t.Results().Len())
		for i := range results {
			results[i] = fc.typeName(t.Results().At(i).Type())
		}
		return fmt.Sprintf("[%s], [%s], %t", strings.Join(params, ", "), strings.Join(results, ", "), t.Variadic())
	case *types.Struct:
		pkgPath := ""
		fields := make([]string, t.NumFields())
		for i := range fields {
			field := t.Field(i)
			if !field.Exported() {
				pkgPath = field.Pkg().Path()
			}
			fields[i] = fmt.Sprintf(`{prop: "%s", name: %s, embedded: %t, exported: %t, typ: %s, tag: %s}`, fieldName(t, i), encodeString(field.Name()), field.Anonymous(), field.Exported(), fc.typeName(field.Type()), encodeString(t.Tag(i)))
		}
		return fmt.Sprintf(`"%s", [%s]`, pkgPath, strings.Join(fields, ", "))
	default:
		err := bailout(fmt.Errorf("%v has unexpected type %T", ty, ty))
		panic(err)
	}
}

func (fc *funcContext) translateTopLevelFunction(fun *ast.FuncDecl) []byte {
	o := fc.pkgCtx.Defs[fun.Name].(*types.Func)
	info := fc.pkgCtx.FuncDeclInfos[o]

	sig := o.Type().(*types.Signature)
	var recv *ast.Ident
	if fun.Recv != nil && fun.Recv.List[0].Names != nil {
		recv = fun.Recv.List[0].Names[0]
	}

	var joinedParams string
	// primaryFunction generates a JS function equivalent of the current Go function
	// and assigns it to the JS variable defined by lvalue.
	primaryFunction := func(lvalue string) []byte {
		if fun.Body == nil {
			return []byte(fmt.Sprintf("\t%s = function() {\n\t\t$throwRuntimeError(\"native function not implemented: %s\");\n\t};\n", lvalue, o.FullName()))
		}

		params, fun := translateFunction(fun.Type, recv, fun.Body, fc, sig, info, lvalue)
		joinedParams = strings.Join(params, ", ")
		return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fun))
	}

	code := bytes.NewBuffer(nil)

	if fun.Recv == nil {
		funcRef := fc.objectName(o)
		code.Write(primaryFunction(funcRef))
		if fun.Name.IsExported() {
			fmt.Fprintf(code, "\t$pkg.%s = %s;\n", encodeIdent(fun.Name.Name), funcRef)
		}
		return code.Bytes()
	}

	recvType := sig.Recv().Type()
	ptr, isPointer := recvType.(*types.Pointer)
	namedRecvType, _ := recvType.(*types.Named)
	if isPointer {
		namedRecvType = ptr.Elem().(*types.Named)
	}
	typeName := fc.objectName(namedRecvType.Obj())
	funName := fun.Name.Name
	if reservedKeywords[funName] {
		funName += "$"
	}

	// Objects the method should be assigned to.
	prototypeVar := fmt.Sprintf("%s.prototype.%s", typeName, funName)
	ptrPrototypeVar := fmt.Sprintf("$ptrType(%s).prototype.%s", typeName, funName)
	isGeneric := signatureTypes{Sig: sig}.IsGeneric()
	if isGeneric {
		// Generic method factories are assigned to the generic type factory
		// properties, to be invoked at type construction time rather than method
		// call time.
		prototypeVar = fmt.Sprintf("%s.methods.%s", typeName, funName)
		ptrPrototypeVar = fmt.Sprintf("%s.ptrMethods.%s", typeName, funName)
	}

	// proxyFunction generates a JS function that forwards the call to the actual
	// method implementation for the alternate receiver (e.g. pointer vs
	// non-pointer).
	proxyFunction := func(lvalue, receiver string) []byte {
		fun := fmt.Sprintf("function(%s) { return %s.%s(%s); }", joinedParams, receiver, funName, joinedParams)
		if isGeneric {
			// For a generic function, we wrap the proxy function in a trivial generic
			// factory function for consistency. It is the same for any possible type
			// arguments, so we simply ignore them.
			fun = fmt.Sprintf("function() { return %s; }", fun)
		}
		return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fun))
	}

	if _, isStruct := namedRecvType.Underlying().(*types.Struct); isStruct {
		code.Write(primaryFunction(ptrPrototypeVar))
		code.Write(proxyFunction(prototypeVar, "this.$val"))
		return code.Bytes()
	}

	if isPointer {
		if _, isArray := ptr.Elem().Underlying().(*types.Array); isArray {
			code.Write(primaryFunction(prototypeVar))
			code.Write(proxyFunction(ptrPrototypeVar, fmt.Sprintf("(new %s(this.$get()))", typeName)))
			return code.Bytes()
		}
		return primaryFunction(ptrPrototypeVar)
	}

	recvExpr := "this.$get()"
	if typesutil.IsGeneric(recvType) {
		recvExpr = fmt.Sprintf("%s.wrap(%s)", typeName, recvExpr)
	} else if isWrapped(recvType) {
		recvExpr = fmt.Sprintf("new %s(%s)", typeName, recvExpr)
	}
	code.Write(primaryFunction(prototypeVar))
	code.Write(proxyFunction(ptrPrototypeVar, recvExpr))
	return code.Bytes()
}

func translateFunction(typ *ast.FuncType, recv *ast.Ident, body *ast.BlockStmt, outerContext *funcContext, sig *types.Signature, info *analysis.FuncInfo, funcRef string) ([]string, string) {
	if info == nil {
		panic("nil info")
	}

	c := &funcContext{
		FuncInfo:    info,
		pkgCtx:      outerContext.pkgCtx,
		genericCtx:  outerContext.genericCtx,
		parent:      outerContext,
		sigTypes:    &signatureTypes{Sig: sig},
		allVars:     make(map[string]int, len(outerContext.allVars)),
		localVars:   []string{},
		flowDatas:   map[*types.Label]*flowData{nil: {}},
		caseCounter: 1,
		labelCases:  make(map[*types.Label]int),
	}
	for k, v := range outerContext.allVars {
		c.allVars[k] = v
	}

	functionName := "" // Function object name, i.e. identifier after the "function" keyword.
	if funcRef == "" {
		// Assign a name for the anonymous function.
		funcRef = "$b"
		functionName = " $b"
	}

	// For regular functions instance is directly in the function variable name.
	instanceVar := funcRef
	if c.sigTypes.IsGeneric() {
		c.genericCtx = &genericCtx{}
		// For generic function, funcRef refers to the generic factory function,
		// allocate a separate variable for a function instance.
		instanceVar = c.newVariable("instance", varGenericFactory)
	}

	prevEV := c.pkgCtx.escapingVars

	var params []string
	for _, param := range typ.Params.List {
		if len(param.Names) == 0 {
			params = append(params, c.newLocalVariable("param"))
			continue
		}
		for _, ident := range param.Names {
			if isBlank(ident) {
				params = append(params, c.newLocalVariable("param"))
				continue
			}
			params = append(params, c.objectName(c.pkgCtx.Defs[ident]))
		}
	}

	bodyOutput := string(c.CatchOutput(c.bodyIndent(), func() {
		if len(c.Blocking) != 0 {
			c.pkgCtx.Scopes[body] = c.pkgCtx.Scopes[typ]
			c.handleEscapingVars(body)
		}

		if c.sigTypes != nil && c.sigTypes.HasNamedResults() {
			c.resultNames = make([]ast.Expr, c.sigTypes.Sig.Results().Len())
			for i := 0; i < c.sigTypes.Sig.Results().Len(); i++ {
				result := c.sigTypes.Sig.Results().At(i)
				c.Printf("%s = %s;", c.objectName(result), c.translateExpr(c.zeroValue(result.Type())).String())
				id := ast.NewIdent("")
				c.pkgCtx.Uses[id] = result
				c.resultNames[i] = c.setType(id, result.Type())
			}
		}

		if recv != nil && !isBlank(recv) {
			this := "this"
			if isWrapped(c.pkgCtx.TypeOf(recv)) {
				this = "this.$val" // Unwrap receiver value.
			}
			c.Printf("%s = %s;", c.translateExpr(recv), this)
		}

		c.translateStmtList(body.List)
		if len(c.Flattened) != 0 && !astutil.EndsWithReturn(body.List) {
			c.translateStmt(&ast.ReturnStmt{}, nil)
		}
	}))

	sort.Strings(c.localVars)

	var prefix, suffix string

	if len(c.Flattened) != 0 {
		c.localVars = append(c.localVars, "$s")
		prefix = prefix + " $s = $s || 0;"
	}

	if c.HasDefer {
		c.localVars = append(c.localVars, "$deferred")
		suffix = " }" + suffix
		if len(c.Blocking) != 0 {
			suffix = " }" + suffix
		}
	}

	localVarDefs := "" // Function-local var declaration at the top.

	if len(c.Blocking) != 0 {
		localVars := append([]string{}, c.localVars...)
		// There are several special variables involved in handling blocking functions:
		// $r is sometimes used as a temporary variable to store blocking call result.
		// $c indicates that a function is being resumed after a blocking call when set to true.
		// $f is an object used to save and restore function context for blocking calls.
		localVars = append(localVars, "$r")
		// If a blocking function is being resumed, initialize local variables from the saved context.
		localVarDefs = fmt.Sprintf("var {%s, $c} = $restore(this, {%s});\n", strings.Join(localVars, ", "), strings.Join(params, ", "))
		// If the function gets blocked, save local variables for future.
		saveContext := fmt.Sprintf("var $f = {$blk: %s, $c: true, $r, %s};", instanceVar, strings.Join(c.localVars, ", "))

		suffix = " " + saveContext + "return $f;" + suffix
	} else if len(c.localVars) > 0 {
		// Non-blocking functions simply declare local variables with no need for restore support.
		localVarDefs = fmt.Sprintf("var %s;\n", strings.Join(c.localVars, ", "))
	}

	if c.HasDefer {
		prefix = prefix + " var $err = null; try {"
		deferSuffix := " } catch(err) { $err = err;"
		if len(c.Blocking) != 0 {
			deferSuffix += " $s = -1;"
		}
		if c.resultNames == nil && c.sigTypes.HasResults() {
			deferSuffix += fmt.Sprintf(" return%s;", c.translateResults(nil))
		}
		deferSuffix += " } finally { $callDeferred($deferred, $err);"
		if c.resultNames != nil {
			deferSuffix += fmt.Sprintf(" if (!$curGoroutine.asleep) { return %s; }", c.translateResults(c.resultNames))
		}
		if len(c.Blocking) != 0 {
			deferSuffix += " if($curGoroutine.asleep) {"
		}
		suffix = deferSuffix + suffix
	}

	if len(c.Flattened) != 0 {
		prefix = prefix + " s: while (true) { switch ($s) { case 0:"
		suffix = " } return; }" + suffix
	}

	if c.HasDefer {
		prefix = prefix + " $deferred = []; $curGoroutine.deferStack.push($deferred);"
	}

	if prefix != "" {
		bodyOutput = c.Indentation(c.bodyIndent()) + "/* */" + prefix + "\n" + bodyOutput
	}
	if suffix != "" {
		bodyOutput = bodyOutput + c.Indentation(c.bodyIndent()) + "/* */" + suffix + "\n"
	}
	if localVarDefs != "" {
		bodyOutput = c.Indentation(c.bodyIndent()) + localVarDefs + bodyOutput
	}

	c.pkgCtx.escapingVars = prevEV

	if !c.sigTypes.IsGeneric() {
		return params, fmt.Sprintf("function%s(%s) {\n%s%s}", functionName, strings.Join(params, ", "), bodyOutput, c.Indentation(0))
	}

	// Generic functions are generated as factories to allow passing type parameters
	// from the call site.
	// TODO(nevkontakte): Cache function instances for a given combination of type
	// parameters.
	typeParams := c.typeParamVars(c.sigTypes.Sig.TypeParams())
	typeParams = append(typeParams, c.typeParamVars(c.sigTypes.Sig.RecvTypeParams())...)

	// anonymous types
	typesInit := strings.Builder{}
	for _, t := range c.genericCtx.anonTypes.Ordered() {
		fmt.Fprintf(&typesInit, "%svar %s = $%sType(%s);\n", c.Indentation(1), t.Name(), strings.ToLower(typeKind(t.Type())[5:]), c.initArgs(t.Type()))
	}

	code := &strings.Builder{}
	fmt.Fprintf(code, "function%s(%s){\n", functionName, strings.Join(typeParams, ", "))
	fmt.Fprintf(code, "%s", typesInit.String())
	fmt.Fprintf(code, "%sconst %s = function(%s) {\n", c.Indentation(1), instanceVar, strings.Join(params, ", "))
	fmt.Fprintf(code, "%s", bodyOutput)
	fmt.Fprintf(code, "%s};\n", c.Indentation(1))
	fmt.Fprintf(code, "%sreturn %s;\n", c.Indentation(1), instanceVar)
	fmt.Fprintf(code, "%s}", c.Indentation(0))
	return params, code.String()
}
