package main

import (
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func (c *PkgContext) translateStmtList(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		c.translateStmt(stmt, "")
	}
}

func (c *PkgContext) translateStmt(stmt ast.Stmt, label string) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		c.Printf("{")
		c.Indent(func() {
			c.translateStmtList(s.List)
		})
		c.Printf("}")

	case *ast.IfStmt:
		c.translateStmt(s.Init, "")
		c.Printf("if (%s) {", c.translateExpr(s.Cond))
		c.Indent(func() {
			c.translateStmtList(s.Body.List)
		})
		if s.Else != nil {
			c.Printf("} else")
			c.translateStmt(s.Else, "")
			return
		}
		c.Printf("}")

	case *ast.SwitchStmt:
		c.translateStmt(s.Init, "")
		translateCond := func(cond ast.Expr) string {
			return c.translateExpr(cond)
		}
		if s.Tag != nil {
			refVar := c.newVarName("_ref")
			c.Printf("var %s = %s;", refVar, c.translateExpr(s.Tag))
			tagType := c.info.Types[s.Tag]
			_, isInterface := tagType.(*types.Interface)
			translateCond = func(cond ast.Expr) string {
				if isInterface {
					return fmt.Sprintf("Go$isEqual(%s, %s)", refVar, c.translateExprToType(cond, tagType))
				}
				return refVar + " === " + c.translateExprToType(cond, tagType)
			}
		}
		c.translateSwitch(s.Body.List, translateCond, "", "", label)

	case *ast.TypeSwitchStmt:
		c.translateStmt(s.Init, "")
		var expr ast.Expr
		var typeSwitchVar string
		switch a := s.Assign.(type) {
		case *ast.AssignStmt:
			expr = a.Rhs[0].(*ast.TypeAssertExpr).X
			typeSwitchVar = a.Lhs[0].(*ast.Ident).Name
			for _, caseClause := range s.Body.List {
				c.objectVars[c.info.Implicits[caseClause]] = typeSwitchVar
			}
		case *ast.ExprStmt:
			expr = a.X.(*ast.TypeAssertExpr).X
		}
		refVar := c.newVarName("_ref")
		typeVar := c.newVarName("_type")
		c.Printf("var %s = %s;", refVar, c.translateExpr(expr))
		c.Printf("var %s = Go$typeOf(%s);", typeVar, refVar)
		translateCond := func(cond ast.Expr) string {
			return typeVar + " === " + c.typeName(c.info.Types[cond])
		}
		c.translateSwitch(s.Body.List, translateCond, refVar, typeSwitchVar, label)

	case *ast.ForStmt:
		c.translateStmt(s.Init, "")
		cond := "true"
		if s.Cond != nil {
			cond = c.translateExpr(s.Cond)
		}
		p := c.postLoopStmt
		defer func() { c.postLoopStmt = p }()
		c.postLoopStmt = s.Post
		c.Printf("%swhile (%s) {", label, cond)
		c.Indent(func() {
			c.translateStmtList(s.Body.List)
			c.translateStmt(s.Post, "")
		})
		c.Printf("}")

	case *ast.RangeStmt:
		p := c.postLoopStmt
		defer func() { c.postLoopStmt = p }()
		c.postLoopStmt = nil

		key := ""
		if s.Key != nil && !isUnderscore(s.Key) {
			key = c.translateExpr(s.Key)
		}
		value := ""
		if s.Value != nil && !isUnderscore(s.Value) {
			value = c.translateExpr(s.Value)
		}
		varKeyword := ""
		if s.Tok == token.DEFINE {
			varKeyword = "var "
		}

		refVar := c.newVarName("_ref")
		c.Printf("var %s = %s;", refVar, c.translateExpr(s.X))

		lenTarget := refVar
		_, isMap := c.info.Types[s.X].Underlying().(*types.Map)
		var keysVar string
		if isMap {
			keysVar = c.newVarName("_keys")
			c.Printf("var %s = %s !== null ? Go$keys(%s) : [];", keysVar, refVar, refVar)
			lenTarget = keysVar
		}

		lenVar := c.newVarName("_len")
		c.Printf("var %s = %s !== null ? %s.length : 0;", lenVar, lenTarget, lenTarget)

		iVar := c.newVarName("_i")
		c.Printf("var %s = 0;", iVar)

		c.Printf("%sfor (; %s < %s; %s++) {", label, iVar, lenVar, iVar)
		c.Indent(func() {
			var entryVar string
			if isMap {
				entryVar = c.newVarName("_entry")
				c.Printf("var %s = %s[%s[%s]];", entryVar, refVar, keysVar, iVar)
				if key != "" {
					c.Printf("%s%s = %s.k;", varKeyword, key, entryVar)
				}
			}
			if !isMap && key != "" {
				c.Printf("%s%s = %s;", varKeyword, key, iVar)
			}
			if value != "" {
				switch t := c.info.Types[s.X].Underlying().(type) {
				case *types.Array:
					c.Printf("%s%s = %s[%s];", varKeyword, value, refVar, iVar)
				case *types.Slice:
					c.Printf("%s%s = %s.Go$get(%s);", varKeyword, value, refVar, iVar)
				case *types.Map:
					c.Printf("%s%s = %s.v;", varKeyword, value, entryVar)
				case *types.Basic:
					c.Printf("%s%s = %s.charCodeAt(%s);", varKeyword, value, refVar, iVar)
				default:
					panic(fmt.Sprintf("Unhandled range type: %T\n", t))
				}
			}
			c.translateStmtList(s.Body.List)
		})
		c.Printf("}")

	case *ast.BranchStmt:
		label := ""
		if s.Label != nil {
			label = " " + s.Label.Name
		}
		switch s.Tok {
		case token.BREAK:
			c.Printf("break%s;", label)
		case token.CONTINUE:
			c.translateStmt(c.postLoopStmt, "")
			c.Printf("continue%s;", label)
		case token.GOTO:
			c.Printf(`throw new GoError("Statement not supported: goto");`)
		case token.FALLTHROUGH:
			// handled in CaseClause
		default:
			panic("Unhandled branch statment: " + s.Tok.String())
		}

	case *ast.ReturnStmt:
		results := s.Results
		if c.resultNames != nil {
			if len(s.Results) != 0 {
				c.translateStmt(&ast.AssignStmt{
					Lhs: c.resultNames,
					Tok: token.ASSIGN,
					Rhs: s.Results,
				}, label)
			}
			results = c.resultNames
		}
		switch len(results) {
		case 0:
			c.Printf("return;")
		case 1:
			if c.functionSig.Results().Len() > 1 {
				c.Printf("return %s;", c.translateExpr(results[0]))
				return
			}
			c.Printf("return %s;", c.translateExprToType(results[0], c.functionSig.Results().At(0).Type()))
		default:
			values := make([]string, len(results))
			for i, result := range results {
				values[i] = c.translateExprToType(result, c.functionSig.Results().At(i).Type())
			}
			c.Printf("return [%s];", strings.Join(values, ", "))
		}

	case *ast.DeferStmt:
		var args []string
		if len(s.Call.Args) > 0 { // skip for call to recover()
			args = c.translateArgs(s.Call)
		}
		if sel, isSelector := s.Call.Fun.(*ast.SelectorExpr); isSelector {
			c.Printf(`Go$deferred.push({ recv: %s, method: "%s", args: [%s] });`, c.translateExpr(sel.X), sel.Sel.Name, strings.Join(args, ", "))
			return
		}
		c.Printf("Go$deferred.push({ fun: %s, args: [%s] });", c.translateExpr(s.Call.Fun), strings.Join(args, ", "))

	case *ast.ExprStmt:
		c.Printf("%s;", c.translateExpr(s.X))

	case *ast.DeclStmt:
		for _, spec := range s.Decl.(*ast.GenDecl).Specs {
			c.translateSpec(spec)
		}

	case *ast.LabeledStmt:
		c.translateStmt(s.Stmt, s.Label.Name+": ")

	case *ast.AssignStmt:
		if s.Tok != token.ASSIGN && s.Tok != token.DEFINE {
			var op token.Token
			switch s.Tok {
			case token.ADD_ASSIGN:
				op = token.ADD
			case token.SUB_ASSIGN:
				op = token.SUB
			case token.MUL_ASSIGN:
				op = token.MUL
			case token.QUO_ASSIGN:
				op = token.QUO
			case token.REM_ASSIGN:
				op = token.REM
			case token.AND_ASSIGN:
				op = token.AND
			case token.OR_ASSIGN:
				op = token.OR
			case token.XOR_ASSIGN:
				op = token.XOR
			case token.SHL_ASSIGN:
				op = token.SHL
			case token.SHR_ASSIGN:
				op = token.SHR
			case token.AND_NOT_ASSIGN:
				op = token.AND_NOT
			default:
				panic(s.Tok)
			}
			parenExpr := &ast.ParenExpr{
				X: s.Rhs[0],
			}
			c.info.Types[parenExpr] = c.info.Types[s.Rhs[0]]
			binaryExpr := &ast.BinaryExpr{
				X:  s.Lhs[0],
				Op: op,
				Y:  parenExpr,
			}
			c.info.Types[binaryExpr] = c.info.Types[s.Lhs[0]]
			c.translateStmt(&ast.AssignStmt{
				Lhs: []ast.Expr{s.Lhs[0]},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{binaryExpr},
			}, label)
			return
		}

		typeOf := func(e ast.Expr) types.Type {
			if s.Tok == token.DEFINE {
				return c.info.Objects[e.(*ast.Ident)].Type()
			}
			return c.info.Types[e]
		}

		rhsExprs := make([]string, len(s.Lhs))

		switch {
		case len(s.Lhs) == 1 && len(s.Rhs) == 1:
			rhsExprs[0] = c.translateExprToType(s.Rhs[0], typeOf(s.Lhs[0]))

		case len(s.Lhs) > 1 && len(s.Rhs) == 1:
			for i := range s.Lhs {
				rhsExprs[i] = fmt.Sprintf("Go$tuple[%d]", i)
			}
			c.Printf("Go$tuple = %s;", c.translateExpr(s.Rhs[0])) // TODO translateExprToType

		case len(s.Lhs) == len(s.Rhs):
			parts := make([]string, len(s.Rhs))
			for i, rhs := range s.Rhs {
				parts[i] = c.translateExprToType(rhs, typeOf(s.Lhs[i]))
				rhsExprs[i] = fmt.Sprintf("Go$tuple[%d]", i)
			}
			c.Printf("Go$tuple = [%s];", strings.Join(parts, ", "))

		default:
			panic("Invalid arity of AssignStmt.")

		}

		for i, lhs := range s.Lhs {
			rhs := rhsExprs[i]
			if isUnderscore(lhs) {
				if len(s.Lhs) == 1 {
					c.Printf("%s;", rhs)
				}
				continue
			}

			if s.Tok == token.DEFINE {
				c.Printf("var %s = %s;", c.translateExpr(lhs), rhs)
				continue
			}

			switch l := lhs.(type) {
			case *ast.StarExpr:
				if _, isStruct := c.info.Types[l].(*types.Struct); !isStruct {
					c.Printf("%s.Go$set(%s);", c.translateExpr(l.X), rhs)
					continue
				}
			case *ast.IndexExpr:
				switch t := c.info.Types[l.X].Underlying().(type) {
				case *types.Slice:
					c.Printf("%s.Go$set(%s, %s);", c.translateExpr(l.X), c.translateExpr(l.Index), rhs)
					continue
				case *types.Map:
					keyVar := c.newVarName("_key")
					c.Printf("var %s = %s;", keyVar, c.translateExprToType(l.Index, t.Key()))
					key := keyVar
					if hasId(t.Key()) {
						key = fmt.Sprintf("(%s || Go$nil).Go$id", key)
					}
					c.Printf("%s[%s] = { k: %s, v: %s };", c.translateExpr(l.X), key, keyVar, rhs)
					continue
				}
			}

			c.Printf("%s = %s;", c.translateExpr(lhs), rhs)
		}

	case *ast.IncDecStmt:
		t := c.info.Types[s.X]
		if iExpr, isIExpr := s.X.(*ast.IndexExpr); isIExpr {
			switch u := c.info.Types[iExpr.X].Underlying().(type) {
			case *types.Array:
				t = u.Elem()
			case *types.Slice:
				t = u.Elem()
			case *types.Map:
				t = u.Elem()
			}
		}

		tok := token.ADD_ASSIGN
		if s.Tok == token.DEC {
			tok = token.SUB_ASSIGN
		}
		one := &ast.BasicLit{
			Kind:  token.INT,
			Value: "1",
		}
		c.info.Types[one] = t
		c.info.Values[one] = exact.MakeInt64(1)
		c.translateStmt(&ast.AssignStmt{
			Lhs: []ast.Expr{s.X},
			Tok: tok,
			Rhs: []ast.Expr{one},
		}, label)

	case *ast.SelectStmt, *ast.GoStmt, *ast.SendStmt:
		c.Printf(`throw new GoError("Statement not supported: %T");`, s)

	case nil:
		// skip

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
}

