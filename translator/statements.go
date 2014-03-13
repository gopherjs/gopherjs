package translator

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
		c.translateBranchingStmt(caseClauses, false, c.translateExpr, nil, "", c.hasGoto[s])

	case *ast.SwitchStmt:
		if s.Init != nil {
			c.Printf("%s;", c.translateSimpleStmt(s.Init))
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
		c.translateBranchingStmt(s.Body.List, true, translateCond, nil, label, c.hasGoto[s])

	case *ast.TypeSwitchStmt:
		if s.Init != nil {
			c.Printf("%s;", c.translateSimpleStmt(s.Init))
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
					value += ".go$val"
				}
			}
			c.Printf("%s = %s;", typeSwitchVar, value)
		}
		c.translateBranchingStmt(s.Body.List, true, translateCond, printCaseBodyPrefix, label, c.hasGoto[s])

	case *ast.ForStmt:
		if s.Init != nil {
			c.Printf("%s;", c.translateSimpleStmt(s.Init))
		}
		cond := "true"
		if s.Cond != nil {
			cond = c.translateExpr(s.Cond).String()
		}
		c.translateLoopingStmt(cond, s.Body, nil, func() {
			if s.Post != nil {
				c.translateStmt(s.Post, "")
			}
		}, label, c.hasGoto[s])

	case *ast.RangeStmt:
		refVar := c.newVariable("_ref")
		c.Printf("%s = %s;", refVar, c.translateExpr(s.X))

		iVar := c.newVariable("_i")
		c.Printf("%s = 0;", iVar)

		switch t := c.p.info.Types[s.X].Type.Underlying().(type) {
		case *types.Basic:
			runeVar := c.newVariable("_rune")
			c.translateLoopingStmt(iVar+" < "+refVar+".length", s.Body, func() {
				c.Printf("%s = go$decodeRune(%s, %s);", runeVar, refVar, iVar)
				if !isBlank(s.Value) {
					c.Printf("%s;", c.translateAssign(s.Value, runeVar+"[0]"))
				}
				if !isBlank(s.Key) {
					c.Printf("%s;", c.translateAssign(s.Key, iVar))
				}
			}, func() {
				c.Printf("%s += %s[1];", iVar, runeVar)
			}, label, c.hasGoto[s])

		case *types.Map:
			keysVar := c.newVariable("_keys")
			c.Printf("%s = go$keys(%s);", keysVar, refVar)
			c.translateLoopingStmt(iVar+" < "+keysVar+".length", s.Body, func() {
				entryVar := c.newVariable("_entry")
				c.Printf("%s = %s[%s[%s]];", entryVar, refVar, keysVar, iVar)
				if !isBlank(s.Value) {
					c.Printf("%s;", c.translateAssign(s.Value, entryVar+".v"))
				}
				if !isBlank(s.Key) {
					c.Printf("%s;", c.translateAssign(s.Key, entryVar+".k"))
				}
			}, func() {
				c.Printf("%s++;", iVar)
			}, label, c.hasGoto[s])

		case *types.Array, *types.Pointer, *types.Slice:
			var length string
			switch t2 := t.(type) {
			case *types.Array:
				length = fmt.Sprintf("%d", t2.Len())
			case *types.Pointer:
				length = fmt.Sprintf("%d", t2.Elem().Underlying().(*types.Array).Len())
			case *types.Slice:
				length = refVar + ".length"
			}
			c.translateLoopingStmt(iVar+" < "+length, s.Body, func() {
				if !isBlank(s.Value) {
					indexExpr := &ast.IndexExpr{
						X:     c.newIdent(refVar, t),
						Index: c.newIdent(iVar, types.Typ[types.Int]),
					}
					et := elemType(t)
					c.p.info.Types[indexExpr] = types.TypeAndValue{Type: et}
					c.Printf("%s;", c.translateAssign(s.Value, c.translateImplicitConversion(indexExpr, et).String()))
				}
				if !isBlank(s.Key) {
					c.Printf("%s;", c.translateAssign(s.Key, iVar))
				}
			}, func() {
				c.Printf("%s++;", iVar)
			}, label, c.hasGoto[s])

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
			c.PrintCond(data.endCase == 0, fmt.Sprintf("break%s;", labelSuffix), fmt.Sprintf("go$s = %d; continue;", data.endCase))
		case token.CONTINUE:
			data.postStmt()
			c.PrintCond(data.beginCase == 0, fmt.Sprintf("continue%s;", labelSuffix), fmt.Sprintf("go$s = %d; continue;", data.beginCase))
		case token.GOTO:
			c.PrintCond(false, "goto "+s.Label.Name, fmt.Sprintf("go$s = %d; continue;", c.labelCases[s.Label.Name]))
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
		switch len(results) {
		case 0:
			c.Printf("return;")
		case 1:
			if c.sig.Results().Len() > 1 {
				c.Printf("return %s;", c.translateExpr(results[0]))
				break
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
		c.printLabel(label)
		if ident, isIdent := s.Call.Fun.(*ast.Ident); isIdent {
			if builtin, isBuiltin := c.p.info.Uses[ident].(*types.Builtin); isBuiltin {
				if builtin.Name() == "recover" {
					c.Printf("go$deferred.push({ fun: go$recover, args: [] });")
					return
				}
				args := make([]ast.Expr, len(s.Call.Args))
				for i, arg := range s.Call.Args {
					args[i] = c.newIdent(c.newVariable("_arg"), c.p.info.Types[arg].Type)
				}
				call := c.translateExpr(&ast.CallExpr{
					Fun:      s.Call.Fun,
					Args:     args,
					Ellipsis: s.Call.Ellipsis,
				})
				c.Printf("go$deferred.push({ fun: function(%s) { %s; }, args: [%s] });", strings.Join(c.translateExprSlice(args, nil), ", "), call, strings.Join(c.translateExprSlice(s.Call.Args, nil), ", "))
				return
			}
		}
		sig := c.p.info.Types[s.Call.Fun].Type.Underlying().(*types.Signature)
		args := c.translateArgs(sig, s.Call.Args, s.Call.Ellipsis.IsValid())
		if sel, isSelector := s.Call.Fun.(*ast.SelectorExpr); isSelector {
			obj := c.p.info.Selections[sel].Obj()
			if !obj.Exported() {
				c.p.dependencies[obj] = true
			}
			c.Printf(`go$deferred.push({ recv: %s, method: "%s", args: [%s] });`, c.translateExpr(sel.X), sel.Sel.Name, args)
			return
		}
		c.Printf("go$deferred.push({ fun: %s, args: [%s] });", c.translateExpr(s.Call.Fun), args)

	case *ast.DeclStmt:
		c.printLabel(label)
		decl := s.Decl.(*ast.GenDecl)
		switch decl.Tok {
		case token.VAR:
			c.Printf("%s;", c.translateSimpleStmt(stmt))
		case token.TYPE:
			for _, spec := range decl.Specs {
				o := c.p.info.Defs[spec.(*ast.TypeSpec).Name].(*types.TypeName)
				c.translateType(o, false)
				c.initType(o)
			}
		case token.CONST:
			// skip, constants are inlined
		}

	case *ast.LabeledStmt:
		c.printLabel(label)
		c.translateStmt(s.Stmt, s.Label.Name)

	case *ast.SelectStmt:
		c.printLabel(label)
		c.Printf(`go$notSupported("select")`)

	case *ast.GoStmt:
		c.printLabel(label)
		c.Printf(`go$notSupported("go")`)

	case *ast.EmptyStmt:
		// skip

	default:
		c.printLabel(label)
		if r := c.translateSimpleStmt(stmt); r != "" {
			c.Printf("%s;", r)
		}
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
			jumpList = append(jumpList, fmt.Sprintf("if (%s) { go$s = %d; continue; }", branch.condition, caseOffset+i-1))
		}
		jumpList = append(jumpList, fmt.Sprintf("{ go$s = %d; continue; }", caseOffset+len(branches)-1))
		jump = strings.Join(jumpList, " else ")
	}
	for i, branch := range branches {
		c.PrintCond(!flatten, fmt.Sprintf("%sif (%s) {", prefix, branch.condition), jump)
		c.Indent(func() {
			if printCaseBodyPrefix != nil {
				printCaseBodyPrefix(branch.clause.List)
			}
			c.translateStmtList(branch.body)
		})
		prefix = "} else "
		jump = fmt.Sprintf("go$s = %d; continue; case %d: ", endCase, caseOffset+i)
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
	c.PrintCond(!flatten, fmt.Sprintf("while (%s) {", cond), fmt.Sprintf("case %d: if(!(%s)) { go$s = %d; continue; }", data.beginCase, cond, data.endCase))
	c.Indent(func() {
		v := &escapeAnalysis{
			info:       c.p.info,
			candidates: make(map[types.Object]bool),
			escaping:   make(map[types.Object]bool),
		}
		ast.Walk(v, body)
		prevEV := c.escapingVars
		for escaping := range v.escaping {
			c.Printf("%s = [undefined];", c.objectName(escaping))
			c.escapingVars = append(c.escapingVars, c.p.objectVars[escaping])
			c.p.objectVars[escaping] += "[0]"
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

		c.escapingVars = prevEV
	})
	c.PrintCond(!flatten, "}", fmt.Sprintf("go$s = %d; continue; case %d:", data.beginCase, data.endCase))

	delete(c.flowDatas, label)
	c.flowDatas[""] = prevFlowData
}

func (c *funcContext) translateSimpleStmt(stmt ast.Stmt) string {
	switch s := stmt.(type) {
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
				parts = append(parts, lhsVar+" = "+c.translateExpr(l.X).String())
				parts = append(parts, indexVar+" = "+c.translateExpr(l.Index).String())
				lhs = &ast.IndexExpr{
					X:     c.newIdent(lhsVar, c.p.info.Types[l.X].Type),
					Index: c.newIdent(indexVar, c.p.info.Types[l.Index].Type),
				}
				c.p.info.Types[lhs] = c.p.info.Types[l]
			case *ast.StarExpr:
				lhsVar := c.newVariable("_lhs")
				parts = append(parts, lhsVar+" = "+c.translateExpr(l.X).String())
				lhs = &ast.StarExpr{
					X: c.newIdent(lhsVar, c.p.info.Types[l.X].Type),
				}
				c.p.info.Types[lhs] = c.p.info.Types[l]
			case *ast.SelectorExpr:
				v := hasCallVisitor{c.p.info, false}
				ast.Walk(&v, l.X)
				if v.hasCall {
					lhsVar := c.newVariable("_lhs")
					parts = append(parts, lhsVar+" = "+c.translateExpr(l.X).String())
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
			c.p.info.Types[binaryExpr] = c.p.info.Types[s.Lhs[0]]
			parts = append(parts, c.translateAssign(lhs, c.translateExpr(binaryExpr).String()))
			return strings.Join(parts, ", ")
		}

		if s.Tok == token.DEFINE {
			for _, lhs := range s.Lhs {
				if !isBlank(lhs) {
					c.p.info.Types[lhs] = types.TypeAndValue{Type: c.p.info.Defs[lhs.(*ast.Ident)].Type()}
				}
			}
		}

		removeParens := func(e ast.Expr) ast.Expr {
			for {
				if p, isParen := e.(*ast.ParenExpr); isParen {
					e = p.X
					continue
				}
				break
			}
			return e
		}

		switch {
		case len(s.Lhs) == 1 && len(s.Rhs) == 1:
			lhs := removeParens(s.Lhs[0])
			if isBlank(lhs) {
				v := hasCallVisitor{c.p.info, false}
				ast.Walk(&v, s.Rhs[0])
				if v.hasCall {
					return c.translateExpr(s.Rhs[0]).String()
				}
				return ""
			}
			return c.translateAssign(lhs, c.translateImplicitConversion(s.Rhs[0], c.p.info.Types[s.Lhs[0]].Type).String())

		case len(s.Lhs) > 1 && len(s.Rhs) == 1:
			tupleVar := c.newVariable("_tuple")
			out := tupleVar + " = " + c.translateExpr(s.Rhs[0]).String()
			tuple := c.p.info.Types[s.Rhs[0]].Type.(*types.Tuple)
			for i, lhs := range s.Lhs {
				lhs = removeParens(lhs)
				if !isBlank(lhs) {
					out += ", " + c.translateAssign(lhs, c.translateImplicitConversion(c.newIdent(fmt.Sprintf("%s[%d]", tupleVar, i), tuple.At(i).Type()), c.p.info.Types[s.Lhs[i]].Type).String())
				}
			}
			return out
		case len(s.Lhs) == len(s.Rhs):
			parts := make([]string, len(s.Rhs))
			for i, rhs := range s.Rhs {
				parts[i] = c.translateImplicitConversion(rhs, c.p.info.Types[s.Lhs[i]].Type).String()
			}
			tupleVar := c.newVariable("_tuple")
			out := tupleVar + " = [" + strings.Join(parts, ", ") + "]"
			for i, lhs := range s.Lhs {
				lhs = removeParens(lhs)
				if !isBlank(lhs) {
					out += ", " + c.translateAssign(lhs, fmt.Sprintf("%s[%d]", tupleVar, i))
				}
			}
			return out

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
		return c.translateSimpleStmt(&ast.AssignStmt{
			Lhs: []ast.Expr{s.X},
			Tok: tok,
			Rhs: []ast.Expr{one},
		})

	case *ast.ExprStmt:
		return c.translateExpr(s.X).String()

	case *ast.DeclStmt:
		var parts []string
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
			parts = append(parts, c.translateSimpleStmt(&ast.AssignStmt{
				Lhs: lhs,
				Tok: token.DEFINE,
				Rhs: rhs,
			}))
		}
		return strings.Join(parts, ", ")

	case *ast.SendStmt:
		return `go$notSupported("send")`

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
}

func (c *funcContext) translateAssign(lhs ast.Expr, rhs string) string {
	if isBlank(lhs) {
		panic("translateAssign with blank lhs")
	}

	for {
		if p, isParenExpr := lhs.(*ast.ParenExpr); isParenExpr {
			lhs = p.X
			continue
		}
		break
	}

	switch l := lhs.(type) {
	case *ast.Ident:
		o := c.p.info.Defs[l]
		if o == nil {
			o = c.p.info.Uses[l]
		}
		return c.objectName(o) + " = " + rhs
	case *ast.SelectorExpr:
		sel := c.p.info.Selections[l]
		switch sel.Kind() {
		case types.FieldVal:
			fields, jsTag := c.translateSelection(sel)
			if jsTag != "" {
				return fmt.Sprintf("%s.%s.%s = %s", c.translateExpr(l.X), strings.Join(fields, "."), jsTag, c.externalize(rhs, sel.Type()))
			}
			return fmt.Sprintf("%s.%s = %s", c.translateExpr(l.X), strings.Join(fields, "."), rhs)
		case types.PackageObj:
			return c.translateExpr(l.X).String() + "." + l.Sel.Name + " = " + rhs
		default:
			panic(int(sel.Kind()))
		}
	case *ast.StarExpr:
		switch u := c.p.info.Types[lhs].Type.Underlying().(type) {
		case *types.Struct:
			lVar := c.newVariable("l")
			rVar := c.newVariable("r")
			out := fmt.Sprintf("%s = %s, %s = %s", lVar, c.translateExpr(l.X), rVar, rhs)
			for i := 0; i < u.NumFields(); i++ {
				name := fieldName(u, i)
				out += fmt.Sprintf(", %s.%s = %s.%s", lVar, name, rVar, name)
			}
			return out
		case *types.Array:
			return fmt.Sprintf("go$copyArray(%s, %s)", c.translateExpr(l.X), rhs)
		default:
			return fmt.Sprintf("%s.go$set(%s)", c.translateExpr(l.X), rhs)
		}
	case *ast.IndexExpr:
		switch t := c.p.info.Types[l.X].Type.Underlying().(type) {
		case *types.Array, *types.Pointer:
			return fmt.Sprintf("%s[%s] = %s", c.translateExpr(l.X), c.flatten64(l.Index), rhs)
		case *types.Slice:
			sliceVar := c.newVariable("_slice")
			indexVar := c.newVariable("_index")
			return fmt.Sprintf("%s = %s, %s = %s", sliceVar, c.translateExpr(l.X), indexVar, c.flatten64(l.Index)) +
				fmt.Sprintf(`, (%s >= 0 && %s < %s.length) ? (%s.array[%s.offset + %s] = %s) : go$throwRuntimeError("index out of range")`, indexVar, indexVar, sliceVar, sliceVar, sliceVar, indexVar, rhs)
		case *types.Map:
			keyVar := c.newVariable("_key")
			return fmt.Sprintf(`%s = %s, (%s || go$throwRuntimeError("assignment to entry in nil map"))[%s] = { k: %s, v: %s }`, keyVar, c.translateImplicitConversion(l.Index, t.Key()), c.translateExpr(l.X), c.makeKey(c.newIdent(keyVar, t.Key()), t.Key()), keyVar, rhs)
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
