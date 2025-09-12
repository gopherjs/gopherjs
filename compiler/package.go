package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/types/typeutil"

	"github.com/gopherjs/gopherjs/compiler/internal/analysis"
	"github.com/gopherjs/gopherjs/compiler/internal/dce"
	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/sources"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
	"github.com/gopherjs/gopherjs/internal/errlist"
)

// pkgContext maintains compiler context for a specific package.
type pkgContext struct {
	*analysis.Info
	dce.Collector
	additionalSelections map[*ast.SelectorExpr]typesutil.Selection

	typesCtx *types.Context
	// List of type names declared in the package, including those defined inside
	// functions.
	typeNames typesutil.TypeNames
	// Mapping from package import paths to JS variables that were assigned to an
	// imported package and can be used to access it.
	pkgVars      map[string]string
	varPtrNames  map[*types.Var]string
	anonTypes    []*types.TypeName
	anonTypeMap  typeutil.Map
	escapingVars map[*types.Var]bool
	indentation  int
	minify       bool
	fileSet      *token.FileSet
	errList      errlist.ErrorList
	instanceSet  *typeparams.PackageInstanceSets
}

// isMain returns true if this is the main package of the program.
func (pc *pkgContext) isMain() bool {
	return pc.Pkg.Name() == "main"
}

// funcContext maintains compiler context for a specific function.
//
// An instance of this type roughly corresponds to a lexical scope for generated
// JavaScript code (as defined for `var` declarations).
type funcContext struct {
	*analysis.FuncInfo
	// Function instance this context corresponds to, or zero if the context is
	// top-level or doesn't correspond to a function. For function literals, this
	// is a synthetic object that assigns a unique identity to the function.
	instance typeparams.Instance
	// JavaScript identifier assigned to the function object (the word after the
	// "function" keyword in the generated code). This identifier can be used
	// within the function scope to reference the function object. It will also
	// appear in the stack trace.
	funcRef string
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
	// Number of function literals encountered within the current function context.
	funcLitCounter int
}

func newRootCtx(tContext *types.Context, srcs *sources.Sources, minify bool) *funcContext {
	funcCtx := &funcContext{
		FuncInfo: srcs.TypeInfo.InitFuncInfo,
		pkgCtx: &pkgContext{
			Info:                 srcs.TypeInfo,
			additionalSelections: make(map[*ast.SelectorExpr]typesutil.Selection),

			typesCtx:     tContext,
			pkgVars:      make(map[string]string),
			varPtrNames:  make(map[*types.Var]string),
			escapingVars: make(map[*types.Var]bool),
			indentation:  1,
			minify:       minify,
			fileSet:      srcs.FileSet,
			instanceSet:  srcs.TypeInfo.InstanceSets,
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
	return funcCtx
}

type flowData struct {
	postStmt  func()
	beginCase int
	endCase   int
}

// Compile the provided Go sources as a single package.
//
// Provided sources must be prepared so that the type information has been determined,
// and the source files have been sorted by name to ensure reproducible JavaScript output.
func Compile(srcs *sources.Sources, tContext *types.Context, minify bool) (_ *Archive, err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if fe, ok := bailingOut(e); ok {
			// Orderly bailout, return whatever clues we already have.
			fmt.Fprintf(fe, `building package %q`, srcs.ImportPath)
			err = fe
			return
		}
		// Some other unexpected panic, catch the stack trace and return as an error.
		err = bailout(fmt.Errorf("unexpected compiler panic while building package %q: %v", srcs.ImportPath, e))
	}()

	rootCtx := newRootCtx(tContext, srcs, minify)

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
	typeDecls, err := rootCtx.namedTypeDecls(rootCtx.pkgCtx.typeNames)
	if err != nil {
		return nil, err
	}

	// Finally, anonymous types are translated the last, to make sure we've
	// discovered all of them referenced in functions, variable and type
	// declarations.
	typeDecls = append(typeDecls, rootCtx.anonTypeDecls(rootCtx.pkgCtx.anonTypes)...)

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

	return &Archive{
		ImportPath:   srcs.ImportPath,
		Name:         srcs.Package.Name(),
		Imports:      importedPaths,
		Package:      srcs.Package,
		Declarations: allDecls,
		FileSet:      srcs.FileSet,
		Minified:     minify,
		GoLinknames:  srcs.GoLinknames,
	}, nil
}

// PrepareAllSources prepares all sources for compilation by
// parsing go linknames, type checking, sorting, simplifying, and
// performing cross package analysis.
// The results are stored in the provided sources.
//
// All sources must be given at the same time for cross package analysis to
// work correctly. For consistency, the sources should be sorted by import path.
func PrepareAllSources(allSources []*sources.Sources, importer sources.Importer, tContext *types.Context) error {
	// Sort the files by name in each source to ensure consistent order of processing.
	for _, srcs := range allSources {
		srcs.Sort()
	}

	// This will be performed recursively for all dependencies
	// to get the packages for the sources.
	// Since some packages might not be recursively reached via the root sources,
	// e.g. runtime, we need to try to TypeCheck all of them here.
	// Any sources that have already been type checked will no-op.
	for _, srcs := range allSources {
		if err := srcs.TypeCheck(importer, sizes32, tContext); err != nil {
			return err
		}
	}

	// Extract all go:linkname compiler directives from the package source.
	for _, srcs := range allSources {
		if err := srcs.ParseGoLinknames(); err != nil {
			return err
		}
	}

	// Simply the source files.
	for _, srcs := range allSources {
		srcs.Simplify()
	}

	// Collect all the generic type instances from all the packages.
	// This must be done for all sources prior to any analysis.
	instances := &typeparams.PackageInstanceSets{}
	tc := &typeparams.Collector{
		TContext:  tContext,
		Instances: instances,
	}
	for _, srcs := range allSources {
		srcs.CollectInstances(tc)
	}
	tc.Finish()

	// Analyze the package to determine type parameters instances, blocking,
	// and other type information. This will not populate the information.
	for _, srcs := range allSources {
		srcs.Analyze(importer, tContext, instances)
	}

	// Propagate the analysis information across all packages.
	allInfo := make([]*analysis.Info, len(allSources))
	for i, src := range allSources {
		allInfo[i] = src.TypeInfo
	}
	analysis.PropagateAnalysis(allInfo)
	return nil
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
			ft := fc.fieldType(t, i)
			fields[i] = fmt.Sprintf(`{prop: "%s", name: %s, embedded: %t, exported: %t, typ: %s, tag: %s}`,
				fieldName(t, i), encodeString(field.Name()), field.Anonymous(), field.Exported(), fc.typeName(ft), encodeString(t.Tag(i)))
		}
		return fmt.Sprintf(`"%s", [%s]`, pkgPath, strings.Join(fields, ", "))
	case *types.TypeParam:
		tr := fc.typeResolver.Substitute(ty)
		if tr != ty {
			return fc.initArgs(tr)
		}
		err := bailout(fmt.Errorf(`"%v" has unexpected generic type parameter %T`, ty, ty))
		panic(err)
	default:
		err := bailout(fmt.Errorf("%v has unexpected type %T", ty, ty))
		panic(err)
	}
}
