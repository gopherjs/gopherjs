package analysis

import (
	"go/ast"
	"go/token"

	"github.com/gopherjs/gopherjs/compiler/util"

	"golang.org/x/tools/go/types"
)

type Info struct {
	*types.Info
	Pkg           *types.Package
	IsBlocking    func(*types.Func) bool
	FuncDeclInfos map[*types.Func]*FuncInfo
	FuncLitInfos  map[*ast.FuncLit]*FuncInfo
	InitFuncInfo  *FuncInfo
	comments      ast.CommentMap
}

type FuncInfo struct {
	HasDefer     bool
	Flattened    map[ast.Node]bool
	Blocking     map[ast.Node]bool
	GotoLabel    map[*types.Label]bool
	LocalCalls   map[*types.Func][][]ast.Node
	p            *Info
	analyzeStack []ast.Node
}

func NewFuncInfo(p *Info) *FuncInfo {
	return &FuncInfo{
		p:          p,
		Flattened:  make(map[ast.Node]bool),
		Blocking:   make(map[ast.Node]bool),
		GotoLabel:  make(map[*types.Label]bool),
		LocalCalls: make(map[*types.Func][][]ast.Node),
	}
}

func AnalyzePkg(files []*ast.File, fileSet *token.FileSet, typesInfo *types.Info, typesPkg *types.Package, isBlocking func(*types.Func) bool) *Info {
	info := &Info{
		Info:          typesInfo,
		Pkg:           typesPkg,
		comments:      make(ast.CommentMap),
		IsBlocking:    isBlocking,
		FuncDeclInfos: make(map[*types.Func]*FuncInfo),
		FuncLitInfos:  make(map[*ast.FuncLit]*FuncInfo),
	}

	info.InitFuncInfo = NewFuncInfo(info)
	for _, file := range files {
		for k, v := range ast.NewCommentMap(fileSet, file, file.Comments) {
			info.comments[k] = v
		}
		ast.Walk(info.InitFuncInfo, file)
	}
	for {
		done := true
		for _, funcInfo := range info.FuncDeclInfos {
			for obj, calls := range funcInfo.LocalCalls {
				if len(info.FuncDeclInfos[obj].Blocking) != 0 {
					for _, call := range calls {
						funcInfo.markBlocking(call)
					}
					delete(funcInfo.LocalCalls, obj)
					done = false
				}
			}
		}
		if done {
			break
		}
	}
	for _, funcInfo := range info.FuncLitInfos {
		for obj, calls := range funcInfo.LocalCalls {
			if len(info.FuncDeclInfos[obj].Blocking) != 0 {
				for _, call := range calls {
					funcInfo.markBlocking(call)
				}
			}
		}
	}

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
		newInfo := NewFuncInfo(c.p)
		c.p.FuncDeclInfos[c.p.Defs[n.Name].(*types.Func)] = newInfo
		return newInfo
	case *ast.FuncLit:
		newInfo := NewFuncInfo(c.p)
		c.p.FuncLitInfos[n] = newInfo
		return newInfo
	case *ast.BranchStmt:
		if n.Tok == token.GOTO {
			for _, n2 := range c.analyzeStack {
				c.Flattened[n2] = true
			}
			c.GotoLabel[c.p.Uses[n.Label].(*types.Label)] = true
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
				if o.Pkg() != c.p.Pkg {
					if c.p.IsBlocking(o) {
						c.markBlocking(c.analyzeStack)
					}
					return
				}
				stack := make([]ast.Node, len(c.analyzeStack))
				copy(stack, c.analyzeStack)
				c.LocalCalls[o] = append(c.LocalCalls[o], stack)
			case *types.Var:
				lookForComment()
			}
		}
		switch f := util.RemoveParens(n.Fun).(type) {
		case *ast.Ident:
			callTo(c.p.Uses[f])
		case *ast.SelectorExpr:
			if sel := c.p.Selections[f]; sel != nil && util.IsJsObject(sel.Recv()) {
				break
			}
			callTo(c.p.Uses[f.Sel])
		}
	case *ast.SendStmt:
		c.markBlocking(c.analyzeStack)
	case *ast.UnaryExpr:
		if n.Op == token.ARROW {
			c.markBlocking(c.analyzeStack)
		}
	case *ast.RangeStmt:
		if _, ok := c.p.Types[n.X].Type.Underlying().(*types.Chan); ok {
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
