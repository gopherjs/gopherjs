package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"time"

	"github.com/gopherjs/gopherjs/compiler/analysis"
	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/internal/symbol"
	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
	"github.com/gopherjs/gopherjs/internal/experiments"
	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/types/typeutil"
)

// pkgContext maintains compiler context for a specific package.
type pkgContext struct {
	*analysis.Info
	additionalSelections map[*ast.SelectorExpr]typesutil.Selection

	typesCtx     *types.Context
	typeNames    typesutil.TypeNames
	pkgVars      map[string]string
	varPtrNames  map[*types.Var]string
	anonTypes    []*types.TypeName
	anonTypeMap  typeutil.Map
	escapingVars map[*types.Var]bool
	indentation  int
	dependencies map[types.Object]bool
	minify       bool
	fileSet      *token.FileSet
	errList      ErrorList
	instanceSet  *typeparams.PackageInstanceSets
}

// funcContext maintains compiler context for a specific function.
//
// An instance of this type roughly corresponds to a lexical scope for generated
// JavaScript code (as defined for `var` declarations).
type funcContext struct {
	*analysis.FuncInfo
	// Surrounding package context.
	pkgCtx *pkgContext
	// Function context, surrounding this function definition. For package-level
	// functions or methods it is the package-level function context (even though
	// it technically doesn't correspond to a function). nil for the package-level
	// function context.
	parent *funcContext
	// Signature of the function this context corresponds to or nil for the
	// package-level function context. For generic functions it is the original
	// generic signature to make sure result variable identity in the signature
	// matches the variable objects referenced in the function body.
	sig *typesutil.Signature
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
	// Generated code that should be emitted at the end of the JS statement.
	delayedOutput []byte
	// Set to true if source position is available and should be emitted for the
	// source map.
	posAvailable bool
	// Current position in the Go source code.
	pos token.Pos
	// For each instantiation of a generic function or method, contains the
	// current mapping between type parameters and corresponding type arguments.
	// The mapping is used to determine concrete types for expressions within the
	// instance's context. Can be nil outside of the generic context, in which
	// case calling its methods is safe and simply does no substitution.
	typeResolver *typeparams.Resolver
	// Mapping from function-level objects to JS variable names they have been assigned.
	objectNames map[types.Object]string
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
			fmt.Fprintf(fe, `building package %q`, importPath)
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

	tContext := types.NewContext()
	typesInfo, typesPkg, err := srcs.TypeCheck(importContext, tContext)
	if err != nil {
		return nil, err
	}
	if genErr := typeparams.RequiresGenericsSupport(typesInfo); genErr != nil && !experiments.Env.Generics {
		return nil, fmt.Errorf("package %s requires generics support (https://github.com/gopherjs/gopherjs/issues/1013): %w", importPath, genErr)
	}
	importContext.Packages[srcs.ImportPath] = typesPkg

	// Extract all go:linkname compiler directives from the package source.
	goLinknames, err := srcs.ParseGoLinknames()
	if err != nil {
		return nil, err
	}

	srcs = srcs.Simplified(typesInfo)

	isBlocking := func(f *types.Func) bool {
		archive, err := importContext.Import(f.Pkg().Path())
		if err != nil {
			panic(err)
		}
		fullName := f.FullName()
		for _, d := range archive.Declarations {
			if string(d.FullName) == fullName {
				return d.Blocking
			}
		}
		panic(fullName)
	}

	tc := typeparams.Collector{
		TContext:  tContext,
		Info:      typesInfo,
		Instances: &typeparams.PackageInstanceSets{},
	}
	tc.Scan(typesPkg, srcs.Files...)
	instancesByObj := map[types.Object][]typeparams.Instance{}
	for _, inst := range tc.Instances.Pkg(typesPkg).Values() {
		instancesByObj[inst.Object] = append(instancesByObj[inst.Object], inst)
	}

	pkgInfo := analysis.AnalyzePkg(srcs.Files, fileSet, typesInfo, typesPkg, isBlocking)
	funcCtx := &funcContext{
		FuncInfo: pkgInfo.InitFuncInfo,
		pkgCtx: &pkgContext{
			Info:                 pkgInfo,
			additionalSelections: make(map[*ast.SelectorExpr]typesutil.Selection),

			typesCtx:     tContext,
			pkgVars:      make(map[string]string),
			varPtrNames:  make(map[*types.Var]string),
			escapingVars: make(map[*types.Var]bool),
			indentation:  1,
			dependencies: make(map[types.Object]bool),
			minify:       minify,
			fileSet:      srcs.FileSet,
			instanceSet:  tc.Instances,
		},
		allVars:     make(map[string]int),
		flowDatas:   map[*types.Label]*flowData{nil: {}},
		caseCounter: 1,
		labelCases:  make(map[*types.Label]int),
		objectNames: map[types.Object]string{},
	}
	for name := range reservedKeywords {
		funcCtx.allVars[name] = 1
	}

	// imports
	var importDecls []*Decl
	var importedPaths []string
	for _, importedPkg := range typesPkg.Imports() {
		if importedPkg == types.Unsafe {
			// Prior to Go 1.9, unsafe import was excluded by Imports() method,
			// but now we do it here to maintain previous behavior.
			continue
		}
		funcCtx.pkgCtx.pkgVars[importedPkg.Path()] = funcCtx.newVariable(importedPkg.Name(), true)
		importedPaths = append(importedPaths, importedPkg.Path())
	}
	sort.Strings(importedPaths)
	for _, impPath := range importedPaths {
		id := funcCtx.newIdent(fmt.Sprintf(`%s.$init`, funcCtx.pkgCtx.pkgVars[impPath]), types.NewSignatureType(nil, nil, nil, nil, nil, false))
		call := &ast.CallExpr{Fun: id}
		funcCtx.Blocking[call] = true
		funcCtx.Flattened[call] = true
		importDecls = append(importDecls, &Decl{
			Vars:     []string{funcCtx.pkgCtx.pkgVars[impPath]},
			DeclCode: []byte(fmt.Sprintf("\t%s = $packages[\"%s\"];\n", funcCtx.pkgCtx.pkgVars[impPath], impPath)),
			InitCode: funcCtx.CatchOutput(1, func() { funcCtx.translateStmt(&ast.ExprStmt{X: call}, nil) }),
		})
	}

	var functions []*ast.FuncDecl
	var vars []*types.Var
	for _, file := range srcs.Files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				sig := funcCtx.pkgCtx.Defs[d.Name].(*types.Func).Type().(*types.Signature)
				if sig.Recv() == nil {
					funcCtx.objectName(funcCtx.pkgCtx.Defs[d.Name].(*types.Func)) // register toplevel name
				}
				if !isBlank(d.Name) {
					functions = append(functions, d)
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					for _, spec := range d.Specs {
						o := funcCtx.pkgCtx.Defs[spec.(*ast.TypeSpec).Name].(*types.TypeName)
						funcCtx.pkgCtx.typeNames.Add(o)
						funcCtx.objectName(o) // register toplevel name
					}
				case token.VAR:
					for _, spec := range d.Specs {
						for _, name := range spec.(*ast.ValueSpec).Names {
							if !isBlank(name) {
								o := funcCtx.pkgCtx.Defs[name].(*types.Var)
								vars = append(vars, o)
								funcCtx.objectName(o) // register toplevel name
							}
						}
					}
				case token.CONST:
					// skip, constants are inlined
				}
			}
		}
	}

	collectDependencies := func(f func()) []string {
		funcCtx.pkgCtx.dependencies = make(map[types.Object]bool)
		f()
		var deps []string
		for o := range funcCtx.pkgCtx.dependencies {
			qualifiedName := o.Pkg().Path() + "." + o.Name()
			if f, ok := o.(*types.Func); ok && f.Type().(*types.Signature).Recv() != nil {
				deps = append(deps, qualifiedName+"~")
				continue
			}
			deps = append(deps, qualifiedName)
		}
		sort.Strings(deps)
		return deps
	}

	// variables
	var varDecls []*Decl
	varsWithInit := make(map[*types.Var]bool)
	for _, init := range funcCtx.pkgCtx.InitOrder {
		for _, o := range init.Lhs {
			varsWithInit[o] = true
		}
	}
	for _, o := range vars {
		var d Decl
		if !o.Exported() {
			d.Vars = []string{funcCtx.objectName(o)}
		}
		if funcCtx.pkgCtx.HasPointer[o] && !o.Exported() {
			d.Vars = append(d.Vars, funcCtx.varPtrName(o))
		}
		if _, ok := varsWithInit[o]; !ok {
			d.DceDeps = collectDependencies(func() {
				d.InitCode = []byte(fmt.Sprintf("\t\t%s = %s;\n", funcCtx.objectName(o), funcCtx.translateExpr(funcCtx.zeroValue(o.Type())).String()))
			})
		}
		d.DceObjectFilter = o.Name()
		varDecls = append(varDecls, &d)
	}
	for _, init := range funcCtx.pkgCtx.InitOrder {
		lhs := make([]ast.Expr, len(init.Lhs))
		for i, o := range init.Lhs {
			ident := ast.NewIdent(o.Name())
			ident.NamePos = o.Pos()
			funcCtx.pkgCtx.Defs[ident] = o
			lhs[i] = funcCtx.setType(ident, o.Type())
			varsWithInit[o] = true
		}
		var d Decl
		d.DceDeps = collectDependencies(func() {
			funcCtx.localVars = nil
			d.InitCode = funcCtx.CatchOutput(1, func() {
				funcCtx.translateStmt(&ast.AssignStmt{
					Lhs: lhs,
					Tok: token.DEFINE,
					Rhs: []ast.Expr{init.Rhs},
				}, nil)
			})
			d.Vars = append(d.Vars, funcCtx.localVars...)
		})
		if len(init.Lhs) == 1 {
			if !analysis.HasSideEffect(init.Rhs, funcCtx.pkgCtx.Info.Info) {
				d.DceObjectFilter = init.Lhs[0].Name()
			}
		}
		varDecls = append(varDecls, &d)
	}

	// functions
	var funcDecls []*Decl
	var mainFunc *types.Func
	for _, fun := range functions {
		o := funcCtx.pkgCtx.Defs[fun.Name].(*types.Func)
		sig := o.Type().(*types.Signature)

		var instances []typeparams.Instance
		if typeparams.SignatureTypeParams(sig) != nil {
			instances = instancesByObj[o]
		} else {
			instances = []typeparams.Instance{{Object: o}}
		}

		if fun.Recv == nil {
			// Auxiliary decl shared by all instances of the function that defines
			// package-level variable by which they all are referenced.
			// TODO(nevkontakte): Set DCE attributes such that it is eliminated if all
			// instances are dead.
			varDecl := Decl{}
			varDecl.Vars = []string{funcCtx.objectName(o)}
			if o.Type().(*types.Signature).TypeParams().Len() != 0 {
				varDecl.DeclCode = funcCtx.CatchOutput(0, func() {
					funcCtx.Printf("%s = {};", funcCtx.objectName(o))
				})
			}
			funcDecls = append(funcDecls, &varDecl)
		}

		for _, inst := range instances {
			funcInfo := funcCtx.pkgCtx.FuncDeclInfos[o]
			d := Decl{
				FullName: o.FullName(),
				Blocking: len(funcInfo.Blocking) != 0,
			}
			d.LinkingName = symbol.New(o)
			if fun.Recv == nil {
				d.RefExpr = funcCtx.instName(inst)
				d.DceObjectFilter = o.Name()
				switch o.Name() {
				case "main":
					mainFunc = o
					d.DceObjectFilter = ""
				case "init":
					d.InitCode = funcCtx.CatchOutput(1, func() {
						id := funcCtx.newIdent("", types.NewSignatureType( /*recv=*/ nil /*rectTypeParams=*/, nil /*typeParams=*/, nil /*params=*/, nil /*results=*/, nil /*variadic=*/, false))
						funcCtx.pkgCtx.Uses[id] = o
						call := &ast.CallExpr{Fun: id}
						if len(funcCtx.pkgCtx.FuncDeclInfos[o].Blocking) != 0 {
							funcCtx.Blocking[call] = true
						}
						funcCtx.translateStmt(&ast.ExprStmt{X: call}, nil)
					})
					d.DceObjectFilter = ""
				}
			} else {
				recvType := o.Type().(*types.Signature).Recv().Type()
				ptr, isPointer := recvType.(*types.Pointer)
				namedRecvType, _ := recvType.(*types.Named)
				if isPointer {
					namedRecvType = ptr.Elem().(*types.Named)
				}
				d.NamedRecvType = funcCtx.objectName(namedRecvType.Obj())
				d.DceObjectFilter = namedRecvType.Obj().Name()
				if !fun.Name.IsExported() {
					d.DceMethodFilter = o.Name() + "~"
				}
			}

			d.DceDeps = collectDependencies(func() {
				d.DeclCode = funcCtx.translateToplevelFunction(fun, funcInfo, inst)
			})
			funcDecls = append(funcDecls, &d)
		}
	}
	if typesPkg.Name() == "main" {
		if mainFunc == nil {
			return nil, fmt.Errorf("missing main function")
		}
		id := funcCtx.newIdent("", types.NewSignatureType( /*recv=*/ nil /*rectTypeParams=*/, nil /*typeParams=*/, nil /*params=*/, nil /*results=*/, nil /*variadic=*/, false))
		funcCtx.pkgCtx.Uses[id] = mainFunc
		call := &ast.CallExpr{Fun: id}
		ifStmt := &ast.IfStmt{
			Cond: funcCtx.newIdent("$pkg === $mainPkg", types.Typ[types.Bool]),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: call},
					&ast.AssignStmt{
						Lhs: []ast.Expr{funcCtx.newIdent("$mainFinished", types.Typ[types.Bool])},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{funcCtx.newConst(types.Typ[types.Bool], constant.MakeBool(true))},
					},
				},
			},
		}
		if len(funcCtx.pkgCtx.FuncDeclInfos[mainFunc].Blocking) != 0 {
			funcCtx.Blocking[call] = true
			funcCtx.Flattened[ifStmt] = true
		}
		funcDecls = append(funcDecls, &Decl{
			InitCode: funcCtx.CatchOutput(1, func() {
				funcCtx.translateStmt(ifStmt, nil)
			}),
		})
	}

	// named types
	var typeDecls []*Decl
	for _, o := range funcCtx.pkgCtx.typeNames.Slice() {
		if o.IsAlias() {
			continue
		}
		typ := o.Type().(*types.Named)
		var instances []typeparams.Instance
		if typ.TypeParams() != nil {
			instances = instancesByObj[o]
		} else {
			instances = []typeparams.Instance{{Object: o}}
		}

		typeName := funcCtx.objectName(o)

		varDecl := Decl{Vars: []string{typeName}}
		if typ.TypeParams() != nil {
			varDecl.DeclCode = funcCtx.CatchOutput(0, func() {
				funcCtx.Printf("%s = {};", funcCtx.objectName(o))
			})
		}
		if isPkgLevel(o) {
			varDecl.TypeInitCode = funcCtx.CatchOutput(0, func() {
				funcCtx.Printf("$pkg.%s = %s;", encodeIdent(o.Name()), funcCtx.objectName(o))
			})
		}
		typeDecls = append(typeDecls, &varDecl)

		for _, inst := range instances {
			funcCtx.typeResolver = typeparams.NewResolver(funcCtx.pkgCtx.typesCtx, typeparams.ToSlice(typ.TypeParams()), inst.TArgs)

			named := typ
			if !inst.IsTrivial() {
				instantiated, err := types.Instantiate(funcCtx.pkgCtx.typesCtx, typ, inst.TArgs, true)
				if err != nil {
					return nil, fmt.Errorf("failed to instantiate type %v with args %v: %w", typ, inst.TArgs, err)
				}
				named = instantiated.(*types.Named)
			}
			underlying := named.Underlying()
			d := Decl{
				DceObjectFilter: o.Name(),
			}
			d.DceDeps = collectDependencies(func() {
				d.DeclCode = funcCtx.CatchOutput(0, func() {
					size := int64(0)
					constructor := "null"
					switch t := underlying.(type) {
					case *types.Struct:
						params := make([]string, t.NumFields())
						for i := 0; i < t.NumFields(); i++ {
							params[i] = fieldName(t, i) + "_"
						}
						constructor = fmt.Sprintf("function(%s) {\n\t\tthis.$val = this;\n\t\tif (arguments.length === 0) {\n", strings.Join(params, ", "))
						for i := 0; i < t.NumFields(); i++ {
							constructor += fmt.Sprintf("\t\t\tthis.%s = %s;\n", fieldName(t, i), funcCtx.translateExpr(funcCtx.zeroValue(t.Field(i).Type())).String())
						}
						constructor += "\t\t\treturn;\n\t\t}\n"
						for i := 0; i < t.NumFields(); i++ {
							constructor += fmt.Sprintf("\t\tthis.%[1]s = %[1]s_;\n", fieldName(t, i))
						}
						constructor += "\t}"
					case *types.Basic, *types.Array, *types.Slice, *types.Chan, *types.Signature, *types.Interface, *types.Pointer, *types.Map:
						size = sizes32.Sizeof(t)
					}
					if tPointer, ok := underlying.(*types.Pointer); ok {
						if _, ok := tPointer.Elem().Underlying().(*types.Array); ok {
							// Array pointers have non-default constructors to support wrapping
							// of the native objects.
							constructor = "$arrayPtrCtor()"
						}
					}
					funcCtx.Printf(`%s = $newType(%d, %s, %q, %t, "%s", %t, %s);`, funcCtx.instName(inst), size, typeKind(typ), inst.TypeString(), o.Name() != "", o.Pkg().Path(), o.Exported(), constructor)
				})
				d.MethodListCode = funcCtx.CatchOutput(0, func() {
					if _, ok := underlying.(*types.Interface); ok {
						return
					}
					var methods []string
					var ptrMethods []string
					for i := 0; i < named.NumMethods(); i++ {
						method := named.Method(i)
						name := method.Name()
						if reservedKeywords[name] {
							name += "$"
						}
						pkgPath := ""
						if !method.Exported() {
							pkgPath = method.Pkg().Path()
						}
						t := method.Type().(*types.Signature)
						entry := fmt.Sprintf(`{prop: "%s", name: %s, pkg: "%s", typ: $funcType(%s)}`, name, encodeString(method.Name()), pkgPath, funcCtx.initArgs(t))
						if _, isPtr := t.Recv().Type().(*types.Pointer); isPtr {
							ptrMethods = append(ptrMethods, entry)
							continue
						}
						methods = append(methods, entry)
					}
					if len(methods) > 0 {
						funcCtx.Printf("%s.methods = [%s];", funcCtx.instName(inst), strings.Join(methods, ", "))
					}
					if len(ptrMethods) > 0 {
						funcCtx.Printf("%s.methods = [%s];", funcCtx.typeName(types.NewPointer(named)), strings.Join(ptrMethods, ", "))
					}
				})
				switch t := underlying.(type) {
				case *types.Array, *types.Chan, *types.Interface, *types.Map, *types.Pointer, *types.Slice, *types.Signature, *types.Struct:
					d.TypeInitCode = funcCtx.CatchOutput(0, func() {
						funcCtx.Printf("%s.init(%s);", funcCtx.instName(inst), funcCtx.initArgs(t))
					})
				}
			})
			typeDecls = append(typeDecls, &d)
		}
		funcCtx.typeResolver = nil
	}

	// anonymous types
	for _, t := range funcCtx.pkgCtx.anonTypes {
		d := Decl{
			Vars:            []string{t.Name()},
			DceObjectFilter: t.Name(),
		}
		d.DceDeps = collectDependencies(func() {
			d.DeclCode = []byte(fmt.Sprintf("\t%s = $%sType(%s);\n", t.Name(), strings.ToLower(typeKind(t.Type())[5:]), funcCtx.initArgs(t.Type())))
		})
		typeDecls = append(typeDecls, &d)
	}

	var allDecls []*Decl
	for _, d := range append(append(append(importDecls, typeDecls...), varDecls...), funcDecls...) {
		d.DeclCode = removeWhitespace(d.DeclCode, minify)
		d.MethodListCode = removeWhitespace(d.MethodListCode, minify)
		d.TypeInitCode = removeWhitespace(d.TypeInitCode, minify)
		d.InitCode = removeWhitespace(d.InitCode, minify)
		allDecls = append(allDecls, d)
	}

	if len(funcCtx.pkgCtx.errList) != 0 {
		return nil, funcCtx.pkgCtx.errList
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
		return fc.typeName(t.Elem())
	case *types.Slice:
		return fc.typeName(t.Elem())
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
	case *types.TypeParam:
		err := bailout(fmt.Errorf(`%v has unexpected generic type parameter %T`, ty, ty))
		panic(err)
	default:
		err := bailout(fmt.Errorf("%v has unexpected type %T", ty, ty))
		panic(err)
	}
}

