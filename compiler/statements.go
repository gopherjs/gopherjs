package compiler

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/analysis"
	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/filter"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

func (fc *funcContext) translateStmtList(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		fc.translateStmt(stmt, nil)
	}
	fc.SetPos(token.NoPos)
}

func (fc *funcContext) translateStmt(stmt ast.Stmt, label *types.Label) {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		if _, yes := bailingOut(err); yes {
			panic(err) // Continue orderly bailout.
		}

		// Oh noes, we've tried to compile something so bad that compiler paniced
		// and ran away. Let's gather some debugging clues.
		bail := bailout(err)
		pos := stmt.Pos()
		if fc.posAvailable && fc.pos.IsValid() {
			pos = fc.pos
		}
		fmt.Fprintf(bail, "Occurred while compiling statement at %s:\n", fc.pkgCtx.fileSet.Position(pos))
		(&printer.Config{Tabwidth: 2, Indent: 1, Mode: printer.UseSpaces}).Fprint(bail, fc.pkgCtx.fileSet, stmt)
		fmt.Fprintf(bail, "\n\nDetailed AST:\n")
		ast.Fprint(bail, fc.pkgCtx.fileSet, stmt, ast.NotNilFilter)
		panic(bail) // Initiate orderly bailout.
	}()

	fc.SetPos(stmt.Pos())

	stmt = filter.IncDecStmt(stmt, fc.pkgCtx.Info.Info)
	stmt = filter.Assign(stmt, fc.pkgCtx.Info.Info, fc.pkgCtx.Info.Pkg)

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		fc.translateStmtList(s.List)

	case *ast.IfStmt:
		var caseClauses []*ast.CaseClause
		ifStmt := s
		for {
			if ifStmt.Init != nil {
				panic("simplification error")
			}
			caseClauses = append(caseClauses, &ast.CaseClause{List: []ast.Expr{ifStmt.Cond}, Body: ifStmt.Body.List})
			elseStmt, ok := ifStmt.Else.(*ast.IfStmt)
			if !ok {
				break
			}
			ifStmt = elseStmt
		}
		var defaultClause *ast.CaseClause
		if block, ok := ifStmt.Else.(*ast.BlockStmt); ok {
			defaultClause = &ast.CaseClause{Body: block.List}
		}
		fc.translateBranchingStmt(caseClauses, defaultClause, false, fc.translateExpr, nil, fc.Flattened[s])

	case *ast.SwitchStmt:
		if s.Init != nil || s.Tag != nil || len(s.Body.List) != 1 {
			panic("simplification error")
		}
		clause := s.Body.List[0].(*ast.CaseClause)
		if len(clause.List) != 0 {
			panic("simplification error")
		}

		prevFlowData := fc.flowDatas[nil]
		data := &flowData{
			postStmt:  prevFlowData.postStmt,  // for "continue" of outer loop
			beginCase: prevFlowData.beginCase, // same
		}
		fc.flowDatas[nil] = data
		fc.flowDatas[label] = data
		defer func() {
			delete(fc.flowDatas, label)
			fc.flowDatas[nil] = prevFlowData
		}()

		if fc.Flattened[s] {
			data.endCase = fc.caseCounter
			fc.caseCounter++

			fc.Indent(func() {
				fc.translateStmtList(clause.Body)
			})
			fc.Printf("case %d:", data.endCase)
			return
		}

		if label != nil || analysis.HasBreak(clause) {
			if label != nil {
				fc.Printf("%s:", label.Name())
			}
			fc.Printf("switch (0) { default:")
			fc.Indent(func() {
				fc.translateStmtList(clause.Body)
			})
			fc.Printf("}")
			return
		}

		fc.translateStmtList(clause.Body)

	case *ast.TypeSwitchStmt:
		if s.Init != nil {
			fc.translateStmt(s.Init, nil)
		}
		refVar := fc.newVariable("_ref")
		var expr ast.Expr
		switch a := s.Assign.(type) {
		case *ast.AssignStmt:
			expr = a.Rhs[0].(*ast.TypeAssertExpr).X
		case *ast.ExprStmt:
			expr = a.X.(*ast.TypeAssertExpr).X
		}
		fc.Printf("%s = %s;", refVar, fc.translateExpr(expr))
		translateCond := func(cond ast.Expr) *expression {
			if types.Identical(fc.pkgCtx.TypeOf(cond), types.Typ[types.UntypedNil]) {
				return fc.formatExpr("%s === $ifaceNil", refVar)
			}
			return fc.formatExpr("$assertType(%s, %s, true)[1]", refVar, fc.typeName(fc.pkgCtx.TypeOf(cond)))
		}
		var caseClauses []*ast.CaseClause
		var defaultClause *ast.CaseClause
		for _, cc := range s.Body.List {
			clause := cc.(*ast.CaseClause)
			var bodyPrefix []ast.Stmt
			if implicit := fc.pkgCtx.Implicits[clause]; implicit != nil {
				value := refVar
				if typesutil.IsJsObject(implicit.Type().Underlying()) {
					value += ".$val.object"
				} else if _, ok := implicit.Type().Underlying().(*types.Interface); !ok {
					value += ".$val"
				}
				bodyPrefix = []ast.Stmt{&ast.AssignStmt{
					Lhs: []ast.Expr{fc.newIdent(fc.objectName(implicit), implicit.Type())},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{fc.newIdent(value, implicit.Type())},
				}}
			}
			c := &ast.CaseClause{
				List: clause.List,
				Body: append(bodyPrefix, clause.Body...),
			}
			if len(c.List) == 0 {
				defaultClause = c
				continue
			}
			caseClauses = append(caseClauses, c)
		}
		fc.translateBranchingStmt(caseClauses, defaultClause, true, translateCond, label, fc.Flattened[s])

	case *ast.ForStmt:
		if s.Init != nil {
			fc.translateStmt(s.Init, nil)
		}
		cond := func() string {
			if s.Cond == nil {
				return "true"
			}
			return fc.translateExpr(s.Cond).String()
		}
		fc.translateLoopingStmt(cond, s.Body, nil, func() {
			if s.Post != nil {
				fc.translateStmt(s.Post, nil)
			}
		}, label, fc.Flattened[s])

	case *ast.RangeStmt:
		refVar := fc.newVariable("_ref")
		fc.Printf("%s = %s;", refVar, fc.translateExpr(s.X))

		switch t := fc.pkgCtx.TypeOf(s.X).Underlying().(type) {
		case *types.Basic:
			iVar := fc.newVariable("_i")
			fc.Printf("%s = 0;", iVar)
			runeVar := fc.newVariable("_rune")
			fc.translateLoopingStmt(func() string { return iVar + " < " + refVar + ".length" }, s.Body, func() {
				fc.Printf("%s = $decodeRune(%s, %s);", runeVar, refVar, iVar)
				if !isBlank(s.Key) {
					fc.Printf("%s", fc.translateAssign(s.Key, fc.newIdent(iVar, types.Typ[types.Int]), s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					fc.Printf("%s", fc.translateAssign(s.Value, fc.newIdent(runeVar+"[0]", types.Typ[types.Rune]), s.Tok == token.DEFINE))
				}
			}, func() {
				fc.Printf("%s += %s[1];", iVar, runeVar)
			}, label, fc.Flattened[s])

		case *types.Map:
			iVar := fc.newVariable("_i")
			fc.Printf("%s = 0;", iVar)
			keysVar := fc.newVariable("_keys")
			fc.Printf("%s = $keys(%s);", keysVar, refVar)
			fc.translateLoopingStmt(func() string { return iVar + " < " + keysVar + ".length" }, s.Body, func() {
				entryVar := fc.newVariable("_entry")
				fc.Printf("%s = %s[%s[%s]];", entryVar, refVar, keysVar, iVar)
				fc.translateStmt(&ast.IfStmt{
					Cond: fc.newIdent(entryVar+" === undefined", types.Typ[types.Bool]),
					Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.CONTINUE}}},
				}, nil)
				if !isBlank(s.Key) {
					fc.Printf("%s", fc.translateAssign(s.Key, fc.newIdent(entryVar+".k", t.Key()), s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					fc.Printf("%s", fc.translateAssign(s.Value, fc.newIdent(entryVar+".v", t.Elem()), s.Tok == token.DEFINE))
				}
			}, func() {
				fc.Printf("%s++;", iVar)
			}, label, fc.Flattened[s])

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
			iVar := fc.newVariable("_i")
			fc.Printf("%s = 0;", iVar)
			fc.translateLoopingStmt(func() string { return iVar + " < " + length }, s.Body, func() {
				if !isBlank(s.Key) {
					fc.Printf("%s", fc.translateAssign(s.Key, fc.newIdent(iVar, types.Typ[types.Int]), s.Tok == token.DEFINE))
				}
				if !isBlank(s.Value) {
					fc.Printf("%s", fc.translateAssign(s.Value, fc.setType(&ast.IndexExpr{
						X:     fc.newIdent(refVar, t),
						Index: fc.newIdent(iVar, types.Typ[types.Int]),
					}, elemType), s.Tok == token.DEFINE))
				}
			}, func() {
				fc.Printf("%s++;", iVar)
			}, label, fc.Flattened[s])

		case *types.Chan:
			okVar := fc.newIdent(fc.newVariable("_ok"), types.Typ[types.Bool])
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
								fc.setType(&ast.UnaryExpr{X: fc.newIdent(refVar, t), Op: token.ARROW}, types.NewTuple(types.NewVar(0, nil, "", t.Elem()), types.NewVar(0, nil, "", types.Typ[types.Bool]))),
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
			fc.Flattened[forStmt] = true
			fc.translateStmt(forStmt, label)

		default:
			panic("")
		}

	case *ast.BranchStmt:
		normalLabel := ""
		blockingLabel := ""
		data := fc.flowDatas[nil]
		if s.Label != nil {
			normalLabel = " " + s.Label.Name
			blockingLabel = " s" // use explicit label "s", because surrounding loop may not be flattened
			data = fc.flowDatas[fc.pkgCtx.Uses[s.Label].(*types.Label)]
		}
		switch s.Tok {
		case token.BREAK:
			fc.PrintCond(data.endCase == 0, fmt.Sprintf("break%s;", normalLabel), fmt.Sprintf("$s = %d; continue%s;", data.endCase, blockingLabel))
		case token.CONTINUE:
			data.postStmt()
			fc.PrintCond(data.beginCase == 0, fmt.Sprintf("continue%s;", normalLabel), fmt.Sprintf("$s = %d; continue%s;", data.beginCase, blockingLabel))
		case token.GOTO:
			fc.PrintCond(false, "goto "+s.Label.Name, fmt.Sprintf("$s = %d; continue;", fc.labelCase(fc.pkgCtx.Uses[s.Label].(*types.Label))))
		case token.FALLTHROUGH:
			// handled in CaseClause
		default:
			panic("Unhandled branch statment: " + s.Tok.String())
		}

	case *ast.ReturnStmt:
		results := s.Results
		if fc.resultNames != nil {
			if len(s.Results) != 0 {
				fc.translateStmt(&ast.AssignStmt{
					Lhs: fc.resultNames,
					Tok: token.ASSIGN,
					Rhs: s.Results,
				}, nil)
			}
			results = fc.resultNames
		}
		rVal := fc.translateResults(results)

		if len(fc.Flattened) == 0 {
			// The function is not flattened and we don't have to worry about
			// resumption. A plain return statement is sufficient.
			fc.Printf("return%s;", rVal)
			return
		}
		if !fc.Blocking[s] {
			// The function is flattened, but the return statement is non-blocking
			// (i.e. doesn't lead to blocking deferred calls). A regular return
			// is sufficient, but we also make sure to not resume function body.
			fc.Printf("$s = -1; return%s;", rVal)
			return
		}

		if rVal != "" {
			// If returned expression is non empty, evaluate and store it in a
			// variable to avoid double-execution in case a deferred function blocks.
			rVar := fc.newVariable("$r")
			fc.Printf("%s =%s;", rVar, rVal)
			rVal = " " + rVar
		}

		// If deferred function is blocking, we need to re-execute return statement
		// upon resumption to make sure the returned value is not lost.
		// See: https://github.com/gopherjs/gopherjs/issues/603.
		nextCase := fc.caseCounter
		fc.caseCounter++
		fc.Printf("$s = %[1]d; case %[1]d: return%[2]s;", nextCase, rVal)
		return

	case *ast.DeferStmt:
		callable, arglist := fc.delegatedCall(s.Call)
		fc.Printf("$deferred.push([%s, %s]);", callable, arglist)

	case *ast.AssignStmt:
		if s.Tok != token.ASSIGN && s.Tok != token.DEFINE {
			panic(s.Tok)
		}

		switch {
		case len(s.Lhs) == 1 && len(s.Rhs) == 1:
			lhs := astutil.RemoveParens(s.Lhs[0])
			if isBlank(lhs) {
				fc.Printf("$unused(%s);", fc.translateImplicitConversion(s.Rhs[0], fc.pkgCtx.TypeOf(s.Lhs[0])))
				return
			}
			fc.Printf("%s", fc.translateAssign(lhs, s.Rhs[0], s.Tok == token.DEFINE))

		case len(s.Lhs) > 1 && len(s.Rhs) == 1:
			tupleVar := fc.newVariable("_tuple")
			fc.Printf("%s = %s;", tupleVar, fc.translateExpr(s.Rhs[0]))
			tuple := fc.pkgCtx.TypeOf(s.Rhs[0]).(*types.Tuple)
			for i, lhs := range s.Lhs {
				lhs = astutil.RemoveParens(lhs)
				if !isBlank(lhs) {
					fc.Printf("%s", fc.translateAssign(lhs, fc.newIdent(fmt.Sprintf("%s[%d]", tupleVar, i), tuple.At(i).Type()), s.Tok == token.DEFINE))
				}
			}
		case len(s.Lhs) == len(s.Rhs):
			tmpVars := make([]string, len(s.Rhs))
			for i, rhs := range s.Rhs {
				tmpVars[i] = fc.newVariable("_tmp")
				if isBlank(astutil.RemoveParens(s.Lhs[i])) {
					fc.Printf("$unused(%s);", fc.translateExpr(rhs))
					continue
				}
				fc.Printf("%s", fc.translateAssign(fc.newIdent(tmpVars[i], fc.pkgCtx.TypeOf(s.Lhs[i])), rhs, true))
			}
			for i, lhs := range s.Lhs {
				lhs = astutil.RemoveParens(lhs)
				if !isBlank(lhs) {
					fc.Printf("%s", fc.translateAssign(lhs, fc.newIdent(tmpVars[i], fc.pkgCtx.TypeOf(lhs)), s.Tok == token.DEFINE))
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
						rhs[i] = fc.zeroValue(fc.pkgCtx.TypeOf(e))
					}
				}
				fc.translateStmt(&ast.AssignStmt{
					Lhs: lhs,
					Tok: token.DEFINE,
					Rhs: rhs,
				}, nil)
			}
		case token.TYPE:
			for _, spec := range decl.Specs {
				o := fc.pkgCtx.Defs[spec.(*ast.TypeSpec).Name].(*types.TypeName)
				fc.pkgCtx.typeNames = append(fc.pkgCtx.typeNames, o)
				fc.pkgCtx.objectNames[o] = fc.newVariableWithLevel(o.Name(), true)
				fc.pkgCtx.dependencies[o] = true
			}
		case token.CONST:
			// skip, constants are inlined
		}

	case *ast.ExprStmt:
		expr := fc.translateExpr(s.X)
		if expr != nil && expr.String() != "" {
			fc.Printf("%s;", expr)
		}

	case *ast.LabeledStmt:
		label := fc.pkgCtx.Defs[s.Label].(*types.Label)
		if fc.GotoLabel[label] {
			fc.PrintCond(false, s.Label.Name+":", fmt.Sprintf("case %d:", fc.labelCase(label)))
		}
		fc.translateStmt(s.Stmt, label)

	case *ast.GoStmt:
		callable, arglist := fc.delegatedCall(s.Call)
		fc.Printf("$go(%s, %s);", callable, arglist)

	case *ast.SendStmt:
		chanType := fc.pkgCtx.TypeOf(s.Chan).Underlying().(*types.Chan)
		call := &ast.CallExpr{
			Fun:  fc.newIdent("$send", types.NewSignature(nil, types.NewTuple(types.NewVar(0, nil, "", chanType), types.NewVar(0, nil, "", chanType.Elem())), nil, false)),
			Args: []ast.Expr{s.Chan, fc.newIdent(fc.translateImplicitConversionWithCloning(s.Value, chanType.Elem()).String(), chanType.Elem())},
		}
		fc.Blocking[call] = true
		fc.translateStmt(&ast.ExprStmt{X: call}, label)

	case *ast.SelectStmt:
		selectionVar := fc.newVariable("_selection")
		var channels []string
		var caseClauses []*ast.CaseClause
		flattened := false
		hasDefault := false
		for i, cc := range s.Body.List {
			clause := cc.(*ast.CommClause)
			switch comm := clause.Comm.(type) {
			case nil:
				channels = append(channels, "[]")
				hasDefault = true
			case *ast.ExprStmt:
				channels = append(channels, fc.formatExpr("[%e]", astutil.RemoveParens(comm.X).(*ast.UnaryExpr).X).String())
			case *ast.AssignStmt:
				channels = append(channels, fc.formatExpr("[%e]", astutil.RemoveParens(comm.Rhs[0]).(*ast.UnaryExpr).X).String())
			case *ast.SendStmt:
				chanType := fc.pkgCtx.TypeOf(comm.Chan).Underlying().(*types.Chan)
				channels = append(channels, fc.formatExpr("[%e, %s]", comm.Chan, fc.translateImplicitConversionWithCloning(comm.Value, chanType.Elem())).String())
			default:
				panic(fmt.Sprintf("unhandled: %T", comm))
			}

			indexLit := &ast.BasicLit{Kind: token.INT}
			fc.pkgCtx.Types[indexLit] = types.TypeAndValue{Type: types.Typ[types.Int], Value: constant.MakeInt64(int64(i))}

			var bodyPrefix []ast.Stmt
			if assign, ok := clause.Comm.(*ast.AssignStmt); ok {
				switch rhsType := fc.pkgCtx.TypeOf(assign.Rhs[0]).(type) {
				case *types.Tuple:
					bodyPrefix = []ast.Stmt{&ast.AssignStmt{Lhs: assign.Lhs, Rhs: []ast.Expr{fc.newIdent(selectionVar+"[1]", rhsType)}, Tok: assign.Tok}}
				default:
					bodyPrefix = []ast.Stmt{&ast.AssignStmt{Lhs: assign.Lhs, Rhs: []ast.Expr{fc.newIdent(selectionVar+"[1][0]", rhsType)}, Tok: assign.Tok}}
				}
			}

			caseClauses = append(caseClauses, &ast.CaseClause{
				List: []ast.Expr{indexLit},
				Body: append(bodyPrefix, clause.Body...),
			})

			flattened = flattened || fc.Flattened[clause]
		}

		selectCall := fc.setType(&ast.CallExpr{
			Fun:  fc.newIdent("$select", types.NewSignature(nil, types.NewTuple(types.NewVar(0, nil, "", types.NewInterface(nil, nil))), types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), false)),
			Args: []ast.Expr{fc.newIdent(fmt.Sprintf("[%s]", strings.Join(channels, ", ")), types.NewInterface(nil, nil))},
		}, types.Typ[types.Int])
		fc.Blocking[selectCall] = !hasDefault
		fc.Printf("%s = %s;", selectionVar, fc.translateExpr(selectCall))

		if len(caseClauses) != 0 {
			translateCond := func(cond ast.Expr) *expression {
				return fc.formatExpr("%s[0] === %e", selectionVar, cond)
			}
			fc.translateBranchingStmt(caseClauses, nil, true, translateCond, label, flattened)
		}

	case *ast.EmptyStmt:
		// skip

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
}

