package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/astutil"
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
	HasPointer    map[*types.Var]bool
	FuncDeclInfos map[*types.Func]*FuncInfo
	FuncLitInfos  map[*ast.FuncLit]*FuncInfo
	InitFuncInfo  *FuncInfo // Context for package variable initialization.

	isImportedBlocking func(*types.Func) bool // For functions from other packages.
	allInfos           []*FuncInfo
}

func (info *Info) newFuncInfo(n ast.Node) *FuncInfo {
	funcInfo := &FuncInfo{
		pkgInfo:      info,
		Flattened:    make(map[ast.Node]bool),
		Blocking:     make(map[ast.Node]bool),
		GotoLabel:    make(map[*types.Label]bool),
		localCallees: make(map[*types.Func][]astPath),
	}

	// Register the function in the appropriate map.
	switch n := n.(type) {
	case *ast.FuncDecl:
		if n.Body == nil {
			// Function body comes from elsewhere (for example, from a go:linkname
			// directive), conservatively assume that it may be blocking.
			// TODO(nevkontakte): It is possible to improve accuracy of this detection.
			// Since GopherJS supports inly "import-style" go:linkname, at this stage
			// the compiler already determined whether the implementation function is
			// blocking, and we could check that.
			funcInfo.Blocking[n] = true
		}
		info.FuncDeclInfos[info.Defs[n.Name].(*types.Func)] = funcInfo
	case *ast.FuncLit:
		info.FuncLitInfos[n] = funcInfo
	}

	// And add it to the list of all functions.
	info.allInfos = append(info.allInfos, funcInfo)

	return funcInfo
}

func (info *Info) IsBlocking(fun *types.Func) bool {
	return len(info.FuncDeclInfos[fun].Blocking) > 0
}

func AnalyzePkg(files []*ast.File, fileSet *token.FileSet, typesInfo *types.Info, typesPkg *types.Package, isBlocking func(*types.Func) bool) *Info {
	info := &Info{
		Info:               typesInfo,
		Pkg:                typesPkg,
		HasPointer:         make(map[*types.Var]bool),
		isImportedBlocking: isBlocking,
		FuncDeclInfos:      make(map[*types.Func]*FuncInfo),
		FuncLitInfos:       make(map[*ast.FuncLit]*FuncInfo),
	}
	info.InitFuncInfo = info.newFuncInfo(nil)

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
	for {
		done := true
		for _, caller := range info.allInfos {
			for callee, callSites := range caller.localCallees {
				if info.IsBlocking(callee) {
					for _, callSite := range callSites {
						caller.markBlocking(callSite)
					}
					delete(caller.localCallees, callee)
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
	// GotoLavel indicates a label referenced by a goto statement, rather than a
	// named loop.
	GotoLabel map[*types.Label]bool
	// List of continue statements in the function.
	continueStmts []continueStmt
	// List of return statements in the function.
	returnStmts []astPath
	// List of other functions from the current package this function calls. If
	// any of them are blocking, this function will become blocking too.
	localCallees map[*types.Func][]astPath

	pkgInfo      *Info // Function's parent package.
	visitorStack astPath
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
	case *ast.FuncDecl, *ast.FuncLit:
		// Analyze the function in its own context.
		return fi.pkgInfo.newFuncInfo(n)
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
		fi.callTo(fi.pkgInfo.Uses[f])
	case *ast.SelectorExpr:
		if sel := fi.pkgInfo.Selections[f]; sel != nil && typesutil.IsJsObject(sel.Recv()) {
			// js.Object methods are known to be non-blocking, but we still must
			// check its arguments.
		} else {
			fi.callTo(fi.pkgInfo.Uses[f.Sel])
		}
	case *ast.FuncLit:
		// Collect info about the function literal itself.
		ast.Walk(fi, n.Fun)

		// Check all argument expressions.
		for _, arg := range n.Args {
			ast.Walk(fi, arg)
		}
		// If the function literal is blocking, this function is blocking to.
		// FIXME(nevkontakte): What if the function literal is calling a blocking
		// function through several layers of indirection? This will only become
		// known at a later stage of analysis.
		if len(fi.pkgInfo.FuncLitInfos[f].Blocking) != 0 {
			fi.markBlocking(fi.visitorStack)
		}
		return nil // No need to walk under this CallExpr, we already did it manually.
	default:
		if astutil.IsTypeExpr(f, fi.pkgInfo.Info) {
			// This is a type assertion, not a call. Type assertion itself is not
			// blocking, but we will visit the expression itself.
		} else {
			// The function is returned by a non-trivial expression. We have to be
			// conservative and assume that function might be blocking.
			fi.markBlocking(fi.visitorStack)
		}
	}

	return fi
}

func (fi *FuncInfo) callTo(callee types.Object) {
	switch o := callee.(type) {
	case *types.Func:
		if recv := o.Type().(*types.Signature).Recv(); recv != nil {
			if _, ok := recv.Type().Underlying().(*types.Interface); ok {
				// Conservatively assume that an interfact implementation might be blocking.
				fi.markBlocking(fi.visitorStack)
				return
			}
		}
		if o.Pkg() != fi.pkgInfo.Pkg {
			if fi.pkgInfo.isImportedBlocking(o) {
				fi.markBlocking(fi.visitorStack)
			}
			return
		}
		// We probably don't know yet whether the callee function is blocking.
		// Record the calls site for the later stage.
		fi.localCallees[o] = append(fi.localCallees[o], fi.visitorStack.copy())
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
