package compiler

import (
	"bytes"
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

type expression struct {
	str    string
	parens bool
}

func (e *expression) String() string {
	return e.str
}

func (e *expression) StringWithParens() string {
	if e.parens {
		return "(" + e.str + ")"
	}
	return e.str
}

func (c *funcContext) translateExpr(expr ast.Expr) *expression {
	exprType := c.p.info.Types[expr].Type
	if value := c.p.info.Types[expr].Value; value != nil {
		basic := types.Typ[types.String]
		if value.Kind() != exact.String { // workaround for bug in go/types
			basic = exprType.Underlying().(*types.Basic)
		}
		switch {
		case basic.Info()&types.IsBoolean != 0:
			return c.formatExpr("%s", strconv.FormatBool(exact.BoolVal(value)))
		case basic.Info()&types.IsInteger != 0:
			if is64Bit(basic) {
				d, _ := exact.Uint64Val(value)
				if basic.Kind() == types.Int64 {
					return c.formatExpr("new %s(%s, %s)", c.typeName(exprType), strconv.FormatInt(int64(d)>>32, 10), strconv.FormatUint(d&(1<<32-1), 10))
				}
				return c.formatExpr("new %s(%s, %s)", c.typeName(exprType), strconv.FormatUint(d>>32, 10), strconv.FormatUint(d&(1<<32-1), 10))
			}
			d, _ := exact.Int64Val(value)
			return c.formatExpr("%s", strconv.FormatInt(d, 10))
		case basic.Info()&types.IsFloat != 0:
			f, _ := exact.Float64Val(value)
			return c.formatExpr("%s", strconv.FormatFloat(f, 'g', -1, 64))
		case basic.Info()&types.IsComplex != 0:
			r, _ := exact.Float64Val(exact.Real(value))
			i, _ := exact.Float64Val(exact.Imag(value))
			if basic.Kind() == types.UntypedComplex {
				exprType = types.Typ[types.Complex128]
			}
			return c.formatExpr("new %s(%s, %s)", c.typeName(exprType), strconv.FormatFloat(r, 'g', -1, 64), strconv.FormatFloat(i, 'g', -1, 64))
		case basic.Info()&types.IsString != 0:
			return c.formatExpr("%s", encodeString(exact.StringVal(value)))
		default:
			panic("Unhandled constant type: " + basic.String())
		}
	}

	switch e := expr.(type) {
	case *ast.CompositeLit:
		if ptrType, isPointer := exprType.(*types.Pointer); isPointer {
			exprType = ptrType.Elem()
		}

		collectIndexedElements := func(elementType types.Type) []string {
			elements := make([]string, 0)
			i := 0
			zero := c.zeroValue(elementType)
			for _, element := range e.Elts {
				if kve, isKve := element.(*ast.KeyValueExpr); isKve {
					key, _ := exact.Int64Val(c.p.info.Types[kve.Key].Value)
					i = int(key)
					element = kve.Value
				}
				for len(elements) <= i {
					elements = append(elements, zero)
				}
				elements[i] = c.translateImplicitConversion(element, elementType).String()
				i++
			}
			return elements
		}

		switch t := exprType.Underlying().(type) {
		case *types.Array:
			elements := collectIndexedElements(t.Elem())
			if len(elements) != 0 {
				zero := c.zeroValue(t.Elem())
				for len(elements) < int(t.Len()) {
					elements = append(elements, zero)
				}
				return c.formatExpr(`$toNativeArray("%s", [%s])`, typeKind(t.Elem()), strings.Join(elements, ", "))
			}
			return c.formatExpr(`$makeNativeArray("%s", %d, function() { return %s; })`, typeKind(t.Elem()), int(t.Len()), c.zeroValue(t.Elem()))
		case *types.Slice:
			return c.formatExpr("new %s([%s])", c.typeName(exprType), strings.Join(collectIndexedElements(t.Elem()), ", "))
		case *types.Map:
			mapVar := c.newVariable("_map")
			keyVar := c.newVariable("_key")
			assignments := ""
			for _, element := range e.Elts {
				kve := element.(*ast.KeyValueExpr)
				assignments += c.formatExpr(`%s = %s, %s[%s] = { k: %s, v: %s }, `, keyVar, c.translateImplicitConversion(kve.Key, t.Key()), mapVar, c.makeKey(c.newIdent(keyVar, t.Key()), t.Key()), keyVar, c.translateImplicitConversion(kve.Value, t.Elem())).String()
			}
			return c.formatExpr("(%s = new $Map(), %s%s)", mapVar, assignments, mapVar)
		case *types.Struct:
			elements := make([]string, t.NumFields())
			isKeyValue := true
			if len(e.Elts) != 0 {
				_, isKeyValue = e.Elts[0].(*ast.KeyValueExpr)
			}
			if !isKeyValue {
				for i, element := range e.Elts {
					elements[i] = c.translateImplicitConversion(element, t.Field(i).Type()).String()
				}
			}
			if isKeyValue {
				for i := range elements {
					elements[i] = c.zeroValue(t.Field(i).Type())
				}
				for _, element := range e.Elts {
					kve := element.(*ast.KeyValueExpr)
					for j := range elements {
						if kve.Key.(*ast.Ident).Name == t.Field(j).Name() {
							elements[j] = c.translateImplicitConversion(kve.Value, t.Field(j).Type()).String()
							break
						}
					}
				}
			}
			return c.formatExpr("new %s.Ptr(%s)", c.typeName(exprType), strings.Join(elements, ", "))
		default:
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", t))
		}

	case *ast.FuncLit:
		params, body := c.translateFunction(e.Type, exprType.(*types.Signature), e.Body.List)
		if len(c.p.escapingVars) != 0 {
			names := make([]string, 0, len(c.p.escapingVars))
			for obj := range c.p.escapingVars {
				names = append(names, c.p.objectVars[obj])
			}
			list := strings.Join(names, ", ")
			return c.formatExpr("(function(%s) { return function(%s) {\n%s%s}; })(%s)", list, strings.Join(params, ", "), string(body), strings.Repeat("\t", c.p.indentation), list)
		}
		return c.formatExpr("(function(%s) {\n%s%s})", strings.Join(params, ", "), string(body), strings.Repeat("\t", c.p.indentation))

	case *ast.UnaryExpr:
		switch e.Op {
		case token.AND:
			switch c.p.info.Types[e.X].Type.Underlying().(type) {
			case *types.Struct, *types.Array:
				return c.translateExpr(e.X)
			}

			switch x := e.X.(type) {
			case *ast.CompositeLit:
				return c.formatExpr("$newDataPointer(%e, %s)", x, c.typeName(c.p.info.Types[e].Type))
			case *ast.Ident:
				if obj := c.p.info.Uses[x]; c.p.escapingVars[obj] {
					return c.formatExpr("new %s(function() { return this.$target[0]; }, function($v) { this.$target[0] = $v; }, %s)", c.typeName(exprType), c.p.objectVars[obj])
				}
				return c.formatExpr("new %s(function() { return %e; }, function($v) { %s })", c.typeName(exprType), x, c.translateAssign(x, "$v"))
			case *ast.SelectorExpr:
				xId := c.newIdent(c.newVariable("_x"), c.p.info.Types[x.X].Type)
				newSel := &ast.SelectorExpr{X: xId, Sel: x.Sel}
				c.p.info.Selections[newSel] = c.p.info.Selections[x]
				return c.formatExpr("(%e = %e, new %s(function() { return %e; }, function($v) { %s }))", xId, x.X, c.typeName(exprType), newSel, c.translateAssign(newSel, "$v"))
			case *ast.IndexExpr:
				xId := c.newIdent(c.newVariable("_x"), c.p.info.Types[x.X].Type)
				newIndex := &ast.IndexExpr{X: xId, Index: x.Index}
				return c.formatExpr("(%e = %e, new %s(function() { return %e; }, function($v) { %s }))", xId, x.X, c.typeName(exprType), newIndex, c.translateAssign(newIndex, "$v"))
			default:
				panic(fmt.Sprintf("Unhandled: %T\n", x))
			}

		case token.ARROW:
			return c.formatExpr("undefined")
		}

		t := c.p.info.Types[e.X].Type
		basic := t.Underlying().(*types.Basic)
		switch e.Op {
		case token.ADD:
			return c.translateExpr(e.X)
		case token.SUB:
			switch {
			case is64Bit(basic):
				return c.formatExpr("new %1s(-%2h, -%2l)", c.typeName(t), e.X)
			case basic.Info()&types.IsComplex != 0:
				return c.formatExpr("new %1s(-%2r, -%2i)", c.typeName(t), e.X)
			case basic.Info()&types.IsUnsigned != 0:
				return c.fixNumber(c.formatExpr("-%e", e.X), basic)
			default:
				return c.formatExpr("-%e", e.X)
			}
		case token.XOR:
			if is64Bit(basic) {
				return c.formatExpr("new %1s(~%2h, ~%2l >>> 0)", c.typeName(t), e.X)
			}
			return c.fixNumber(c.formatExpr("~%e", e.X), basic)
		case token.NOT:
			x := c.translateExpr(e.X)
			if x.String() == "true" {
				return c.formatExpr("false")
			}
			if x.String() == "false" {
				return c.formatExpr("true")
			}
			return c.formatExpr("!%s", x)
		default:
			panic(e.Op)
		}

	case *ast.BinaryExpr:
		if e.Op == token.NEQ {
			return c.formatExpr("!(%s)", c.translateExpr(&ast.BinaryExpr{
				X:  e.X,
				Op: token.EQL,
				Y:  e.Y,
			}))
		}

		t := c.p.info.Types[e.X].Type
		t2 := c.p.info.Types[e.Y].Type
		_, isInterface := t2.Underlying().(*types.Interface)
		if isInterface {
			t = t2
		}

		if basic, isBasic := t.Underlying().(*types.Basic); isBasic && basic.Info()&types.IsNumeric != 0 {
			if is64Bit(basic) {
				switch e.Op {
				case token.MUL:
					return c.formatExpr("$mul64(%e, %e)", e.X, e.Y)
				case token.QUO:
					return c.formatExpr("$div64(%e, %e, false)", e.X, e.Y)
				case token.REM:
					return c.formatExpr("$div64(%e, %e, true)", e.X, e.Y)
				case token.SHL:
					return c.formatExpr("$shiftLeft64(%e, %f)", e.X, e.Y)
				case token.SHR:
					return c.formatExpr("$shiftRight%s(%e, %f)", toJavaScriptType(basic), e.X, e.Y)
				case token.EQL:
					return c.formatExpr("(%1h === %2h && %1l === %2l)", e.X, e.Y)
				case token.LSS:
					return c.formatExpr("(%1h < %2h || (%1h === %2h && %1l < %2l))", e.X, e.Y)
				case token.LEQ:
					return c.formatExpr("(%1h < %2h || (%1h === %2h && %1l <= %2l))", e.X, e.Y)
				case token.GTR:
					return c.formatExpr("(%1h > %2h || (%1h === %2h && %1l > %2l))", e.X, e.Y)
				case token.GEQ:
					return c.formatExpr("(%1h > %2h || (%1h === %2h && %1l >= %2l))", e.X, e.Y)
				case token.ADD, token.SUB:
					return c.formatExpr("new %3s(%1h %4t %2h, %1l %4t %2l)", e.X, e.Y, c.typeName(t), e.Op)
				case token.AND, token.OR, token.XOR:
					return c.formatExpr("new %3s(%1h %4t %2h, (%1l %4t %2l) >>> 0)", e.X, e.Y, c.typeName(t), e.Op)
				case token.AND_NOT:
					return c.formatExpr("new %3s(%1h &~ %2h, (%1l &~ %2l) >>> 0)", e.X, e.Y, c.typeName(t))
				default:
					panic(e.Op)
				}
			}

			if basic.Info()&types.IsComplex != 0 {
				switch e.Op {
				case token.EQL:
					if basic.Kind() == types.Complex64 {
						return c.formatExpr("($float32IsEqual(%1r, %2r) && $float32IsEqual(%1i, %2i))", e.X, e.Y)
					}
					return c.formatExpr("(%1r === %2r && %1i === %2i)", e.X, e.Y)
				case token.ADD, token.SUB:
					return c.formatExpr("new %3s(%1r %4t %2r, %1i %4t %2i)", e.X, e.Y, c.typeName(t), e.Op)
				case token.MUL:
					return c.formatExpr("new %3s(%1r * %2r - %1i * %2i, %1r * %2i + %1i * %2r)", e.X, e.Y, c.typeName(t))
				case token.QUO:
					return c.formatExpr("$divComplex(%e, %e)", e.X, e.Y)
				default:
					panic(e.Op)
				}
			}

			switch e.Op {
			case token.EQL:
				if basic.Kind() == types.Float32 {
					return c.formatParenExpr("$float32IsEqual(%e, %e)", e.X, e.Y)
				}
				return c.formatParenExpr("%e === %e", e.X, e.Y)
			case token.LSS, token.LEQ, token.GTR, token.GEQ:
				return c.formatExpr("%e %t %e", e.X, e.Op, e.Y)
			case token.ADD, token.SUB:
				if basic.Info()&types.IsInteger != 0 {
					return c.fixNumber(c.formatExpr("%e %t %e", e.X, e.Op, e.Y), basic)
				}
				return c.formatExpr("%e %t %e", e.X, e.Op, e.Y)
			case token.MUL:
				switch basic.Kind() {
				case types.Int32, types.Int:
					return c.formatParenExpr("(((%1e >>> 16 << 16) * %2e >> 0) + (%1e << 16 >>> 16) * %2e) >> 0", e.X, e.Y)
				case types.Uint32, types.Uint, types.Uintptr:
					return c.formatParenExpr("(((%1e >>> 16 << 16) * %2e >>> 0) + (%1e << 16 >>> 16) * %2e) >>> 0", e.X, e.Y)
				case types.Float32, types.Float64:
					return c.formatExpr("%e * %e", e.X, e.Y)
				default:
					return c.fixNumber(c.formatExpr("%e * %e", e.X, e.Y), basic)
				}
			case token.QUO:
				if basic.Info()&types.IsInteger != 0 {
					// cut off decimals
					shift := ">>"
					if basic.Info()&types.IsUnsigned != 0 {
						shift = ">>>"
					}
					return c.formatExpr(`(%1s = %2e / %3e, (%1s === %1s && %1s !== 1/0 && %1s !== -1/0) ? %1s %4s 0 : $throwRuntimeError("integer divide by zero"))`, c.newVariable("_q"), e.X, e.Y, shift)
				}
				return c.formatExpr("%e / %e", e.X, e.Y)
			case token.REM:
				return c.formatExpr(`(%1s = %2e %% %3e, %1s === %1s ? %1s : $throwRuntimeError("integer divide by zero"))`, c.newVariable("_r"), e.X, e.Y)
			case token.SHL, token.SHR:
				op := e.Op.String()
				if e.Op == token.SHR && basic.Info()&types.IsUnsigned != 0 {
					op = ">>>"
				}
				if c.p.info.Types[e.Y].Value != nil {
					return c.fixNumber(c.formatExpr("%e %s %e", e.X, op, e.Y), basic)
				}
				if e.Op == token.SHR && basic.Info()&types.IsUnsigned == 0 {
					return c.fixNumber(c.formatParenExpr("%e >> $min(%e, 31)", e.X, e.Y), basic)
				}
				y := c.newVariable("y")
				return c.fixNumber(c.formatExpr("(%s = %s, %s < 32 ? (%e %s %s) : 0)", y, c.translateImplicitConversion(e.Y, types.Typ[types.Uint]), y, e.X, op, y), basic)
			case token.AND, token.OR:
				if basic.Info()&types.IsUnsigned != 0 {
					return c.formatParenExpr("(%e %t %e) >>> 0", e.X, e.Op, e.Y)
				}
				return c.formatParenExpr("%e %t %e", e.X, e.Op, e.Y)
			case token.AND_NOT:
				return c.formatParenExpr("%e & ~%e", e.X, e.Y)
			case token.XOR:
				return c.fixNumber(c.formatParenExpr("%e ^ %e", e.X, e.Y), basic)
			default:
				panic(e.Op)
			}
		}

		switch e.Op {
		case token.ADD, token.LSS, token.LEQ, token.GTR, token.GEQ:
			return c.formatExpr("%e %t %e", e.X, e.Op, e.Y)
		case token.LAND:
			x := c.translateExpr(e.X)
			y := c.translateExpr(e.Y)
			if x.String() == "false" {
				return c.formatExpr("false")
			}
			return c.formatExpr("%s && %s", x, y)
		case token.LOR:
			x := c.translateExpr(e.X)
			y := c.translateExpr(e.Y)
			if x.String() == "true" {
				return c.formatExpr("true")
			}
			return c.formatExpr("%s || %s", x, y)
		case token.EQL:
			switch u := t.Underlying().(type) {
			case *types.Struct:
				x := c.newVariable("x")
				y := c.newVariable("y")
				var conds []string
				for i := 0; i < u.NumFields(); i++ {
					field := u.Field(i)
					if field.Name() == "_" {
						continue
					}
					conds = append(conds, c.translateExpr(&ast.BinaryExpr{
						X:  c.newIdent(x+"."+fieldName(u, i), field.Type()),
						Op: token.EQL,
						Y:  c.newIdent(y+"."+fieldName(u, i), field.Type()),
					}).String())
				}
				if len(conds) == 0 {
					conds = []string{"true"}
				}
				return c.formatExpr("(%s = %e, %s = %e, %s)", x, e.X, y, e.Y, strings.Join(conds, " && "))
			case *types.Interface:
				if isJsObject(t) {
					return c.formatExpr("%s === %s", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
				}
				return c.formatExpr("$interfaceIsEqual(%s, %s)", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
			case *types.Array:
				return c.formatExpr("$arrayIsEqual(%e, %e)", e.X, e.Y)
			case *types.Pointer:
				xUnary, xIsUnary := e.X.(*ast.UnaryExpr)
				yUnary, yIsUnary := e.Y.(*ast.UnaryExpr)
				if xIsUnary && xUnary.Op == token.AND && yIsUnary && yUnary.Op == token.AND {
					xIndex, xIsIndex := xUnary.X.(*ast.IndexExpr)
					yIndex, yIsIndex := yUnary.X.(*ast.IndexExpr)
					if xIsIndex && yIsIndex {
						return c.formatExpr("$sliceIsEqual(%e, %f, %e, %f)", xIndex.X, xIndex.Index, yIndex.X, yIndex.Index)
					}
				}
				switch u.Elem().Underlying().(type) {
				case *types.Struct, *types.Interface:
					return c.formatExpr("%s === %s", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
				case *types.Array:
					return c.formatExpr("$arrayIsEqual(%s, %s)", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
				default:
					return c.formatExpr("$pointerIsEqual(%s, %s)", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
				}
			default:
				return c.formatExpr("%s === %s", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
			}
		default:
			panic(e.Op)
		}

	case *ast.ParenExpr:
		x := c.translateExpr(e.X)
		if x.String() == "true" || x.String() == "false" {
			return x
		}
		return c.formatParenExpr("%s", x)

	case *ast.IndexExpr:
		switch t := c.p.info.Types[e.X].Type.Underlying().(type) {
		case *types.Array, *types.Pointer:
			return c.formatExpr("%e[%f]", e.X, e.Index)
		case *types.Slice:
			return c.formatExpr(`((%2f < 0 || %2f >= %1e.length) ? $throwRuntimeError("index out of range") : %1e.array[%1e.offset + %2f])`, e.X, e.Index)
		case *types.Map:
			key := c.makeKey(e.Index, t.Key())
			if _, isTuple := exprType.(*types.Tuple); isTuple {
				return c.formatExpr(`(%1s = %2e[%3s], %1s !== undefined ? [%1s.v, true] : [%4s, false])`, c.newVariable("_entry"), e.X, key, c.zeroValue(t.Elem()))
			}
			return c.formatExpr(`(%1s = %2e[%3s], %1s !== undefined ? %1s.v : %4s)`, c.newVariable("_entry"), e.X, key, c.zeroValue(t.Elem()))
		case *types.Basic:
			return c.formatExpr("%e.charCodeAt(%f)", e.X, e.Index)
		default:
			panic(fmt.Sprintf("Unhandled IndexExpr: %T\n", t))
		}

	case *ast.SliceExpr:
		if b, isBasic := c.p.info.Types[e.X].Type.Underlying().(*types.Basic); isBasic && b.Info()&types.IsString != 0 {
			switch {
			case e.Low == nil && e.High == nil:
				return c.translateExpr(e.X)
			case e.Low == nil:
				return c.formatExpr("%e.substring(0, %f)", e.X, e.High)
			case e.High == nil:
				return c.formatExpr("%e.substring(%f)", e.X, e.Low)
			default:
				return c.formatExpr("%e.substring(%f, %f)", e.X, e.Low, e.High)
			}
		}
		slice := c.translateConversionToSlice(e.X, exprType)
		switch {
		case e.Low == nil && e.High == nil:
			return c.formatExpr("%s", slice)
		case e.Low == nil:
			if e.Max != nil {
				return c.formatExpr("$subslice(%s, 0, %f, %f)", slice, e.High, e.Max)
			}
			return c.formatExpr("$subslice(%s, 0, %f)", slice, e.High)
		case e.High == nil:
			return c.formatExpr("$subslice(%s, %f)", slice, e.Low)
		default:
			if e.Max != nil {
				return c.formatExpr("$subslice(%s, %f, %f, %f)", slice, e.Low, e.High, e.Max)
			}
			return c.formatExpr("$subslice(%s, %f, %f)", slice, e.Low, e.High)
		}

	case *ast.SelectorExpr:
		sel := c.p.info.Selections[e]
		parameterName := func(v *types.Var) string {
			if v.Anonymous() || v.Name() == "" {
				return c.newVariable("param")
			}
			return c.newVariable(v.Name())
		}
		makeParametersList := func() []string {
			params := sel.Obj().Type().(*types.Signature).Params()
			names := make([]string, params.Len())
			for i := 0; i < params.Len(); i++ {
				names[i] = parameterName(params.At(i))
			}
			return names
		}

		switch sel.Kind() {
		case types.FieldVal:
			fields, jsTag := c.translateSelection(sel)
			if jsTag != "" {
				if _, ok := sel.Type().(*types.Signature); ok {
					return c.formatExpr("$internalize(%1e.%2s.%3s, %4s, %1e.%2s)", e.X, strings.Join(fields, "."), jsTag, c.typeName(sel.Type()))
				}
				return c.internalize(c.formatExpr("%e.%s.%s", e.X, strings.Join(fields, "."), jsTag), sel.Type())
			}
			return c.formatExpr("%e.%s", e.X, strings.Join(fields, "."))
		case types.MethodVal:
			if !sel.Obj().Exported() {
				c.p.dependencies[sel.Obj()] = true
			}
			parameters := makeParametersList()
			target := c.translateExpr(e.X)
			if isWrapped(sel.Recv()) {
				target = c.formatParenExpr("new %s(%s)", c.typeName(sel.Recv()), target)
			}
			recv := c.newVariable("_recv")
			return c.formatExpr("(%s = %s, function(%s) { return %s.%s(%s); })", recv, target, strings.Join(parameters, ", "), recv, e.Sel.Name, strings.Join(parameters, ", "))
		case types.MethodExpr:
			if !sel.Obj().Exported() {
				c.p.dependencies[sel.Obj()] = true
			}
			recv := "recv"
			if isWrapped(sel.Recv()) {
				recv = fmt.Sprintf("(new %s(recv))", c.typeName(sel.Recv()))
			}
			parameters := makeParametersList()
			return c.formatExpr("(function(%s) { return %s.%s(%s); })", strings.Join(append([]string{"recv"}, parameters...), ", "), recv, sel.Obj().(*types.Func).Name(), strings.Join(parameters, ", "))
		case types.PackageObj:
			if isJsPackage(sel.Obj().Pkg()) {
				switch sel.Obj().Name() {
				case "Global":
					return c.formatExpr("$global")
				case "This":
					if c.flattened {
						return c.formatExpr("$this")
					}
					return c.formatExpr("this")
				case "Arguments":
					args := "arguments"
					if c.flattened {
						args = "$args"
					}

					return c.formatExpr(`new ($sliceType(%s.Object))($global.Array.prototype.slice.call(%s))`, c.p.pkgVars["github.com/gopherjs/gopherjs/js"], args)
				default:
					panic("Invalid js package object: " + sel.Obj().Name())
				}
			}
			return c.formatExpr("%s", c.objectName(sel.Obj()))
		}
		panic("")

	case *ast.CallExpr:
		plainFun := e.Fun
		for {
			if p, isParen := plainFun.(*ast.ParenExpr); isParen {
				plainFun = p.X
				continue
			}
			break
		}

		var isType func(ast.Expr) bool
		isType = func(expr ast.Expr) bool {
			switch e := expr.(type) {
			case *ast.ArrayType, *ast.ChanType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.StructType:
				return true
			case *ast.StarExpr:
				return isType(e.X)
			case *ast.Ident:
				_, ok := c.p.info.Uses[e].(*types.TypeName)
				return ok
			case *ast.SelectorExpr:
				_, ok := c.p.info.Uses[e.Sel].(*types.TypeName)
				return ok
			case *ast.ParenExpr:
				return isType(e.X)
			default:
				return false
			}
		}

		if isType(plainFun) {
			return c.formatExpr("%s", c.translateConversion(e.Args[0], c.p.info.Types[plainFun].Type))
		}

		var fun *expression
		switch f := plainFun.(type) {
		case *ast.Ident:
			if o, ok := c.p.info.Uses[f].(*types.Builtin); ok {
				switch o.Name() {
				case "new":
					t := c.p.info.Types[e].Type.(*types.Pointer)
					if c.p.pkg.Path() == "syscall" && types.Identical(t.Elem().Underlying(), types.Typ[types.Uintptr]) {
						return c.formatExpr("new Uint8Array(8)")
					}
					switch t.Elem().Underlying().(type) {
					case *types.Struct, *types.Array:
						return c.formatExpr("%s", c.zeroValue(t.Elem()))
					default:
						return c.formatExpr("$newDataPointer(%s, %s)", c.zeroValue(t.Elem()), c.typeName(t))
					}
				case "make":
					switch argType := c.p.info.Types[e.Args[0]].Type.Underlying().(type) {
					case *types.Slice:
						t := c.typeName(c.p.info.Types[e.Args[0]].Type)
						if len(e.Args) == 3 {
							return c.formatExpr("%s.make(%f, %f, function() { return %s; })", t, e.Args[1], e.Args[2], c.zeroValue(argType.Elem()))
						}
						return c.formatExpr("%s.make(%f, 0, function() { return %s; })", t, e.Args[1], c.zeroValue(argType.Elem()))
					case *types.Map:
						return c.formatExpr("new $Map()")
					case *types.Chan:
						return c.formatExpr("new %s()", c.typeName(c.p.info.Types[e.Args[0]].Type))
					default:
						panic(fmt.Sprintf("Unhandled make type: %T\n", argType))
					}
				case "len":
					arg := c.translateExpr(e.Args[0])
					switch argType := c.p.info.Types[e.Args[0]].Type.Underlying().(type) {
					case *types.Basic, *types.Slice:
						return c.formatExpr("%s.length", arg)
					case *types.Pointer:
						return c.formatExpr("(%s, %d)", arg, argType.Elem().(*types.Array).Len())
					case *types.Map:
						return c.formatExpr("$keys(%s).length", arg)
					case *types.Chan:
						return c.formatExpr("0")
					// length of array is constant
					default:
						panic(fmt.Sprintf("Unhandled len type: %T\n", argType))
					}
				case "cap":
					arg := c.translateExpr(e.Args[0])
					switch argType := c.p.info.Types[e.Args[0]].Type.Underlying().(type) {
					case *types.Slice:
						return c.formatExpr("%s.capacity", arg)
					case *types.Pointer:
						return c.formatExpr("(%s, %d)", arg, argType.Elem().(*types.Array).Len())
					case *types.Chan:
						return c.formatExpr("0")
					// capacity of array is constant
					default:
						panic(fmt.Sprintf("Unhandled cap type: %T\n", argType))
					}
				case "panic":
					return c.formatExpr("throw $panic(%s)", c.translateImplicitConversion(e.Args[0], types.NewInterface(nil, nil)))
				case "append":
					if len(e.Args) == 1 {
						return c.translateExpr(e.Args[0])
					}
					if e.Ellipsis.IsValid() {
						return c.formatExpr("$appendSlice(%e, %s)", e.Args[0], c.translateConversionToSlice(e.Args[1], exprType))
					}
					sliceType := exprType.Underlying().(*types.Slice)
					return c.formatExpr("$append(%e, %s)", e.Args[0], strings.Join(c.translateExprSlice(e.Args[1:], sliceType.Elem()), ", "))
				case "delete":
					return c.formatExpr(`delete %e[%s]`, e.Args[0], c.makeKey(e.Args[1], c.p.info.Types[e.Args[0]].Type.Underlying().(*types.Map).Key()))
				case "copy":
					if basic, isBasic := c.p.info.Types[e.Args[1]].Type.Underlying().(*types.Basic); isBasic && basic.Info()&types.IsString != 0 {
						return c.formatExpr("$copyString(%e, %e)", e.Args[0], e.Args[1])
					}
					return c.formatExpr("$copySlice(%e, %e)", e.Args[0], e.Args[1])
				case "print", "println":
					return c.formatExpr("console.log(%s)", strings.Join(c.translateExprSlice(e.Args, nil), ", "))
				case "complex":
					return c.formatExpr("new %s(%e, %e)", c.typeName(c.p.info.Types[e].Type), e.Args[0], e.Args[1])
				case "real":
					return c.formatExpr("%e.real", e.Args[0])
				case "imag":
					return c.formatExpr("%e.imag", e.Args[0])
				case "recover":
					return c.formatExpr("$recover()")
				case "close":
					return c.formatExpr(`$notSupported("close")`)
				default:
					panic(fmt.Sprintf("Unhandled builtin: %s\n", o.Name()))
				}
			}
			fun = c.translateExpr(plainFun)

		case *ast.SelectorExpr:
			sel := c.p.info.Selections[f]
			o := sel.Obj()

			externalizeExpr := func(e ast.Expr) string {
				t := c.p.info.Types[e].Type
				if types.Identical(t, types.Typ[types.UntypedNil]) {
					return "null"
				}
				return c.externalize(c.translateExpr(e).String(), t)
			}
			externalizeArgs := func(args []ast.Expr) string {
				s := make([]string, len(args))
				for i, arg := range args {
					s[i] = externalizeExpr(arg)
				}
				return strings.Join(s, ", ")
			}

			switch sel.Kind() {
			case types.MethodVal:
				if !sel.Obj().Exported() {
					c.p.dependencies[sel.Obj()] = true
				}

				methodName := o.Name()
				if reservedKeywords[methodName] {
					methodName += "$"
				}

				fun = c.translateExpr(f.X)
				t := sel.Recv()
				for _, index := range sel.Index()[:len(sel.Index())-1] {
					if ptr, isPtr := t.(*types.Pointer); isPtr {
						t = ptr.Elem()
					}
					s := t.Underlying().(*types.Struct)
					fun = c.formatExpr("%s.%s", fun, fieldName(s, index))
					t = s.Field(index).Type()
				}

				if isJsPackage(o.Pkg()) {
					globalRef := func(id string) string {
						if fun.String() == "$global" && id[0] == '$' {
							return id
						}
						return fun.String() + "." + id
					}
					switch o.Name() {
					case "Get":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							return c.formatExpr("%s", globalRef(id))
						}
						return c.formatExpr("%s[$externalize(%e, $String)]", fun, e.Args[0])
					case "Set":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							return c.formatExpr("%s = %s", globalRef(id), externalizeExpr(e.Args[1]))
						}
						return c.formatExpr("%s[$externalize(%e, $String)] = %s", fun, e.Args[0], externalizeExpr(e.Args[1]))
					case "Delete":
						return c.formatExpr("delete %s[$externalize(%e, $String)]", fun, e.Args[0])
					case "Length":
						return c.formatExpr("$parseInt(%s.length)", fun)
					case "Index":
						return c.formatExpr("%s[%e]", fun, e.Args[0])
					case "SetIndex":
						return c.formatExpr("%s[%e] = %s", fun, e.Args[0], externalizeExpr(e.Args[1]))
					case "Call":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							if e.Ellipsis.IsValid() {
								objVar := c.newVariable("obj")
								return c.formatExpr("(%s = %s, %s.%s.apply(%s, %s))", objVar, fun, objVar, id, objVar, externalizeExpr(e.Args[1]))
							}
							return c.formatExpr("%s(%s)", globalRef(id), externalizeArgs(e.Args[1:]))
						}
						if e.Ellipsis.IsValid() {
							objVar := c.newVariable("obj")
							return c.formatExpr("(%s = %s, %s[$externalize(%e, $String)].apply(%s, %s))", objVar, fun, objVar, e.Args[0], objVar, externalizeExpr(e.Args[1]))
						}
						return c.formatExpr("%s[$externalize(%e, $String)](%s)", fun, e.Args[0], externalizeArgs(e.Args[1:]))
					case "Invoke":
						if e.Ellipsis.IsValid() {
							return c.formatExpr("%s.apply(undefined, %s)", fun, externalizeExpr(e.Args[0]))
						}
						return c.formatExpr("%s(%s)", fun, externalizeArgs(e.Args))
					case "New":
						if e.Ellipsis.IsValid() {
							return c.formatExpr("new ($global.Function.prototype.bind.apply(%s, [undefined].concat(%s)))", fun, externalizeExpr(e.Args[0]))
						}
						return c.formatExpr("new (%s)(%s)", fun, externalizeArgs(e.Args))
					case "Bool":
						return c.internalize(fun, types.Typ[types.Bool])
					case "Str":
						return c.internalize(fun, types.Typ[types.String])
					case "Int":
						return c.internalize(fun, types.Typ[types.Int])
					case "Int64":
						return c.internalize(fun, types.Typ[types.Int64])
					case "Uint64":
						return c.internalize(fun, types.Typ[types.Uint64])
					case "Float":
						return c.internalize(fun, types.Typ[types.Float64])
					case "Interface":
						return c.internalize(fun, types.NewInterface(nil, nil))
					case "Unsafe":
						return fun
					case "IsUndefined":
						return c.formatParenExpr("%s === undefined", fun)
					case "IsNull":
						return c.formatParenExpr("%s === null", fun)
					default:
						panic("Invalid js package object: " + o.Name())
					}
				}

				methodsRecvType := o.Type().(*types.Signature).Recv().Type()
				_, pointerExpected := methodsRecvType.(*types.Pointer)
				_, isPointer := t.Underlying().(*types.Pointer)
				_, isStruct := t.Underlying().(*types.Struct)
				_, isArray := t.Underlying().(*types.Array)
				if pointerExpected && !isPointer && !isStruct && !isArray {
					vVar := c.newVariable("v")
					fun = c.formatExpr("(new %s(function() { return %s; }, function(%s) { %s = %s; })).%s", c.typeName(methodsRecvType), fun, vVar, fun, vVar, methodName)
					break
				}

				if isWrapped(t) {
					fun = c.formatExpr("(new %s(%s)).%s", c.typeName(t), fun, methodName)
					break
				}
				fun = c.formatExpr("%s.%s", fun, methodName)

			case types.PackageObj:
				if isJsPackage(o.Pkg()) && o.Name() == "InternalObject" {
					return c.translateExpr(e.Args[0])
				}
				fun = c.translateExpr(f)

			case types.FieldVal:
				fields, jsTag := c.translateSelection(sel)
				if jsTag != "" {
					sig := sel.Type().(*types.Signature)
					return c.internalize(c.formatExpr("%e.%s.%s(%s)", f.X, strings.Join(fields, "."), jsTag, externalizeArgs(e.Args)), sig.Results().At(0).Type())
				}
				fun = c.formatExpr("%e.%s", f.X, strings.Join(fields, "."))

			case types.MethodExpr:
				fun = c.translateExpr(f)

			default:
				panic("")
			}
		default:
			fun = c.translateExpr(plainFun)
		}

		sig := c.p.info.Types[plainFun].Type.Underlying().(*types.Signature)
		if len(e.Args) == 1 {
			if tuple, isTuple := c.p.info.Types[e.Args[0]].Type.(*types.Tuple); isTuple {
				tupleVar := c.newVariable("_tuple")
				args := make([]ast.Expr, tuple.Len())
				for i := range args {
					args[i] = c.newIdent(c.formatExpr("%s[%d]", tupleVar, i).String(), tuple.At(i).Type())
				}
				return c.formatExpr("(%s = %e, %s(%s))", tupleVar, e.Args[0], fun, strings.Join(c.translateArgs(sig, args, false), ", "))
			}
		}
		return c.formatExpr("%s(%s)", fun, strings.Join(c.translateArgs(sig, e.Args, e.Ellipsis.IsValid()), ", "))

	case *ast.StarExpr:
		if c1, isCall := e.X.(*ast.CallExpr); isCall && len(c1.Args) == 1 {
			if c2, isCall := c1.Args[0].(*ast.CallExpr); isCall && len(c2.Args) == 1 && types.Identical(c.p.info.Types[c2.Fun].Type, types.Typ[types.UnsafePointer]) {
				if unary, isUnary := c2.Args[0].(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
					return c.translateExpr(unary.X) // unsafe conversion
				}
			}
		}
		switch exprType.Underlying().(type) {
		case *types.Struct, *types.Array:
			return c.translateExpr(e.X)
		}
		return c.formatExpr("%e.$get()", e.X)

	case *ast.TypeAssertExpr:
		if e.Type == nil {
			return c.translateExpr(e.X)
		}
		t := c.p.info.Types[e.Type].Type
		check := "%1e !== null && " + c.typeCheck("%1e.constructor", t)
		valueSuffix := ""
		if _, isInterface := t.Underlying().(*types.Interface); !isInterface {
			valueSuffix = ".$val"
		}
		if _, isTuple := exprType.(*types.Tuple); isTuple {
			return c.formatExpr("("+check+" ? [%1e%2s, true] : [%3s, false])", e.X, valueSuffix, c.zeroValue(c.p.info.Types[e.Type].Type))
		}
		return c.formatExpr("("+check+" ? %1e%2s : $typeAssertionFailed(%1e, %3s))", e.X, valueSuffix, c.typeName(t))

	case *ast.Ident:
		if e.Name == "_" {
			panic("Tried to translate underscore identifier.")
		}
		switch o := c.p.info.Uses[e].(type) {
		case *types.PkgName:
			return c.formatExpr("%s", c.p.pkgVars[o.Pkg().Path()])
		case *types.Var, *types.Const:
			return c.formatExpr("%s", c.objectName(o))
		case *types.Func:
			return c.formatExpr("%s", c.objectName(o))
		case *types.TypeName:
			return c.formatExpr("%s", c.typeName(o.Type()))
		case *types.Nil:
			return c.formatExpr("%s", c.zeroValue(c.p.info.Types[e].Type))
		default:
			panic(fmt.Sprintf("Unhandled object: %T\n", o))
		}

	case *This:
		this := "this"
		if c.flattened {
			this = "$this"
		}
		if isWrapped(c.p.info.Types[e].Type) {
			this += ".$val"
		}
		return c.formatExpr(this)

	case nil:
		return c.formatExpr("")

	default:
		panic(fmt.Sprintf("Unhandled expression: %T\n", e))

	}
}

func (c *funcContext) identifierConstant(expr ast.Expr) (string, bool) {
	val := c.p.info.Types[expr].Value
	if val == nil {
		return "", false
	}
	s := exact.StringVal(val)
	if len(s) == 0 {
		return "", false
	}
	for i, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (i > 0 && c >= '0' && c <= '9') || c == '_' || c == '$') {
			return "", false
		}
	}
	return s, true
}

func (c *funcContext) translateExprSlice(exprs []ast.Expr, desiredType types.Type) []string {
	parts := make([]string, len(exprs))
	for i, expr := range exprs {
		parts[i] = c.translateImplicitConversion(expr, desiredType).String()
	}
	return parts
}

func (c *funcContext) translateConversion(expr ast.Expr, desiredType types.Type) *expression {
	exprType := c.p.info.Types[expr].Type
	if types.Identical(exprType, desiredType) {
		return c.translateExpr(expr)
	}

	if c.p.pkg.Path() == "reflect" {
		if call, isCall := expr.(*ast.CallExpr); isCall && types.Identical(c.p.info.Types[call.Fun].Type, types.Typ[types.UnsafePointer]) {
			if ptr, isPtr := desiredType.(*types.Pointer); isPtr {
				if named, isNamed := ptr.Elem().(*types.Named); isNamed {
					return c.formatExpr("%e.%s", call.Args[0], named.Obj().Name()) // unsafe conversion
				}
			}
		}
	}

	switch t := desiredType.Underlying().(type) {
	case *types.Basic:
		switch {
		case t.Info()&types.IsInteger != 0:
			basicExprType := exprType.Underlying().(*types.Basic)
			switch {
			case is64Bit(t):
				if !is64Bit(basicExprType) {
					if basicExprType.Kind() == types.Uintptr { // this might be an Object returned from reflect.Value.Pointer()
						return c.formatExpr("new %1s(0, %2e.constructor === Number ? %2e : 1)", c.typeName(desiredType), expr)
					}
					return c.formatExpr("new %s(0, %e)", c.typeName(desiredType), expr)
				}
				return c.formatExpr("new %1s(%2h, %2l)", c.typeName(desiredType), expr)
			case is64Bit(basicExprType):
				if t.Info()&types.IsUnsigned == 0 && basicExprType.Info()&types.IsUnsigned == 0 {
					return c.fixNumber(c.formatParenExpr("%1l + ((%1h >> 31) * 4294967296)", expr), t)
				}
				return c.fixNumber(c.formatExpr("%s.low", c.translateExpr(expr)), t)
			case basicExprType.Info()&types.IsFloat != 0:
				return c.formatParenExpr("%e >> 0", expr)
			case types.Identical(exprType, types.Typ[types.UnsafePointer]):
				return c.translateExpr(expr)
			default:
				return c.fixNumber(c.translateExpr(expr), t)
			}
		case t.Info()&types.IsFloat != 0:
			if t.Kind() == types.Float64 && exprType.Underlying().(*types.Basic).Kind() == types.Float32 {
				return c.formatExpr("$coerceFloat32(%f)", expr)
			}
			return c.formatExpr("%f", expr)
		case t.Info()&types.IsComplex != 0:
			return c.formatExpr("new %1s(%2r, %2i)", c.typeName(desiredType), expr)
		case t.Info()&types.IsString != 0:
			value := c.translateExpr(expr)
			switch et := exprType.Underlying().(type) {
			case *types.Basic:
				if is64Bit(et) {
					value = c.formatExpr("%s.low", value)
				}
				if et.Info()&types.IsNumeric != 0 {
					return c.formatExpr("$encodeRune(%s)", value)
				}
				return value
			case *types.Slice:
				if types.Identical(et.Elem().Underlying(), types.Typ[types.Rune]) {
					return c.formatExpr("$runesToString(%s)", value)
				}
				return c.formatExpr("$bytesToString(%s)", value)
			default:
				panic(fmt.Sprintf("Unhandled conversion: %v\n", et))
			}
		case t.Kind() == types.UnsafePointer:
			if unary, isUnary := expr.(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
				if indexExpr, isIndexExpr := unary.X.(*ast.IndexExpr); isIndexExpr {
					return c.formatExpr("$sliceToArray(%s)", c.translateConversionToSlice(indexExpr.X, types.NewSlice(types.Typ[types.Uint8])))
				}
				if ident, isIdent := unary.X.(*ast.Ident); isIdent && ident.Name == "_zero" {
					return c.formatExpr("new Uint8Array(0)")
				}
			}
			if ptr, isPtr := c.p.info.Types[expr].Type.(*types.Pointer); c.p.pkg.Path() == "syscall" && isPtr {
				if s, isStruct := ptr.Elem().Underlying().(*types.Struct); isStruct {
					array := c.newVariable("_array")
					target := c.newVariable("_struct")
					c.Printf("%s = new Uint8Array(%d);", array, sizes32.Sizeof(s))
					c.Delayed(func() {
						c.Printf("%s = %s, %s;", target, c.translateExpr(expr), c.loadStruct(array, target, s))
					})
					return c.formatExpr("%s", array)
				}
			}
		}

	case *types.Slice:
		switch et := exprType.Underlying().(type) {
		case *types.Basic:
			if et.Info()&types.IsString != 0 {
				if types.Identical(t.Elem().Underlying(), types.Typ[types.Rune]) {
					return c.formatExpr("new %s($stringToRunes(%e))", c.typeName(desiredType), expr)
				}
				return c.formatExpr("new %s($stringToBytes(%e))", c.typeName(desiredType), expr)
			}
		case *types.Array, *types.Pointer:
			return c.formatExpr("new %s(%e)", c.typeName(desiredType), expr)
		}

	case *types.Pointer:
		if s, isStruct := t.Elem().Underlying().(*types.Struct); isStruct {
			if c.p.pkg.Path() == "syscall" && types.Identical(exprType, types.Typ[types.UnsafePointer]) {
				array := c.newVariable("_array")
				target := c.newVariable("_struct")
				return c.formatExpr("(%s = %e, %s = %s, %s, %s)", array, expr, target, c.zeroValue(t.Elem()), c.loadStruct(array, target, s), target)
			}
			return c.clone(c.translateExpr(expr), t.Elem())
		}

		if !types.Identical(exprType, types.Typ[types.UnsafePointer]) {
			return c.formatExpr("new %1s(%2e.$get, %2e.$set)", c.typeName(desiredType), expr)
		}

	case *types.Interface:
		if types.Identical(exprType, types.Typ[types.UnsafePointer]) {
			return c.translateExpr(expr)
		}
	}

	return c.translateImplicitConversion(expr, desiredType)
}

func (c *funcContext) translateImplicitConversion(expr ast.Expr, desiredType types.Type) *expression {
	if desiredType == nil {
		return c.translateExpr(expr)
	}
	if expr == nil {
		return c.formatExpr("%s", c.zeroValue(desiredType))
	}

	switch desiredType.Underlying().(type) {
	case *types.Struct, *types.Array:
		if _, isComposite := expr.(*ast.CompositeLit); !isComposite {
			return c.clone(c.translateExpr(expr), desiredType)
		}
	}

	exprType := c.p.info.Types[expr].Type
	if types.Identical(exprType, desiredType) {
		return c.translateExpr(expr)
	}

	basicExprType, isBasicExpr := exprType.Underlying().(*types.Basic)
	if isBasicExpr && basicExprType.Kind() == types.UntypedNil {
		return c.formatExpr("%s", c.zeroValue(desiredType))
	}

	switch desiredType.Underlying().(type) {
	case *types.Slice:
		return c.formatExpr("$subslice(new %1s(%2e.array), %2e.offset, %2e.offset + %2e.length)", c.typeName(desiredType), expr)

	case *types.Interface:
		if isWrapped(exprType) {
			return c.formatExpr("new %s(%e)", c.typeName(exprType), expr)
		}
		if _, isStruct := exprType.Underlying().(*types.Struct); isStruct {
			return c.formatExpr("new %1e.constructor.Struct(%1e)", expr)
		}
	}

	return c.translateExpr(expr)
}

func (c *funcContext) translateConversionToSlice(expr ast.Expr, desiredType types.Type) *expression {
	switch c.p.info.Types[expr].Type.Underlying().(type) {
	case *types.Basic:
		return c.formatExpr("new %s($stringToBytes(%e))", c.typeName(desiredType), expr)
	case *types.Array, *types.Pointer:
		return c.formatExpr("new %s(%e)", c.typeName(desiredType), expr)
	}
	return c.translateExpr(expr)
}

func (c *funcContext) clone(src *expression, ty types.Type) *expression {
	switch t := ty.Underlying().(type) {
	case *types.Struct:
		structVar := c.newVariable("_struct")
		fields := make([]string, t.NumFields())
		for i := range fields {
			fields[i] = c.clone(c.formatExpr("%s.%s", structVar, fieldName(t, i)), t.Field(i).Type()).String()
		}
		constructor := structVar + ".constructor"
		if named, isNamed := ty.(*types.Named); isNamed {
			constructor = c.objectName(named.Obj()) + ".Ptr"
		}
		return c.formatExpr("(%s = %s, new %s(%s))", structVar, src, constructor, strings.Join(fields, ", "))
	case *types.Array:
		return c.formatExpr("$mapArray(%s, function(entry) { return %s; })", src, c.clone(c.formatExpr("entry"), t.Elem()))
	default:
		return src
	}
}

func (c *funcContext) loadStruct(array, target string, s *types.Struct) string {
	view := c.newVariable("_view")
	code := fmt.Sprintf("%s = new DataView(%s.buffer, %s.byteOffset)", view, array, array)
	var fields []*types.Var
	var collectFields func(s *types.Struct, path string)
	collectFields = func(s *types.Struct, path string) {
		for i := 0; i < s.NumFields(); i++ {
			field := s.Field(i)
			if fs, isStruct := field.Type().Underlying().(*types.Struct); isStruct {
				collectFields(fs, path+"."+fieldName(s, i))
				continue
			}
			fields = append(fields, types.NewVar(0, nil, path+"."+fieldName(s, i), field.Type()))
		}
	}
	collectFields(s, target)
	offsets := sizes32.Offsetsof(fields)
	for i, field := range fields {
		switch t := field.Type().Underlying().(type) {
		case *types.Basic:
			if t.Info()&types.IsNumeric != 0 {
				if is64Bit(t) {
					code += fmt.Sprintf(", %s = new %s(%s.getUint32(%d, true), %s.getUint32(%d, true))", field.Name(), c.typeName(field.Type()), view, offsets[i]+4, view, offsets[i])
					break
				}
				code += fmt.Sprintf(", %s = %s.get%s(%d, true)", field.Name(), view, toJavaScriptType(t), offsets[i])
			}
		case *types.Array:
			code += fmt.Sprintf(`, %s = new ($nativeArray("%s"))(%s.buffer, $min(%s.byteOffset + %d, %s.buffer.byteLength))`, field.Name(), typeKind(t.Elem()), array, array, offsets[i], array)
		}
	}
	return code
}

func (c *funcContext) typeCheck(of string, to types.Type) string {
	if isJsObject(to) {
		return "true"
	}
	if in, isInterface := to.Underlying().(*types.Interface); isInterface {
		if in.Empty() {
			return "true"
		}
		return fmt.Sprintf("%s.implementedBy.indexOf(%s) !== -1", c.typeName(to), of)
	}
	return of + " === " + c.typeName(to)
}

func (c *funcContext) fixNumber(value *expression, basic *types.Basic) *expression {
	switch basic.Kind() {
	case types.Int8:
		return c.formatParenExpr("%s << 24 >> 24", value)
	case types.Uint8:
		return c.formatParenExpr("%s << 24 >>> 24", value)
	case types.Int16:
		return c.formatParenExpr("%s << 16 >> 16", value)
	case types.Uint16:
		return c.formatParenExpr("%s << 16 >>> 16", value)
	case types.Int32, types.Int:
		return c.formatParenExpr("%s >> 0", value)
	case types.Uint32, types.Uint, types.Uintptr:
		return c.formatParenExpr("%s >>> 0", value)
	default:
		panic(int(basic.Kind()))
	}
}

func (c *funcContext) internalize(s *expression, t types.Type) *expression {
	if isJsObject(t) {
		return s
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Info()&types.IsBoolean != 0:
			return c.formatExpr("!!(%s)", s)
		case u.Info()&types.IsInteger != 0 && !is64Bit(u):
			return c.fixNumber(c.formatExpr("$parseInt(%s)", s), u)
		case u.Info()&types.IsFloat != 0:
			return c.formatExpr("$parseFloat(%s)", s)
		}
	}
	return c.formatExpr("$internalize(%s, %s)", s, c.typeName(t))
}

func (c *funcContext) formatExpr(format string, a ...interface{}) *expression {
	return c.formatExprInternal(format, a, false)
}

func (c *funcContext) formatParenExpr(format string, a ...interface{}) *expression {
	return c.formatExprInternal(format, a, true)
}

func (c *funcContext) formatExprInternal(format string, a []interface{}, parens bool) *expression {
	processFormat := func(f func(uint8, uint8, int)) {
		n := 0
		for i := 0; i < len(format); i++ {
			b := format[i]
			if b == '%' {
				i++
				k := format[i]
				if k >= '0' && k <= '9' {
					n = int(k - '0' - 1)
					i++
					k = format[i]
				}
				f(0, k, n)
				n++
				continue
			}
			f(b, 0, 0)
		}
	}

	counts := make([]int, len(a))
	processFormat(func(b, k uint8, n int) {
		switch k {
		case 'e', 'f', 'h', 'l', 'r', 'i':
			counts[n]++
		}
	})

	out := bytes.NewBuffer(nil)
	vars := make([]string, len(a))
	hasAssignments := false
	for i, e := range a {
		if counts[i] <= 1 {
			continue
		}
		if _, isIdent := e.(*ast.Ident); isIdent {
			continue
		}
		if val := c.p.info.Types[e.(ast.Expr)].Value; val != nil {
			continue
		}
		if !hasAssignments {
			hasAssignments = true
			out.WriteByte('(')
			parens = false
		}
		v := c.newVariable("x")
		out.WriteString(v + " = " + c.translateExpr(e.(ast.Expr)).String() + ", ")
		vars[i] = v
	}

	processFormat(func(b, k uint8, n int) {
		writeExpr := func(suffix string) {
			if vars[n] != "" {
				out.WriteString(vars[n] + suffix)
				return
			}
			out.WriteString(c.translateExpr(a[n].(ast.Expr)).StringWithParens() + suffix)
		}
		switch k {
		case 0:
			out.WriteByte(b)
		case 's':
			if e, ok := a[n].(*expression); ok {
				out.WriteString(e.StringWithParens())
				return
			}
			out.WriteString(a[n].(string))
		case 'd':
			out.WriteString(strconv.Itoa(a[n].(int)))
		case 't':
			out.WriteString(a[n].(token.Token).String())
		case 'e':
			e := a[n].(ast.Expr)
			if val := c.p.info.Types[e].Value; val != nil {
				out.WriteString(c.translateExpr(e).String())
				return
			}
			writeExpr("")
		case 'f':
			e := a[n].(ast.Expr)
			if val := c.p.info.Types[e].Value; val != nil {
				d, _ := exact.Int64Val(val)
				out.WriteString(strconv.FormatInt(d, 10))
				return
			}
			if is64Bit(c.p.info.Types[e].Type.Underlying().(*types.Basic)) {
				out.WriteString("$flatten64(")
				writeExpr("")
				out.WriteString(")")
				return
			}
			writeExpr("")
		case 'h':
			e := a[n].(ast.Expr)
			if val := c.p.info.Types[e].Value; val != nil {
				d, _ := exact.Uint64Val(val)
				if c.p.info.Types[e].Type.Underlying().(*types.Basic).Kind() == types.Int64 {
					out.WriteString(strconv.FormatInt(int64(d)>>32, 10))
					return
				}
				out.WriteString(strconv.FormatUint(d>>32, 10))
				return
			}
			writeExpr(".high")
		case 'l':
			if val := c.p.info.Types[a[n].(ast.Expr)].Value; val != nil {
				d, _ := exact.Uint64Val(val)
				out.WriteString(strconv.FormatUint(d&(1<<32-1), 10))
				return
			}
			writeExpr(".low")
		case 'r':
			if val := c.p.info.Types[a[n].(ast.Expr)].Value; val != nil {
				r, _ := exact.Float64Val(exact.Real(val))
				out.WriteString(strconv.FormatFloat(r, 'g', -1, 64))
				return
			}
			writeExpr(".real")
		case 'i':
			if val := c.p.info.Types[a[n].(ast.Expr)].Value; val != nil {
				i, _ := exact.Float64Val(exact.Imag(val))
				out.WriteString(strconv.FormatFloat(i, 'g', -1, 64))
				return
			}
			writeExpr(".imag")
		case '%':
			out.WriteRune('%')
		default:
			panic(fmt.Sprintf("formatExpr: %%%c%d", k, n))
		}
	})

	if hasAssignments {
		out.WriteByte(')')
	}
	return &expression{str: out.String(), parens: parens}
}
