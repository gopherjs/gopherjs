package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/types"
)

type this struct {
	ast.Ident
}

func (c *funcContext) translateStmtList(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		c.translateStmt(stmt, "")
	}
	c.WritePos(token.NoPos)
}

func (c *funcContext) translateStmt(stmt ast.Stmt, label string) {
	c.WritePos(stmt.Pos())

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		c.translateStmtList(s.List)

	case *ast.IfStmt:
		if s.Init != nil {
			c.translateStmt(s.Init, "")
		}
		var caseClauses []ast.Stmt
		ifStmt := s
		for {
			caseClauses = append(caseClauses, &ast.CaseClause{List: []ast.Expr{ifStmt.Cond}, Body: ifStmt.Body.List})
			switch elseStmt := ifStmt.Else.(type) {
			case *ast.IfStmt:
				if elseStmt.Init != nil {
					caseClauses = append(caseClauses, &ast.CaseClause{List: nil, Body: []ast.Stmt{elseStmt}})
					break
				}
				ifStmt = elseStmt
				continue
			case *ast.BlockStmt:
				caseClauses = append(caseClauses, &ast.CaseClause{List: nil, Body: elseStmt.List})
			case *ast.EmptyStmt, nil:
				// no else clause
			default:
				panic(fmt.Sprintf("Unhandled else: %T\n", elseStmt))
			}
			break
		}
		c.translateBranchingStmt(caseClauses, false, c.translateExpr, nil, "", c.flattened[s])

	case *ast.SwitchStmt:
		if s.Init != nil {
			c.translateStmt(s.Init, "")
		}
		translateCond := func(cond ast.Expr) *expression {
			return c.translateExpr(cond)
		}
		if s.Tag != nil {
			refVar := c.newVariable("_ref")
			c.Printf("%s = %s;", refVar, c.translateExpr(s.Tag))
			translateCond = func(cond ast.Expr) *expression {
				return c.translateExpr(&ast.BinaryExpr{
					X:  c.newIdent(refVar, c.p.info.Types[s.Tag].Type),
					Op: token.EQL,
					Y:  cond,
				})
			}
		}
		c.translateBranchingStmt(s.Body.List, true, translateCond, nil, label, c.flattened[s])

	case *ast.TypeSwitchStmt:
		if s.Init != nil {
			c.translateStmt(s.Init, "")
		}
		var expr ast.Expr
		var typeSwitchVar string
		switch a := s.Assign.(type) {
		case *ast.AssignStmt:
			expr = a.Rhs[0].(*ast.TypeAssertExpr).X
			typeSwitchVar = c.newVariable(a.Lhs[0].(*ast.Ident).Name)
			for _, caseClause := range s.Body.List {
				c.p.objectVars[c.p.info.Implicits[caseClause]] = typeSwitchVar
			}
		case *ast.ExprStmt:
			expr = a.X.(*ast.TypeAssertExpr).X
		}
		refVar := c.newVariable("_ref")
		c.Printf("%s = %s;", refVar, c.translateExpr(expr))
		translateCond := func(cond ast.Expr) *expression {
			if types.Identical(c.p.info.Types[cond].Type, types.Typ[types.UntypedNil]) {
				return c.formatExpr("%s === $ifaceNil", refVar)
			}
			return c.formatExpr("$assertType(%s, %s, true)[1]", refVar, c.typeName(c.p.info.Types[cond].Type))
		}
		printCaseBodyPrefix := func(index int) {
			if typeSwitchVar == "" {
				return
			}
			value := refVar
			if conds := s.Body.List[index].(*ast.CaseClause).List; len(conds) == 1 {
				t := c.p.info.Types[conds[0]].Type
				if _, isInterface := t.Underlying().(*types.Interface); !isInterface && !types.Identical(t, types.Typ[types.UntypedNil]) {
					value += ".$val"
				}
			}
			c.Printf("%s = %s;", typeSwitchVar, value)
		}
		c.translateBranchingStmt(s.Body.List, true, translateCond, printCaseBodyPrefix, label, c.flattened[s])

	case *ast.ForStmt:
		if s.Init != nil {
			c.translateStmt(s.Init, "")
		}
		cond := "true"
		if s.Cond != nil {
			cond = c.translateExpr(s.Cond).String()
		}
		c.translateLoopingStmt(cond, s.Body, nil, func() {
			if s.Post != nil {
				c.translateStmt(s.Post, "")
			}
		}, label, c.flattened[s])

	case *ast.RangeStmt:
		refVar := c.newVariable("_ref")
		c.Printf("%s = %s;", refVar, c.translateExpr(s.X))

		switch t := c.p.info.Types[s.X].Type.Underlying().(type) {
		case *types.Basic:
			iVar := c.newVariable("_i")
			c.Printf("%s = 0;", iVar)
			runeVar := c.newVariable("_rune")
			c.translateLoopingStmt(iVar+" < "+refVar+".length", s.Body, func() {
				c.Printf("%s = $decodeRune(%s, %s);", runeVar, refVar, iVar)
				if !isBlank(s.Key) {
					c.Printf("%s", c.translateAssign(s.Key, iVar, types.Typ[types.Int], s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					c.Printf("%s", c.translateAssign(s.Value, runeVar+"[0]", types.Typ[types.Rune], s.Tok == token.DEFINE))
				}
			}, func() {
				c.Printf("%s += %s[1];", iVar, runeVar)
			}, label, c.flattened[s])

		case *types.Map:
			iVar := c.newVariable("_i")
			c.Printf("%s = 0;", iVar)
			keysVar := c.newVariable("_keys")
			c.Printf("%s = $keys(%s);", keysVar, refVar)
			c.translateLoopingStmt(iVar+" < "+keysVar+".length", s.Body, func() {
				entryVar := c.newVariable("_entry")
				c.Printf("%s = %s[%s[%s]];", entryVar, refVar, keysVar, iVar)
				c.translateStmt(&ast.IfStmt{
					Cond: c.newIdent(entryVar+" === undefined", types.Typ[types.Bool]),
					Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.CONTINUE}}},
				}, "")
				if !isBlank(s.Key) {
					c.Printf("%s", c.translateAssign(s.Key, entryVar+".k", t.Key(), s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					c.Printf("%s", c.translateAssign(s.Value, entryVar+".v", t.Elem(), s.Tok == token.DEFINE))
				}
			}, func() {
				c.Printf("%s++;", iVar)
			}, label, c.flattened[s])

		case *types.Array, *types.Pointer, *types.Slice:
			var length string
			var elemType types.Type
			switch t2 := t.(type) {
			case *types.Array:
				length = fmt.Sprintf("%d", t2.Len())
				elemType = t2.Elem()
			case *types.Pointer:
				length = fmt.Sprintf("%d", t2.Elem().Underlying().(*types.Array).Len())
				elemType = t2.Elem().Underlying().(*types.Array).Elem()
			case *types.Slice:
				length = refVar + ".$length"
				elemType = t2.Elem()
			}
			iVar := c.newVariable("_i")
			c.Printf("%s = 0;", iVar)
			c.translateLoopingStmt(iVar+" < "+length, s.Body, func() {
				if !isBlank(s.Key) {
					c.Printf("%s", c.translateAssign(s.Key, iVar, types.Typ[types.Int], s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					c.Printf("%s", c.translateAssign(s.Value, c.translateImplicitConversion(c.setType(&ast.IndexExpr{
						X:     c.newIdent(refVar, t),
						Index: c.newIdent(iVar, types.Typ[types.Int]),
					}, elemType), elemType).String(), elemType, s.Tok == token.DEFINE))
				}
			}, func() {
				c.Printf("%s++;", iVar)
			}, label, c.flattened[s])

		case *types.Chan:
			okVar := c.newIdent(c.newVariable("_ok"), types.Typ[types.Bool])
			key := s.Key
			tok := s.Tok
			if key == nil {
				key = ast.NewIdent("_")
				tok = token.ASSIGN
			}
			forStmt := &ast.ForStmt{
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{
								key,
								okVar,
							},
							Rhs: []ast.Expr{
								c.setType(&ast.UnaryExpr{X: c.newIdent(refVar, t), Op: token.ARROW}, types.NewTuple(types.NewVar(0, nil, "", t.Elem()), types.NewVar(0, nil, "", types.Typ[types.Bool]))),
							},
							Tok: tok,
						},
						&ast.IfStmt{
							Cond: &ast.UnaryExpr{X: okVar, Op: token.NOT},
							Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}}},
						},
						s.Body,
					},
				},
			}
			c.flattened[forStmt] = true
			c.translateStmt(forStmt, label)

		default:
			panic("")
		}

	case *ast.BranchStmt:
		normalLabel := ""
		blockingLabel := ""
		data := c.flowDatas[""]
		if s.Label != nil {
			normalLabel = " " + s.Label.Name
			blockingLabel = " s" // use explicit label "s", because surrounding loop may not be flattened
			data = c.flowDatas[s.Label.Name]
		}
		switch s.Tok {
		case token.BREAK:
			c.PrintCond(data.endCase == 0, fmt.Sprintf("break%s;", normalLabel), fmt.Sprintf("$s = %d; continue%s;", data.endCase, blockingLabel))
		case token.CONTINUE:
			data.postStmt()
			c.PrintCond(data.beginCase == 0, fmt.Sprintf("continue%s;", normalLabel), fmt.Sprintf("$s = %d; continue%s;", data.beginCase, blockingLabel))
		case token.GOTO:
			c.PrintCond(false, "goto "+s.Label.Name, fmt.Sprintf("$s = %d; continue;", c.labelCases[s.Label.Name]))
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
				}, "")
			}
			results = c.resultNames
		}
		switch len(results) {
		case 0:
			c.Printf("return;")
		case 1:
			if c.sig.Results().Len() > 1 {
				c.Printf("return %s;", c.translateExpr(results[0]))
				return
			}
			v := c.translateImplicitConversion(results[0], c.sig.Results().At(0).Type())
			c.delayedOutput = nil
			c.Printf("return %s;", v)
		default:
			values := make([]string, len(results))
			for i, result := range results {
				values[i] = c.translateImplicitConversion(result, c.sig.Results().At(i).Type()).String()
			}
			c.delayedOutput = nil
			c.Printf("return [%s];", strings.Join(values, ", "))
		}

	case *ast.DeferStmt:
		isBuiltin := false
		isJs := false
		switch fun := s.Call.Fun.(type) {
		case *ast.Ident:
			var builtin *types.Builtin
			builtin, isBuiltin = c.p.info.Uses[fun].(*types.Builtin)
			if isBuiltin && builtin.Name() == "recover" {
				c.Printf("$deferred.push([$recover, []]);")
				return
			}
		case *ast.SelectorExpr:
			isJs = isJsPackage(c.p.info.Uses[fun.Sel].Pkg())
		}
		if isBuiltin || isJs {
			args := make([]ast.Expr, len(s.Call.Args))
			for i, arg := range s.Call.Args {
				args[i] = c.newIdent(c.newVariable("_arg"), c.p.info.Types[arg].Type)
			}
			call := c.translateExpr(&ast.CallExpr{
				Fun:      s.Call.Fun,
				Args:     args,
				Ellipsis: s.Call.Ellipsis,
			})
			c.Printf("$deferred.push([function(%s) { %s; }, [%s]]);", strings.Join(c.translateExprSlice(args, nil), ", "), call, strings.Join(c.translateExprSlice(s.Call.Args, nil), ", "))
			return
		}
		sig := c.p.info.Types[s.Call.Fun].Type.Underlying().(*types.Signature)
		args := c.translateArgs(sig, s.Call.Args, s.Call.Ellipsis.IsValid())
		if len(c.blocking) != 0 {
			args = append(args, "true")
		}
		c.Printf("$deferred.push([%s, [%s]]);", c.translateExpr(s.Call.Fun), strings.Join(args, ", "))

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

			var parts []string
			lhs := s.Lhs[0]
			switch l := lhs.(type) {
			case *ast.IndexExpr:
				lhsVar := c.newVariable("_lhs")
				indexVar := c.newVariable("_index")
				parts = append(parts, lhsVar+" = "+c.translateExpr(l.X).String()+";")
				parts = append(parts, indexVar+" = "+c.translateExpr(l.Index).String()+";")
				lhs = c.setType(&ast.IndexExpr{
					X:     c.newIdent(lhsVar, c.p.info.Types[l.X].Type),
					Index: c.newIdent(indexVar, c.p.info.Types[l.Index].Type),
				}, c.p.info.Types[l].Type)
			case *ast.StarExpr:
				lhsVar := c.newVariable("_lhs")
				parts = append(parts, lhsVar+" = "+c.translateExpr(l.X).String()+";")
				lhs = c.setType(&ast.StarExpr{
					X: c.newIdent(lhsVar, c.p.info.Types[l.X].Type),
				}, c.p.info.Types[l].Type)
			case *ast.SelectorExpr:
				v := hasCallVisitor{c.p.info, false}
				ast.Walk(&v, l.X)
				if v.hasCall {
					lhsVar := c.newVariable("_lhs")
					parts = append(parts, lhsVar+" = "+c.translateExpr(l.X).String()+";")
					lhs = c.setType(&ast.SelectorExpr{
						X:   c.newIdent(lhsVar, c.p.info.Types[l.X].Type),
						Sel: l.Sel,
					}, c.p.info.Types[l].Type)
					c.p.info.Selections[lhs.(*ast.SelectorExpr)] = c.p.info.Selections[l]
				}
			}

			lhsType := c.p.info.Types[s.Lhs[0]].Type
			parts = append(parts, c.translateAssignOfExpr(lhs, c.setType(&ast.BinaryExpr{
				X:  lhs,
				Op: op,
				Y:  c.setType(&ast.ParenExpr{X: s.Rhs[0]}, c.p.info.Types[s.Rhs[0]].Type),
			}, lhsType), lhsType, s.Tok == token.DEFINE))
			c.Printf("%s", strings.Join(parts, " "))
			return
		}

		if s.Tok == token.DEFINE {
			for _, lhs := range s.Lhs {
				if !isBlank(lhs) {
					obj := c.p.info.Defs[lhs.(*ast.Ident)]
					if obj == nil {
						obj = c.p.info.Uses[lhs.(*ast.Ident)]
					}
					c.setType(lhs, obj.Type())
				}
			}
		}

		switch {
		case len(s.Lhs) == 1 && len(s.Rhs) == 1:
			lhs := removeParens(s.Lhs[0])
			if isBlank(lhs) {
				v := hasCallVisitor{c.p.info, false}
				ast.Walk(&v, s.Rhs[0])
				if v.hasCall {
					c.Printf("%s;", c.translateExpr(s.Rhs[0]).String())
				}
				return
			}
			lhsType := c.p.info.Types[s.Lhs[0]].Type
			c.Printf("%s", c.translateAssignOfExpr(lhs, s.Rhs[0], lhsType, s.Tok == token.DEFINE))

		case len(s.Lhs) > 1 && len(s.Rhs) == 1:
			tupleVar := c.newVariable("_tuple")
			out := tupleVar + " = " + c.translateExpr(s.Rhs[0]).String() + ";"
			tuple := c.p.info.Types[s.Rhs[0]].Type.(*types.Tuple)
			for i, lhs := range s.Lhs {
				lhs = removeParens(lhs)
				if !isBlank(lhs) {
					lhsType := c.p.info.Types[s.Lhs[i]].Type
					out += " " + c.translateAssignOfExpr(lhs, c.newIdent(fmt.Sprintf("%s[%d]", tupleVar, i), tuple.At(i).Type()), lhsType, s.Tok == token.DEFINE)
				}
			}
			c.Printf("%s", out)
		case len(s.Lhs) == len(s.Rhs):
			tmpVars := make([]string, len(s.Rhs))
			var parts []string
			for i, rhs := range s.Rhs {
				tmpVars[i] = c.newVariable("_tmp")
				if isBlank(removeParens(s.Lhs[i])) {
					v := hasCallVisitor{c.p.info, false}
					ast.Walk(&v, rhs)
					if v.hasCall {
						c.Printf("%s;", c.translateExpr(rhs).String())
					}
					continue
				}
				lhsType := c.p.info.Types[s.Lhs[i]].Type
				parts = append(parts, c.translateAssignOfExpr(c.newIdent(tmpVars[i], c.p.info.Types[s.Lhs[i]].Type), rhs, lhsType, true))
			}
			for i, lhs := range s.Lhs {
				lhs = removeParens(lhs)
				if !isBlank(lhs) {
					parts = append(parts, c.translateAssign(lhs, tmpVars[i], c.p.info.Types[lhs].Type, s.Tok == token.DEFINE))
				}
			}
			c.Printf("%s", strings.Join(parts, " "))

		default:
			panic("Invalid arity of AssignStmt.")

		}

	case *ast.IncDecStmt:
		t := c.p.info.Types[s.X].Type
		if iExpr, isIExpr := s.X.(*ast.IndexExpr); isIExpr {
			switch u := c.p.info.Types[iExpr.X].Type.Underlying().(type) {
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
		c.translateStmt(&ast.AssignStmt{
			Lhs: []ast.Expr{s.X},
			Tok: tok,
			Rhs: []ast.Expr{c.newInt(1, t)},
		}, label)

	case *ast.DeclStmt:
		decl := s.Decl.(*ast.GenDecl)
		switch decl.Tok {
		case token.VAR:
			for _, spec := range s.Decl.(*ast.GenDecl).Specs {
				valueSpec := spec.(*ast.ValueSpec)
				lhs := make([]ast.Expr, len(valueSpec.Names))
				for i, name := range valueSpec.Names {
					lhs[i] = name
				}
				rhs := valueSpec.Values
				isTuple := false
				if len(rhs) == 1 {
					_, isTuple = c.p.info.Types[rhs[0]].Type.(*types.Tuple)
				}
				for len(rhs) < len(lhs) && !isTuple {
					rhs = append(rhs, nil)
				}
				c.translateStmt(&ast.AssignStmt{
					Lhs: lhs,
					Tok: token.DEFINE,
					Rhs: rhs,
				}, "")
			}
		case token.TYPE:
			for _, spec := range decl.Specs {
				o := c.p.info.Defs[spec.(*ast.TypeSpec).Name].(*types.TypeName)
				c.translateType(o, false)
				c.initType(o)
			}
		case token.CONST:
			// skip, constants are inlined
		}

	case *ast.ExprStmt:
		expr := c.translateExpr(s.X)
		if expr != nil {
			c.Printf("%s;", expr)
		}

	case *ast.LabeledStmt:
		if labelCase, ok := c.labelCases[s.Label.Name]; ok {
			c.PrintCond(false, s.Label.Name+":", fmt.Sprintf("case %d:", labelCase))
		}
		c.translateStmt(s.Stmt, s.Label.Name)

	case *ast.GoStmt:
		c.Printf("$go(%s, [%s]);", c.translateExpr(s.Call.Fun), strings.Join(c.translateArgs(c.p.info.Types[s.Call.Fun].Type.Underlying().(*types.Signature), s.Call.Args, s.Call.Ellipsis.IsValid()), ", "))

	case *ast.SendStmt:
		chanType := c.p.info.Types[s.Chan].Type.Underlying().(*types.Chan)
		call := &ast.CallExpr{
			Fun:  c.newIdent("$send", types.NewSignature(nil, nil, types.NewTuple(types.NewVar(0, nil, "", chanType), types.NewVar(0, nil, "", chanType.Elem())), nil, false)),
			Args: []ast.Expr{s.Chan, s.Value},
		}
		c.blocking[call] = true
		c.translateStmt(&ast.ExprStmt{X: call}, label)

	case *ast.SelectStmt:
		var channels []string
		var caseClauses []ast.Stmt
		flattened := false
		hasDefault := false
		for i, s := range s.Body.List {
			clause := s.(*ast.CommClause)
			switch comm := clause.Comm.(type) {
			case nil:
				channels = append(channels, "[]")
				hasDefault = true
			case *ast.ExprStmt:
				channels = append(channels, c.formatExpr("[%e]", removeParens(comm.X).(*ast.UnaryExpr).X).String())
			case *ast.AssignStmt:
				channels = append(channels, c.formatExpr("[%e]", removeParens(comm.Rhs[0]).(*ast.UnaryExpr).X).String())
			case *ast.SendStmt:
				channels = append(channels, c.formatExpr("[%e, %e]", comm.Chan, comm.Value).String())
			default:
				panic(fmt.Sprintf("unhandled: %T", comm))
			}
			caseClauses = append(caseClauses, &ast.CaseClause{
				List: []ast.Expr{c.newInt(i, types.Typ[types.Int])},
				Body: clause.Body,
			})
			flattened = flattened || c.flattened[clause]
		}

		selectCall := c.setType(&ast.CallExpr{
			Fun:  c.newIdent("$select", types.NewSignature(nil, nil, types.NewTuple(types.NewVar(0, nil, "", types.NewInterface(nil, nil))), types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), false)),
			Args: []ast.Expr{c.newIdent(fmt.Sprintf("[%s]", strings.Join(channels, ", ")), types.NewInterface(nil, nil))},
		}, types.Typ[types.Int])
		c.blocking[selectCall] = !hasDefault
		selectionVar := c.newVariable("_selection")
		c.Printf("%s = %s;", selectionVar, c.translateExpr(selectCall))

		translateCond := func(cond ast.Expr) *expression {
			return c.formatExpr("%s[0] === %e", selectionVar, cond)
		}
		printCaseBodyPrefix := func(index int) {
			if assign, ok := s.Body.List[index].(*ast.CommClause).Comm.(*ast.AssignStmt); ok {
				switch rhsType := c.p.info.Types[assign.Rhs[0]].Type.(type) {
				case *types.Tuple:
					c.translateStmt(&ast.AssignStmt{Lhs: assign.Lhs, Rhs: []ast.Expr{c.newIdent(selectionVar+"[1]", rhsType)}, Tok: assign.Tok}, "")
				default:
					c.translateStmt(&ast.AssignStmt{Lhs: assign.Lhs, Rhs: []ast.Expr{c.newIdent(selectionVar+"[1][0]", rhsType)}, Tok: assign.Tok}, "")
				}
			}
		}
		c.translateBranchingStmt(caseClauses, true, translateCond, printCaseBodyPrefix, label, flattened)

	case *ast.EmptyStmt:
		// skip

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
}