func (c *PkgContext) translateSwitch(caseClauses []ast.Stmt, translateCond func(ast.Expr) string, typeSwitchValue, typeSwitchVar string, label string) {
	if len(caseClauses) == 0 {
		return
	}
	if len(caseClauses) == 1 && caseClauses[0].(*ast.CaseClause).List == nil {
		c.translateStmtList(caseClauses[0].(*ast.CaseClause).Body)
		return
	}

	clauseStmts := make([][]ast.Stmt, len(caseClauses))
	openClauses := make([]int, 0)
	for i, child := range caseClauses {
		caseClause := child.(*ast.CaseClause)
		openClauses = append(openClauses, i)
		for _, j := range openClauses {
			clauseStmts[j] = append(clauseStmts[j], caseClause.Body...)
		}
		if !hasFallthrough(caseClause) {
			openClauses = nil
		}
	}

	c.Printf("%sswitch (undefined) {", label)
	c.Printf("default:")
	c.Indent(func() {
		var defaultClause []ast.Stmt
		for i, child := range caseClauses {
			caseClause := child.(*ast.CaseClause)
			if len(caseClause.List) == 0 {
				defaultClause = clauseStmts[i]
				continue
			}
			conds := make([]string, len(caseClause.List))
			for i, cond := range caseClause.List {
				conds[i] = translateCond(cond)
			}
			c.Printf("if (%s) {", strings.Join(conds, " || "))
			c.Indent(func() {
				if typeSwitchVar != "" {
					value := typeSwitchValue
					if len(caseClause.List) == 1 {
						if isWrapped(c.info.Types[caseClause.List[0]]) {
							value += ".v"
						}
					}
					c.Printf("var %s = %s;", typeSwitchVar, value)
				}
				c.translateStmtList(clauseStmts[i])
			})
			if i < len(caseClauses)-1 || defaultClause != nil {
				c.Printf("} else")
				continue
			}
			c.Printf("}")
		}
		c.Printf("{")
		c.Indent(func() {
			if typeSwitchVar != "" {
				c.Printf("var %s = %s;", typeSwitchVar, typeSwitchValue)
			}
			c.translateStmtList(defaultClause)
		})
		c.Printf("}")
	})
	c.Printf("} while (false);")
}

func hasFallthrough(caseClause *ast.CaseClause) bool {
	if len(caseClause.Body) == 0 {
		return false
	}
	b, isBranchStmt := caseClause.Body[len(caseClause.Body)-1].(*ast.BranchStmt)
	return isBranchStmt && b.Tok == token.FALLTHROUGH
}
