package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

type continueStmt struct {
	forStmt      *ast.ForStmt
	analyzeStack astPath
}

func newContinueStmt(forStmt *ast.ForStmt, stack astPath) continueStmt {
	cs := continueStmt{
		forStmt:      forStmt,
		analyzeStack: stack.copy(),
	}
	return cs
}

// astPath is a list of AST nodes where each previous node is a parent of the
// next node.
type astPath []ast.Node

func (src astPath) copy() astPath {
	dst := make(astPath, len(src))
	copy(dst, src)
	return dst
}

func (ap astPath) String() string {
	s := &strings.Builder{}
	s.WriteString("[")
	for i, n := range ap {
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "%T(%p)", n, n)
	}
	s.WriteString("]")
	return s.String()
}

type Info struct {
	*types.Info
	Pkg           *types.Package
	typeCtx       *types.Context
	instanceSets  *typeparams.PackageInstanceSets
	HasPointer    map[*types.Var]bool
	funcInstInfos *typeparams.InstanceMap[*FuncInfo]
	funcLitInfos  map[*ast.FuncLit]*FuncInfo
	InitFuncInfo  *FuncInfo // Context for package variable initialization.

	isImportedBlocking func(typeparams.Instance) bool // For functions from other packages.
	allInfos           []*FuncInfo
}

func (info *Info) newFuncInfo(n ast.Node, inst *typeparams.Instance) *FuncInfo {
	funcInfo := &FuncInfo{
		pkgInfo:            info,
		Flattened:          make(map[ast.Node]bool),
		Blocking:           make(map[ast.Node]bool),
		GotoLabel:          make(map[*types.Label]bool),
		localInstCallees:   new(typeparams.InstanceMap[[]astPath]),
		literalFuncCallees: make(map[*ast.FuncLit][]astPath),
	}

	// Register the function in the appropriate map.
	switch n := n.(type) {
	case *ast.FuncDecl:
		if n.Body == nil {
			// Function body comes from elsewhere (for example, from a go:linkname
			// directive), conservatively assume that it may be blocking.
			// TODO(nevkontakte): It is possible to improve accuracy of this detection.
			// Since GopherJS supports only "import-style" go:linkname, at this stage
			// the compiler already determined whether the implementation function is
			// blocking, and we could check that.
			funcInfo.Blocking[n] = true
		}

		if inst == nil {
			inst = &typeparams.Instance{Object: info.Defs[n.Name]}
		}
		info.funcInstInfos.Set(*inst, funcInfo)

	case *ast.FuncLit:
		info.funcLitInfos[n] = funcInfo
	}

	// And add it to the list of all functions.
	info.allInfos = append(info.allInfos, funcInfo)

	return funcInfo
}

func (info *Info) newFuncInfoInstances(fd *ast.FuncDecl) []*FuncInfo {
	obj := info.Defs[fd.Name]
	instances := info.instanceSets.Pkg(info.Pkg).ForObj(obj)
	if len(instances) == 0 {
		// No instances found, this is a non-generic function.
		return []*FuncInfo{info.newFuncInfo(fd, nil)}
	}

	funcInfos := make([]*FuncInfo, 0, len(instances))
	for _, inst := range instances {
		fi := info.newFuncInfo(fd, &inst)
		if sig, ok := obj.Type().(*types.Signature); ok {
			tp := typeparams.ToSlice(typeparams.SignatureTypeParams(sig))
			fi.resolver = typeparams.NewResolver(info.typeCtx, tp, inst.TArgs)
		}
		funcInfos = append(funcInfos, fi)
	}
	return funcInfos
}

// IsBlocking returns true if the function may contain blocking calls or operations.
func (info *Info) IsBlocking(inst typeparams.Instance) bool {
	if funInfo := info.FuncInfo(inst); funInfo != nil {
		return funInfo.HasBlocking()
	}
	panic(fmt.Errorf(`info did not have function declaration instance for %q`, inst))
}

// FuncInfo returns information about the given function instance.
// The object in the instance must be a function declaration.
// If not information is found, nil is returned.
func (info *Info) FuncInfo(inst typeparams.Instance) *FuncInfo {
	return info.funcInstInfos.Get(inst)
}