type branch struct {
	index     int
	clause    *ast.CaseClause
	condition string
	body      []ast.Stmt
}

func (c *funcContext) translateBranchingStmt(caseClauses []ast.Stmt, isSwitch bool, translateCond func(ast.Expr) *expression, printCaseBodyPrefix func(int), label string, flatten bool) {
	var branches []*branch
	var defaultBranch *branch
	var openBranches []*branch
clauseLoop:
	for i, cc := range caseClauses {
		clause := cc.(*ast.CaseClause)

		branch := &branch{i, clause, "", nil}
		openBranches = append(openBranches, branch)
		for _, openBranch := range openBranches {
			openBranch.body = append(openBranch.body, clause.Body...)
		}
		if !hasFallthrough(clause) {
			openBranches = nil
		}

		if len(clause.List) == 0 {
			defaultBranch = branch
			continue
		}

		var conds []string
		for _, cond := range clause.List {
			x := translateCond(cond).String()
			if x == "true" {
				defaultBranch = branch
				break clauseLoop
			}
			if x != "false" {
				conds = append(conds, x)
			}
		}
		if len(conds) == 0 {
			continue
		}
		branch.condition = strings.Join(conds, " || ")

		branches = append(branches, branch)
	}

	for defaultBranch == nil && len(branches) != 0 && len(branches[len(branches)-1].body) == 0 {
		branches = branches[:len(branches)-1]
	}

	if len(branches) == 0 {
		if defaultBranch != nil {
			c.translateStmtList(defaultBranch.body)
			return
		}
		return
	}

	hasBreak := false
	if isSwitch {
		switch label {
		case "":
			v := hasBreakVisitor{}
			for _, child := range caseClauses {
				ast.Walk(&v, child)
			}
			hasBreak = v.hasBreak
		default:
			hasBreak = true // always assume break if label is given
		}
	}

	var caseOffset, endCase int
	if flatten {
		caseOffset = c.caseCounter
		endCase = caseOffset + len(branches) - 1
		if defaultBranch != nil {
			endCase++
		}
		c.caseCounter = endCase + 1
	}

	if isSwitch {
		prevFlowData := c.flowDatas[""]
		data := &flowData{
			postStmt:  prevFlowData.postStmt,  // for "continue" of outer loop
			beginCase: prevFlowData.beginCase, // same
			endCase:   endCase,
		}
		c.flowDatas[""] = data
		c.flowDatas[label] = data
		defer func() {
			delete(c.flowDatas, label)
			c.flowDatas[""] = prevFlowData
		}()
	}

	if isSwitch && !flatten && label != "" {
		c.Printf("%s:", label)
	}
	prefix := ""
	if hasBreak {
		prefix = "switch (0) { default: "
	}
	jump := ""
	if flatten {
		var jumpList []string
		for i, branch := range branches {
			if i == 0 {
				jumpList = append(jumpList, fmt.Sprintf("if (%s) {}", branch.condition))
				continue
			}
			jumpList = append(jumpList, fmt.Sprintf("if (%s) { $s = %d; continue; }", branch.condition, caseOffset+i-1))
		}
		jumpList = append(jumpList, fmt.Sprintf("{ $s = %d; continue; }", caseOffset+len(branches)-1))
		jump = strings.Join(jumpList, " else ")
	}
	for i, branch := range branches {
		c.WritePos(branch.clause.Pos())
		c.PrintCond(!flatten, fmt.Sprintf("%sif (%s) {", prefix, branch.condition), jump)
		c.Indent(func() {
			if printCaseBodyPrefix != nil {
				printCaseBodyPrefix(branch.index)
			}
			c.translateStmtList(branch.body)
		})
		prefix = "} else "
		jump = fmt.Sprintf("$s = %d; continue; case %d: ", endCase, caseOffset+i)
	}
	if defaultBranch != nil {
		c.PrintCond(!flatten, "} else {", jump)
		c.Indent(func() {
			if printCaseBodyPrefix != nil {
				printCaseBodyPrefix(defaultBranch.index)
			}
			c.translateStmtList(defaultBranch.body)
		})
	}
	if hasBreak {
		c.PrintCond(!flatten, "} }", fmt.Sprintf("case %d:", endCase))
		return
	}
	c.PrintCond(!flatten, "}", fmt.Sprintf("case %d:", endCase))
}

