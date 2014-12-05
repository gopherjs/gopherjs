package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/exact"
	"golang.org/x/tools/go/types"
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

	var obj types.Object
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		obj = c.p.info.Uses[e.Sel]
	case *ast.Ident:
		obj = c.p.info.Defs[e]
		if obj == nil {
			obj = c.p.info.Uses[e]
		}
	}

	if obj != nil && isJsPackage(obj.Pkg()) {
		switch obj.Name() {
		case "Global":
			return c.formatExpr("$global")
		case "This":
			if len(c.flattened) != 0 {
				return c.formatExpr("$this")
			}
			return c.formatExpr("this")
		case "Arguments":
			args := "arguments"
			if len(c.flattened) != 0 {
				args = "$args"
			}
			return c.formatExpr(`new ($sliceType(%s.Object))($global.Array.prototype.slice.call(%s, []))`, c.p.pkgVars["github.com/gopherjs/gopherjs/js"], args)
		case "Module":
			return c.formatExpr("$module")
		}
	}

	switch e := expr.(type) {
	case *ast.CompositeLit:
		if ptrType, isPointer := exprType.(*types.Pointer); isPointer {
			exprType = ptrType.Elem()
		}

		collectIndexedElements := func(elementType types.Type) []string {
			var elements []string
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
				elements[i] = c.translateImplicitConversionWithCloning(element, elementType).String()
				i++
			}
			return elements
		}

		switch t := exprType.Underlying().(type) {
		case *types.Array:
			elements := collectIndexedElements(t.Elem())
			if len(elements) == 0 {
				return c.formatExpr("%s", c.zeroValue(t))
			}
			zero := c.zeroValue(t.Elem())
			for len(elements) < int(t.Len()) {
				elements = append(elements, zero)
			}
			return c.formatExpr(`$toNativeArray(%s, [%s])`, typeKind(t.Elem()), strings.Join(elements, ", "))
		case *types.Slice:
			return c.formatExpr("new %s([%s])", c.typeName(exprType), strings.Join(collectIndexedElements(t.Elem()), ", "))
		case *types.Map:
			mapVar := c.newVariable("_map")
			keyVar := c.newVariable("_key")
			assignments := ""
			for _, element := range e.Elts {
				kve := element.(*ast.KeyValueExpr)
				assignments += c.formatExpr(`%s = %s, %s[%s] = { k: %s, v: %s }, `, keyVar, c.translateImplicitConversionWithCloning(kve.Key, t.Key()), mapVar, c.makeKey(c.newIdent(keyVar, t.Key()), t.Key()), keyVar, c.translateImplicitConversionWithCloning(kve.Value, t.Elem())).String()
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
					elements[i] = c.translateImplicitConversionWithCloning(element, t.Field(i).Type()).String()
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
							elements[j] = c.translateImplicitConversionWithCloning(kve.Value, t.Field(j).Type()).String()
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
		innerContext := c.p.analyzeFunction(exprType.(*types.Signature), e.Body)
		params, body := innerContext.translateFunction(e.Type, e.Body.List, c.allVars)
		if len(c.p.escapingVars) != 0 {
			names := make([]string, 0, len(c.p.escapingVars))
			for obj := range c.p.escapingVars {
				names = append(names, c.p.objectVars[obj])
			}
			sort.Strings(names)
			list := strings.Join(names, ", ")
			return c.formatExpr("(function(%s) { return function(%s) {\n%s%s}; })(%s)", list, strings.Join(params, ", "), string(body), strings.Repeat("\t", c.p.indentation), list)
		}
		return c.formatExpr("(function(%s) {\n%s%s})", strings.Join(params, ", "), string(body), strings.Repeat("\t", c.p.indentation))

	case *ast.UnaryExpr:
		t := c.p.info.Types[e.X].Type
		switch e.Op {
		case token.AND:
			switch t.Underlying().(type) {
			case *types.Struct, *types.Array:
				return c.translateExpr(e.X)
			}

			switch x := removeParens(e.X).(type) {
			case *ast.CompositeLit:
				return c.formatExpr("$newDataPointer(%e, %s)", x, c.typeName(c.p.info.Types[e].Type))
			case *ast.Ident:
				if obj := c.p.info.Uses[x]; c.p.escapingVars[obj] {
					return c.formatExpr("new %s(function() { return this.$target[0]; }, function($v) { this.$target[0] = $v; }, %s)", c.typeName(exprType), c.p.objectVars[obj])
				}
				return c.formatExpr("new %s(function() { return %e; }, function($v) { %s })", c.typeName(exprType), x, c.translateAssign(x, "$v", exprType, false))
			case *ast.SelectorExpr:
				newSel := &ast.SelectorExpr{X: c.newIdent("this.$target", c.p.info.Types[x.X].Type), Sel: x.Sel}
				c.p.info.Selections[newSel] = c.p.info.Selections[x]
				return c.formatExpr("new %s(function() { return %e; }, function($v) { %s }, %e)", c.typeName(exprType), newSel, c.translateAssign(newSel, "$v", exprType, false), x.X)
			case *ast.IndexExpr:
				newIndex := &ast.IndexExpr{X: c.newIdent("this.$target", c.p.info.Types[x.X].Type), Index: x.Index}
				return c.formatExpr("new %s(function() { return %e; }, function($v) { %s }, %e)", c.typeName(exprType), newIndex, c.translateAssign(newIndex, "$v", exprType, false), x.X)
			case *ast.StarExpr:
				return c.translateExpr(x.X)
			default:
				panic(fmt.Sprintf("Unhandled: %T\n", x))
			}

		case token.ARROW:
			call := &ast.CallExpr{
				Fun:  c.newIdent("$recv", types.NewSignature(nil, nil, types.NewTuple(types.NewVar(0, nil, "", t)), types.NewTuple(types.NewVar(0, nil, "", exprType), types.NewVar(0, nil, "", types.Typ[types.Bool])), false)),
				Args: []ast.Expr{e.X},
			}
			c.blocking[call] = true
			if _, isTuple := exprType.(*types.Tuple); isTuple {
				return c.formatExpr("%e", call)
			}
			return c.formatExpr("%e[0]", call)
		}

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
			case *types.Array, *types.Struct:
				return c.formatExpr("$equal(%e, %e, %s)", e.X, e.Y, c.typeName(t))
			case *types.Interface:
				if isJsObject(t) {
					return c.formatExpr("%s === %s", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
				}
				return c.formatExpr("$interfaceIsEqual(%s, %s)", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
			case *types.Pointer:
				switch u.Elem().Underlying().(type) {
				case *types.Struct, *types.Interface:
					return c.formatExpr("%s === %s", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
				case *types.Array:
					return c.formatExpr("$equal(%s, %s, %s)", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t), c.typeName(u.Elem()))
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
			pattern := "%1e[%2f]"
			if c.p.info.Types[e.Index].Value == nil { // add range check if not constant
				pattern = `((%2f < 0 || %2f >= %1e.length) ? $throwRuntimeError("index out of range") : ` + pattern + `)`
			}
			if _, ok := t.(*types.Pointer); ok { // check pointer for nix (attribute getter causes a panic)
				pattern = `(%1e.nilCheck, ` + pattern + `)`
			}
			return c.formatExpr(pattern, e.X, e.Index)
		case *types.Slice:
			return c.formatExpr(`((%2f < 0 || %2f >= %1e.$length) ? $throwRuntimeError("index out of range") : %1e.$array[%1e.$offset + %2f])`, e.X, e.Index)
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
		sel, ok := c.p.info.Selections[e]
		if !ok {
			// qualified identifier
			return c.formatExpr("%s", c.objectName(obj))
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
			recv := c.makeReceiver(e.X, sel)
			return c.formatExpr(`$methodVal(%s, "%s")`, recv, sel.Obj().(*types.Func).Name())
		case types.MethodExpr:
			if !sel.Obj().Exported() {
				c.p.dependencies[sel.Obj()] = true
			}
			return c.formatExpr("$methodExpr(%s.prototype.%s)", c.typeName(sel.Recv()), sel.Obj().(*types.Func).Name())
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
				return c.translateBuiltin(o.Name(), e.Args, e.Ellipsis.IsValid(), exprType)
			}
			fun = c.translateExpr(plainFun)

		case *ast.SelectorExpr:
			sel, ok := c.p.info.Selections[f]
			if !ok {
				// qualified identifier
				obj := c.p.info.Uses[f.Sel]
				if isJsPackage(obj.Pkg()) {
					switch obj.Name() {
					case "Debugger":
						return c.formatExpr("debugger")
					case "InternalObject":
						return c.translateExpr(e.Args[0])
					}
				}
				fun = c.translateExpr(f)
				break
			}

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
				recv := c.makeReceiver(f.X, sel)

				if isJsPackage(sel.Obj().Pkg()) {
					globalRef := func(id string) string {
						if recv.String() == "$global" && id[0] == '$' {
							return id
						}
						return recv.String() + "." + id
					}
					switch sel.Obj().Name() {
					case "Get":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							return c.formatExpr("%s", globalRef(id))
						}
						return c.formatExpr("%s[$externalize(%e, $String)]", recv, e.Args[0])
					case "Set":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							return c.formatExpr("%s = %s", globalRef(id), externalizeExpr(e.Args[1]))
						}
						return c.formatExpr("%s[$externalize(%e, $String)] = %s", recv, e.Args[0], externalizeExpr(e.Args[1]))
					case "Delete":
						return c.formatExpr("delete %s[$externalize(%e, $String)]", recv, e.Args[0])
					case "Length":
						return c.formatExpr("$parseInt(%s.length)", recv)
					case "Index":
						return c.formatExpr("%s[%e]", recv, e.Args[0])
					case "SetIndex":
						return c.formatExpr("%s[%e] = %s", recv, e.Args[0], externalizeExpr(e.Args[1]))
					case "Call":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							if e.Ellipsis.IsValid() {
								objVar := c.newVariable("obj")
								return c.formatExpr("(%s = %s, %s.%s.apply(%s, %s))", objVar, recv, objVar, id, objVar, externalizeExpr(e.Args[1]))
							}
							return c.formatExpr("%s(%s)", globalRef(id), externalizeArgs(e.Args[1:]))
						}
						if e.Ellipsis.IsValid() {
							objVar := c.newVariable("obj")
							return c.formatExpr("(%s = %s, %s[$externalize(%e, $String)].apply(%s, %s))", objVar, recv, objVar, e.Args[0], objVar, externalizeExpr(e.Args[1]))
						}
						return c.formatExpr("%s[$externalize(%e, $String)](%s)", recv, e.Args[0], externalizeArgs(e.Args[1:]))
					case "Invoke":
						if e.Ellipsis.IsValid() {
							return c.formatExpr("%s.apply(undefined, %s)", recv, externalizeExpr(e.Args[0]))
						}
						return c.formatExpr("%s(%s)", recv, externalizeArgs(e.Args))
					case "New":
						if e.Ellipsis.IsValid() {
							return c.formatExpr("new ($global.Function.prototype.bind.apply(%s, [undefined].concat(%s)))", recv, externalizeExpr(e.Args[0]))
						}
						return c.formatExpr("new (%s)(%s)", recv, externalizeArgs(e.Args))
					case "Bool":
						return c.internalize(recv, types.Typ[types.Bool])
					case "Str":
						return c.internalize(recv, types.Typ[types.String])
					case "Int":
						return c.internalize(recv, types.Typ[types.Int])
					case "Int64":
						return c.internalize(recv, types.Typ[types.Int64])
					case "Uint64":
						return c.internalize(recv, types.Typ[types.Uint64])
					case "Float":
						return c.internalize(recv, types.Typ[types.Float64])
					case "Interface":
						return c.internalize(recv, types.NewInterface(nil, nil))
					case "Unsafe":
						return recv
					case "IsUndefined":
						return c.formatParenExpr("%s === undefined", recv)
					case "IsNull":
						return c.formatParenExpr("%s === null", recv)
					default:
						panic("Invalid js package object: " + sel.Obj().Name())
					}
				}

				methodName := sel.Obj().Name()
				if reservedKeywords[methodName] {
					methodName += "$"
				}
				fun = c.formatExpr("%s.%s", recv, methodName)

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
		args := c.translateArgs(sig, e.Args, e.Ellipsis.IsValid())
		if c.blocking[e] {
			resumeCase := c.caseCounter
			c.caseCounter++
			returnVar := "$r"
			if sig.Results().Len() != 0 {
				returnVar = c.newVariable("_r")
			}
			c.Printf("%[1]s = %[2]s(%[3]s); /* */ $s = %[4]d; case %[4]d: if (%[1]s && %[1]s.$blocking) { %[1]s = %[1]s(); }", returnVar, fun, strings.Join(append(args, "true"), ", "), resumeCase)
			if sig.Results().Len() != 0 {
				return c.formatExpr("%s", returnVar)
			}
			return nil
		}
		return c.formatExpr("%s(%s)", fun, strings.Join(args, ", "))

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
		if _, isTuple := exprType.(*types.Tuple); isTuple {
			return c.formatExpr("$assertType(%e, %s, true)", e.X, c.typeName(t))
		}
		return c.formatExpr("$assertType(%e, %s)", e.X, c.typeName(t))

	case *ast.Ident:
		if e.Name == "_" {
			panic("Tried to translate underscore identifier.")
		}
		switch o := obj.(type) {
		case *types.PkgName:
			return c.formatExpr("%s", c.p.pkgVars[o.Imported().Path()])
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

	case *this:
		this := "this"
		if len(c.flattened) != 0 {
			this = "$this"
		}
		if isWrapped(c.p.info.Types[e].Type) {
			return c.formatExpr("%1s.$val !== undefined ? %1s.$val : %1s", this)
		}
		return c.formatExpr("%s", this)

	case nil:
		return c.formatExpr("")

	default:
		panic(fmt.Sprintf("Unhandled expression: %T\n", e))

	}
}

func (c *funcContext) makeReceiver(x ast.Expr, sel *types.Selection) *expression {
	if !sel.Obj().Exported() {
		c.p.dependencies[sel.Obj()] = true
	}

	recvType := sel.Recv()
	_, isPointer := recvType.Underlying().(*types.Pointer)
	methodsRecvType := sel.Obj().Type().(*types.Signature).Recv().Type()
	_, pointerExpected := methodsRecvType.(*types.Pointer)
	var recv *expression
	switch {
	case !isPointer && pointerExpected:
		recv = c.translateExpr(c.setType(&ast.UnaryExpr{Op: token.AND, X: x}, methodsRecvType))
	default:
		recv = c.translateExpr(x)
	}

	for _, index := range sel.Index()[:len(sel.Index())-1] {
		if ptr, isPtr := recvType.(*types.Pointer); isPtr {
			recvType = ptr.Elem()
		}
		s := recvType.Underlying().(*types.Struct)
		recv = c.formatExpr("%s.%s", recv, fieldName(s, index))
		recvType = s.Field(index).Type()
	}

	if isWrapped(methodsRecvType) {
		recv = c.formatExpr("new %s(%s)", c.typeName(methodsRecvType), recv)
	}

	return recv
}

func (c *funcContext) translateBuiltin(name string, args []ast.Expr, ellipsis bool, typ types.Type) *expression {
	switch name {
	case "new":
		t := typ.(*types.Pointer)
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
		switch argType := c.p.info.Types[args[0]].Type.Underlying().(type) {
		case *types.Slice:
			t := c.typeName(c.p.info.Types[args[0]].Type)
			if len(args) == 3 {
				return c.formatExpr("%s.make(%f, %f)", t, args[1], args[2])
			}
			return c.formatExpr("%s.make(%f)", t, args[1])
		case *types.Map:
			return c.formatExpr("new $Map()")
		case *types.Chan:
			length := "0"
			if len(args) == 2 {
				length = c.translateExpr(args[1]).String()
			}
			return c.formatExpr("new %s(%s)", c.typeName(c.p.info.Types[args[0]].Type), length)
		default:
			panic(fmt.Sprintf("Unhandled make type: %T\n", argType))
		}
	case "len":
		switch argType := c.p.info.Types[args[0]].Type.Underlying().(type) {
		case *types.Basic:
			return c.formatExpr("%e.length", args[0])
		case *types.Slice:
			return c.formatExpr("%e.$length", args[0])
		case *types.Pointer:
			return c.formatExpr("(%e, %d)", args[0], argType.Elem().(*types.Array).Len())
		case *types.Map:
			return c.formatExpr("$keys(%e).length", args[0])
		case *types.Chan:
			return c.formatExpr("%e.$buffer.length", args[0])
		// length of array is constant
		default:
			panic(fmt.Sprintf("Unhandled len type: %T\n", argType))
		}
	case "cap":
		switch argType := c.p.info.Types[args[0]].Type.Underlying().(type) {
		case *types.Slice, *types.Chan:
			return c.formatExpr("%e.$capacity", args[0])
		case *types.Pointer:
			return c.formatExpr("(%e, %d)", args[0], argType.Elem().(*types.Array).Len())
		// capacity of array is constant
		default:
			panic(fmt.Sprintf("Unhandled cap type: %T\n", argType))
		}
	case "panic":
		return c.formatExpr("$panic(%s)", c.translateImplicitConversion(args[0], types.NewInterface(nil, nil)))
	case "append":
		if len(args) == 1 {
			return c.translateExpr(args[0])
		}
		if ellipsis {
			return c.formatExpr("$appendSlice(%e, %s)", args[0], c.translateConversionToSlice(args[1], typ))
		}
		sliceType := typ.Underlying().(*types.Slice)
		return c.formatExpr("$append(%e, %s)", args[0], strings.Join(c.translateExprSlice(args[1:], sliceType.Elem()), ", "))
	case "delete":
		return c.formatExpr(`delete %e[%s]`, args[0], c.makeKey(args[1], c.p.info.Types[args[0]].Type.Underlying().(*types.Map).Key()))
	case "copy":
		if basic, isBasic := c.p.info.Types[args[1]].Type.Underlying().(*types.Basic); isBasic && basic.Info()&types.IsString != 0 {
			return c.formatExpr("$copyString(%e, %e)", args[0], args[1])
		}
		return c.formatExpr("$copySlice(%e, %e)", args[0], args[1])
	case "print", "println":
		return c.formatExpr("console.log(%s)", strings.Join(c.translateExprSlice(args, nil), ", "))
	case "complex":
		return c.formatExpr("new %s(%e, %e)", c.typeName(typ), args[0], args[1])
	case "real":
		return c.formatExpr("%e.$real", args[0])
	case "imag":
		return c.formatExpr("%e.$imag", args[0])
	case "recover":
		return c.formatExpr("$recover()")
	case "close":
		return c.formatExpr(`$close(%e)`, args[0])
	default:
		panic(fmt.Sprintf("Unhandled builtin: %s\n", name))
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
					switch named.Obj().Name() {
					case "arrayType", "chanType", "funcType", "interfaceType", "mapType", "ptrType", "sliceType", "structType":
						return c.formatExpr("%e.%s", call.Args[0], named.Obj().Name()) // unsafe conversion
					default:
						return c.translateExpr(expr)
					}
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
				return c.fixNumber(c.formatExpr("%s.$low", c.translateExpr(expr)), t)
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
					value = c.formatExpr("%s.$low", value)
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
			if call, ok := expr.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "new" {
					return c.formatExpr("new Uint8Array(%d)", int(sizes32.Sizeof(c.p.info.Types[call.Args[0]].Type)))
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
			return c.formatExpr("$clone(%e, %s)", expr, c.typeName(t.Elem()))
		}

		if !types.Identical(exprType, types.Typ[types.UnsafePointer]) {
			return c.formatExpr("new %1s(%2e.$get, %2e.$set)", c.typeName(desiredType), expr)
		}

	case *types.Interface:
		if types.Identical(exprType, types.Typ[types.UnsafePointer]) {
			return c.translateExpr(expr)
		}
	}

	return c.translateImplicitConversionWithCloning(expr, desiredType)
}

func (c *funcContext) translateImplicitConversionWithCloning(expr ast.Expr, desiredType types.Type) *expression {
	switch desiredType.Underlying().(type) {
	case *types.Struct, *types.Array:
		switch expr.(type) {
		case nil, *ast.CompositeLit:
			// nothing
		default:
			return c.formatExpr("$clone(%e, %s)", expr, c.typeName(desiredType))
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
		return c.formatExpr("$subslice(new %1s(%2e.$array), %2e.$offset, %2e.$offset + %2e.$length)", c.typeName(desiredType), expr)

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
			code += fmt.Sprintf(`, %s = new ($nativeArray(%s))(%s.buffer, $min(%s.byteOffset + %d, %s.buffer.byteLength))`, field.Name(), typeKind(t.Elem()), array, array, offsets[i], array)
		}
	}
	return code
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
			writeExpr(".$high")
		case 'l':
			if val := c.p.info.Types[a[n].(ast.Expr)].Value; val != nil {
				d, _ := exact.Uint64Val(val)
				out.WriteString(strconv.FormatUint(d&(1<<32-1), 10))
				return
			}
			writeExpr(".$low")
		case 'r':
			if val := c.p.info.Types[a[n].(ast.Expr)].Value; val != nil {
				r, _ := exact.Float64Val(exact.Real(val))
				out.WriteString(strconv.FormatFloat(r, 'g', -1, 64))
				return
			}
			writeExpr(".$real")
		case 'i':
			if val := c.p.info.Types[a[n].(ast.Expr)].Value; val != nil {
				i, _ := exact.Float64Val(exact.Imag(val))
				out.WriteString(strconv.FormatFloat(i, 'g', -1, 64))
				return
			}
			writeExpr(".$imag")
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
