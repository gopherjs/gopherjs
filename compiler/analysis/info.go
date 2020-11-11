package analysis

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/goplusjs/gopherjs/compiler/astutil"
	"github.com/goplusjs/gopherjs/compiler/typesutil"
)

type continueStmt struct {
	forStmt      *ast.ForStmt
	analyzeStack []ast.Node
}

type Info struct {
	*types.Info
	Pkg           *types.Package
	IsBlocking    func(*types.Func) bool
	HasPointer    map[*types.Var]bool
	FuncDeclInfos map[*types.Func]*FuncInfo
	FuncLitInfos  map[*ast.FuncLit]*FuncInfo
	InitFuncInfo  *FuncInfo
	allInfos      []*FuncInfo
}

type FuncInfo struct {
	HasDefer bool
	// Flattened map tracks which AST nodes within function body must be
	// translated into re-enterant blocks.
	//
	// Function body needs to be "flattened" if an option to jump an arbitrary
	// position in the code is required. Typical examples are a "goto" operator or
	// resuming goroutine execution after a blocking call.
	Flattened map[ast.Node]bool
	// Blocking map tracks which AST nodes lead to potentially blocking calls.
	//
	// Blocking calls require special handling on JS side to avoid blocking the
	// event loop and freezing the page.
	Blocking map[ast.Node]bool
	// GotoLabel keeps track of labels referenced by a goto operator.
	//
	// JS doesn't support "goto" natively and it needs to be emulated with a
	// switch/case statement. This is distinct from labeled loop statements, which
	// have native JS syntax and don't require special handling.
	GotoLabel map[*types.Label]bool

	// All callsite AST paths for all functions called by this function.
	localCalls map[*types.Func][][]ast.Node
	// All "continue" operators in the function body.
	//
	// "continue" operator may trigger blocking calls in for loop condition or
	// post-iteration statement, so they may require special handling.
	continueStmts []continueStmt
	packageInfo   *Info
	analyzeStack  []ast.Node
}

func (info *Info) newFuncInfo() *FuncInfo {
	funcInfo := &FuncInfo{
		packageInfo: info,
		Flattened:   make(map[ast.Node]bool),
		Blocking:    make(map[ast.Node]bool),
		GotoLabel:   make(map[*types.Label]bool),
		localCalls:  make(map[*types.Func][][]ast.Node),
	}
	info.allInfos = append(info.allInfos, funcInfo)
	return funcInfo
}

func AnalyzePkg(files []*ast.File, fileSet *token.FileSet, typesInfo *types.Info, typesPkg *types.Package, isBlocking func(*types.Func) bool) *Info {
	info := &Info{
		Info:          typesInfo,
		Pkg:           typesPkg,
		HasPointer:    make(map[*types.Var]bool),
		IsBlocking:    isBlocking,
		FuncDeclInfos: make(map[*types.Func]*FuncInfo),
		FuncLitInfos:  make(map[*ast.FuncLit]*FuncInfo),
	}
	info.InitFuncInfo = info.newFuncInfo()

	for _, file := range files {
		ast.Walk(info.InitFuncInfo, file)
	}

	// Propagate information about blocking calls through the AST tree.
	// TODO: This can probably be done more efficiently while traversing the AST
	// tree.
	for {
		done := true
		for _, funcInfo := range info.allInfos {
			for obj, calls := range funcInfo.localCalls {
				if len(info.FuncDeclInfos[obj].Blocking) != 0 {
					for _, call := range calls {
						funcInfo.markBlocking(call)
					}
					delete(funcInfo.localCalls, obj)
					done = false
				}
			}
		}
		if done {
			break
		}
	}

	// Detect all "continue" statements that lead to blocking calls.
	for _, funcInfo := range info.allInfos {
		for _, continueStmt := range funcInfo.continueStmts {
			if funcInfo.Blocking[continueStmt.forStmt.Post] {
				funcInfo.markBlocking(continueStmt.analyzeStack)
			}
		}
		funcInfo.continueStmts = nil // We no longer need this information.
	}

	info.allInfos = nil // Let GC reclaim memory we no longer need.

	return info
}