// FuncLitInfo returns information about the given function literal.
// It panics if the function literal is not found in this package info.
// If not information is found, nil is returned.
func (info *Info) FuncLitInfo(fun *ast.FuncLit) *FuncInfo {
	return info.funcLitInfos[fun]
}

// VarsWithInitializers returns a set of package-level variables that have
// explicit initializers.
func (info *Info) VarsWithInitializers() map[*types.Var]bool {
	result := map[*types.Var]bool{}
	for _, init := range info.InitOrder {
		for _, o := range init.Lhs {
			result[o] = true
		}
	}
	return result
}

func AnalyzePkg(files []*ast.File, fileSet *token.FileSet, typesInfo *types.Info, typeCtx *types.Context, typesPkg *types.Package, instanceSets *typeparams.PackageInstanceSets, isBlocking func(typeparams.Instance) bool) *Info {
	info := &Info{
		Info:               typesInfo,
		Pkg:                typesPkg,
		typeCtx:            typeCtx,
		instanceSets:       instanceSets,
		HasPointer:         make(map[*types.Var]bool),
		isImportedBlocking: isBlocking,
		funcInstInfos:      new(typeparams.InstanceMap[*FuncInfo]),
		funcLitInfos:       make(map[*ast.FuncLit]*FuncInfo),
	}
	info.InitFuncInfo = info.newFuncInfo(nil, nil)

	// Traverse the full AST of the package and collect information about existing
	// functions.
	for _, file := range files {
		ast.Walk(info.InitFuncInfo, file)
	}

	for _, funcInfo := range info.allInfos {
		if !funcInfo.HasDefer {
			continue
		}
		// Conservatively assume that if a function has a deferred call, it might be
		// blocking, and therefore all return statements need to be treated as
		// blocking.
		// TODO(nevkontakte): This could be improved by detecting whether a deferred
		// call is actually blocking. Doing so might reduce generated code size a
		// bit.
		for _, returnStmt := range funcInfo.returnStmts {
			funcInfo.markBlocking(returnStmt)
		}
	}

	// Propagate information about blocking calls to the caller functions.
	// For each function we check all other functions it may call and if any of
	// them are blocking, we mark the caller blocking as well. The process is
	// repeated until no new blocking functions is detected.
	for {
		done := true
		for _, caller := range info.allInfos {
			// Check calls to named functions and function-typed variables.
			caller.localInstCallees.Iterate(func(callee typeparams.Instance, callSites []astPath) {
				if info.funcInstInfos.Get(callee).HasBlocking() {
					for _, callSite := range callSites {
						caller.markBlocking(callSite)
					}
					caller.localInstCallees.Delete(callee)
					done = false
				}
			})

			// Check direct calls to function literals.
			for callee, callSites := range caller.literalFuncCallees {
				if info.funcLitInfos[callee].HasBlocking() {
					for _, callSite := range callSites {
						caller.markBlocking(callSite)
					}
					delete(caller.literalFuncCallees, callee)
					done = false
				}
			}
		}
		if done {
			break
		}
	}

	// After all function blocking information was propagated, mark flow control
	// statements as blocking whenever they may lead to a blocking function call.
	for _, funcInfo := range info.allInfos {
		for _, continueStmt := range funcInfo.continueStmts {
			if funcInfo.Blocking[continueStmt.forStmt.Post] {
				// If a for-loop post-expression is blocking, the continue statement
				// that leads to it must be treated as blocking.
				funcInfo.markBlocking(continueStmt.analyzeStack)
			}
		}
	}

	return info
}