func (c *funcContext) translateLoopingStmt(cond string, body *ast.BlockStmt, bodyPrefix, post func(), label string, flatten bool) {
	prevFlowData := c.flowDatas[""]
	data := &flowData{
		postStmt: post,
	}
	if flatten {
		data.beginCase = c.caseCounter
		data.endCase = c.caseCounter + 1
		c.caseCounter += 2
	}
	c.flowDatas[""] = data
	c.flowDatas[label] = data
	defer func() {
		delete(c.flowDatas, label)
		c.flowDatas[""] = prevFlowData
	}()

	if !flatten && label != "" {
		c.Printf("%s:", label)
	}
	c.PrintCond(!flatten, fmt.Sprintf("while (%s) {", cond), fmt.Sprintf("case %d: if(!(%s)) { $s = %d; continue; }", data.beginCase, cond, data.endCase))
	c.Indent(func() {
		prevEV := c.p.escapingVars
		c.p.escapingVars = make(map[types.Object]bool)
		for escaping := range prevEV {
			c.p.escapingVars[escaping] = true
		}

		v := &escapeAnalysis{
			info:       c.p.info,
			candidates: make(map[types.Object]bool),
			escaping:   make(map[types.Object]bool),
		}
		ast.Walk(v, body)
		names := make([]string, 0, len(c.p.escapingVars))
		for obj := range v.escaping {
			names = append(names, c.objectName(obj))
			c.p.escapingVars[obj] = true
		}
		sort.Strings(names)
		for _, name := range names {
			c.Printf("%s = [undefined];", name)
		}

		if bodyPrefix != nil {
			bodyPrefix()
		}
		c.translateStmtList(body.List)
		isTerminated := false
		if len(body.List) != 0 {
			switch body.List[len(body.List)-1].(type) {
			case *ast.ReturnStmt, *ast.BranchStmt:
				isTerminated = true
			}
		}
		if !isTerminated {
			post()
		}

		c.p.escapingVars = prevEV
	})
	c.PrintCond(!flatten, "}", fmt.Sprintf("$s = %d; continue; case %d:", data.beginCase, data.endCase))
}