func (fc *funcContext) translateBranchingStmt(caseClauses []*ast.CaseClause, defaultClause *ast.CaseClause, canBreak bool, translateCond func(ast.Expr) *expression, label *types.Label, flatten bool) {
	var caseOffset, defaultCase, endCase int
	if flatten {
		caseOffset = fc.caseCounter
		defaultCase = caseOffset + len(caseClauses)
		endCase = defaultCase
		if defaultClause != nil {
			endCase++
		}
		fc.caseCounter = endCase + 1
	}

	hasBreak := false
	if canBreak {
		prevFlowData := fc.flowDatas[nil]
		data := &flowData{
			postStmt:  prevFlowData.postStmt,  // for "continue" of outer loop
			beginCase: prevFlowData.beginCase, // same
			endCase:   endCase,
		}
		fc.flowDatas[nil] = data
		fc.flowDatas[label] = data
		defer func() {
			delete(fc.flowDatas, label)
			fc.flowDatas[nil] = prevFlowData
		}()

		for _, child := range caseClauses {
			if analysis.HasBreak(child) {
				hasBreak = true
				break
			}
		}
		if defaultClause != nil && analysis.HasBreak(defaultClause) {
			hasBreak = true
		}
	}

	if label != nil && !flatten {
		fc.Printf("%s:", label.Name())
	}

	condStrs := make([]string, len(caseClauses))
	for i, clause := range caseClauses {
		conds := make([]string, len(clause.List))
		for j, cond := range clause.List {
			conds[j] = translateCond(cond).String()
		}
		condStrs[i] = strings.Join(conds, " || ")
		if flatten {
			fc.Printf("/* */ if (%s) { $s = %d; continue; }", condStrs[i], caseOffset+i)
		}
	}

	if flatten {
		fc.Printf("/* */ $s = %d; continue;", defaultCase)
	}

	prefix := ""
	suffix := ""
	if label != nil || hasBreak {
		prefix = "switch (0) { default: "
		suffix = " }"
	}

	for i, clause := range caseClauses {
		fc.SetPos(clause.Pos())
		fc.PrintCond(!flatten, fmt.Sprintf("%sif (%s) {", prefix, condStrs[i]), fmt.Sprintf("case %d:", caseOffset+i))
		fc.Indent(func() {
			fc.translateStmtList(clause.Body)
			if flatten && (i < len(caseClauses)-1 || defaultClause != nil) && !astutil.EndsWithReturn(clause.Body) {
				fc.Printf("$s = %d; continue;", endCase)
			}
		})
		prefix = "} else "
	}

	if defaultClause != nil {
		fc.PrintCond(!flatten, prefix+"{", fmt.Sprintf("case %d:", caseOffset+len(caseClauses)))
		fc.Indent(func() {
			fc.translateStmtList(defaultClause.Body)
		})
	}

	fc.PrintCond(!flatten, "}"+suffix, fmt.Sprintf("case %d:", endCase))
}