func (fc *funcContext) translateToplevelFunction(fun *ast.FuncDecl, info *analysis.FuncInfo, inst typeparams.Instance) []byte {
	o := inst.Object.(*types.Func)
	sig := o.Type().(*types.Signature)
	var recv *ast.Ident
	if fun.Recv != nil && fun.Recv.List[0].Names != nil {
		recv = fun.Recv.List[0].Names[0]
	}

	var joinedParams string
	primaryFunction := func(funcRef string) []byte {
		if fun.Body == nil {
			return []byte(fmt.Sprintf("\t%s = function() {\n\t\t$throwRuntimeError(\"native function not implemented: %s\");\n\t};\n", funcRef, o.FullName()))
		}

		params, fun := translateFunction(fun.Type, recv, fun.Body, fc, sig, info, funcRef, inst)
		joinedParams = strings.Join(params, ", ")
		return []byte(fmt.Sprintf("\t%s = %s;\n", funcRef, fun))
	}

	code := bytes.NewBuffer(nil)

	if fun.Recv == nil {
		funcRef := fc.instName(inst)
		code.Write(primaryFunction(funcRef))
		if fun.Name.IsExported() {
			fmt.Fprintf(code, "\t$pkg.%s = %s;\n", encodeIdent(fun.Name.Name), funcRef)
		}
		return code.Bytes()
	}

	recvInst := inst.Recv()
	recvInstName := fc.instName(recvInst)
	recvType := recvInst.Object.Type().(*types.Named)
	funName := fun.Name.Name
	if reservedKeywords[funName] {
		funName += "$"
	}

	if _, isStruct := recvType.Underlying().(*types.Struct); isStruct {
		code.Write(primaryFunction(recvInstName + ".ptr.prototype." + funName))
		fmt.Fprintf(code, "\t%s.prototype.%s = function(%s) { return this.$val.%s(%s); };\n", recvInstName, funName, joinedParams, funName, joinedParams)
		return code.Bytes()
	}

	if ptr, isPointer := sig.Recv().Type().(*types.Pointer); isPointer {
		if _, isArray := ptr.Elem().Underlying().(*types.Array); isArray {
			code.Write(primaryFunction(recvInstName + ".prototype." + funName))
			fmt.Fprintf(code, "\t$ptrType(%s).prototype.%s = function(%s) { return (new %s(this.$get())).%s(%s); };\n", recvInstName, funName, joinedParams, recvInstName, funName, joinedParams)
			return code.Bytes()
		}
		return primaryFunction(fmt.Sprintf("$ptrType(%s).prototype.%s", recvInstName, funName))
	}

	value := "this.$get()"
	if isWrapped(recvType) {
		value = fmt.Sprintf("new %s(%s)", recvInstName, value)
	}
	code.Write(primaryFunction(recvInstName + ".prototype." + funName))
	fmt.Fprintf(code, "\t$ptrType(%s).prototype.%s = function(%s) { return %s.%s(%s); };\n", recvInstName, funName, joinedParams, value, funName, joinedParams)
	return code.Bytes()
}