func (c *funcContext) translateAssignOfExpr(lhs, rhs ast.Expr, typ types.Type, define bool) string {
	if l, ok := lhs.(*ast.IndexExpr); ok {
		if t, ok := c.p.info.Types[l.X].Type.Underlying().(*types.Map); ok {
			keyVar := c.newVariable("_key")
			return fmt.Sprintf(`%s = %s; (%s || $throwRuntimeError("assignment to entry in nil map"))[%s] = { k: %s, v: %s };`, keyVar, c.translateImplicitConversionWithCloning(l.Index, t.Key()), c.translateExpr(l.X), c.makeKey(c.newIdent(keyVar, t.Key()), t.Key()), keyVar, c.translateImplicitConversionWithCloning(rhs, t.Elem()))
		}
	}

	if _, ok := rhs.(*ast.CompositeLit); ok && define {
		return fmt.Sprintf("%s = %s;", c.translateExpr(lhs), c.translateImplicitConversion(rhs, typ)) // skip $copy
	}
	return c.translateAssign(lhs, c.translateImplicitConversion(rhs, typ).String(), typ, define)
}

func (c *funcContext) translateAssign(lhs ast.Expr, rhs string, typ types.Type, define bool) string {
	lhs = removeParens(lhs)
	if isBlank(lhs) {
		panic("translateAssign with blank lhs")
	}

	switch typ.Underlying().(type) {
	case *types.Array, *types.Struct:
		if define {
			return fmt.Sprintf("%[1]s = %[2]s; $copy(%[1]s, %[3]s, %[4]s);", c.translateExpr(lhs), c.zeroValue(typ), rhs, c.typeName(typ))
		}
		return fmt.Sprintf("$copy(%s, %s, %s);", c.translateExpr(lhs), rhs, c.typeName(typ))
	}

	switch l := lhs.(type) {
	case *ast.Ident:
		o := c.p.info.Defs[l]
		if o == nil {
			o = c.p.info.Uses[l]
		}
		return fmt.Sprintf("%s = %s;", c.objectName(o), rhs)
	case *ast.SelectorExpr:
		sel, ok := c.p.info.Selections[l]
		if !ok {
			// qualified identifier
			return fmt.Sprintf("%s.%s = %s;", c.translateExpr(l.X), l.Sel.Name, rhs)
		}
		fields, jsTag := c.translateSelection(sel)
		if jsTag != "" {
			return fmt.Sprintf("%s.%s.%s = %s;", c.translateExpr(l.X), strings.Join(fields, "."), jsTag, c.externalize(rhs, sel.Type()))
		}
		return fmt.Sprintf("%s.%s = %s;", c.translateExpr(l.X), strings.Join(fields, "."), rhs)
	case *ast.StarExpr:
		return fmt.Sprintf("%s.$set(%s);", c.translateExpr(l.X), rhs)
	case *ast.IndexExpr:
		switch t := c.p.info.Types[l.X].Type.Underlying().(type) {
		case *types.Array, *types.Pointer:
			pattern := "%1e[%2f] = %3s"
			if c.p.info.Types[l.Index].Value == nil { // add range check if not constant
				pattern = `(%2f < 0 || %2f >= %1e.length) ? $throwRuntimeError("index out of range") : ` + pattern
			}
			if _, ok := t.(*types.Pointer); ok { // check pointer for nix (attribute getter causes a panic)
				pattern = `%1e.nilCheck, ` + pattern
			}
			return c.formatExpr(pattern, l.X, l.Index, rhs).String() + ";"
		case *types.Slice:
			return c.formatExpr(`(%2f < 0 || %2f >= %1e.$length) ? $throwRuntimeError("index out of range") : %1e.$array[%1e.$offset + %2f] = %3s`, l.X, l.Index, rhs).String() + ";"
		default:
			panic(fmt.Sprintf("Unhandled lhs type: %T\n", t))
		}
	default:
		panic(fmt.Sprintf("Unhandled lhs type: %T\n", l))
	}
}