type FuncInfo struct {
	HasDefer bool
	// Nodes are "flattened" into a switch-case statement when we need to be able
	// to jump into an arbitrary position in the code with a GOTO statement, or
	// resume a goroutine after a blocking call unblocks.
	Flattened map[ast.Node]bool
	// Blocking indicates that either the AST node itself or its descendant may
	// block goroutine execution (for example, a channel operation).
	Blocking map[ast.Node]bool
	// GotoLabel indicates a label referenced by a goto statement, rather than a
	// named loop.
	GotoLabel map[*types.Label]bool
	// List of continue statements in the function.
	continueStmts []continueStmt
	// List of return statements in the function.
	returnStmts []astPath
	// List of other named functions from the current package this function calls.
	// If any of them are blocking, this function will become blocking too.
	localInstCallees *typeparams.InstanceMap[[]astPath]
	// List of function literals directly called from this function (for example:
	// `func() { /* do stuff */ }()`). This is distinct from function literals
	// assigned to named variables (for example: `doStuff := func() {};
	// doStuff()`), which are handled by localNamedCallees. If any of them are
	// identified as blocking, this function will become blocking too.
	literalFuncCallees map[*ast.FuncLit][]astPath
	// resolver is used by this function instance to resolve any type arguments
	// for internal function calls.
	// This may be nil if not an instance of a generic function.
	resolver *typeparams.Resolver

	pkgInfo      *Info // Function's parent package.
	visitorStack astPath
}

// HasBlocking indicates if this function may block goroutine execution.
//
// For example, a channel operation in a function or a call to another
// possibly blocking function may block the function.
func (fi *FuncInfo) HasBlocking() bool {
	return fi == nil || len(fi.Blocking) != 0
}

func (fi *FuncInfo) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		if len(fi.visitorStack) != 0 {
			fi.visitorStack = fi.visitorStack[:len(fi.visitorStack)-1]
		}
		return nil
	}
	fi.visitorStack = append(fi.visitorStack, node)

	switch n := node.(type) {
	case *ast.FuncDecl:
		// Analyze all the instances of the function declarations
		// in their own context with their own type arguments.
		fis := fi.pkgInfo.newFuncInfoInstances(n)
		if n.Body != nil {
			for _, fi := range fis {
				ast.Walk(fi, n.Body)
			}
		}
		return nil
	case *ast.FuncLit:
		// Analyze the function literal in its own context.
		return fi.pkgInfo.newFuncInfo(n, nil)
	case *ast.BranchStmt:
		switch n.Tok {
		case token.GOTO:
			// Emulating GOTO in JavaScript requires the code to be flattened into a
			// switch-statement.
			fi.markFlattened(fi.visitorStack)
			fi.GotoLabel[fi.pkgInfo.Uses[n.Label].(*types.Label)] = true
		case token.CONTINUE:
			loopStmt := astutil.FindLoopStmt(fi.visitorStack, n, fi.pkgInfo.Info)
			if forStmt, ok := (loopStmt).(*ast.ForStmt); ok {
				// In `for x; y; z { ... }` loops `z` may be potentially blocking
				// and therefore continue expression that triggers it would have to
				// be treated as blocking.
				fi.continueStmts = append(fi.continueStmts, newContinueStmt(forStmt, fi.visitorStack))
			}
		}
		return fi
	case *ast.CallExpr:
		return fi.visitCallExpr(n)
	case *ast.SendStmt:
		// Sending into a channel is blocking.
		fi.markBlocking(fi.visitorStack)
		return fi
	case *ast.UnaryExpr:
		switch n.Op {
		case token.AND:
			if id, ok := astutil.RemoveParens(n.X).(*ast.Ident); ok {
				fi.pkgInfo.HasPointer[fi.pkgInfo.Uses[id].(*types.Var)] = true
			}
		case token.ARROW:
			// Receiving from a channel is blocking.
			fi.markBlocking(fi.visitorStack)
		}
		return fi
	case *ast.RangeStmt:
		if _, ok := fi.pkgInfo.TypeOf(n.X).Underlying().(*types.Chan); ok {
			// for-range loop over a channel is blocking.
			fi.markBlocking(fi.visitorStack)
		}
		return fi
	case *ast.SelectStmt:
		for _, s := range n.Body.List {
			if s.(*ast.CommClause).Comm == nil { // default clause
				return fi
			}
		}
		// Select statements without a default case are blocking.
		fi.markBlocking(fi.visitorStack)
		return fi
	case *ast.CommClause:
		// FIXME(nevkontakte): Does this need to be manually spelled out? Presumably
		// ast.Walk would visit all those nodes anyway, and we are not creating any
		// new contexts here.
		// https://github.com/gopherjs/gopherjs/issues/230 seems to be relevant?
		switch comm := n.Comm.(type) {
		case *ast.SendStmt:
			ast.Walk(fi, comm.Chan)
			ast.Walk(fi, comm.Value)
		case *ast.ExprStmt:
			ast.Walk(fi, comm.X.(*ast.UnaryExpr).X)
		case *ast.AssignStmt:
			ast.Walk(fi, comm.Rhs[0].(*ast.UnaryExpr).X)
		}
		for _, s := range n.Body {
			ast.Walk(fi, s)
		}
		return nil // The subtree was manually checked, no need to visit it again.
	case *ast.GoStmt:
		// Unlike a regular call, the function in a go statement doesn't block the
		// caller goroutine, but the expression that determines the function and its
		// arguments still need to be checked.
		ast.Walk(fi, n.Call.Fun)
		for _, arg := range n.Call.Args {
			ast.Walk(fi, arg)
		}
		return nil // The subtree was manually checked, no need to visit it again.
	case *ast.DeferStmt:
		fi.HasDefer = true
		if funcLit, ok := n.Call.Fun.(*ast.FuncLit); ok {
			ast.Walk(fi, funcLit.Body)
		}
		return fi
	case *ast.ReturnStmt:
		// Capture all return statements in the function. They could become blocking
		// if the function has a blocking deferred call.
		fi.returnStmts = append(fi.returnStmts, fi.visitorStack.copy())
		return fi
	default:
		return fi
	}
	// Deliberately no return here to make sure that each of the cases above is
	// self-sufficient and explicitly decides in which context the its AST subtree
	// needs to be analyzed.
}