func (fc *funcContext) translateLoopingStmt(cond func() string, body *ast.BlockStmt, bodyPrefix, post func(), label *types.Label, flatten bool) {
	prevFlowData := fc.flowDatas[nil]
	data := &flowData{
		postStmt: post,
	}
	if flatten {
		data.beginCase = fc.caseCounter
		data.endCase = fc.caseCounter + 1
		fc.caseCounter += 2
	}
	fc.flowDatas[nil] = data
	fc.flowDatas[label] = data
	defer func() {
		delete(fc.flowDatas, label)
		fc.flowDatas[nil] = prevFlowData
	}()

	if !flatten && label != nil {
		fc.Printf("%s:", label.Name())
	}
	isTerminated := false
	fc.PrintCond(!flatten, "while (true) {", fmt.Sprintf("case %d:", data.beginCase))
	fc.Indent(func() {
		condStr := cond()
		if condStr != "true" {
			fc.PrintCond(!flatten, fmt.Sprintf("if (!(%s)) { break; }", condStr), fmt.Sprintf("if(!(%s)) { $s = %d; continue; }", condStr, data.endCase))
		}

		prevEV := fc.pkgCtx.escapingVars
		fc.handleEscapingVars(body)

		if bodyPrefix != nil {
			bodyPrefix()
		}
		fc.translateStmtList(body.List)
		if len(body.List) != 0 {
			switch body.List[len(body.List)-1].(type) {
			case *ast.ReturnStmt, *ast.BranchStmt:
				isTerminated = true
			}
		}
		if !isTerminated {
			post()
		}

		fc.pkgCtx.escapingVars = prevEV
	})
	if flatten {
		// If the last statement of the loop is a return or unconditional branching
		// statement, there's no need for an instruction to go back to the beginning
		// of the loop.
		if !isTerminated {
			fc.Printf("$s = %d; continue;", data.beginCase)
		}
		fc.Printf("case %d:", data.endCase)
	} else {
		fc.Printf("}")
	}
}

