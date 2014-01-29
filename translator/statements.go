package translator

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
		var caseClauses []ast.Stmt
		var initStmts []ast.Stmt
		ifStmt := s
		for {
			caseClauses = append(caseClauses, &ast.CaseClause{List: []ast.Expr{ifStmt.Cond}, Body: ifStmt.Body.List})
			initStmts = append(initStmts, ifStmt.Init)
			switch elseStmt := ifStmt.Else.(type) {
			case *ast.IfStmt:
				ifStmt = elseStmt
				continue
			case *ast.BlockStmt:
				caseClauses = append(caseClauses, &ast.CaseClause{List: nil, Body: elseStmt.List})
				initStmts = append(initStmts, nil)
			case *ast.EmptyStmt, nil:
				// no else clause
			default:
				panic(fmt.Sprintf("Unhandled else: %T\n", elseStmt))
			}
			break
		}
		c.translateBranchingStmt(caseClauses, initStmts, false, c.translateExpr, nil, label)

	case *ast.SwitchStmt:
		if s.Init != nil {
			c.Printf("%s;", c.translateSimpleStmt(s.Init))
		}
		translateCond := func(cond ast.Expr) string {
			return c.translateExpr(cond)
		}
		if s.Tag != nil {
			refVar := c.newVariable("_ref")
			c.Printf("%s = %s;", refVar, c.translateExpr(s.Tag))
			translateCond = func(cond ast.Expr) string {
				refId := c.newIdent(refVar, c.info.Types[s.Tag].Type)
				return c.translateExpr(&ast.BinaryExpr{
					X:  refId,
					Op: token.EQL,
					Y:  cond,
				})
			}
		}
		c.translateBranchingStmt(s.Body.List, nil, true, translateCond, nil, label)

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
				c.objectVars[c.info.Implicits[caseClause]] = typeSwitchVar
			}
		case *ast.ExprStmt:
			expr = a.X.(*ast.TypeAssertExpr).X
		}
		refVar := c.newVariable("_ref")
		typeVar := c.newVariable("_type")
		c.Printf("%s = %s;", refVar, c.translateExpr(expr))
		c.Printf("%s = %s !== null ? %s.constructor : null;", typeVar, refVar, refVar)
		translateCond := func(cond ast.Expr) string {
			return c.typeCheck(typeVar, c.info.Types[cond].Type)
		}
		printCaseBodyPrefix := func(conds []ast.Expr) {
			if typeSwitchVar == "" {
				return
			}
			value := refVar
			if len(conds) == 1 {
				t := c.info.Types[conds[0]].Type
				if _, isInterface := t.Underlying().(*types.Interface); !isInterface && !types.Identical(t, types.Typ[types.UntypedNil]) {
					value += ".go$val"
				}
			}
			c.Printf("%s = %s;", typeSwitchVar, value)
		}
		c.translateBranchingStmt(s.Body.List, nil, true, translateCond, printCaseBodyPrefix, label)

	case *ast.ForStmt:
		if s.Init != nil {
			c.Printf("%s;", c.translateSimpleStmt(s.Init))
		}
		cond := "true"
		if s.Cond != nil {
			cond = c.translateExpr(s.Cond)
		}
		p := c.postLoopStmt[""]
		defer func() {
			delete(c.postLoopStmt, label)
			c.postLoopStmt[""] = p
		}()
		c.postLoopStmt[""] = s.Post
		c.postLoopStmt[label] = s.Post

		c.Printf("%swhile (%s) {", label, cond)
		c.Indent(func() {
			c.handleEscapingVariables(s.Body, func() {
				c.translateStmtList(s.Body.List)
				if s.Post != nil {
					if len(s.Body.List) != 0 {
						switch s.Body.List[len(s.Body.List)-1].(type) {
						case *ast.ReturnStmt, *ast.BranchStmt:
							return
						}
					}
					c.Printf("%s;", c.translateSimpleStmt(s.Post))
				}
			})
		})
		c.Printf("}")

	case *ast.RangeStmt:
		p := c.postLoopStmt[""]
		defer func() { c.postLoopStmt[""] = p }()
		delete(c.postLoopStmt, "")

		refVar := c.newVariable("_ref")
		c.Printf("%s = %s;", refVar, c.translateExpr(s.X))

		iVar := c.newVariable("_i")
		c.Printf("%s = 0;", iVar)

		switch t := c.info.Types[s.X].Type.Underlying().(type) {
		case *types.Basic:
			runeVar := c.newVariable("_rune")
			c.Printf("%sfor (; %s < %s.length; %s += %s[1]) {", label, iVar, refVar, iVar, runeVar)
			c.Indent(func() {
				c.handleEscapingVariables(s.Body, func() {
					c.Printf("%s = go$decodeRune(%s, %s);", runeVar, refVar, iVar)
					if !isBlank(s.Value) {
						c.Printf("%s;", c.translateAssign(s.Value, runeVar+"[0]"))
					}
					if !isBlank(s.Key) {
						c.Printf("%s;", c.translateAssign(s.Key, iVar))
					}
					c.translateStmtList(s.Body.List)
				})
			})
			c.Printf("}")

		case *types.Map:
			keysVar := c.newVariable("_keys")
			c.Printf("%s = go$keys(%s);", keysVar, refVar)
			c.Printf("%sfor (; %s < %s.length; %s += 1) {", label, iVar, keysVar, iVar)
			c.Indent(func() {
				c.handleEscapingVariables(s.Body, func() {
					entryVar := c.newVariable("_entry")
					c.Printf("%s = %s[%s[%s]];", entryVar, refVar, keysVar, iVar)
					if !isBlank(s.Value) {
						c.Printf("%s;", c.translateAssign(s.Value, entryVar+".v"))
					}
					if !isBlank(s.Key) {
						c.Printf("%s;", c.translateAssign(s.Key, entryVar+".k"))
					}
					c.translateStmtList(s.Body.List)
				})
			})
			c.Printf("}")

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
			c.Printf("%sfor (; %s < %s; %s += 1) {", label, iVar, length, iVar)
			c.Indent(func() {
				c.handleEscapingVariables(s.Body, func() {
					if !isBlank(s.Value) {
						indexExpr := &ast.IndexExpr{
							X:     c.newIdent(refVar, t),
							Index: c.newIdent(iVar, types.Typ[types.Int]),
						}
						et := elemType(t)
						c.info.Types[indexExpr] = types.TypeAndValue{Type: et}
						c.Printf("%s;", c.translateAssign(s.Value, c.translateImplicitConversion(indexExpr, et)))
					}
					if !isBlank(s.Key) {
						c.Printf("%s;", c.translateAssign(s.Key, iVar))
					}
					c.translateStmtList(s.Body.List)
				})
			})
			c.Printf("}")

		case *types.Chan:
			// skip

		default:
			panic("")
		}

	case *ast.BranchStmt:
		label := ""
		postLoopStmt := c.postLoopStmt[""]
		if s.Label != nil {
			label = " " + s.Label.Name
			postLoopStmt = c.postLoopStmt[s.Label.Name+": "]
		}
		switch s.Tok {
		case token.BREAK:
			c.Printf("break%s;", label)
		case token.CONTINUE:
			if postLoopStmt != nil {
				c.Printf("%s;", c.translateSimpleStmt(postLoopStmt))
			}
			c.Printf("continue%s;", label)
		case token.GOTO:
			c.Printf(`go$notSupported("goto");`)
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
				break
			}
			v := c.translateImplicitConversion(results[0], c.functionSig.Results().At(0).Type())
			c.delayedOutput = nil
			c.Printf("return %s;", v)
		default:
			values := make([]string, len(results))
			for i, result := range results {
				values[i] = c.translateImplicitConversion(result, c.functionSig.Results().At(i).Type())
			}
			c.delayedOutput = nil
			c.Printf("return [%s];", strings.Join(values, ", "))
		}

	case *ast.DeferStmt:
		if ident, isIdent := s.Call.Fun.(*ast.Ident); isIdent {
			if builtin, isBuiltin := c.info.Objects[ident].(*types.Builtin); isBuiltin {
				if builtin.Name() == "recover" {
					c.Printf("go$deferred.push({ fun: go$recover, args: [] });")
					return
				}
				args := make([]ast.Expr, len(s.Call.Args))
				for i, arg := range s.Call.Args {
					args[i] = c.newIdent(c.newVariable("_arg"), c.info.Types[arg].Type)
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
		sig := c.info.Types[s.Call.Fun].Type.Underlying().(*types.Signature)
		args := c.translateArgs(sig, s.Call.Args, s.Call.Ellipsis.IsValid())
		if sel, isSelector := s.Call.Fun.(*ast.SelectorExpr); isSelector {
			c.Printf(`go$deferred.push({ recv: %s, method: "%s", args: [%s] });`, c.translateExpr(sel.X), sel.Sel.Name, args)
			return
		}
		c.Printf("go$deferred.push({ fun: %s, args: [%s] });", c.translateExpr(s.Call.Fun), args)

	case *ast.DeclStmt:
		decl := s.Decl.(*ast.GenDecl)
		switch decl.Tok {
		case token.VAR:
			c.Printf("%s%s;", label, c.translateSimpleStmt(stmt))
		case token.TYPE:
			for _, spec := range decl.Specs {
				o := c.info.Objects[spec.(*ast.TypeSpec).Name].(*types.TypeName)
				c.translateType(o)
				c.initType(o)
			}
		case token.CONST:
			// skip, constants are inlined
		}

	case *ast.LabeledStmt:
		c.translateStmt(s.Stmt, s.Label.Name+": ")

	case *ast.SelectStmt:
		c.Printf(`go$notSupported("select")`)

	case *ast.GoStmt:
		c.Printf(`go$notSupported("go")`)

	case *ast.EmptyStmt:
		// skip

	default:
		if r := c.translateSimpleStmt(stmt); r != "" {
			c.Printf("%s%s;", label, r)
		}
	}
}

type branch struct {
	clause    *ast.CaseClause
	initStmt  ast.Stmt
	condition string
	body      []ast.Stmt
}

func (c *PkgContext) translateBranchingStmt(caseClauses []ast.Stmt, initStmts []ast.Stmt, isSwitch bool, translateCond func(ast.Expr) string, printCaseBodyPrefix func([]ast.Expr), label string) {
	var branches []*branch
	var defaultBranch *branch
	var openBranches []*branch
clauseLoop:
	for i, cc := range caseClauses {
		clause := cc.(*ast.CaseClause)

		var initStmt ast.Stmt
		if initStmts != nil {
			initStmt = initStmts[i]
		}
		branch := &branch{clause, initStmt, "", nil}
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
			x := translateCond(cond)
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

	printBody := func() {
		elsePrefix := ""
		for _, branch := range branches {
			initStmt := ""
			if branch.initStmt != nil {
				initStmt = c.translateSimpleStmt(branch.initStmt) + ", "
			}
			c.Printf("%sif (%s%s) {", elsePrefix, initStmt, branch.condition)
			c.Indent(func() {
				if printCaseBodyPrefix != nil {
					printCaseBodyPrefix(branch.clause.List)
				}
				c.translateStmtList(branch.body)
			})
			elsePrefix = "} else "
		}
		if defaultBranch != nil {
			c.Printf("} else {")
			c.Indent(func() {
				if printCaseBodyPrefix != nil {
					printCaseBodyPrefix(nil)
				}
				c.translateStmtList(defaultBranch.body)
			})
		}
		c.Printf("}")
	}

	if !isSwitch {
		printBody()
		return
	}

	v := HasBreakVisitor{}
	for _, child := range caseClauses {
		ast.Walk(&v, child)
	}
	if !v.hasBreak && label == "" {
		printBody()
		return
	}

	c.Printf("%sswitch (undefined) {", label)
	c.Printf("default:")
	c.Indent(func() {
		printBody()
	})
	c.Printf("}")
}

func (c *PkgContext) translateSimpleStmt(stmt ast.Stmt) string {
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
				parts = append(parts, lhsVar+" = "+c.translateExpr(l.X))
				parts = append(parts, indexVar+" = "+c.translateExpr(l.Index))
				lhs = &ast.IndexExpr{
					X:     c.newIdent(lhsVar, c.info.Types[l.X].Type),
					Index: c.newIdent(indexVar, c.info.Types[l.Index].Type),
				}
				c.info.Types[lhs] = c.info.Types[l]
			case *ast.StarExpr:
				lhsVar := c.newVariable("_lhs")
				parts = append(parts, lhsVar+" = "+c.translateExpr(l.X))
				lhs = &ast.StarExpr{
					X: c.newIdent(lhsVar, c.info.Types[l.X].Type),
				}
				c.info.Types[lhs] = c.info.Types[l]
			case *ast.SelectorExpr:
				v := HasCallVisitor{c.info, false}
				ast.Walk(&v, l.X)
				if v.hasCall {
					lhsVar := c.newVariable("_lhs")
					parts = append(parts, lhsVar+" = "+c.translateExpr(l.X))
					lhs = &ast.SelectorExpr{
						X:   c.newIdent(lhsVar, c.info.Types[l.X].Type),
						Sel: l.Sel,
					}
					c.info.Types[lhs] = c.info.Types[l]
					c.info.Selections[lhs.(*ast.SelectorExpr)] = c.info.Selections[l]
				}
			}

			parenExpr := &ast.ParenExpr{X: s.Rhs[0]}
			c.info.Types[parenExpr] = c.info.Types[s.Rhs[0]]
			binaryExpr := &ast.BinaryExpr{
				X:  lhs,
				Op: op,
				Y:  parenExpr,
			}
			c.info.Types[binaryExpr] = c.info.Types[s.Lhs[0]]
			parts = append(parts, c.translateAssign(lhs, c.translateExpr(binaryExpr)))
			return strings.Join(parts, ", ")
		}

		if s.Tok == token.DEFINE {
			for _, lhs := range s.Lhs {
				if !isBlank(lhs) {
					c.info.Types[lhs] = types.TypeAndValue{Type: c.info.Objects[lhs.(*ast.Ident)].Type()}
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
				v := HasCallVisitor{c.info, false}
				ast.Walk(&v, s.Rhs[0])
				if v.hasCall {
					return c.translateExpr(s.Rhs[0])
				}
				return ""
			}
			return c.translateAssign(lhs, c.translateImplicitConversion(s.Rhs[0], c.info.Types[s.Lhs[0]].Type))

		case len(s.Lhs) > 1 && len(s.Rhs) == 1:
			tupleVar := c.newVariable("_tuple")
			out := tupleVar + " = " + c.translateExpr(s.Rhs[0])
			tuple := c.info.Types[s.Rhs[0]].Type.(*types.Tuple)
			for i, lhs := range s.Lhs {
				lhs = removeParens(lhs)
				if !isBlank(lhs) {
					out += ", " + c.translateAssign(lhs, c.translateImplicitConversion(c.newIdent(fmt.Sprintf("%s[%d]", tupleVar, i), tuple.At(i).Type()), c.info.Types[s.Lhs[i]].Type))
				}
			}
			return out
		case len(s.Lhs) == len(s.Rhs):
			parts := make([]string, len(s.Rhs))
			for i, rhs := range s.Rhs {
				parts[i] = c.translateImplicitConversion(rhs, c.info.Types[s.Lhs[i]].Type)
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
		t := c.info.Types[s.X].Type
		if iExpr, isIExpr := s.X.(*ast.IndexExpr); isIExpr {
			switch u := c.info.Types[iExpr.X].Type.Underlying().(type) {
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
		c.info.Types[one] = types.TypeAndValue{Type: t, Value: exact.MakeInt64(1)}
		return c.translateSimpleStmt(&ast.AssignStmt{
			Lhs: []ast.Expr{s.X},
			Tok: tok,
			Rhs: []ast.Expr{one},
		})

	case *ast.ExprStmt:
		return c.translateExpr(s.X)

	case *ast.DeclStmt:
		var parts []string
		for _, spec := range s.Decl.(*ast.GenDecl).Specs {
			for _, singleSpec := range c.splitValueSpec(spec.(*ast.ValueSpec)) {
				lhs := make([]ast.Expr, len(singleSpec.Names))
				for i, name := range singleSpec.Names {
					lhs[i] = name
				}
				parts = append(parts, c.translateSimpleStmt(&ast.AssignStmt{
					Lhs: lhs,
					Tok: token.DEFINE,
					Rhs: singleSpec.Values,
				}))
			}
		}
		return strings.Join(parts, ", ")

	case *ast.SendStmt:
		return `go$notSupported("send")`

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
}

func (c *PkgContext) translateAssign(lhs ast.Expr, rhs string) string {
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
		return c.objectName(c.info.Objects[l]) + " = " + rhs
	case *ast.SelectorExpr:
		sel := c.info.Selections[l]
		switch sel.Kind() {
		case types.FieldVal:
			fields, jsTag := c.translateSelection(sel)
			if jsTag != "" {
				return fmt.Sprintf("%s.%s.%s = %s", c.translateExpr(l.X), strings.Join(fields, "."), jsTag, c.externalize(rhs, sel.Type()))
			}
			return fmt.Sprintf("%s.%s = %s", c.translateExpr(l.X), strings.Join(fields, "."), rhs)
		case types.PackageObj:
			return c.translateExpr(l.X) + "." + l.Sel.Name + " = " + rhs
		default:
			panic(int(sel.Kind()))
		}
	case *ast.StarExpr:
		switch u := c.info.Types[lhs].Type.Underlying().(type) {
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
		switch t := c.info.Types[l.X].Type.Underlying().(type) {
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

func (c *PkgContext) handleEscapingVariables(node ast.Node, f func()) {
	v := &EscapeAnalysis{
		info:       c.info,
		candidates: make(map[types.Object]bool),
		escaping:   make(map[types.Object]bool),
	}
	ast.Walk(v, node)
	ev := c.escapingVars
	for escaping := range v.escaping {
		c.Printf("%s = [undefined];", c.objectName(escaping))
		c.escapingVars = append(c.escapingVars, c.objectVars[escaping])
		c.objectVars[escaping] += "[0]"
	}
	f()
	c.escapingVars = ev
}

func hasFallthrough(caseClause *ast.CaseClause) bool {
	if len(caseClause.Body) == 0 {
		return false
	}
	b, isBranchStmt := caseClause.Body[len(caseClause.Body)-1].(*ast.BranchStmt)
	return isBranchStmt && b.Tok == token.FALLTHROUGH
}

type HasBreakVisitor struct {
	hasBreak bool
}

func (v *HasBreakVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if v.hasBreak {
		return nil
	}
	switch n := node.(type) {
	case *ast.BranchStmt:
		if n.Tok == token.BREAK && n.Label == nil {
			v.hasBreak = true
			return nil
		}
	case *ast.FuncLit, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return nil
	}
	return v
}

type HasCallVisitor struct {
	info    *types.Info
	hasCall bool
}

func (v *HasCallVisitor) Visit(node ast.Node) (w ast.Visitor) {
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

type EscapeAnalysis struct {
	info       *types.Info
	candidates map[types.Object]bool
	escaping   map[types.Object]bool
}

// huge overapproximation
func (v *EscapeAnalysis) Visit(node ast.Node) (w ast.Visitor) {
	switch n := node.(type) {
	case *ast.ValueSpec:
		for _, name := range n.Names {
			v.candidates[v.info.Objects[name]] = true
		}
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			for _, name := range n.Lhs {
				v.candidates[v.info.Objects[name.(*ast.Ident)]] = true
			}
		}
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			switch v.info.Types[n.X].Type.Underlying().(type) {
			case *types.Struct, *types.Array:
				// always by reference
				return nil
			default:
				return &EscapingObjectCollector{v}
			}
		}
	case *ast.FuncLit:
		return &EscapingObjectCollector{v}
	case *ast.ForStmt, *ast.RangeStmt:
		return nil
	}
	return v
}

type EscapingObjectCollector struct {
	analysis *EscapeAnalysis
}

func (v *EscapingObjectCollector) Visit(node ast.Node) (w ast.Visitor) {
	if id, isIdent := node.(*ast.Ident); isIdent {
		obj := v.analysis.info.Objects[id]
		if v.analysis.candidates[obj] {
			v.analysis.escaping[obj] = true
		}
	}
	return v
}