func (fi *FuncInfo) visitCallExpr(n *ast.CallExpr) ast.Visitor {
	switch f := astutil.RemoveParens(n.Fun).(type) {
	case *ast.Ident:
		fi.callToNamedFunc(fi.instanceForIdent(f))
	case *ast.SelectorExpr:
		if sel := fi.pkgInfo.Selections[f]; sel != nil {
			if typesutil.IsJsObject(sel.Recv()) {
				// js.Object methods are known to be non-blocking, but we still must
				// check its arguments.
			} else {
				// selection is a method call like `foo.Bar()`, where `foo` might
				// be generic and needs to be substituted with the type argument.
				fi.callToNamedFunc(fi.instanceFoSelection(sel))
			}
		} else {
			fi.callToNamedFunc(fi.instanceForIdent(f.Sel))
		}
	case *ast.FuncLit:
		// Collect info about the function literal itself.
		ast.Walk(fi, n.Fun)

		// Check all argument expressions.
		for _, arg := range n.Args {
			ast.Walk(fi, arg)
		}
		// Register literal function call site in case it is identified as blocking.
		fi.literalFuncCallees[f] = append(fi.literalFuncCallees[f], fi.visitorStack.copy())
		return nil // No need to walk under this CallExpr, we already did it manually.
	case *ast.IndexExpr:
		// Collect info about the instantiated type or function, or index expression.
		if astutil.IsTypeExpr(f, fi.pkgInfo.Info) {
			// This is a type conversion to an instance of a generic type,
			// not a call. Type assertion itself is not blocking, but we will
			// visit the input expression.
		} else if astutil.IsTypeExpr(f.Index, fi.pkgInfo.Info) {
			// This is a call of an instantiation of a generic function,
			// e.g. `foo[int]` in `func foo[T any]() { ... }; func main() { foo[int]() }`
			fi.callToNamedFunc(fi.instanceForIdent(f.X.(*ast.Ident)))
		} else {
			// The called function is gotten with an index or key from a map, array, or slice.
			// e.g. `m := map[string]func(){}; m["key"]()`, `s := []func(); s[0]()`.
			// Since we can't predict if the returned function will be blocking
			// or not, we have to be conservative and assume that function might be blocking.
			fi.markBlocking(fi.visitorStack)
		}
	case *ast.IndexListExpr:
		// Collect info about the instantiated type or function.
		if astutil.IsTypeExpr(f, fi.pkgInfo.Info) {
			// This is a type conversion to an instance of a generic type,
			// not a call. Type assertion itself is not blocking, but we will
			// visit the input expression.
		} else {
			// This is a call of an instantiation of a generic function,
			// e.g. `foo[int, bool]` in `func foo[T1, T2 any]() { ... }; func main() { foo[int, bool]() }`
			fi.callToNamedFunc(fi.instanceForIdent(f.X.(*ast.Ident)))
		}
	default:
		if astutil.IsTypeExpr(f, fi.pkgInfo.Info) {
			// This is a type conversion, not a call. Type assertion itself is not
			// blocking, but we will visit the input expression.
		} else {
			// The function is returned by a non-trivial expression. We have to be
			// conservative and assume that function might be blocking.
			fi.markBlocking(fi.visitorStack)
		}
	}

	return fi
}

