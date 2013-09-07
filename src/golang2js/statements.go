package main

import (
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func (c *PkgContext) translateStmtList(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		c.translateStmt(stmt)
	}
}

func (c *PkgContext) translateStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		c.Printf("{")
		c.Indent(func() {
			c.translateStmtList(s.List)
		})
		c.Printf("}")

	case *ast.IfStmt:
		c.translateStmt(s.Init)
		c.Printf("if (%s) {", c.translateExpr(s.Cond))
		c.Indent(func() {
			c.translateStmtList(s.Body.List)
		})
		if s.Else != nil {
			c.Printf("} else")
			c.translateStmt(s.Else)
			return
		}
		c.Printf("}")

	case *ast.SwitchStmt:
		c.translateStmt(s.Init)

		if s.Tag == nil {
			if s.Body.List == nil {
				return
			}
			if len(s.Body.List) == 1 && s.Body.List[0].(*ast.CaseClause).List == nil {
				c.translateStmtList(s.Body.List[0].(*ast.CaseClause).Body)
				return
			}

			clauseStmts := make([][]ast.Stmt, len(s.Body.List))
			openClauses := make([]int, 0)
			for i, child := range s.Body.List {
				caseClause := child.(*ast.CaseClause)
				openClauses = append(openClauses, i)
				for _, j := range openClauses {
					clauseStmts[j] = append(clauseStmts[j], caseClause.Body...)
				}
				if !hasFallthrough(caseClause) {
					openClauses = nil
				}
			}

			var defaultClause []ast.Stmt
			for i, child := range s.Body.List {
				caseClause := child.(*ast.CaseClause)
				if len(caseClause.List) == 0 {
					defaultClause = clauseStmts[i]
					continue
				}
				conds := make([]string, len(caseClause.List))
				for i, cond := range caseClause.List {
					conds[i] = c.translateExpr(cond)
				}
				c.Printf("if (%s) {", strings.Join(conds, " || "))
				c.Indent(func() {
					c.translateStmtList(clauseStmts[i])
				})
				if i < len(s.Body.List)-1 || defaultClause != nil {
					c.Printf("} else")
					continue
				}
				c.Printf("}")
			}
			if defaultClause != nil {
				c.Printf("{")
				c.Indent(func() {
					c.translateStmtList(defaultClause)
				})
				c.Printf("}")
			}
			return
		}

		c.Printf("switch (%s) {", c.translateExpr(s.Tag))
		hasDefault := false
		for _, child := range s.Body.List {
			caseClause := child.(*ast.CaseClause)
			for _, cond := range caseClause.List {
				c.Printf("case %s:", c.translateExpr(cond))
			}
			if len(caseClause.List) == 0 {
				c.Printf("default:")
				hasDefault = true
			}
			c.Indent(func() {
				c.translateStmtList(caseClause.Body)
				if !hasFallthrough(caseClause) {
					c.Printf("break;")
				}
			})
		}
		if !hasDefault {
			c.Printf("default:")
			c.Printf("  // empty")
			c.Printf("  break;")
		}
		c.Printf("}")

	case *ast.TypeSwitchStmt:
		c.translateStmt(s.Init)
		expr := ""
		if assign, isAssign := s.Assign.(*ast.AssignStmt); isAssign {
			id := assign.Lhs[0].(*ast.Ident)
			expr = c.newVarName(id.Name)
			obj := &types.Var{}
			c.info.Objects[id] = obj
			c.objectVars[obj] = expr
			c.translateStmt(s.Assign)
			for _, caseClause := range s.Body.List {
				c.objectVars[c.info.Implicits[caseClause]] = expr
			}
		}
		if expr == "" {
			expr = c.translateExpr(s.Assign.(*ast.ExprStmt).X)
		}
		c.Printf("switch (typeOf(%s)) {", expr)
		for _, child := range s.Body.List {
			caseClause := child.(*ast.CaseClause)
			for _, cond := range caseClause.List {
				c.Printf("case %s:", c.translateExpr(cond))
			}
			if len(caseClause.List) == 0 {
				c.Printf("default:")
			}
			c.Indent(func() {
				c.translateStmtList(caseClause.Body)
				c.Printf("break;")
			})
		}
		c.Printf("}")

	case *ast.ForStmt:
		c.translateStmt(s.Init)
		post := strings.TrimSuffix(strings.TrimSpace(c.CatchOutput(func() { c.translateStmt(s.Post) })), ";") // TODO ugly
		c.Printf("for (; %s; %s) {", c.translateExpr(s.Cond), post)
		c.Indent(func() {
			c.translateStmtList(s.Body.List)
		})
		c.Printf("}")

	case *ast.RangeStmt:
		refVar := c.newVarName("_ref")
		lenVar := c.newVarName("_len")
		iVar := c.newVarName("_i")
		vars := []string{refVar, lenVar, iVar}

		key := c.translateExpr(s.Key)
		value := c.translateExpr(s.Value)
		keyAssign := ""
		if key != "" {
			keyAssign = fmt.Sprintf(", %s = %s", key, iVar)
			if s.Tok == token.DEFINE {
				vars = append(vars, key)
			}
		}
		if value != "" {
			if s.Tok == token.DEFINE {
				vars = append(vars, value)
			}
		}

		c.Printf("var %s;", strings.Join(vars, ", "))
		forParams := "" +
			fmt.Sprintf("%s = %s", refVar, c.translateExpr(s.X)) +
			fmt.Sprintf(", %s = %s.length", lenVar, refVar) +
			fmt.Sprintf(", %s = 0", iVar) +
			keyAssign +
			fmt.Sprintf("; %s < %s", iVar, lenVar) +
			fmt.Sprintf("; %s++", iVar) +
			keyAssign
		c.Printf("for (%s) {", forParams)
		c.Indent(func() {
			if value != "" {
				switch t := c.info.Types[s.X].Underlying().(type) {
				case *types.Array:
					c.Printf("var %s = %s[%s];", value, refVar, iVar)
				case *types.Slice:
					c.Printf("var %s = %s.get(%s);", value, refVar, iVar)
				case *types.Basic:
					c.Printf("var %s = %s.charCodeAt(%s);", value, refVar, iVar)
				default:
					panic(fmt.Sprintf("Unhandled range type: %T\n", t))
				}
			}
			c.translateStmtList(s.Body.List)
		})
		c.Printf("}")

	case *ast.BranchStmt:
		switch s.Tok {
		case token.BREAK:
			c.Printf("break;")
		case token.CONTINUE:
			c.Printf("continue;")
		case token.GOTO:
			c.Printf(`throw "goto not implemented";`)
		case token.FALLTHROUGH:
			// handled in CaseClause
		default:
			panic("Unhandled branch statment: " + s.Tok.String())
		}

	case *ast.ReturnStmt:
		results := make([]string, len(s.Results))
		for i, result := range s.Results {
			results[i] = c.translateExpr(result)
			if c.namedResults != nil {
				c.Printf("%s = %s;", c.namedResults[i], results[i])
			}
		}
		if c.namedResults != nil {
			results = c.namedResults
		}
		switch len(results) {
		case 0:
			c.Printf("return;")
		case 1:
			c.Printf("return %s;", results[0])
		default:
			c.Printf("return [%s];", strings.Join(results, ", "))
		}

	case *ast.DeferStmt:
		args := c.translateArgs(s.Call)
		c.Printf("_deferred.push({ fun: %s, recv: %s, args: [%s] });", c.translateExpr(s.Call.Fun), "this", strings.Join(args, ", ")) // TODO fix receiver

	case *ast.ExprStmt:
		c.Printf("%s;", c.translateExpr(s.X))

	case *ast.DeclStmt:
		for _, spec := range s.Decl.(*ast.GenDecl).Specs {
			c.translateSpec(spec)
		}

	case *ast.LabeledStmt:
		c.Printf("// label: %s", s.Label.Name)
		c.translateStmt(s.Stmt)

	case *ast.AssignStmt:
		rhsExprs := make([]string, len(s.Rhs))
		// rhsTypes := make([]types.Type, len(s.Rhs))
		for i, rhs := range s.Rhs {
			rhsExprs[i] = c.translateExpr(rhs)
			// rhsTypes[i] = c.info.Types[rhs]
		}
		rhs := rhsExprs[0]
		// completeRhsType := rhsTypes[0]
		if len(rhsExprs) > 1 {
			rhs = "[" + strings.Join(rhsExprs, ", ") + "]"
			// completeRhsType = types.NewTuple(rhsTypes...)
		}

		if len(s.Lhs) > 1 {
			c.Printf("_tuple = %s;", rhs)
		}

		for i, l := range s.Lhs {
			lhs := c.translateExpr(l)
			// lhsType := c.info.Types[l]

			// rhsType := completeRhsType
			if len(s.Lhs) > 1 {
				if lhs == "" {
					continue
				}
				rhs = fmt.Sprintf("_tuple[%d]", i)
				// rhsType = completeRhsType.(*types.Tuple).At(i)
			}

			if lhs == "" {
				c.Printf("%s;", rhs)
				continue
			}

			if s.Tok == token.DEFINE {
				c.Printf("var %s = %s;", lhs, rhs)
				continue
			}

			if iExpr, ok := s.Lhs[0].(*ast.IndexExpr); ok && s.Tok == token.ASSIGN {
				if _, isSlice := c.info.Types[iExpr.X].Underlying().(*types.Slice); isSlice {
					c.Printf("%s.set(%s, %s);", c.translateExpr(iExpr.X), c.translateExpr(iExpr.Index), rhs)
					continue
				}
			}

			tok := s.Tok.String()
			if s.Tok == token.AND_NOT_ASSIGN {
				tok = "&=~"
			}
			c.Printf("%s %s %s;", lhs, tok, rhs)
		}

	case *ast.IncDecStmt:
		c.Printf("%s%s;", c.translateExpr(s.X), s.Tok)

	case nil:
		// skip

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
}

func hasFallthrough(caseClause *ast.CaseClause) bool {
	if len(caseClause.Body) == 0 {
		return false
	}
	b, isBranchStmt := caseClause.Body[len(caseClause.Body)-1].(*ast.BranchStmt)
	return isBranchStmt && b.Tok == token.FALLTHROUGH
}