func translateFunction(typ *ast.FuncType, recv *ast.Ident, body *ast.BlockStmt, outerContext *funcContext, sig *types.Signature, info *analysis.FuncInfo, funcRef string, inst typeparams.Instance) ([]string, string) {
	if info == nil {
		panic("nil info")
	}

	c := &funcContext{
		FuncInfo:     info,
		pkgCtx:       outerContext.pkgCtx,
		parent:       outerContext,
		allVars:      make(map[string]int, len(outerContext.allVars)),
		localVars:    []string{},
		flowDatas:    map[*types.Label]*flowData{nil: {}},
		caseCounter:  1,
		labelCases:   make(map[*types.Label]int),
		typeResolver: outerContext.typeResolver,
		objectNames:  map[types.Object]string{},
		sig:          &typesutil.Signature{Sig: sig},
	}
	for k, v := range outerContext.allVars {
		c.allVars[k] = v
	}
	prevEV := c.pkgCtx.escapingVars

	if sig.TypeParams().Len() > 0 {
		c.typeResolver = typeparams.NewResolver(c.pkgCtx.typesCtx, typeparams.ToSlice(sig.TypeParams()), inst.TArgs)
	} else if sig.RecvTypeParams().Len() > 0 {
		c.typeResolver = typeparams.NewResolver(c.pkgCtx.typesCtx, typeparams.ToSlice(sig.RecvTypeParams()), inst.TArgs)
	}
	if c.objectNames == nil {
		c.objectNames = map[types.Object]string{}
	}

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

	bodyOutput := string(c.CatchOutput(1, func() {
		if len(c.Blocking) != 0 {
			c.pkgCtx.Scopes[body] = c.pkgCtx.Scopes[typ]
			c.handleEscapingVars(body)
		}

		if c.sig != nil && c.sig.HasNamedResults() {
			c.resultNames = make([]ast.Expr, c.sig.Sig.Results().Len())
			for i := 0; i < c.sig.Sig.Results().Len(); i++ {
				result := c.sig.Sig.Results().At(i)
				typ := c.typeResolver.Substitute(result.Type())
				c.Printf("%s = %s;", c.objectName(result), c.translateExpr(c.zeroValue(typ)).String())
				id := ast.NewIdent("")
				c.pkgCtx.Uses[id] = result
				c.resultNames[i] = c.setType(id, typ)
			}
		}

		if recv != nil && !isBlank(recv) {
			this := "this"
			if isWrapped(c.typeOf(recv)) {
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

	var prefix, suffix, functionName string

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
		if funcRef == "" {
			funcRef = "$b"
			functionName = " $b"
		}

		localVars := append([]string{}, c.localVars...)
		// There are several special variables involved in handling blocking functions:
		// $r is sometimes used as a temporary variable to store blocking call result.
		// $c indicates that a function is being resumed after a blocking call when set to true.
		// $f is an object used to save and restore function context for blocking calls.
		localVars = append(localVars, "$r")
		// If a blocking function is being resumed, initialize local variables from the saved context.
		localVarDefs = fmt.Sprintf("var {%s, $c} = $restore(this, {%s});\n", strings.Join(localVars, ", "), strings.Join(params, ", "))
		// If the function gets blocked, save local variables for future.
		saveContext := fmt.Sprintf("var $f = {$blk: "+funcRef+", $c: true, $r, %s};", strings.Join(c.localVars, ", "))

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
		if c.resultNames == nil && c.sig.HasResults() {
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
		bodyOutput = c.Indentation(1) + "/* */" + prefix + "\n" + bodyOutput
	}
	if suffix != "" {
		bodyOutput = bodyOutput + c.Indentation(1) + "/* */" + suffix + "\n"
	}
	if localVarDefs != "" {
		bodyOutput = c.Indentation(1) + localVarDefs + bodyOutput
	}

	c.pkgCtx.escapingVars = prevEV

	return params, fmt.Sprintf("function%s(%s) {\n%s%s}", functionName, strings.Join(params, ", "), bodyOutput, c.Indentation(0))
}