func (fi *FuncInfo) instanceForIdent(fnId *ast.Ident) typeparams.Instance {
	tArgs := fi.pkgInfo.Info.Instances[fnId].TypeArgs
	return typeparams.Instance{
		Object: fi.pkgInfo.Uses[fnId],
		TArgs:  fi.resolver.SubstituteAll(tArgs),
	}
}

func (fi *FuncInfo) instanceFoSelection(sel *types.Selection) typeparams.Instance {
	if _, ok := sel.Obj().Type().(*types.Signature); ok {
		// Substitute the selection to ensure that the receiver has the correct
		// type arguments propagated down from the caller.
		resolved := fi.resolver.SubstituteSelection(sel)
		sig := resolved.Obj().Type().(*types.Signature)

		// Using the substituted receiver type, find the instance of this call.
		// This does require looking up the original method in the receiver type
		// that may or may not have been the receiver prior to the substitution.
		if recv := sig.Recv(); recv != nil {
			typ := recv.Type()
			if ptrType, ok := typ.(*types.Pointer); ok {
				typ = ptrType.Elem()
			}

			if rt, ok := typ.(*types.Named); ok {
				origMethod, _, _ := types.LookupFieldOrMethod(rt.Origin(), true, rt.Obj().Pkg(), resolved.Obj().Name())
				if origMethod == nil {
					panic(fmt.Errorf(`failed to lookup field %q in type %v`, resolved.Obj().Name(), rt.Origin()))
				}
				return typeparams.Instance{
					Object: origMethod,
					TArgs:  fi.resolver.SubstituteAll(rt.TypeArgs()),
				}
			}
		}
	}
	return typeparams.Instance{Object: sel.Obj()}
}

func (fi *FuncInfo) callToNamedFunc(callee typeparams.Instance) {
	switch o := callee.Object.(type) {
	case *types.Func:
		o = o.Origin()
		if recv := o.Type().(*types.Signature).Recv(); recv != nil {
			if _, ok := recv.Type().Underlying().(*types.Interface); ok {
				// Conservatively assume that an interface implementation may be blocking.
				fi.markBlocking(fi.visitorStack)
				return
			}
		}
		if o.Pkg() != fi.pkgInfo.Pkg {
			if fi.pkgInfo.isImportedBlocking(callee) {
				fi.markBlocking(fi.visitorStack)
			}
			return
		}
		// We probably don't know yet whether the callee function is blocking.
		// Record the calls site for the later stage.
		paths := fi.localInstCallees.Get(callee)
		paths = append(paths, fi.visitorStack.copy())
		fi.localInstCallees.Set(callee, paths)
	case *types.Var:
		// Conservatively assume that a function in a variable might be blocking.
		fi.markBlocking(fi.visitorStack)
	}
}

func (fi *FuncInfo) markBlocking(stack astPath) {
	for _, n := range stack {
		fi.Blocking[n] = true
		fi.Flattened[n] = true
	}
}

func (fi *FuncInfo) markFlattened(stack astPath) {
	for _, n := range stack {
		fi.Flattened[n] = true
	}
}
