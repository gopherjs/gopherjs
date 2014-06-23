package compiler

import (
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type This struct {
	ast.Ident
}

func (c *funcContext) translateStmtList(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		c.translateStmt(stmt, "")
	}
}

func (c *funcContext) translateStmt(stmt ast.Stmt, label string) {
	c.WritePos(stmt.Pos())

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		c.translateStmtList(s.List)

	case *ast.IfStmt:
		c.printLabel(label)
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
				refIdent := c.newIdent(refVar, c.p.info.Types[s.Tag].Type)
				return c.translateExpr(&ast.BinaryExpr{
					X:  refIdent,
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
		typeVar := c.newVariable("_type")
		c.Printf("%s = %s;", refVar, c.translateExpr(expr))
		c.Printf("%s = %s !== null ? %s.constructor : null;", typeVar, refVar, refVar)
		translateCond := func(cond ast.Expr) *expression {
			return c.formatExpr("%s", c.typeCheck(typeVar, c.p.info.Types[cond].Type))
		}
		printCaseBodyPrefix := func(conds []ast.Expr) {
			if typeSwitchVar == "" {
				return
			}
			value := refVar
			if len(conds) == 1 {
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

		iVar := c.newVariable("_i")
		c.Printf("%s = 0;", iVar)

		switch t := c.p.info.Types[s.X].Type.Underlying().(type) {
		case *types.Basic:
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
			keysVar := c.newVariable("_keys")
			c.Printf("%s = $keys(%s);", keysVar, refVar)
			c.translateLoopingStmt(iVar+" < "+keysVar+".length", s.Body, func() {
				entryVar := c.newVariable("_entry")
				c.Printf("%s = %s[%s[%s]];", entryVar, refVar, keysVar, iVar)
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
				length = refVar + ".length"
				elemType = t2.Elem()
			}
			c.translateLoopingStmt(iVar+" < "+length, s.Body, func() {
				if !isBlank(s.Key) {
					c.Printf("%s", c.translateAssign(s.Key, iVar, types.Typ[types.Int], s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					indexExpr := &ast.IndexExpr{
						X:     c.newIdent(refVar, t),
						Index: c.newIdent(iVar, types.Typ[types.Int]),
					}
					c.p.info.Types[indexExpr] = types.TypeAndValue{Type: elemType}
					c.Printf("%s", c.translateAssign(s.Value, c.translateImplicitConversion(indexExpr, elemType).String(), elemType, s.Tok == token.DEFINE))
				}
			}, func() {
				c.Printf("%s++;", iVar)
			}, label, c.flattened[s])

		case *types.Chan:
			c.printLabel(label)
			// skip

		default:
			panic("")
		}

	case *ast.BranchStmt:
		c.printLabel(label)
		labelSuffix := ""
		data := c.flowDatas[""]
		if s.Label != nil {
			labelSuffix = " " + s.Label.Name
			data = c.flowDatas[s.Label.Name]
		}
		switch s.Tok {
		case token.BREAK:
			c.PrintCond(data.endCase == 0, fmt.Sprintf("break%s;", labelSuffix), fmt.Sprintf("$s = %d; continue;", data.endCase))
		case token.CONTINUE:
			data.postStmt()
			c.PrintCond(data.beginCase == 0, fmt.Sprintf("continue%s;", labelSuffix), fmt.Sprintf("$s = %d; continue;", data.beginCase))
		case token.GOTO:
			c.PrintCond(false, "goto "+s.Label.Name, fmt.Sprintf("$s = %d; continue;", c.labelCases[s.Label.Name]))
		case token.FALLTHROUGH:
			// handled in CaseClause
		default:
			panic("Unhandled branch statment: " + s.Tok.String())
		}

	case *ast.ReturnStmt:
		c.printLabel(label)
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
		value := ""
		switch len(results) {
		case 0:
			c.PrintCond(len(c.blocking) == 0, "return;", "$c(); return;")
			return
		case 1:
			if c.sig.Results().Len() > 1 {
				value = c.translateExpr(results[0]).String()
				break
			}
			value = c.translateImplicitConversion(results[0], c.sig.Results().At(0).Type()).String()
			c.delayedOutput = nil
		default:
			values := make([]string, len(results))
			for i, result := range results {
				values[i] = c.translateImplicitConversion(result, c.sig.Results().At(i).Type()).String()
			}
			value = "[" + strings.Join(values, ", ") + "]"
			c.delayedOutput = nil
		}
		c.PrintCond(len(c.blocking) == 0, fmt.Sprintf("return %s;", value), fmt.Sprintf("$c(%s); return;", value))

	case *ast.DeferStmt:
		c.printLabel(label)
		isBuiltin := false
		isJs := false
		switch fun := s.Call.Fun.(type) {
		case *ast.Ident:
			var builtin *types.Builtin
			builtin, isBuiltin = c.p.info.Uses[fun].(*types.Builtin)
			if isBuiltin && builtin.Name() == "recover" {
				c.Printf("$deferred.push({ fun: $recover, args: [] });")
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
			c.Printf("$deferred.push({ fun: function(%s) { %s; }, args: [%s] });", strings.Join(c.translateExprSlice(args, nil), ", "), call, strings.Join(c.translateExprSlice(s.Call.Args, nil), ", "))
			return
		}
		sig := c.p.info.Types[s.Call.Fun].Type.Underlying().(*types.Signature)
		args := strings.Join(c.translateArgs(sig, s.Call.Args, s.Call.Ellipsis.IsValid()), ", ")
		if s, isSelector := s.Call.Fun.(*ast.SelectorExpr); isSelector {
			obj := c.p.info.Uses[s.Sel]
			if !obj.Exported() {
				c.p.dependencies[obj] = true
			}
			recv := c.translateExpr(s.X)
			sel, ok := c.p.info.Selections[s]
			if ok && sel.Kind() == types.MethodVal && isWrapped(sel.Recv()) {
				recv = c.formatParenExpr("new %s(%s)", c.typeName(sel.Recv()), recv)
			}
			c.Printf(`$deferred.push({ recv: %s, method: "%s", args: [%s] });`, recv, s.Sel.Name, args)
			return
		}
		c.Printf("$deferred.push({ fun: %s, args: [%s] });", c.translateExpr(s.Call.Fun), args)

	case *ast.AssignStmt:
		c.printLabel(label)
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
				lhs = &ast.IndexExpr{
					X:     c.newIdent(lhsVar, c.p.info.Types[l.X].Type),
					Index: c.newIdent(indexVar, c.p.info.Types[l.Index].Type),
				}
				c.p.info.Types[lhs] = c.p.info.Types[l]
			case *ast.StarExpr:
				lhsVar := c.newVariable("_lhs")
				parts = append(parts, lhsVar+" = "+c.translateExpr(l.X).String()+";")
				lhs = &ast.StarExpr{
					X: c.newIdent(lhsVar, c.p.info.Types[l.X].Type),
				}
				c.p.info.Types[lhs] = c.p.info.Types[l]
			case *ast.SelectorExpr:
				v := hasCallVisitor{c.p.info, false}
				ast.Walk(&v, l.X)
				if v.hasCall {
					lhsVar := c.newVariable("_lhs")
					parts = append(parts, lhsVar+" = "+c.translateExpr(l.X).String()+";")
					lhs = &ast.SelectorExpr{
						X:   c.newIdent(lhsVar, c.p.info.Types[l.X].Type),
						Sel: l.Sel,
					}
					c.p.info.Types[lhs] = c.p.info.Types[l]
					c.p.info.Selections[lhs.(*ast.SelectorExpr)] = c.p.info.Selections[l]
				}
			}

			parenExpr := &ast.ParenExpr{X: s.Rhs[0]}
			c.p.info.Types[parenExpr] = c.p.info.Types[s.Rhs[0]]
			binaryExpr := &ast.BinaryExpr{
				X:  lhs,
				Op: op,
				Y:  parenExpr,
			}
			lhsType := c.p.info.Types[s.Lhs[0]]
			c.p.info.Types[binaryExpr] = lhsType
			parts = append(parts, c.translateAssign(lhs, c.translateExpr(binaryExpr).String(), lhsType.Type, s.Tok == token.DEFINE))
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
					c.p.info.Types[lhs] = types.TypeAndValue{Type: obj.Type()}
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
			c.Printf("%s", c.translateAssign(lhs, c.translateImplicitConversion(s.Rhs[0], lhsType).String(), lhsType, s.Tok == token.DEFINE))

		case len(s.Lhs) > 1 && len(s.Rhs) == 1:
			tupleVar := c.newVariable("_tuple")
			out := tupleVar + " = " + c.translateExpr(s.Rhs[0]).String() + ";"
			tuple := c.p.info.Types[s.Rhs[0]].Type.(*types.Tuple)
			for i, lhs := range s.Lhs {
				lhs = removeParens(lhs)
				if !isBlank(lhs) {
					lhsType := c.p.info.Types[s.Lhs[i]].Type
					out += " " + c.translateAssign(lhs, c.translateImplicitConversion(c.newIdent(fmt.Sprintf("%s[%d]", tupleVar, i), tuple.At(i).Type()), lhsType).String(), lhsType, s.Tok == token.DEFINE)
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
				parts = append(parts, c.translateAssign(c.newIdent(tmpVars[i], c.p.info.Types[s.Lhs[i]].Type), c.translateImplicitConversion(rhs, lhsType).String(), lhsType, true))
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
		one := &ast.BasicLit{
			Kind:  token.INT,
			Value: "1",
		}
		c.p.info.Types[one] = types.TypeAndValue{Type: t, Value: exact.MakeInt64(1)}
		c.translateStmt(&ast.AssignStmt{
			Lhs: []ast.Expr{s.X},
			Tok: tok,
			Rhs: []ast.Expr{one},
		}, label)

	case *ast.DeclStmt:
		c.printLabel(label)
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
		c.printLabel(label)
		expr := c.translateExpr(s.X).String()
		if expr != "" {
			c.Printf("%s;", expr)
		}

	case *ast.LabeledStmt:
		c.printLabel(label)
		c.translateStmt(s.Stmt, s.Label.Name)

	case *ast.SelectStmt:
		c.printLabel(label)
		c.Printf(`$notSupported("select");`)

	case *ast.GoStmt:
		c.printLabel(label)
		if !GOROUTINES {
			c.Printf(`$notSupported("go");`)
			return
		}
		c.Printf("$go(%s, [%s]);", c.translateExpr(s.Call.Fun), strings.Join(c.translateArgs(c.p.info.Types[s.Call.Fun].Type.(*types.Signature), s.Call.Args, s.Call.Ellipsis.IsValid()), ", "))

	case *ast.SendStmt:
		c.printLabel(label)
		c.Printf(`$notSupported("send");`)

	case *ast.EmptyStmt:
		// skip

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
}

type branch struct {
	clause    *ast.CaseClause
	condition string
	body      []ast.Stmt
}

func (c *funcContext) translateBranchingStmt(caseClauses []ast.Stmt, isSwitch bool, translateCond func(ast.Expr) *expression, printCaseBodyPrefix func([]ast.Expr), label string, flatten bool) {
	var branches []*branch
	var defaultBranch *branch
	var openBranches []*branch
clauseLoop:
	for _, cc := range caseClauses {
		clause := cc.(*ast.CaseClause)

		branch := &branch{clause, "", nil}
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
	if _, ok := c.labelCases[label]; ok {
		flatten = true // always flatten if label is referenced by goto
	}
	if flatten {
		caseOffset = c.caseCounter
		endCase = caseOffset + len(branches) - 1
		if defaultBranch != nil {
			endCase++
		}
		c.caseCounter = endCase + 1
	}

	var prevFlowData *flowData
	if isSwitch {
		prevFlowData = c.flowDatas[""]
		data := &flowData{
			postStmt:  prevFlowData.postStmt,  // for "continue" of outer loop
			beginCase: prevFlowData.beginCase, // same
			endCase:   endCase,
		}
		c.flowDatas[""] = data
		c.flowDatas[label] = data
	}

	c.printLabel(label)
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
				printCaseBodyPrefix(branch.clause.List)
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
				printCaseBodyPrefix(nil)
			}
			c.translateStmtList(defaultBranch.body)
		})
	}
	if hasBreak {
		c.PrintCond(!flatten, "} }", fmt.Sprintf("case %d:", endCase))
		return
	}
	c.PrintCond(!flatten, "}", fmt.Sprintf("case %d:", endCase))

	if isSwitch {
		delete(c.flowDatas, label)
		c.flowDatas[""] = prevFlowData
	}
}

func (c *funcContext) translateLoopingStmt(cond string, body *ast.BlockStmt, bodyPrefix, post func(), label string, flatten bool) {
	prevFlowData := c.flowDatas[""]
	data := &flowData{
		postStmt: post,
	}
	if _, ok := c.labelCases[label]; ok {
		flatten = true // always flatten if label is referenced by goto
	}
	if flatten {
		data.beginCase = c.caseCounter
		data.endCase = c.caseCounter + 1
		c.caseCounter += 2
	}
	c.flowDatas[""] = data
	c.flowDatas[label] = data

	c.printLabel(label)
	c.PrintCond(!flatten, fmt.Sprintf("while (%s) {", cond), fmt.Sprintf("case %d: if(!(%s)) { $s = %d; continue; }", data.beginCase, cond, data.endCase))
	c.Indent(func() {
		v := &escapeAnalysis{
			info:       c.p.info,
			candidates: make(map[types.Object]bool),
			escaping:   make(map[types.Object]bool),
		}
		ast.Walk(v, body)
		prevEV := c.p.escapingVars
		c.p.escapingVars = make(map[types.Object]bool)
		for escaping := range prevEV {
			c.p.escapingVars[escaping] = true
		}
		for escaping := range v.escaping {
			c.Printf("%s = [undefined];", c.objectName(escaping))
			c.p.escapingVars[escaping] = true
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

	delete(c.flowDatas, label)
	c.flowDatas[""] = prevFlowData
}

func (c *funcContext) translateAssign(lhs ast.Expr, rhs string, typ types.Type, define bool) string {
	lhs = removeParens(lhs)
	if isBlank(lhs) {
		panic("translateAssign with blank lhs")
	}

	if l, ok := lhs.(*ast.IndexExpr); ok {
		if t, ok := c.p.info.Types[l.X].Type.Underlying().(*types.Map); ok {
			keyVar := c.newVariable("_key")
			return fmt.Sprintf(`%s = %s; (%s || $throwRuntimeError("assignment to entry in nil map"))[%s] = { k: %s, v: %s };`, keyVar, c.translateImplicitConversion(l.Index, t.Key()), c.translateExpr(l.X), c.makeKey(c.newIdent(keyVar, t.Key()), t.Key()), keyVar, rhs)
		}
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
			return fmt.Sprintf("%s[%s] = %s;", c.translateExpr(l.X), c.formatExpr("%f", l.Index).String(), rhs)
		case *types.Slice:
			return c.formatExpr(`(%2f < 0 || %2f >= %1e.length) ? $throwRuntimeError("index out of range") : %1e.array[%1e.offset + %2f] = %3s`, l.X, l.Index, rhs).String() + ";"
		default:
			panic(fmt.Sprintf("Unhandled lhs type: %T\n", t))
		}
	default:
		panic(fmt.Sprintf("Unhandled lhs type: %T\n", l))
	}
}

func (c *funcContext) printLabel(label string) {
	if label != "" {
		labelCase, ok := c.labelCases[label]
		c.PrintCond(!ok, label+":", fmt.Sprintf("case %d:", labelCase))
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