func hasFallthrough(caseClause *ast.CaseClause) bool {
	if len(caseClause.Body) == 0 {
		return false
	}
	b, isBranchStmt := caseClause.Body[len(caseClause.Body)-1].(*ast.BranchStmt)
	return isBranchStmt && b.Tok == token.FALLTHROUGH
}

type hasBreakVisitor struct {
	hasBreak bool
}

func (v *hasBreakVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if v.hasBreak {
		return nil
	}
	switch n := node.(type) {
	case *ast.BranchStmt:
		if n.Tok == token.BREAK && n.Label == nil {
			v.hasBreak = true
			return nil
		}
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, ast.Expr:
		return nil
	}
	return v
}

type hasCallVisitor struct {
	info    *types.Info
	hasCall bool
}

func (v *hasCallVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if v.hasCall {
		return nil
	}
	if call, isCall := node.(*ast.CallExpr); isCall {
		if _, isSig := v.info.Types[call.Fun].Type.(*types.Signature); isSig { // skip conversions
			v.hasCall = true
			return nil
		}
	}
	return v
}

type escapeAnalysis struct {
	info       *types.Info
	candidates map[types.Object]bool
	escaping   map[types.Object]bool
}

func (v *escapeAnalysis) Visit(node ast.Node) (w ast.Visitor) {
	// huge overapproximation
	switch n := node.(type) {
	case *ast.ValueSpec:
		for _, name := range n.Names {
			v.candidates[v.info.Defs[name]] = true
		}
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			for _, name := range n.Lhs {
				def := v.info.Defs[name.(*ast.Ident)]
				if def != nil {
					v.candidates[def] = true
				}
			}
		}
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			switch v.info.Types[n.X].Type.Underlying().(type) {
			case *types.Struct, *types.Array:
				// always by reference
				return nil
			default:
				return &escapingObjectCollector{v}
			}
		}
	case *ast.FuncLit:
		return &escapingObjectCollector{v}
	case *ast.ForStmt, *ast.RangeStmt:
		return nil
	}
	return v
}

type escapingObjectCollector struct {
	analysis *escapeAnalysis
}

func (v *escapingObjectCollector) Visit(node ast.Node) (w ast.Visitor) {
	if id, isIdent := node.(*ast.Ident); isIdent {
		obj := v.analysis.info.Uses[id]
		if v.analysis.candidates[obj] {
			v.analysis.escaping[obj] = true
		}
	}
	return v
}
