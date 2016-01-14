package compiler

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/analysis"
	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/filter"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

type this struct {
	ast.Ident
}

func (c *funcContext) translateStmtList(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		c.translateStmt(stmt, nil)
	}
	c.SetPos(token.NoPos)
}

func (c *funcContext) translateStmt(stmt ast.Stmt, label *types.Label) {
	c.SetPos(stmt.Pos())

	stmt = filter.IncDecStmt(stmt, c.p.Info)
	stmt = filter.Assign(stmt, c.p.Info)

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		c.translateStmtList(s.List)

	case *ast.IfStmt:
		if s.Init != nil {
			c.translateStmt(s.Init, nil)
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
		c.translateBranchingStmt(caseClauses, false, nil, nil, nil, c.Flattened[s])

	case *ast.SwitchStmt:
		if s.Init != nil {
			c.translateStmt(s.Init, nil)
		}

		tag := s.Tag
		if tag == nil {
			tag = ast.NewIdent("true")
			c.p.Types[tag] = types.TypeAndValue{Type: types.Typ[types.Bool], Value: constant.MakeBool(true)}
		}

		if c.p.Types[tag].Value == nil {
			refVar := c.newVariable("_ref")
			c.Printf("%s = %s;", refVar, c.translateExpr(tag))
			tag = c.newIdent(refVar, c.p.TypeOf(tag))
		}

		translateCond := func(cond ast.Expr) *expression {
			return c.translateExpr(&ast.BinaryExpr{
				X:  tag,
				Op: token.EQL,
				Y:  cond,
			})
		}
		c.translateBranchingStmt(s.Body.List, true, translateCond, nil, label, c.Flattened[s])

	case *ast.TypeSwitchStmt:
		if s.Init != nil {
			c.translateStmt(s.Init, nil)
		}
		refVar := c.newVariable("_ref")
		var expr ast.Expr
		var printCaseBodyPrefix func(index int)
		switch a := s.Assign.(type) {
		case *ast.AssignStmt:
			expr = a.Rhs[0].(*ast.TypeAssertExpr).X
			printCaseBodyPrefix = func(index int) {
				value := refVar
				caseClause := s.Body.List[index].(*ast.CaseClause)
				if len(caseClause.List) == 1 {
					t := c.p.TypeOf(caseClause.List[0])
					if _, isInterface := t.Underlying().(*types.Interface); !isInterface && !types.Identical(t, types.Typ[types.UntypedNil]) {
						value += ".$val"
					}
				}
				c.Printf("%s = %s;", c.objectName(c.p.Implicits[caseClause]), value)
			}
		case *ast.ExprStmt:
			expr = a.X.(*ast.TypeAssertExpr).X
		}
		c.Printf("%s = %s;", refVar, c.translateExpr(expr))
		translateCond := func(cond ast.Expr) *expression {
			if types.Identical(c.p.TypeOf(cond), types.Typ[types.UntypedNil]) {
				return c.formatExpr("%s === $ifaceNil", refVar)
			}
			return c.formatExpr("$assertType(%s, %s, true)[1]", refVar, c.typeName(c.p.TypeOf(cond)))
		}
		c.translateBranchingStmt(s.Body.List, true, translateCond, printCaseBodyPrefix, label, c.Flattened[s])

	case *ast.ForStmt:
		if s.Init != nil {
			c.translateStmt(s.Init, nil)
		}
		cond := func() string {
			if s.Cond == nil {
				return "true"
			}
			return c.translateExpr(s.Cond).String()
		}
		c.translateLoopingStmt(cond, s.Body, nil, func() {
			if s.Post != nil {
				c.translateStmt(s.Post, nil)
			}
		}, label, c.Flattened[s])

	case *ast.RangeStmt:
		refVar := c.newVariable("_ref")
		c.Printf("%s = %s;", refVar, c.translateExpr(s.X))

		switch t := c.p.TypeOf(s.X).Underlying().(type) {
		case *types.Basic:
			iVar := c.newVariable("_i")
			c.Printf("%s = 0;", iVar)
			runeVar := c.newVariable("_rune")
			c.translateLoopingStmt(func() string { return iVar + " < " + refVar + ".length" }, s.Body, func() {
				c.Printf("%s = $decodeRune(%s, %s);", runeVar, refVar, iVar)
				if !isBlank(s.Key) {
					c.Printf("%s", c.translateAssign(s.Key, c.newIdent(iVar, types.Typ[types.Int]), s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					c.Printf("%s", c.translateAssign(s.Value, c.newIdent(runeVar+"[0]", types.Typ[types.Rune]), s.Tok == token.DEFINE))
				}
			}, func() {
				c.Printf("%s += %s[1];", iVar, runeVar)
			}, label, c.Flattened[s])

		case *types.Map:
			iVar := c.newVariable("_i")
			c.Printf("%s = 0;", iVar)
			keysVar := c.newVariable("_keys")
			c.Printf("%s = $keys(%s);", keysVar, refVar)
			c.translateLoopingStmt(func() string { return iVar + " < " + keysVar + ".length" }, s.Body, func() {
				entryVar := c.newVariable("_entry")
				c.Printf("%s = %s[%s[%s]];", entryVar, refVar, keysVar, iVar)
				c.translateStmt(&ast.IfStmt{
					Cond: c.newIdent(entryVar+" === undefined", types.Typ[types.Bool]),
					Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.CONTINUE}}},
				}, nil)
				if !isBlank(s.Key) {
					c.Printf("%s", c.translateAssign(s.Key, c.newIdent(entryVar+".k", t.Key()), s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					c.Printf("%s", c.translateAssign(s.Value, c.newIdent(entryVar+".v", t.Elem()), s.Tok == token.DEFINE))
				}
			}, func() {
				c.Printf("%s++;", iVar)
			}, label, c.Flattened[s])

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
			c.translateLoopingStmt(func() string { return iVar + " < " + length }, s.Body, func() {
				if !isBlank(s.Key) {
					c.Printf("%s", c.translateAssign(s.Key, c.newIdent(iVar, types.Typ[types.Int]), s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					c.Printf("%s", c.translateAssign(s.Value, c.setType(&ast.IndexExpr{
						X:     c.newIdent(refVar, t),
						Index: c.newIdent(iVar, types.Typ[types.Int]),
					}, elemType), s.Tok == token.DEFINE))
				}
			}, func() {
				c.Printf("%s++;", iVar)
			}, label, c.Flattened[s])

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
			c.Flattened[forStmt] = true
			c.translateStmt(forStmt, label)

		default:
			panic("")
		}

	case *ast.BranchStmt:
		normalLabel := ""
		blockingLabel := ""
		data := c.flowDatas[nil]
		if s.Label != nil {
			normalLabel = " " + s.Label.Name
			blockingLabel = " s" // use explicit label "s", because surrounding loop may not be flattened
			data = c.flowDatas[c.p.Uses[s.Label].(*types.Label)]
		}
		switch s.Tok {
		case token.BREAK:
			c.PrintCond(data.endCase == 0, fmt.Sprintf("break%s;", normalLabel), fmt.Sprintf("$s = %d; continue%s;", data.endCase, blockingLabel))
		case token.CONTINUE:
			data.postStmt()
			c.PrintCond(data.beginCase == 0, fmt.Sprintf("continue%s;", normalLabel), fmt.Sprintf("$s = %d; continue%s;", data.beginCase, blockingLabel))
		case token.GOTO:
			c.PrintCond(false, "goto "+s.Label.Name, fmt.Sprintf("$s = %d; continue;", c.labelCase(c.p.Uses[s.Label].(*types.Label))))
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
				}, nil)
			}
			results = c.resultNames
		}
		rVal := c.translateResults(results)
		if c.Flattened[s] {
			resumeCase := c.caseCounter
			c.caseCounter++
			c.Printf("/* */ $s = %[1]d; case %[1]d:", resumeCase)
		}
		c.Printf("return%s;", rVal)

	case *ast.DeferStmt:
		isBuiltin := false
		isJs := false
		switch fun := s.Call.Fun.(type) {
		case *ast.Ident:
			var builtin *types.Builtin
			builtin, isBuiltin = c.p.Uses[fun].(*types.Builtin)
			if isBuiltin && builtin.Name() == "recover" {
				c.Printf("$deferred.push([$recover, []]);")
				return
			}
		case *ast.SelectorExpr:
			isJs = typesutil.IsJsPackage(c.p.Uses[fun.Sel].Pkg())
		}
		sig := c.p.TypeOf(s.Call.Fun).Underlying().(*types.Signature)
		args := c.translateArgs(sig, s.Call.Args, s.Call.Ellipsis.IsValid(), true)
		if isBuiltin || isJs {
			vars := make([]string, len(s.Call.Args))
			callArgs := make([]ast.Expr, len(s.Call.Args))
			for i, arg := range s.Call.Args {
				v := c.newVariable("_arg")
				vars[i] = v
				callArgs[i] = c.newIdent(v, c.p.TypeOf(arg))
			}
			call := c.translateExpr(&ast.CallExpr{
				Fun:      s.Call.Fun,
				Args:     callArgs,
				Ellipsis: s.Call.Ellipsis,
			})
			c.Printf("$deferred.push([function(%s) { %s; }, [%s]]);", strings.Join(vars, ", "), call, strings.Join(args, ", "))
			return
		}
		c.Printf("$deferred.push([%s, [%s]]);", c.translateExpr(s.Call.Fun), strings.Join(args, ", "))

	case *ast.AssignStmt:
		if s.Tok != token.ASSIGN && s.Tok != token.DEFINE {
			panic(s.Tok)
		}

		switch {
		case len(s.Lhs) == 1 && len(s.Rhs) == 1:
			lhs := astutil.RemoveParens(s.Lhs[0])
			if isBlank(lhs) {
				if analysis.HasSideEffect(s.Rhs[0], c.p.Info.Info) {
					c.Printf("%s;", c.translateExpr(s.Rhs[0]))
				}
				return
			}
			c.Printf("%s", c.translateAssign(lhs, s.Rhs[0], s.Tok == token.DEFINE))

		case len(s.Lhs) > 1 && len(s.Rhs) == 1:
			tupleVar := c.newVariable("_tuple")
			c.Printf("%s = %s;", tupleVar, c.translateExpr(s.Rhs[0]))
			tuple := c.p.TypeOf(s.Rhs[0]).(*types.Tuple)
			for i, lhs := range s.Lhs {
				lhs = astutil.RemoveParens(lhs)
				if !isBlank(lhs) {
					c.Printf("%s", c.translateAssign(lhs, c.newIdent(fmt.Sprintf("%s[%d]", tupleVar, i), tuple.At(i).Type()), s.Tok == token.DEFINE))
				}
			}
		case len(s.Lhs) == len(s.Rhs):
			tmpVars := make([]string, len(s.Rhs))
			for i, rhs := range s.Rhs {
				tmpVars[i] = c.newVariable("_tmp")
				if isBlank(astutil.RemoveParens(s.Lhs[i])) {
					if analysis.HasSideEffect(rhs, c.p.Info.Info) {
						c.Printf("%s;", c.translateExpr(rhs))
					}
					continue
				}
				c.Printf("%s", c.translateAssign(c.newIdent(tmpVars[i], c.p.TypeOf(s.Lhs[i])), rhs, true))
			}
			for i, lhs := range s.Lhs {
				lhs = astutil.RemoveParens(lhs)
				if !isBlank(lhs) {
					c.Printf("%s", c.translateAssign(lhs, c.newIdent(tmpVars[i], c.p.TypeOf(lhs)), s.Tok == token.DEFINE))
				}
			}

		default:
			panic("Invalid arity of AssignStmt.")

		}

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
				if len(rhs) == 0 {
					rhs = make([]ast.Expr, len(lhs))
					for i, e := range lhs {
						rhs[i] = c.zeroValue(c.p.TypeOf(e))
					}
				}
				c.translateStmt(&ast.AssignStmt{
					Lhs: lhs,
					Tok: token.DEFINE,
					Rhs: rhs,
				}, nil)
			}
		case token.TYPE:
			for _, spec := range decl.Specs {
				o := c.p.Defs[spec.(*ast.TypeSpec).Name].(*types.TypeName)
				c.p.typeNames = append(c.p.typeNames, o)
				c.p.objectNames[o] = c.newVariableWithLevel(o.Name(), true)
				c.p.dependencies[o] = true
			}
		case token.CONST:
			// skip, constants are inlined
		}

	case *ast.ExprStmt:
		expr := c.translateExpr(s.X)
		if expr != nil && expr.String() != "" {
			c.Printf("%s;", expr)
		}

	case *ast.LabeledStmt:
		label := c.p.Defs[s.Label].(*types.Label)
		if c.GotoLabel[label] {
			c.PrintCond(false, s.Label.Name+":", fmt.Sprintf("case %d:", c.labelCase(label)))
		}
		c.translateStmt(s.Stmt, label)

	case *ast.GoStmt:
		c.Printf("$go(%s, [%s]);", c.translateExpr(s.Call.Fun), strings.Join(c.translateArgs(c.p.TypeOf(s.Call.Fun).Underlying().(*types.Signature), s.Call.Args, s.Call.Ellipsis.IsValid(), true), ", "))

	case *ast.SendStmt:
		chanType := c.p.TypeOf(s.Chan).Underlying().(*types.Chan)
		call := &ast.CallExpr{
			Fun:  c.newIdent("$send", types.NewSignature(nil, types.NewTuple(types.NewVar(0, nil, "", chanType), types.NewVar(0, nil, "", chanType.Elem())), nil, false)),
			Args: []ast.Expr{s.Chan, c.newIdent(c.translateImplicitConversionWithCloning(s.Value, chanType.Elem()).String(), chanType.Elem())},
		}
		c.Blocking[call] = true
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
				channels = append(channels, c.formatExpr("[%e]", astutil.RemoveParens(comm.X).(*ast.UnaryExpr).X).String())
			case *ast.AssignStmt:
				channels = append(channels, c.formatExpr("[%e]", astutil.RemoveParens(comm.Rhs[0]).(*ast.UnaryExpr).X).String())
			case *ast.SendStmt:
				chanType := c.p.TypeOf(comm.Chan).Underlying().(*types.Chan)
				channels = append(channels, c.formatExpr("[%e, %s]", comm.Chan, c.translateImplicitConversionWithCloning(comm.Value, chanType.Elem())).String())
			default:
				panic(fmt.Sprintf("unhandled: %T", comm))
			}
			indexLit := &ast.BasicLit{Kind: token.INT}
			c.p.Types[indexLit] = types.TypeAndValue{Type: types.Typ[types.Int], Value: constant.MakeInt64(int64(i))}
			caseClauses = append(caseClauses, &ast.CaseClause{
				List: []ast.Expr{indexLit},
				Body: clause.Body,
			})
			flattened = flattened || c.Flattened[clause]
		}

		selectCall := c.setType(&ast.CallExpr{
			Fun:  c.newIdent("$select", types.NewSignature(nil, types.NewTuple(types.NewVar(0, nil, "", types.NewInterface(nil, nil))), types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), false)),
			Args: []ast.Expr{c.newIdent(fmt.Sprintf("[%s]", strings.Join(channels, ", ")), types.NewInterface(nil, nil))},
		}, types.Typ[types.Int])
		c.Blocking[selectCall] = !hasDefault
		selectionVar := c.newVariable("_selection")
		c.Printf("%s = %s;", selectionVar, c.translateExpr(selectCall))

		translateCond := func(cond ast.Expr) *expression {
			return c.formatExpr("%s[0] === %e", selectionVar, cond)
		}
		printCaseBodyPrefix := func(index int) {
			if assign, ok := s.Body.List[index].(*ast.CommClause).Comm.(*ast.AssignStmt); ok {
				switch rhsType := c.p.TypeOf(assign.Rhs[0]).(type) {
				case *types.Tuple:
					c.translateStmt(&ast.AssignStmt{Lhs: assign.Lhs, Rhs: []ast.Expr{c.newIdent(selectionVar+"[1]", rhsType)}, Tok: assign.Tok}, nil)
				default:
					c.translateStmt(&ast.AssignStmt{Lhs: assign.Lhs, Rhs: []ast.Expr{c.newIdent(selectionVar+"[1][0]", rhsType)}, Tok: assign.Tok}, nil)
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
	index   int
	clause  *ast.CaseClause
	conds   []ast.Expr
	condStr string
	body    []ast.Stmt
}

func (c *funcContext) translateBranchingStmt(caseClauses []ast.Stmt, isSwitch bool, translateCond func(ast.Expr) *expression, printCaseBodyPrefix func(int), label *types.Label, flatten bool) {
	var branches []*branch
	var defaultBranch *branch
	var openBranches []*branch
clauseLoop:
	for i, cc := range caseClauses {
		clause := cc.(*ast.CaseClause)

		branch := &branch{index: i, clause: clause}
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

		for _, cond := range clause.List {
			if translateCond == nil {
				if b, ok := analysis.BoolValue(cond, c.p.Info.Info); ok {
					if b {
						defaultBranch = branch
						break clauseLoop
					}
					continue
				}
			}
			branch.conds = append(branch.conds, cond)
		}
		if len(branch.conds) == 0 {
			continue
		}

		branches = append(branches, branch)
	}

	for defaultBranch == nil && len(branches) != 0 && len(branches[len(branches)-1].body) == 0 && printCaseBodyPrefix == nil {
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
		case nil:
			for _, child := range caseClauses {
				if analysis.HasBreak(child) {
					hasBreak = true
					break
				}
			}
		default:
			hasBreak = true // always assume break if label is given
		}
	}

	var caseOffset, endCase int
	if flatten {
		caseOffset = c.caseCounter
		endCase = caseOffset + len(branches)
		if defaultBranch != nil {
			endCase++
		}
		c.caseCounter = endCase + 1
	}

	if isSwitch {
		prevFlowData := c.flowDatas[nil]
		data := &flowData{
			postStmt:  prevFlowData.postStmt,  // for "continue" of outer loop
			beginCase: prevFlowData.beginCase, // same
			endCase:   endCase,
		}
		c.flowDatas[nil] = data
		c.flowDatas[label] = data
		defer func() {
			delete(c.flowDatas, label)
			c.flowDatas[nil] = prevFlowData
		}()
	}

	if isSwitch && !flatten && label != nil {
		c.Printf("%s:", label.Name())
	}
	prefix := ""
	if hasBreak {
		prefix = "switch (0) { default: "
	}
	for i, b := range branches {
		conds := make([]string, len(b.conds))
		for i, cond := range b.conds {
			if translateCond == nil {
				conds[i] = c.translateExpr(cond).String()
				continue
			}
			conds[i] = translateCond(cond).String()
		}
		b.condStr = strings.Join(conds, " || ")
		if flatten {
			c.Printf("/* */ if (%s) { $s = %d; continue; }", b.condStr, caseOffset+i)
		}
	}
	if flatten {
		c.Printf("/* */ $s = %d; continue;", caseOffset+len(branches))
	}
	for i, b := range branches {
		c.SetPos(b.clause.Pos())
		c.PrintCond(!flatten, fmt.Sprintf("%sif (%s) {", prefix, b.condStr), fmt.Sprintf("case %d:", caseOffset+i))
		c.Indent(func() {
			if printCaseBodyPrefix != nil {
				printCaseBodyPrefix(b.index)
			}
			c.translateStmtList(b.body)
			if flatten && (defaultBranch != nil || i != len(branches)-1) && !endsWithReturn(b.body) {
				c.Printf("$s = %d; continue;", endCase)
			}
		})
		prefix = "} else "
	}
	if defaultBranch != nil {
		c.PrintCond(!flatten, "} else {", fmt.Sprintf("case %d:", caseOffset+len(branches)))
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

func (c *funcContext) translateLoopingStmt(cond func() string, body *ast.BlockStmt, bodyPrefix, post func(), label *types.Label, flatten bool) {
	prevFlowData := c.flowDatas[nil]
	data := &flowData{
		postStmt: post,
	}
	if flatten {
		data.beginCase = c.caseCounter
		data.endCase = c.caseCounter + 1
		c.caseCounter += 2
	}
	c.flowDatas[nil] = data
	c.flowDatas[label] = data
	defer func() {
		delete(c.flowDatas, label)
		c.flowDatas[nil] = prevFlowData
	}()

	if !flatten && label != nil {
		c.Printf("%s:", label.Name())
	}
	c.PrintCond(!flatten, "while (true) {", fmt.Sprintf("case %d:", data.beginCase))
	c.Indent(func() {
		condStr := cond()
		if condStr != "true" {
			c.PrintCond(!flatten, fmt.Sprintf("if (!(%s)) { break; }", condStr), fmt.Sprintf("if(!(%s)) { $s = %d; continue; }", condStr, data.endCase))
		}

		prevEV := c.p.escapingVars
		c.handleEscapingVars(body)

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

func (c *funcContext) translateAssign(lhs, rhs ast.Expr, define bool) string {
	lhs = astutil.RemoveParens(lhs)
	if isBlank(lhs) {
		panic("translateAssign with blank lhs")
	}

	if l, ok := lhs.(*ast.IndexExpr); ok {
		if t, ok := c.p.TypeOf(l.X).Underlying().(*types.Map); ok {
			if typesutil.IsJsObject(c.p.TypeOf(l.Index)) {
				c.p.errList = append(c.p.errList, types.Error{Fset: c.p.fileSet, Pos: l.Index.Pos(), Msg: "cannot use js.Object as map key"})
			}
			keyVar := c.newVariable("_key")
			return fmt.Sprintf(`%s = %s; (%s || $throwRuntimeError("assignment to entry in nil map"))[%s.keyFor(%s)] = { k: %s, v: %s };`, keyVar, c.translateImplicitConversionWithCloning(l.Index, t.Key()), c.translateExpr(l.X), c.typeName(t.Key()), keyVar, keyVar, c.translateImplicitConversionWithCloning(rhs, t.Elem()))
		}
	}

	lhsType := c.p.TypeOf(lhs)
	rhsExpr := c.translateImplicitConversion(rhs, lhsType)
	if _, ok := rhs.(*ast.CompositeLit); ok && define {
		return fmt.Sprintf("%s = %s;", c.translateExpr(lhs), rhsExpr) // skip $copy
	}

	isReflectValue := false
	if named, ok := lhsType.(*types.Named); ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "reflect" && named.Obj().Name() == "Value" {
		isReflectValue = true
	}
	if !isReflectValue { // this is a performance hack, but it is safe since reflect.Value has no exported fields and the reflect package does not violate this assumption
		switch lhsType.Underlying().(type) {
		case *types.Array, *types.Struct:
			if define {
				return fmt.Sprintf("%s = $clone(%s, %s);", c.translateExpr(lhs), rhsExpr, c.typeName(lhsType))
			}
			return fmt.Sprintf("%s.copy(%s, %s);", c.typeName(lhsType), c.translateExpr(lhs), rhsExpr)
		}
	}

	switch l := lhs.(type) {
	case *ast.Ident:
		o := c.p.Defs[l]
		if o == nil {
			o = c.p.Uses[l]
		}
		return fmt.Sprintf("%s = %s;", c.objectName(o), rhsExpr)
	case *ast.SelectorExpr:
		sel, ok := c.p.SelectionOf(l)
		if !ok {
			// qualified identifier
			return fmt.Sprintf("%s = %s;", c.objectName(c.p.Uses[l.Sel]), rhsExpr)
		}
		fields, jsTag := c.translateSelection(sel, l.Pos())
		if jsTag != "" {
			return fmt.Sprintf("%s.%s.%s = %s;", c.translateExpr(l.X), strings.Join(fields, "."), jsTag, c.externalize(rhsExpr.String(), sel.Type()))
		}
		return fmt.Sprintf("%s.%s = %s;", c.translateExpr(l.X), strings.Join(fields, "."), rhsExpr)
	case *ast.StarExpr:
		return fmt.Sprintf("%s.$set(%s);", c.translateExpr(l.X), rhsExpr)
	case *ast.IndexExpr:
		switch t := c.p.TypeOf(l.X).Underlying().(type) {
		case *types.Array, *types.Pointer:
			pattern := rangeCheck("%1e[%2f] = %3s", c.p.Types[l.Index].Value != nil, true)
			if _, ok := t.(*types.Pointer); ok { // check pointer for nil (attribute getter causes a panic)
				pattern = `%1e.nilCheck, ` + pattern
			}
			return c.formatExpr(pattern, l.X, l.Index, rhsExpr).String() + ";"
		case *types.Slice:
			return c.formatExpr(rangeCheck("%1e.$array[%1e.$offset + %2f] = %3s", c.p.Types[l.Index].Value != nil, false), l.X, l.Index, rhsExpr).String() + ";"
		default:
			panic(fmt.Sprintf("Unhandled lhs type: %T\n", t))
		}
	default:
		panic(fmt.Sprintf("Unhandled lhs type: %T\n", l))
	}
}

func (c *funcContext) translateResults(results []ast.Expr) string {
	tuple := c.sig.Results()
	switch tuple.Len() {
	case 0:
		return ""
	case 1:
		result := c.zeroValue(tuple.At(0).Type())
		if results != nil {
			result = results[0]
		}
		v := c.translateImplicitConversion(result, tuple.At(0).Type())
		c.delayedOutput = nil
		return " " + v.String()
	default:
		if len(results) == 1 {
			return " " + c.translateExpr(results[0]).String()
		}
		values := make([]string, tuple.Len())
		for i := range values {
			result := c.zeroValue(tuple.At(i).Type())
			if results != nil {
				result = results[i]
			}
			values[i] = c.translateImplicitConversion(result, tuple.At(i).Type()).String()
		}
		c.delayedOutput = nil
		return " [" + strings.Join(values, ", ") + "]"
	}
}

func hasFallthrough(caseClause *ast.CaseClause) bool {
	if len(caseClause.Body) == 0 {
		return false
	}
	b, isBranchStmt := caseClause.Body[len(caseClause.Body)-1].(*ast.BranchStmt)
	return isBranchStmt && b.Tok == token.FALLTHROUGH
}

func (c *funcContext) labelCase(label *types.Label) int {
	labelCase, ok := c.labelCases[label]
	if !ok {
		labelCase = c.caseCounter
		c.caseCounter++
		c.labelCases[label] = labelCase
	}
	return labelCase
}