func (fc *funcContext) translateAssign(lhs, rhs ast.Expr, define bool) string {
	lhs = astutil.RemoveParens(lhs)
	if isBlank(lhs) {
		panic("translateAssign with blank lhs")
	}

	if l, ok := lhs.(*ast.IndexExpr); ok {
		if t, ok := fc.pkgCtx.TypeOf(l.X).Underlying().(*types.Map); ok {
			if typesutil.IsJsObject(fc.pkgCtx.TypeOf(l.Index)) {
				fc.pkgCtx.errList = append(fc.pkgCtx.errList, types.Error{Fset: fc.pkgCtx.fileSet, Pos: l.Index.Pos(), Msg: "cannot use js.Object as map key"})
			}
			keyVar := fc.newVariable("_key")
			return fmt.Sprintf(`%s = %s; (%s || $throwRuntimeError("assignment to entry in nil map"))[%s.keyFor(%s)] = { k: %s, v: %s };`, keyVar, fc.translateImplicitConversionWithCloning(l.Index, t.Key()), fc.translateExpr(l.X), fc.typeName(t.Key()), keyVar, keyVar, fc.translateImplicitConversionWithCloning(rhs, t.Elem()))
		}
	}

	lhsType := fc.pkgCtx.TypeOf(lhs)
	rhsExpr := fc.translateImplicitConversion(rhs, lhsType)
	if _, ok := rhs.(*ast.CompositeLit); ok && define {
		return fmt.Sprintf("%s = %s;", fc.translateExpr(lhs), rhsExpr) // skip $copy
	}

	isReflectValue := false
	if named, ok := lhsType.(*types.Named); ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "reflect" && named.Obj().Name() == "Value" {
		isReflectValue = true
	}
	if !isReflectValue { // this is a performance hack, but it is safe since reflect.Value has no exported fields and the reflect package does not violate this assumption
		switch lhsType.Underlying().(type) {
		case *types.Array, *types.Struct:
			if define {
				return fmt.Sprintf("%s = $clone(%s, %s);", fc.translateExpr(lhs), rhsExpr, fc.typeName(lhsType))
			}
			return fmt.Sprintf("%s.copy(%s, %s);", fc.typeName(lhsType), fc.translateExpr(lhs), rhsExpr)
		}
	}

	switch l := lhs.(type) {
	case *ast.Ident:
		return fmt.Sprintf("%s = %s;", fc.objectName(fc.pkgCtx.ObjectOf(l)), rhsExpr)
	case *ast.SelectorExpr:
		sel, ok := fc.pkgCtx.SelectionOf(l)
		if !ok {
			// qualified identifier
			return fmt.Sprintf("%s = %s;", fc.objectName(fc.pkgCtx.Uses[l.Sel]), rhsExpr)
		}
		fields, jsTag := fc.translateSelection(sel, l.Pos())
		if jsTag != "" {
			return fmt.Sprintf("%s.%s%s = %s;", fc.translateExpr(l.X), strings.Join(fields, "."), formatJSStructTagVal(jsTag), fc.externalize(rhsExpr.String(), sel.Type()))
		}
		return fmt.Sprintf("%s.%s = %s;", fc.translateExpr(l.X), strings.Join(fields, "."), rhsExpr)
	case *ast.StarExpr:
		return fmt.Sprintf("%s.$set(%s);", fc.translateExpr(l.X), rhsExpr)
	case *ast.IndexExpr:
		switch t := fc.pkgCtx.TypeOf(l.X).Underlying().(type) {
		case *types.Array, *types.Pointer:
			pattern := rangeCheck("%1e[%2f] = %3s", fc.pkgCtx.Types[l.Index].Value != nil, true)
			if _, ok := t.(*types.Pointer); ok { // check pointer for nil (attribute getter causes a panic)
				pattern = `%1e.nilCheck, ` + pattern
			}
			return fc.formatExpr(pattern, l.X, l.Index, rhsExpr).String() + ";"
		case *types.Slice:
			return fc.formatExpr(rangeCheck("%1e.$array[%1e.$offset + %2f] = %3s", fc.pkgCtx.Types[l.Index].Value != nil, false), l.X, l.Index, rhsExpr).String() + ";"
		default:
			panic(fmt.Sprintf("Unhandled lhs type: %T\n", t))
		}
	default:
		panic(fmt.Sprintf("Unhandled lhs type: %T\n", l))
	}
}