func (c *FuncInfo) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		if len(c.analyzeStack) != 0 {
			c.analyzeStack = c.analyzeStack[:len(c.analyzeStack)-1]
		}
		return nil
	}
	c.analyzeStack = append(c.analyzeStack, node)

	switch n := node.(type) {
	case *ast.FuncDecl:
		newInfo := c.packageInfo.newFuncInfo()
		c.packageInfo.FuncDeclInfos[c.packageInfo.Defs[n.Name].(*types.Func)] = newInfo
		return newInfo
	case *ast.FuncLit:
		newInfo := c.packageInfo.newFuncInfo()
		c.packageInfo.FuncLitInfos[n] = newInfo
		return newInfo
	case *ast.BranchStmt:
		switch n.Tok {
		case token.GOTO:
			for _, n2 := range c.analyzeStack {
				c.Flattened[n2] = true
			}
			c.GotoLabel[c.packageInfo.Uses[n.Label].(*types.Label)] = true
		case token.CONTINUE:
			if n.Label != nil {
				label := c.packageInfo.Uses[n.Label].(*types.Label)
				for i := len(c.analyzeStack) - 1; i >= 0; i-- {
					if labelStmt, ok := c.analyzeStack[i].(*ast.LabeledStmt); ok && c.packageInfo.Defs[labelStmt.Label] == label {
						if _, ok := labelStmt.Stmt.(*ast.RangeStmt); ok {
							return nil
						}
						stack := make([]ast.Node, len(c.analyzeStack))
						copy(stack, c.analyzeStack)
						c.continueStmts = append(c.continueStmts, continueStmt{labelStmt.Stmt.(*ast.ForStmt), stack})
						return nil
					}
				}
				return nil
			}
			for i := len(c.analyzeStack) - 1; i >= 0; i-- {
				if _, ok := c.analyzeStack[i].(*ast.RangeStmt); ok {
					return nil
				}
				if forStmt, ok := c.analyzeStack[i].(*ast.ForStmt); ok {
					stack := make([]ast.Node, len(c.analyzeStack))
					copy(stack, c.analyzeStack)
					c.continueStmts = append(c.continueStmts, continueStmt{forStmt, stack})
					return nil
				}
			}
		}
	case *ast.CallExpr:
		callTo := func(obj types.Object) {
			switch o := obj.(type) {
			case *types.Func:
				if recv := o.Type().(*types.Signature).Recv(); recv != nil {
					if _, ok := recv.Type().Underlying().(*types.Interface); ok {
						c.markBlocking(c.analyzeStack)
						return
					}
				}
				if o.Pkg() != c.packageInfo.Pkg {
					if c.packageInfo.IsBlocking(o) {
						c.markBlocking(c.analyzeStack)
					}
					return
				}
				stack := make([]ast.Node, len(c.analyzeStack))
				copy(stack, c.analyzeStack)
				c.localCalls[o] = append(c.localCalls[o], stack)
			case *types.Var:
				c.markBlocking(c.analyzeStack)
			}
		}
		switch f := astutil.RemoveParens(n.Fun).(type) {
		case *ast.Ident:
			callTo(c.packageInfo.Uses[f])
		case *ast.SelectorExpr:
			if sel := c.packageInfo.Selections[f]; sel != nil && typesutil.IsJsObject(sel.Recv()) {
				break
			}
			callTo(c.packageInfo.Uses[f.Sel])
		case *ast.FuncLit:
			ast.Walk(c, n.Fun)
			for _, arg := range n.Args {
				ast.Walk(c, arg)
			}
			if len(c.packageInfo.FuncLitInfos[f].Blocking) != 0 {
				c.markBlocking(c.analyzeStack)
			}
			return nil
		default:
			if !astutil.IsTypeExpr(f, c.packageInfo.Info) {
				c.markBlocking(c.analyzeStack)
			}
		}
	case *ast.SendStmt:
		c.markBlocking(c.analyzeStack)
	case *ast.UnaryExpr:
		switch n.Op {
		case token.AND:
			if id, ok := astutil.RemoveParens(n.X).(*ast.Ident); ok {
				c.packageInfo.HasPointer[c.packageInfo.Uses[id].(*types.Var)] = true
			}
		case token.ARROW:
			c.markBlocking(c.analyzeStack)
		}
	case *ast.RangeStmt:
		if _, ok := c.packageInfo.TypeOf(n.X).Underlying().(*types.Chan); ok {
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
		switch comm := n.Comm.(type) {
		case *ast.SendStmt:
			ast.Walk(c, comm.Chan)
			ast.Walk(c, comm.Value)
		case *ast.ExprStmt:
			ast.Walk(c, comm.X.(*ast.UnaryExpr).X)
		case *ast.AssignStmt:
			ast.Walk(c, comm.Rhs[0].(*ast.UnaryExpr).X)
		}
		for _, s := range n.Body {
			ast.Walk(c, s)
		}
		return nil
	case *ast.GoStmt:
		ast.Walk(c, n.Call.Fun)
		for _, arg := range n.Call.Args {
			ast.Walk(c, arg)
		}
		return nil
	case *ast.DeferStmt:
		c.HasDefer = true
		if funcLit, ok := n.Call.Fun.(*ast.FuncLit); ok {
			ast.Walk(c, funcLit.Body)
		}
	}
	return c
}

func (c *FuncInfo) markBlocking(stack []ast.Node) {
	for _, n := range stack {
		c.Blocking[n] = true
		c.Flattened[n] = true
	}
}