func (fc *funcContext) translateResults(results []ast.Expr) string {
	tuple := fc.sig.Results()
	switch tuple.Len() {
	case 0:
		return ""
	case 1:
		result := fc.zeroValue(tuple.At(0).Type())
		if results != nil {
			result = results[0]
		}
		v := fc.translateImplicitConversion(result, tuple.At(0).Type())
		fc.delayedOutput = nil
		return " " + v.String()
	default:
		if len(results) == 1 {
			resultTuple := fc.pkgCtx.TypeOf(results[0]).(*types.Tuple)

			if resultTuple.Len() != tuple.Len() {
				panic("invalid tuple return assignment")
			}

			resultExpr := fc.translateExpr(results[0]).String()

			if types.Identical(resultTuple, tuple) {
				return " " + resultExpr
			}

			tmpVar := fc.newVariable("_returncast")
			fc.Printf("%s = %s;", tmpVar, resultExpr)

			// Not all the return types matched, map everything out for implicit casting
			results = make([]ast.Expr, resultTuple.Len())
			for i := range results {
				results[i] = fc.newIdent(fmt.Sprintf("%s[%d]", tmpVar, i), resultTuple.At(i).Type())
			}
		}
		values := make([]string, tuple.Len())
		for i := range values {
			result := fc.zeroValue(tuple.At(i).Type())
			if results != nil {
				result = results[i]
			}
			values[i] = fc.translateImplicitConversion(result, tuple.At(i).Type()).String()
		}
		fc.delayedOutput = nil
		return " [" + strings.Join(values, ", ") + "]"
	}
}

func (fc *funcContext) labelCase(label *types.Label) int {
	labelCase, ok := fc.labelCases[label]
	if !ok {
		labelCase = fc.caseCounter
		fc.caseCounter++
		fc.labelCases[label] = labelCase
	}
	return labelCase
}
