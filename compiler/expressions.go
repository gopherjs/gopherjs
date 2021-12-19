package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/analysis"
	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
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

func (fc *funcContext) translateExpr(expr ast.Expr) *expression {
	exprType := fc.pkgCtx.TypeOf(expr)
	if value := fc.pkgCtx.Types[expr].Value; value != nil {
		basic := exprType.Underlying().(*types.Basic)
		switch {
		case isBoolean(basic):
			return fc.formatExpr("%s", strconv.FormatBool(constant.BoolVal(value)))
		case isInteger(basic):
			if is64Bit(basic) {
				if basic.Kind() == types.Int64 {
					d, ok := constant.Int64Val(constant.ToInt(value))
					if !ok {
						panic("could not get exact uint")
					}
					return fc.formatExpr("new %s(%s, %s)", fc.typeName(exprType), strconv.FormatInt(d>>32, 10), strconv.FormatUint(uint64(d)&(1<<32-1), 10))
				}
				d, ok := constant.Uint64Val(constant.ToInt(value))
				if !ok {
					panic("could not get exact uint")
				}
				return fc.formatExpr("new %s(%s, %s)", fc.typeName(exprType), strconv.FormatUint(d>>32, 10), strconv.FormatUint(d&(1<<32-1), 10))
			}
			d, ok := constant.Int64Val(constant.ToInt(value))
			if !ok {
				panic("could not get exact int")
			}
			return fc.formatExpr("%s", strconv.FormatInt(d, 10))
		case isFloat(basic):
			f, _ := constant.Float64Val(value)
			return fc.formatExpr("%s", strconv.FormatFloat(f, 'g', -1, 64))
		case isComplex(basic):
			r, _ := constant.Float64Val(constant.Real(value))
			i, _ := constant.Float64Val(constant.Imag(value))
			if basic.Kind() == types.UntypedComplex {
				exprType = types.Typ[types.Complex128]
			}
			return fc.formatExpr("new %s(%s, %s)", fc.typeName(exprType), strconv.FormatFloat(r, 'g', -1, 64), strconv.FormatFloat(i, 'g', -1, 64))
		case isString(basic):
			return fc.formatExpr("%s", encodeString(constant.StringVal(value)))
		default:
			panic("Unhandled constant type: " + basic.String())
		}
	}

	var obj types.Object
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		obj = fc.pkgCtx.Uses[e.Sel]
	case *ast.Ident:
		obj = fc.pkgCtx.Defs[e]
		if obj == nil {
			obj = fc.pkgCtx.Uses[e]
		}
	}

	if obj != nil && typesutil.IsJsPackage(obj.Pkg()) {
		switch obj.Name() {
		case "Global":
			return fc.formatExpr("$global")
		case "Module":
			return fc.formatExpr("$module")
		case "Undefined":
			return fc.formatExpr("undefined")
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
			zero := fc.translateExpr(fc.zeroValue(elementType)).String()
			for _, element := range e.Elts {
				if kve, isKve := element.(*ast.KeyValueExpr); isKve {
					key, ok := constant.Int64Val(constant.ToInt(fc.pkgCtx.Types[kve.Key].Value))
					if !ok {
						panic("could not get exact int")
					}
					i = int(key)
					element = kve.Value
				}
				for len(elements) <= i {
					elements = append(elements, zero)
				}
				elements[i] = fc.translateImplicitConversionWithCloning(element, elementType).String()
				i++
			}
			return elements
		}

		switch t := exprType.Underlying().(type) {
		case *types.Array:
			elements := collectIndexedElements(t.Elem())
			if len(elements) == 0 {
				return fc.formatExpr("%s.zero()", fc.typeName(t))
			}
			zero := fc.translateExpr(fc.zeroValue(t.Elem())).String()
			for len(elements) < int(t.Len()) {
				elements = append(elements, zero)
			}
			return fc.formatExpr(`$toNativeArray(%s, [%s])`, typeKind(t.Elem()), strings.Join(elements, ", "))
		case *types.Slice:
			return fc.formatExpr("new %s([%s])", fc.typeName(exprType), strings.Join(collectIndexedElements(t.Elem()), ", "))
		case *types.Map:
			entries := make([]string, len(e.Elts))
			for i, element := range e.Elts {
				kve := element.(*ast.KeyValueExpr)
				entries[i] = fmt.Sprintf("{ k: %s, v: %s }", fc.translateImplicitConversionWithCloning(kve.Key, t.Key()), fc.translateImplicitConversionWithCloning(kve.Value, t.Elem()))
			}
			return fc.formatExpr("$makeMap(%s.keyFor, [%s])", fc.typeName(t.Key()), strings.Join(entries, ", "))
		case *types.Struct:
			elements := make([]string, t.NumFields())
			isKeyValue := true
			if len(e.Elts) != 0 {
				_, isKeyValue = e.Elts[0].(*ast.KeyValueExpr)
			}
			if !isKeyValue {
				for i, element := range e.Elts {
					elements[i] = fc.translateImplicitConversionWithCloning(element, t.Field(i).Type()).String()
				}
			}
			if isKeyValue {
				for i := range elements {
					elements[i] = fc.translateExpr(fc.zeroValue(t.Field(i).Type())).String()
				}
				for _, element := range e.Elts {
					kve := element.(*ast.KeyValueExpr)
					for j := range elements {
						if kve.Key.(*ast.Ident).Name == t.Field(j).Name() {
							elements[j] = fc.translateImplicitConversionWithCloning(kve.Value, t.Field(j).Type()).String()
							break
						}
					}
				}
			}
			return fc.formatExpr("new %s.ptr(%s)", fc.typeName(exprType), strings.Join(elements, ", "))
		default:
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", t))
		}

	case *ast.FuncLit:
		_, fun := translateFunction(e.Type, nil, e.Body, fc, exprType.(*types.Signature), fc.pkgCtx.FuncLitInfos[e], "")
		if len(fc.pkgCtx.escapingVars) != 0 {
			names := make([]string, 0, len(fc.pkgCtx.escapingVars))
			for obj := range fc.pkgCtx.escapingVars {
				names = append(names, fc.pkgCtx.objectNames[obj])
			}
			sort.Strings(names)
			list := strings.Join(names, ", ")
			return fc.formatExpr("(function(%s) { return %s; })(%s)", list, fun, list)
		}
		return fc.formatExpr("(%s)", fun)

	case *ast.UnaryExpr:
		t := fc.pkgCtx.TypeOf(e.X)
		switch e.Op {
		case token.AND:
			if typesutil.IsJsObject(exprType) {
				return fc.formatExpr("%e.object", e.X)
			}

			switch t.Underlying().(type) {
			case *types.Struct, *types.Array:
				// JavaScript's pass-by-reference semantics makes passing array's or
				// struct's object semantically equivalent to passing a pointer
				// TODO(nevkontakte): Evaluate if performance gain justifies complexity
				// introduced by the special case.
				return fc.translateExpr(e.X)
			}

			switch x := astutil.RemoveParens(e.X).(type) {
			case *ast.CompositeLit:
				return fc.formatExpr("$newDataPointer(%e, %s)", x, fc.typeName(fc.pkgCtx.TypeOf(e)))
			case *ast.Ident:
				obj := fc.pkgCtx.Uses[x].(*types.Var)
				if fc.pkgCtx.escapingVars[obj] {
					return fc.formatExpr("(%1s.$ptr || (%1s.$ptr = new %2s(function() { return this.$target[0]; }, function($v) { this.$target[0] = $v; }, %1s)))", fc.pkgCtx.objectNames[obj], fc.typeName(exprType))
				}
				return fc.formatExpr(`(%1s || (%1s = new %2s(function() { return %3s; }, function($v) { %4s })))`, fc.varPtrName(obj), fc.typeName(exprType), fc.objectName(obj), fc.translateAssign(x, fc.newIdent("$v", exprType), false))
			case *ast.SelectorExpr:
				sel, ok := fc.pkgCtx.SelectionOf(x)
				if !ok {
					// qualified identifier
					obj := fc.pkgCtx.Uses[x.Sel].(*types.Var)
					return fc.formatExpr(`(%1s || (%1s = new %2s(function() { return %3s; }, function($v) { %4s })))`, fc.varPtrName(obj), fc.typeName(exprType), fc.objectName(obj), fc.translateAssign(x, fc.newIdent("$v", exprType), false))
				}
				newSel := &ast.SelectorExpr{X: fc.newIdent("this.$target", fc.pkgCtx.TypeOf(x.X)), Sel: x.Sel}
				fc.setType(newSel, exprType)
				fc.pkgCtx.additionalSelections[newSel] = sel
				return fc.formatExpr("(%1e.$ptr_%2s || (%1e.$ptr_%2s = new %3s(function() { return %4e; }, function($v) { %5s }, %1e)))", x.X, x.Sel.Name, fc.typeName(exprType), newSel, fc.translateAssign(newSel, fc.newIdent("$v", exprType), false))
			case *ast.IndexExpr:
				if _, ok := fc.pkgCtx.TypeOf(x.X).Underlying().(*types.Slice); ok {
					return fc.formatExpr("$indexPtr(%1e.$array, %1e.$offset + %2e, %3s)", x.X, x.Index, fc.typeName(exprType))
				}
				return fc.formatExpr("$indexPtr(%e, %e, %s)", x.X, x.Index, fc.typeName(exprType))
			case *ast.StarExpr:
				return fc.translateExpr(x.X)
			default:
				panic(fmt.Sprintf("Unhandled: %T\n", x))
			}

		case token.ARROW:
			call := &ast.CallExpr{
				Fun:  fc.newIdent("$recv", types.NewSignature(nil, types.NewTuple(types.NewVar(0, nil, "", t)), types.NewTuple(types.NewVar(0, nil, "", exprType), types.NewVar(0, nil, "", types.Typ[types.Bool])), false)),
				Args: []ast.Expr{e.X},
			}
			fc.Blocking[call] = true
			if _, isTuple := exprType.(*types.Tuple); isTuple {
				return fc.formatExpr("%e", call)
			}
			return fc.formatExpr("%e[0]", call)
		}

		basic := t.Underlying().(*types.Basic)
		switch e.Op {
		case token.ADD:
			return fc.translateExpr(e.X)
		case token.SUB:
			switch {
			case is64Bit(basic):
				return fc.formatExpr("new %1s(-%2h, -%2l)", fc.typeName(t), e.X)
			case isComplex(basic):
				return fc.formatExpr("new %1s(-%2r, -%2i)", fc.typeName(t), e.X)
			case isUnsigned(basic):
				return fc.fixNumber(fc.formatExpr("-%e", e.X), basic)
			default:
				return fc.formatExpr("-%e", e.X)
			}
		case token.XOR:
			if is64Bit(basic) {
				return fc.formatExpr("new %1s(~%2h, ~%2l >>> 0)", fc.typeName(t), e.X)
			}
			return fc.fixNumber(fc.formatExpr("~%e", e.X), basic)
		case token.NOT:
			return fc.formatExpr("!%e", e.X)
		default:
			panic(e.Op)
		}

	case *ast.BinaryExpr:
		if e.Op == token.NEQ {
			return fc.formatExpr("!(%s)", fc.translateExpr(&ast.BinaryExpr{
				X:  e.X,
				Op: token.EQL,
				Y:  e.Y,
			}))
		}

		t := fc.pkgCtx.TypeOf(e.X)
		t2 := fc.pkgCtx.TypeOf(e.Y)
		_, isInterface := t2.Underlying().(*types.Interface)
		if isInterface || types.Identical(t, types.Typ[types.UntypedNil]) {
			t = t2
		}

		if basic, isBasic := t.Underlying().(*types.Basic); isBasic && isNumeric(basic) {
			if is64Bit(basic) {
				switch e.Op {
				case token.MUL:
					return fc.formatExpr("$mul64(%e, %e)", e.X, e.Y)
				case token.QUO:
					return fc.formatExpr("$div64(%e, %e, false)", e.X, e.Y)
				case token.REM:
					return fc.formatExpr("$div64(%e, %e, true)", e.X, e.Y)
				case token.SHL:
					return fc.formatExpr("$shiftLeft64(%e, %f)", e.X, e.Y)
				case token.SHR:
					return fc.formatExpr("$shiftRight%s(%e, %f)", toJavaScriptType(basic), e.X, e.Y)
				case token.EQL:
					return fc.formatExpr("(%1h === %2h && %1l === %2l)", e.X, e.Y)
				case token.LSS:
					return fc.formatExpr("(%1h < %2h || (%1h === %2h && %1l < %2l))", e.X, e.Y)
				case token.LEQ:
					return fc.formatExpr("(%1h < %2h || (%1h === %2h && %1l <= %2l))", e.X, e.Y)
				case token.GTR:
					return fc.formatExpr("(%1h > %2h || (%1h === %2h && %1l > %2l))", e.X, e.Y)
				case token.GEQ:
					return fc.formatExpr("(%1h > %2h || (%1h === %2h && %1l >= %2l))", e.X, e.Y)
				case token.ADD, token.SUB:
					return fc.formatExpr("new %3s(%1h %4t %2h, %1l %4t %2l)", e.X, e.Y, fc.typeName(t), e.Op)
				case token.AND, token.OR, token.XOR:
					return fc.formatExpr("new %3s(%1h %4t %2h, (%1l %4t %2l) >>> 0)", e.X, e.Y, fc.typeName(t), e.Op)
				case token.AND_NOT:
					return fc.formatExpr("new %3s(%1h & ~%2h, (%1l & ~%2l) >>> 0)", e.X, e.Y, fc.typeName(t))
				default:
					panic(e.Op)
				}
			}

			if isComplex(basic) {
				switch e.Op {
				case token.EQL:
					return fc.formatExpr("(%1r === %2r && %1i === %2i)", e.X, e.Y)
				case token.ADD, token.SUB:
					return fc.formatExpr("new %3s(%1r %4t %2r, %1i %4t %2i)", e.X, e.Y, fc.typeName(t), e.Op)
				case token.MUL:
					return fc.formatExpr("new %3s(%1r * %2r - %1i * %2i, %1r * %2i + %1i * %2r)", e.X, e.Y, fc.typeName(t))
				case token.QUO:
					return fc.formatExpr("$divComplex(%e, %e)", e.X, e.Y)
				default:
					panic(e.Op)
				}
			}

			switch e.Op {
			case token.EQL:
				return fc.formatParenExpr("%e === %e", e.X, e.Y)
			case token.LSS, token.LEQ, token.GTR, token.GEQ:
				return fc.formatExpr("%e %t %e", e.X, e.Op, e.Y)
			case token.ADD, token.SUB:
				return fc.fixNumber(fc.formatExpr("%e %t %e", e.X, e.Op, e.Y), basic)
			case token.MUL:
				switch basic.Kind() {
				case types.Int32, types.Int:
					return fc.formatParenExpr("$imul(%e, %e)", e.X, e.Y)
				case types.Uint32, types.Uintptr:
					return fc.formatParenExpr("$imul(%e, %e) >>> 0", e.X, e.Y)
				}
				return fc.fixNumber(fc.formatExpr("%e * %e", e.X, e.Y), basic)
			case token.QUO:
				if isInteger(basic) {
					// cut off decimals
					shift := ">>"
					if isUnsigned(basic) {
						shift = ">>>"
					}
					return fc.formatExpr(`(%1s = %2e / %3e, (%1s === %1s && %1s !== 1/0 && %1s !== -1/0) ? %1s %4s 0 : $throwRuntimeError("integer divide by zero"))`, fc.newVariable("_q"), e.X, e.Y, shift)
				}
				if basic.Kind() == types.Float32 {
					return fc.fixNumber(fc.formatExpr("%e / %e", e.X, e.Y), basic)
				}
				return fc.formatExpr("%e / %e", e.X, e.Y)
			case token.REM:
				return fc.formatExpr(`(%1s = %2e %% %3e, %1s === %1s ? %1s : $throwRuntimeError("integer divide by zero"))`, fc.newVariable("_r"), e.X, e.Y)
			case token.SHL, token.SHR:
				op := e.Op.String()
				if e.Op == token.SHR && isUnsigned(basic) {
					op = ">>>"
				}
				if v := fc.pkgCtx.Types[e.Y].Value; v != nil {
					i, _ := constant.Uint64Val(constant.ToInt(v))
					if i >= 32 {
						return fc.formatExpr("0")
					}
					return fc.fixNumber(fc.formatExpr("%e %s %s", e.X, op, strconv.FormatUint(i, 10)), basic)
				}
				if e.Op == token.SHR && !isUnsigned(basic) {
					return fc.fixNumber(fc.formatParenExpr("%e >> $min(%f, 31)", e.X, e.Y), basic)
				}
				y := fc.newVariable("y")
				return fc.fixNumber(fc.formatExpr("(%s = %f, %s < 32 ? (%e %s %s) : 0)", y, e.Y, y, e.X, op, y), basic)
			case token.AND, token.OR:
				if isUnsigned(basic) {
					return fc.formatParenExpr("(%e %t %e) >>> 0", e.X, e.Op, e.Y)
				}
				return fc.formatParenExpr("%e %t %e", e.X, e.Op, e.Y)
			case token.AND_NOT:
				return fc.fixNumber(fc.formatParenExpr("%e & ~%e", e.X, e.Y), basic)
			case token.XOR:
				return fc.fixNumber(fc.formatParenExpr("%e ^ %e", e.X, e.Y), basic)
			default:
				panic(e.Op)
			}
		}

		switch e.Op {
		case token.ADD, token.LSS, token.LEQ, token.GTR, token.GEQ:
			return fc.formatExpr("%e %t %e", e.X, e.Op, e.Y)
		case token.LAND:
			if fc.Blocking[e.Y] {
				skipCase := fc.caseCounter
				fc.caseCounter++
				resultVar := fc.newVariable("_v")
				fc.Printf("if (!(%s)) { %s = false; $s = %d; continue s; }", fc.translateExpr(e.X), resultVar, skipCase)
				fc.Printf("%s = %s; case %d:", resultVar, fc.translateExpr(e.Y), skipCase)
				return fc.formatExpr("%s", resultVar)
			}
			return fc.formatExpr("%e && %e", e.X, e.Y)
		case token.LOR:
			if fc.Blocking[e.Y] {
				skipCase := fc.caseCounter
				fc.caseCounter++
				resultVar := fc.newVariable("_v")
				fc.Printf("if (%s) { %s = true; $s = %d; continue s; }", fc.translateExpr(e.X), resultVar, skipCase)
				fc.Printf("%s = %s; case %d:", resultVar, fc.translateExpr(e.Y), skipCase)
				return fc.formatExpr("%s", resultVar)
			}
			return fc.formatExpr("%e || %e", e.X, e.Y)
		case token.EQL:
			switch u := t.Underlying().(type) {
			case *types.Array, *types.Struct:
				return fc.formatExpr("$equal(%e, %e, %s)", e.X, e.Y, fc.typeName(t))
			case *types.Interface:
				return fc.formatExpr("$interfaceIsEqual(%s, %s)", fc.translateImplicitConversion(e.X, t), fc.translateImplicitConversion(e.Y, t))
			case *types.Basic:
				if isBoolean(u) {
					if b, ok := analysis.BoolValue(e.X, fc.pkgCtx.Info.Info); ok && b {
						return fc.translateExpr(e.Y)
					}
					if b, ok := analysis.BoolValue(e.Y, fc.pkgCtx.Info.Info); ok && b {
						return fc.translateExpr(e.X)
					}
				}
			}
			return fc.formatExpr("%s === %s", fc.translateImplicitConversion(e.X, t), fc.translateImplicitConversion(e.Y, t))
		default:
			panic(e.Op)
		}

	case *ast.ParenExpr:
		return fc.formatParenExpr("%e", e.X)

	case *ast.IndexExpr:
		switch t := fc.pkgCtx.TypeOf(e.X).Underlying().(type) {
		case *types.Pointer:
			if _, ok := t.Elem().Underlying().(*types.Array); !ok {
				// Should never happen in type-checked code.
				panic(fmt.Errorf("non-array pointers can't be used with index expression"))
			}
			// Rewrite arrPtr[i] → (*arrPtr)[i] to concentrate array dereferencing
			// logic in one place.
			x := &ast.StarExpr{
				Star: e.X.Pos(),
				X:    e.X,
			}
			astutil.SetType(fc.pkgCtx.Info.Info, t.Elem(), x)
			e.X = x
			return fc.translateExpr(e)
		case *types.Array:
			pattern := rangeCheck("%1e[%2f]", fc.pkgCtx.Types[e.Index].Value != nil, true)
			return fc.formatExpr(pattern, e.X, e.Index)
		case *types.Slice:
			return fc.formatExpr(rangeCheck("%1e.$array[%1e.$offset + %2f]", fc.pkgCtx.Types[e.Index].Value != nil, false), e.X, e.Index)
		case *types.Map:
			if typesutil.IsJsObject(fc.pkgCtx.TypeOf(e.Index)) {
				fc.pkgCtx.errList = append(fc.pkgCtx.errList, types.Error{Fset: fc.pkgCtx.fileSet, Pos: e.Index.Pos(), Msg: "cannot use js.Object as map key"})
			}
			key := fmt.Sprintf("%s.keyFor(%s)", fc.typeName(t.Key()), fc.translateImplicitConversion(e.Index, t.Key()))
			if _, isTuple := exprType.(*types.Tuple); isTuple {
				return fc.formatExpr(`(%1s = %2e[%3s], %1s !== undefined ? [%1s.v, true] : [%4e, false])`, fc.newVariable("_entry"), e.X, key, fc.zeroValue(t.Elem()))
			}
			return fc.formatExpr(`(%1s = %2e[%3s], %1s !== undefined ? %1s.v : %4e)`, fc.newVariable("_entry"), e.X, key, fc.zeroValue(t.Elem()))
		case *types.Basic:
			return fc.formatExpr("%e.charCodeAt(%f)", e.X, e.Index)
		default:
			panic(fmt.Sprintf("Unhandled IndexExpr: %T\n", t))
		}

	case *ast.SliceExpr:
		if b, isBasic := fc.pkgCtx.TypeOf(e.X).Underlying().(*types.Basic); isBasic && isString(b) {
			switch {
			case e.Low == nil && e.High == nil:
				return fc.translateExpr(e.X)
			case e.Low == nil:
				return fc.formatExpr("$substring(%e, 0, %f)", e.X, e.High)
			case e.High == nil:
				return fc.formatExpr("$substring(%e, %f)", e.X, e.Low)
			default:
				return fc.formatExpr("$substring(%e, %f, %f)", e.X, e.Low, e.High)
			}
		}
		slice := fc.translateConversionToSlice(e.X, exprType)
		switch {
		case e.Low == nil && e.High == nil:
			return fc.formatExpr("%s", slice)
		case e.Low == nil:
			if e.Max != nil {
				return fc.formatExpr("$subslice(%s, 0, %f, %f)", slice, e.High, e.Max)
			}
			return fc.formatExpr("$subslice(%s, 0, %f)", slice, e.High)
		case e.High == nil:
			return fc.formatExpr("$subslice(%s, %f)", slice, e.Low)
		default:
			if e.Max != nil {
				return fc.formatExpr("$subslice(%s, %f, %f, %f)", slice, e.Low, e.High, e.Max)
			}
			return fc.formatExpr("$subslice(%s, %f, %f)", slice, e.Low, e.High)
		}

	case *ast.SelectorExpr:
		sel, ok := fc.pkgCtx.SelectionOf(e)
		if !ok {
			// qualified identifier
			return fc.formatExpr("%s", fc.objectName(obj))
		}

		switch sel.Kind() {
		case types.FieldVal:
			fields, jsTag := fc.translateSelection(sel, e.Pos())
			if jsTag != "" {
				if _, ok := sel.Type().(*types.Signature); ok {
					return fc.formatExpr("$internalize(%1e.%2s%3s, %4s, %1e.%2s)", e.X, strings.Join(fields, "."), formatJSStructTagVal(jsTag), fc.typeName(sel.Type()))
				}
				return fc.internalize(fc.formatExpr("%e.%s%s", e.X, strings.Join(fields, "."), formatJSStructTagVal(jsTag)), sel.Type())
			}
			return fc.formatExpr("%e.%s", e.X, strings.Join(fields, "."))
		case types.MethodVal:
			return fc.formatExpr(`$methodVal(%s, "%s")`, fc.makeReceiver(e), sel.Obj().(*types.Func).Name())
		case types.MethodExpr:
			if !sel.Obj().Exported() {
				fc.pkgCtx.dependencies[sel.Obj()] = true
			}
			if _, ok := sel.Recv().Underlying().(*types.Interface); ok {
				return fc.formatExpr(`$ifaceMethodExpr("%s")`, sel.Obj().(*types.Func).Name())
			}
			return fc.formatExpr(`$methodExpr(%s, "%s")`, fc.typeName(sel.Recv()), sel.Obj().(*types.Func).Name())
		default:
			panic(fmt.Sprintf("unexpected sel.Kind(): %T", sel.Kind()))
		}

	case *ast.CallExpr:
		plainFun := astutil.RemoveParens(e.Fun)

		if astutil.IsTypeExpr(plainFun, fc.pkgCtx.Info.Info) {
			return fc.formatExpr("(%s)", fc.translateConversion(e.Args[0], fc.pkgCtx.TypeOf(plainFun)))
		}

		sig := fc.pkgCtx.TypeOf(plainFun).Underlying().(*types.Signature)

		switch f := plainFun.(type) {
		case *ast.Ident:
			obj := fc.pkgCtx.Uses[f]
			if o, ok := obj.(*types.Builtin); ok {
				return fc.translateBuiltin(o.Name(), sig, e.Args, e.Ellipsis.IsValid())
			}
			if typesutil.IsJsPackage(obj.Pkg()) && obj.Name() == "InternalObject" {
				return fc.translateExpr(e.Args[0])
			}
			return fc.translateCall(e, sig, fc.translateExpr(f))

		case *ast.SelectorExpr:
			sel, ok := fc.pkgCtx.SelectionOf(f)
			if !ok {
				// qualified identifier
				obj := fc.pkgCtx.Uses[f.Sel]
				if typesutil.IsJsPackage(obj.Pkg()) {
					switch obj.Name() {
					case "Debugger":
						return fc.formatExpr("debugger")
					case "InternalObject":
						return fc.translateExpr(e.Args[0])
					}
				}
				return fc.translateCall(e, sig, fc.translateExpr(f))
			}

			externalizeExpr := func(e ast.Expr) string {
				t := fc.pkgCtx.TypeOf(e)
				if types.Identical(t, types.Typ[types.UntypedNil]) {
					return "null"
				}
				return fc.externalize(fc.translateExpr(e).String(), t)
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
				recv := fc.makeReceiver(f)
				declaredFuncRecv := sel.Obj().(*types.Func).Type().(*types.Signature).Recv().Type()
				if typesutil.IsJsObject(declaredFuncRecv) {
					globalRef := func(id string) string {
						if recv.String() == "$global" && id[0] == '$' && len(id) > 1 {
							return id
						}
						return recv.String() + "." + id
					}
					switch sel.Obj().Name() {
					case "Get":
						if id, ok := fc.identifierConstant(e.Args[0]); ok {
							return fc.formatExpr("%s", globalRef(id))
						}
						return fc.formatExpr("%s[$externalize(%e, $String)]", recv, e.Args[0])
					case "Set":
						if id, ok := fc.identifierConstant(e.Args[0]); ok {
							return fc.formatExpr("%s = %s", globalRef(id), externalizeExpr(e.Args[1]))
						}
						return fc.formatExpr("%s[$externalize(%e, $String)] = %s", recv, e.Args[0], externalizeExpr(e.Args[1]))
					case "Delete":
						return fc.formatExpr("delete %s[$externalize(%e, $String)]", recv, e.Args[0])
					case "Length":
						return fc.formatExpr("$parseInt(%s.length)", recv)
					case "Index":
						return fc.formatExpr("%s[%e]", recv, e.Args[0])
					case "SetIndex":
						return fc.formatExpr("%s[%e] = %s", recv, e.Args[0], externalizeExpr(e.Args[1]))
					case "Call":
						if id, ok := fc.identifierConstant(e.Args[0]); ok {
							if e.Ellipsis.IsValid() {
								objVar := fc.newVariable("obj")
								return fc.formatExpr("(%s = %s, %s.%s.apply(%s, %s))", objVar, recv, objVar, id, objVar, externalizeExpr(e.Args[1]))
							}
							return fc.formatExpr("%s(%s)", globalRef(id), externalizeArgs(e.Args[1:]))
						}
						if e.Ellipsis.IsValid() {
							objVar := fc.newVariable("obj")
							return fc.formatExpr("(%s = %s, %s[$externalize(%e, $String)].apply(%s, %s))", objVar, recv, objVar, e.Args[0], objVar, externalizeExpr(e.Args[1]))
						}
						return fc.formatExpr("%s[$externalize(%e, $String)](%s)", recv, e.Args[0], externalizeArgs(e.Args[1:]))
					case "Invoke":
						if e.Ellipsis.IsValid() {
							return fc.formatExpr("%s.apply(undefined, %s)", recv, externalizeExpr(e.Args[0]))
						}
						return fc.formatExpr("%s(%s)", recv, externalizeArgs(e.Args))
					case "New":
						if e.Ellipsis.IsValid() {
							return fc.formatExpr("new ($global.Function.prototype.bind.apply(%s, [undefined].concat(%s)))", recv, externalizeExpr(e.Args[0]))
						}
						return fc.formatExpr("new (%s)(%s)", recv, externalizeArgs(e.Args))
					case "Bool":
						return fc.internalize(recv, types.Typ[types.Bool])
					case "String":
						return fc.internalize(recv, types.Typ[types.String])
					case "Int":
						return fc.internalize(recv, types.Typ[types.Int])
					case "Int64":
						return fc.internalize(recv, types.Typ[types.Int64])
					case "Uint64":
						return fc.internalize(recv, types.Typ[types.Uint64])
					case "Float":
						return fc.internalize(recv, types.Typ[types.Float64])
					case "Interface":
						return fc.internalize(recv, types.NewInterface(nil, nil))
					case "Unsafe":
						return recv
					default:
						panic("Invalid js package object: " + sel.Obj().Name())
					}
				}

				methodName := sel.Obj().Name()
				if reservedKeywords[methodName] {
					methodName += "$"
				}
				return fc.translateCall(e, sig, fc.formatExpr("%s.%s", recv, methodName))

			case types.FieldVal:
				fields, jsTag := fc.translateSelection(sel, f.Pos())
				if jsTag != "" {
					call := fc.formatExpr("%e.%s%s(%s)", f.X, strings.Join(fields, "."), formatJSStructTagVal(jsTag), externalizeArgs(e.Args))
					switch sig.Results().Len() {
					case 0:
						return call
					case 1:
						return fc.internalize(call, sig.Results().At(0).Type())
					default:
						fc.pkgCtx.errList = append(fc.pkgCtx.errList, types.Error{Fset: fc.pkgCtx.fileSet, Pos: f.Pos(), Msg: "field with js tag can not have func type with multiple results"})
					}
				}
				return fc.translateCall(e, sig, fc.formatExpr("%e.%s", f.X, strings.Join(fields, ".")))

			case types.MethodExpr:
				return fc.translateCall(e, sig, fc.translateExpr(f))

			default:
				panic(fmt.Sprintf("unexpected sel.Kind(): %T", sel.Kind()))
			}
		default:
			return fc.translateCall(e, sig, fc.translateExpr(plainFun))
		}

	case *ast.StarExpr:
		if typesutil.IsJsObject(fc.pkgCtx.TypeOf(e.X)) {
			return fc.formatExpr("new $jsObjectPtr(%e)", e.X)
		}
		if c1, isCall := e.X.(*ast.CallExpr); isCall && len(c1.Args) == 1 {
			if c2, isCall := c1.Args[0].(*ast.CallExpr); isCall && len(c2.Args) == 1 && types.Identical(fc.pkgCtx.TypeOf(c2.Fun), types.Typ[types.UnsafePointer]) {
				if unary, isUnary := c2.Args[0].(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
					return fc.translateExpr(unary.X) // unsafe conversion
				}
			}
		}
		switch exprType.Underlying().(type) {
		case *types.Struct, *types.Array:
			return fc.translateExpr(e.X)
		}
		return fc.formatExpr("%e.$get()", e.X)

	case *ast.TypeAssertExpr:
		if e.Type == nil {
			return fc.translateExpr(e.X)
		}
		t := fc.pkgCtx.TypeOf(e.Type)
		if _, isTuple := exprType.(*types.Tuple); isTuple {
			return fc.formatExpr("$assertType(%e, %s, true)", e.X, fc.typeName(t))
		}
		return fc.formatExpr("$assertType(%e, %s)", e.X, fc.typeName(t))

	case *ast.Ident:
		if e.Name == "_" {
			panic("Tried to translate underscore identifier.")
		}
		switch o := obj.(type) {
		case *types.Var, *types.Const:
			return fc.formatExpr("%s", fc.objectName(o))
		case *types.Func:
			return fc.formatExpr("%s", fc.objectName(o))
		case *types.TypeName:
			return fc.formatExpr("%s", fc.typeName(o.Type()))
		case *types.Nil:
			if typesutil.IsJsObject(exprType) {
				return fc.formatExpr("null")
			}
			switch t := exprType.Underlying().(type) {
			case *types.Basic:
				if t.Kind() != types.UnsafePointer {
					panic("unexpected basic type")
				}
				return fc.formatExpr("0")
			case *types.Slice, *types.Pointer:
				return fc.formatExpr("%s.nil", fc.typeName(exprType))
			case *types.Chan:
				return fc.formatExpr("$chanNil")
			case *types.Map:
				return fc.formatExpr("false")
			case *types.Interface:
				return fc.formatExpr("$ifaceNil")
			case *types.Signature:
				return fc.formatExpr("$throwNilPointerError")
			default:
				panic(fmt.Sprintf("unexpected type: %T", t))
			}
		default:
			panic(fmt.Sprintf("Unhandled object: %T\n", o))
		}

	case nil:
		return fc.formatExpr("")

	default:
		panic(fmt.Sprintf("Unhandled expression: %T\n", e))

	}
}

func (fc *funcContext) translateCall(e *ast.CallExpr, sig *types.Signature, fun *expression) *expression {
	args := fc.translateArgs(sig, e.Args, e.Ellipsis.IsValid())
	if fc.Blocking[e] {
		resumeCase := fc.caseCounter
		fc.caseCounter++
		returnVar := "$r"
		if sig.Results().Len() != 0 {
			returnVar = fc.newVariable("_r")
		}
		fc.Printf("%[1]s = %[2]s(%[3]s); /* */ $s = %[4]d; case %[4]d: if($c) { $c = false; %[1]s = %[1]s.$blk(); } if (%[1]s && %[1]s.$blk !== undefined) { break s; }", returnVar, fun, strings.Join(args, ", "), resumeCase)
		if sig.Results().Len() != 0 {
			return fc.formatExpr("%s", returnVar)
		}
		return fc.formatExpr("")
	}
	return fc.formatExpr("%s(%s)", fun, strings.Join(args, ", "))
}

// delegatedCall returns a pair of JS expresions representing a callable function
// and its arguments to be invoked elsewhere.
//
// This function is necessary in conjunction with keywords such as `go` and `defer`,
// where we need to compute function and its arguments at the the keyword site,
// but the call itself will happen elsewhere (hence "delegated").
//
// Built-in functions and cetrain `js.Object` methods don't translate into JS
// function calls, and need to be wrapped before they can be delegated, which
// this function handles and returns JS expressions that are safe to delegate
// and behave like a regular JS function and a list of its argument values.
func (fc *funcContext) delegatedCall(expr *ast.CallExpr) (callable *expression, arglist *expression) {
	isBuiltin := false
	isJs := false
	switch fun := expr.Fun.(type) {
	case *ast.Ident:
		_, isBuiltin = fc.pkgCtx.Uses[fun].(*types.Builtin)
	case *ast.SelectorExpr:
		isJs = typesutil.IsJsPackage(fc.pkgCtx.Uses[fun.Sel].Pkg())
	}
	sig := fc.pkgCtx.TypeOf(expr.Fun).Underlying().(*types.Signature)
	sigTypes := signatureTypes{Sig: sig}
	args := fc.translateArgs(sig, expr.Args, expr.Ellipsis.IsValid())

	if !isBuiltin && !isJs {
		// Normal function calls don't require wrappers.
		callable = fc.translateExpr(expr.Fun)
		arglist = fc.formatExpr("[%s]", strings.Join(args, ", "))
		return callable, arglist
	}

	// Since some builtins or js.Object methods may not transpile into
	// callable expressions, we need to wrap then in a proxy lambda in order
	// to push them onto the deferral stack.
	vars := make([]string, len(expr.Args))
	callArgs := make([]ast.Expr, len(expr.Args))
	ellipsis := expr.Ellipsis

	for i := range expr.Args {
		v := fc.newVariable("_arg")
		vars[i] = v
		// Subtle: the proxy lambda argument needs to be assigned with the type
		// that the original function expects, and not with the argument
		// expression result type, or we may do implicit type conversion twice.
		callArgs[i] = fc.newIdent(v, sigTypes.Param(i, ellipsis.IsValid()))
	}
	wrapper := &ast.CallExpr{
		Fun:      expr.Fun,
		Args:     callArgs,
		Ellipsis: expr.Ellipsis,
	}
	callable = fc.formatExpr("function(%s) { %e; }", strings.Join(vars, ", "), wrapper)
	arglist = fc.formatExpr("[%s]", strings.Join(args, ", "))
	return callable, arglist
}

func (fc *funcContext) makeReceiver(e *ast.SelectorExpr) *expression {
	sel, _ := fc.pkgCtx.SelectionOf(e)
	if !sel.Obj().Exported() {
		fc.pkgCtx.dependencies[sel.Obj()] = true
	}

	x := e.X
	recvType := sel.Recv()
	if len(sel.Index()) > 1 {
		for _, index := range sel.Index()[:len(sel.Index())-1] {
			if ptr, isPtr := recvType.(*types.Pointer); isPtr {
				recvType = ptr.Elem()
			}
			s := recvType.Underlying().(*types.Struct)
			recvType = s.Field(index).Type()
		}

		fakeSel := &ast.SelectorExpr{X: x, Sel: ast.NewIdent("o")}
		fc.pkgCtx.additionalSelections[fakeSel] = &fakeSelection{
			kind:  types.FieldVal,
			recv:  sel.Recv(),
			index: sel.Index()[:len(sel.Index())-1],
			typ:   recvType,
		}
		x = fc.setType(fakeSel, recvType)
	}

	_, isPointer := recvType.Underlying().(*types.Pointer)
	methodsRecvType := sel.Obj().Type().(*types.Signature).Recv().Type()
	_, pointerExpected := methodsRecvType.(*types.Pointer)
	if !isPointer && pointerExpected {
		recvType = types.NewPointer(recvType)
		x = fc.setType(&ast.UnaryExpr{Op: token.AND, X: x}, recvType)
	}
	if isPointer && !pointerExpected {
		x = fc.setType(x, methodsRecvType)
	}

	recv := fc.translateImplicitConversionWithCloning(x, methodsRecvType)
	if isWrapped(recvType) {
		// Wrap JS-native value to have access to the Go type's methods.
		recv = fc.formatExpr("new %s(%s)", fc.typeName(methodsRecvType), recv)
	}
	return recv
}

func (fc *funcContext) translateBuiltin(name string, sig *types.Signature, args []ast.Expr, ellipsis bool) *expression {
	switch name {
	case "new":
		t := sig.Results().At(0).Type().(*types.Pointer)
		if fc.pkgCtx.Pkg.Path() == "syscall" && types.Identical(t.Elem().Underlying(), types.Typ[types.Uintptr]) {
			return fc.formatExpr("new Uint8Array(8)")
		}
		switch t.Elem().Underlying().(type) {
		case *types.Struct, *types.Array:
			return fc.formatExpr("%e", fc.zeroValue(t.Elem()))
		default:
			return fc.formatExpr("$newDataPointer(%e, %s)", fc.zeroValue(t.Elem()), fc.typeName(t))
		}
	case "make":
		switch argType := fc.pkgCtx.TypeOf(args[0]).Underlying().(type) {
		case *types.Slice:
			t := fc.typeName(fc.pkgCtx.TypeOf(args[0]))
			if len(args) == 3 {
				return fc.formatExpr("$makeSlice(%s, %f, %f)", t, args[1], args[2])
			}
			return fc.formatExpr("$makeSlice(%s, %f)", t, args[1])
		case *types.Map:
			if len(args) == 2 && fc.pkgCtx.Types[args[1]].Value == nil {
				return fc.formatExpr(`((%1f < 0 || %1f > 2147483647) ? $throwRuntimeError("makemap: size out of range") : {})`, args[1])
			}
			return fc.formatExpr("{}")
		case *types.Chan:
			length := "0"
			if len(args) == 2 {
				length = fc.formatExpr("%f", args[1]).String()
			}
			return fc.formatExpr("new $Chan(%s, %s)", fc.typeName(fc.pkgCtx.TypeOf(args[0]).Underlying().(*types.Chan).Elem()), length)
		default:
			panic(fmt.Sprintf("Unhandled make type: %T\n", argType))
		}
	case "len":
		switch argType := fc.pkgCtx.TypeOf(args[0]).Underlying().(type) {
		case *types.Basic:
			return fc.formatExpr("%e.length", args[0])
		case *types.Slice:
			return fc.formatExpr("%e.$length", args[0])
		case *types.Pointer:
			return fc.formatExpr("(%e, %d)", args[0], argType.Elem().(*types.Array).Len())
		case *types.Map:
			return fc.formatExpr("$keys(%e).length", args[0])
		case *types.Chan:
			return fc.formatExpr("%e.$buffer.length", args[0])
		// length of array is constant
		default:
			panic(fmt.Sprintf("Unhandled len type: %T\n", argType))
		}
	case "cap":
		switch argType := fc.pkgCtx.TypeOf(args[0]).Underlying().(type) {
		case *types.Slice, *types.Chan:
			return fc.formatExpr("%e.$capacity", args[0])
		case *types.Pointer:
			return fc.formatExpr("(%e, %d)", args[0], argType.Elem().(*types.Array).Len())
		// capacity of array is constant
		default:
			panic(fmt.Sprintf("Unhandled cap type: %T\n", argType))
		}
	case "panic":
		return fc.formatExpr("$panic(%s)", fc.translateImplicitConversion(args[0], types.NewInterface(nil, nil)))
	case "append":
		if ellipsis || len(args) == 1 {
			argStr := fc.translateArgs(sig, args, ellipsis)
			return fc.formatExpr("$appendSlice(%s, %s)", argStr[0], argStr[1])
		}
		sliceType := sig.Results().At(0).Type().Underlying().(*types.Slice)
		return fc.formatExpr("$append(%e, %s)", args[0], strings.Join(fc.translateExprSlice(args[1:], sliceType.Elem()), ", "))
	case "delete":
		args = fc.expandTupleArgs(args)
		keyType := fc.pkgCtx.TypeOf(args[0]).Underlying().(*types.Map).Key()
		return fc.formatExpr(`delete %e[%s.keyFor(%s)]`, args[0], fc.typeName(keyType), fc.translateImplicitConversion(args[1], keyType))
	case "copy":
		args = fc.expandTupleArgs(args)
		if basic, isBasic := fc.pkgCtx.TypeOf(args[1]).Underlying().(*types.Basic); isBasic && isString(basic) {
			return fc.formatExpr("$copyString(%e, %e)", args[0], args[1])
		}
		return fc.formatExpr("$copySlice(%e, %e)", args[0], args[1])
	case "print":
		args = fc.expandTupleArgs(args)
		return fc.formatExpr("$print(%s)", strings.Join(fc.translateExprSlice(args, nil), ", "))
	case "println":
		args = fc.expandTupleArgs(args)
		return fc.formatExpr("console.log(%s)", strings.Join(fc.translateExprSlice(args, nil), ", "))
	case "complex":
		argStr := fc.translateArgs(sig, args, ellipsis)
		return fc.formatExpr("new %s(%s, %s)", fc.typeName(sig.Results().At(0).Type()), argStr[0], argStr[1])
	case "real":
		return fc.formatExpr("%e.$real", args[0])
	case "imag":
		return fc.formatExpr("%e.$imag", args[0])
	case "recover":
		return fc.formatExpr("$recover()")
	case "close":
		return fc.formatExpr(`$close(%e)`, args[0])
	default:
		panic(fmt.Sprintf("Unhandled builtin: %s\n", name))
	}
}

func (fc *funcContext) identifierConstant(expr ast.Expr) (string, bool) {
	val := fc.pkgCtx.Types[expr].Value
	if val == nil {
		return "", false
	}
	s := constant.StringVal(val)
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

func (fc *funcContext) translateExprSlice(exprs []ast.Expr, desiredType types.Type) []string {
	parts := make([]string, len(exprs))
	for i, expr := range exprs {
		parts[i] = fc.translateImplicitConversion(expr, desiredType).String()
	}
	return parts
}

func (fc *funcContext) translateConversion(expr ast.Expr, desiredType types.Type) *expression {
	exprType := fc.pkgCtx.TypeOf(expr)
	if types.Identical(exprType, desiredType) {
		return fc.translateExpr(expr)
	}

	if fc.pkgCtx.Pkg.Path() == "reflect" || fc.pkgCtx.Pkg.Path() == "internal/reflectlite" {
		if call, isCall := expr.(*ast.CallExpr); isCall && types.Identical(fc.pkgCtx.TypeOf(call.Fun), types.Typ[types.UnsafePointer]) {
			if ptr, isPtr := desiredType.(*types.Pointer); isPtr {
				if named, isNamed := ptr.Elem().(*types.Named); isNamed {
					switch named.Obj().Name() {
					case "arrayType", "chanType", "funcType", "interfaceType", "mapType", "ptrType", "sliceType", "structType":
						return fc.formatExpr("%e.kindType", call.Args[0]) // unsafe conversion
					default:
						return fc.translateExpr(expr)
					}
				}
			}
		}
	}

	switch t := desiredType.Underlying().(type) {
	case *types.Basic:
		switch {
		case isInteger(t):
			basicExprType := exprType.Underlying().(*types.Basic)
			switch {
			case is64Bit(t):
				if !is64Bit(basicExprType) {
					if basicExprType.Kind() == types.Uintptr { // this might be an Object returned from reflect.Value.Pointer()
						return fc.formatExpr("new %1s(0, %2e.constructor === Number ? %2e : 1)", fc.typeName(desiredType), expr)
					}
					return fc.formatExpr("new %s(0, %e)", fc.typeName(desiredType), expr)
				}
				return fc.formatExpr("new %1s(%2h, %2l)", fc.typeName(desiredType), expr)
			case is64Bit(basicExprType):
				if !isUnsigned(t) && !isUnsigned(basicExprType) {
					return fc.fixNumber(fc.formatParenExpr("%1l + ((%1h >> 31) * 4294967296)", expr), t)
				}
				return fc.fixNumber(fc.formatExpr("%s.$low", fc.translateExpr(expr)), t)
			case isFloat(basicExprType):
				return fc.formatParenExpr("%e >> 0", expr)
			case types.Identical(exprType, types.Typ[types.UnsafePointer]):
				return fc.translateExpr(expr)
			default:
				return fc.fixNumber(fc.translateExpr(expr), t)
			}
		case isFloat(t):
			if t.Kind() == types.Float32 && exprType.Underlying().(*types.Basic).Kind() == types.Float64 {
				return fc.formatExpr("$fround(%e)", expr)
			}
			return fc.formatExpr("%f", expr)
		case isComplex(t):
			return fc.formatExpr("new %1s(%2r, %2i)", fc.typeName(desiredType), expr)
		case isString(t):
			value := fc.translateExpr(expr)
			switch et := exprType.Underlying().(type) {
			case *types.Basic:
				if is64Bit(et) {
					value = fc.formatExpr("%s.$low", value)
				}
				if isNumeric(et) {
					return fc.formatExpr("$encodeRune(%s)", value)
				}
				return value
			case *types.Slice:
				if types.Identical(et.Elem().Underlying(), types.Typ[types.Rune]) {
					return fc.formatExpr("$runesToString(%s)", value)
				}
				return fc.formatExpr("$bytesToString(%s)", value)
			default:
				panic(fmt.Sprintf("Unhandled conversion: %v\n", et))
			}
		case t.Kind() == types.UnsafePointer:
			if unary, isUnary := expr.(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
				if indexExpr, isIndexExpr := unary.X.(*ast.IndexExpr); isIndexExpr {
					return fc.formatExpr("$sliceToNativeArray(%s)", fc.translateConversionToSlice(indexExpr.X, types.NewSlice(types.Typ[types.Uint8])))
				}
				if ident, isIdent := unary.X.(*ast.Ident); isIdent && ident.Name == "_zero" {
					return fc.formatExpr("new Uint8Array(0)")
				}
			}
			if ptr, isPtr := fc.pkgCtx.TypeOf(expr).(*types.Pointer); fc.pkgCtx.Pkg.Path() == "syscall" && isPtr {
				if s, isStruct := ptr.Elem().Underlying().(*types.Struct); isStruct {
					array := fc.newVariable("_array")
					target := fc.newVariable("_struct")
					fc.Printf("%s = new Uint8Array(%d);", array, sizes32.Sizeof(s))
					fc.Delayed(func() {
						fc.Printf("%s = %s, %s;", target, fc.translateExpr(expr), fc.loadStruct(array, target, s))
					})
					return fc.formatExpr("%s", array)
				}
			}
			if call, ok := expr.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "new" {
					return fc.formatExpr("new Uint8Array(%d)", int(sizes32.Sizeof(fc.pkgCtx.TypeOf(call.Args[0]))))
				}
			}
		}

	case *types.Slice:
		switch et := exprType.Underlying().(type) {
		case *types.Basic:
			if isString(et) {
				if types.Identical(t.Elem().Underlying(), types.Typ[types.Rune]) {
					return fc.formatExpr("new %s($stringToRunes(%e))", fc.typeName(desiredType), expr)
				}
				return fc.formatExpr("new %s($stringToBytes(%e))", fc.typeName(desiredType), expr)
			}
		case *types.Array, *types.Pointer:
			return fc.formatExpr("new %s(%e)", fc.typeName(desiredType), expr)
		}

	case *types.Pointer:
		if types.Identical(exprType, types.Typ[types.UntypedNil]) {
			// Fall through to the fc.translateImplicitConversionWithCloning(), which
			// handles conversion from untyped nil to a pointer type.
			break
		}

		switch ptrElType := t.Elem().Underlying().(type) {
		case *types.Array: // (*[N]T)(expr) — converting expr to a pointer to an array.
			if _, ok := exprType.Underlying().(*types.Slice); ok {
				return fc.formatExpr("$sliceToGoArray(%e, %s)", expr, fc.typeName(desiredType))
			}
			// TODO(nevkontakte): Is this just for aliased types (e.g. `type a [4]byte`)?
			return fc.translateExpr(expr)
		case *types.Struct: // (*StructT)(expr) — converting expr to a pointer to a struct.
			if fc.pkgCtx.Pkg.Path() == "syscall" && types.Identical(exprType, types.Typ[types.UnsafePointer]) {
				// Special case: converting an unsafe pointer to a byte array into a
				// struct pointer when handling syscalls.
				// TODO(nevkontakte): Add a runtime assertion that the unsafe.Pointer is
				// indeed pointing at a byte array.
				array := fc.newVariable("_array")
				target := fc.newVariable("_struct")
				return fc.formatExpr("(%s = %e, %s = %e, %s, %s)", array, expr, target, fc.zeroValue(t.Elem()), fc.loadStruct(array, target, ptrElType), target)
			}
			// Convert between structs of different types but identical layouts,
			// for example:
			// type A struct { foo int }; type B A; var a *A = &A{42}; var b *B = (*B)(a)
			//
			// TODO(nevkontakte): Should this only apply when exprType is a pointer to a
			// struct as well?
			return fc.formatExpr("$pointerOfStructConversion(%e, %s)", expr, fc.typeName(t))
		}

		if types.Identical(exprType, types.Typ[types.UnsafePointer]) {
			// TODO(nevkontakte): Why do we fall through to the implicit conversion here?
			// Conversion from unsafe.Pointer() requires explicit type conversion: https://play.golang.org/p/IQxtmpn1wgc.
			// Possibly related to https://github.com/gopherjs/gopherjs/issues/1001.
			break // Fall through to fc.translateImplicitConversionWithCloning() below.
		}
		// Handle remaining cases, for example:
		// type iPtr *int; var c int = 42; println((iPtr)(&c));
		// TODO(nevkontakte): Are there any other cases that fall into this case?
		exprTypeElem := exprType.Underlying().(*types.Pointer).Elem()
		ptrVar := fc.newVariable("_ptr")
		getterConv := fc.translateConversion(fc.setType(&ast.StarExpr{X: fc.newIdent(ptrVar, exprType)}, exprTypeElem), t.Elem())
		setterConv := fc.translateConversion(fc.newIdent("$v", t.Elem()), exprTypeElem)
		return fc.formatExpr("(%1s = %2e, new %3s(function() { return %4s; }, function($v) { %1s.$set(%5s); }, %1s.$target))", ptrVar, expr, fc.typeName(desiredType), getterConv, setterConv)

	case *types.Interface:
		if types.Identical(exprType, types.Typ[types.UnsafePointer]) {
			return fc.translateExpr(expr)
		}
	}

	return fc.translateImplicitConversionWithCloning(expr, desiredType)
}

func (fc *funcContext) translateImplicitConversionWithCloning(expr ast.Expr, desiredType types.Type) *expression {
	switch desiredType.Underlying().(type) {
	case *types.Struct, *types.Array:
		switch expr.(type) {
		case nil, *ast.CompositeLit:
			// nothing
		default:
			return fc.formatExpr("$clone(%e, %s)", expr, fc.typeName(desiredType))
		}
	}

	return fc.translateImplicitConversion(expr, desiredType)
}

func (fc *funcContext) translateImplicitConversion(expr ast.Expr, desiredType types.Type) *expression {
	if desiredType == nil {
		return fc.translateExpr(expr)
	}

	exprType := fc.pkgCtx.TypeOf(expr)
	if types.Identical(exprType, desiredType) {
		return fc.translateExpr(expr)
	}

	basicExprType, isBasicExpr := exprType.Underlying().(*types.Basic)
	if isBasicExpr && basicExprType.Kind() == types.UntypedNil {
		return fc.formatExpr("%e", fc.zeroValue(desiredType))
	}

	switch desiredType.Underlying().(type) {
	case *types.Slice:
		return fc.formatExpr("$convertSliceType(%1e, %2s)", expr, fc.typeName(desiredType))

	case *types.Interface:
		if typesutil.IsJsObject(exprType) {
			// wrap JS object into js.Object struct when converting to interface
			return fc.formatExpr("new $jsObjectPtr(%e)", expr)
		}
		if isWrapped(exprType) {
			return fc.formatExpr("new %s(%e)", fc.typeName(exprType), expr)
		}
		if _, isStruct := exprType.Underlying().(*types.Struct); isStruct {
			return fc.formatExpr("new %1e.constructor.elem(%1e)", expr)
		}
	}

	return fc.translateExpr(expr)
}

func (fc *funcContext) translateConversionToSlice(expr ast.Expr, desiredType types.Type) *expression {
	switch fc.pkgCtx.TypeOf(expr).Underlying().(type) {
	case *types.Array, *types.Pointer:
		return fc.formatExpr("new %s(%e)", fc.typeName(desiredType), expr)
	}
	return fc.translateExpr(expr)
}

func (fc *funcContext) loadStruct(array, target string, s *types.Struct) string {
	view := fc.newVariable("_view")
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
			if isNumeric(t) {
				if is64Bit(t) {
					code += fmt.Sprintf(", %s = new %s(%s.getUint32(%d, true), %s.getUint32(%d, true))", field.Name(), fc.typeName(field.Type()), view, offsets[i]+4, view, offsets[i])
					break
				}
				code += fmt.Sprintf(", %s = %s.get%s(%d, true)", field.Name(), view, toJavaScriptType(t), offsets[i])
			}
		case *types.Array:
			code += fmt.Sprintf(`, %s = new ($nativeArray(%s))(%s.buffer, $min(%s.byteOffset + %d, %s.buffer.byteLength))`, field.Name(), typeKind(t.Elem()), array, array, offsets[i], array)
		}
		// TODO(nevkontakte): Explicitly panic if unsupported field type is encountered?
	}
	return code
}

func (fc *funcContext) fixNumber(value *expression, basic *types.Basic) *expression {
	switch basic.Kind() {
	case types.Int8:
		return fc.formatParenExpr("%s << 24 >> 24", value)
	case types.Uint8:
		return fc.formatParenExpr("%s << 24 >>> 24", value)
	case types.Int16:
		return fc.formatParenExpr("%s << 16 >> 16", value)
	case types.Uint16:
		return fc.formatParenExpr("%s << 16 >>> 16", value)
	case types.Int32, types.Int, types.UntypedInt:
		return fc.formatParenExpr("%s >> 0", value)
	case types.Uint32, types.Uint, types.Uintptr:
		return fc.formatParenExpr("%s >>> 0", value)
	case types.Float32:
		return fc.formatExpr("$fround(%s)", value)
	case types.Float64:
		return value
	default:
		panic(fmt.Sprintf("fixNumber: unhandled basic.Kind(): %s", basic.String()))
	}
}

func (fc *funcContext) internalize(s *expression, t types.Type) *expression {
	if typesutil.IsJsObject(t) {
		return s
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch {
		case isBoolean(u):
			return fc.formatExpr("!!(%s)", s)
		case isInteger(u) && !is64Bit(u):
			return fc.fixNumber(fc.formatExpr("$parseInt(%s)", s), u)
		case isFloat(u):
			return fc.formatExpr("$parseFloat(%s)", s)
		}
	}
	return fc.formatExpr("$internalize(%s, %s)", s, fc.typeName(t))
}

func (fc *funcContext) formatExpr(format string, a ...interface{}) *expression {
	return fc.formatExprInternal(format, a, false)
}

func (fc *funcContext) formatParenExpr(format string, a ...interface{}) *expression {
	return fc.formatExprInternal(format, a, true)
}

func (fc *funcContext) formatExprInternal(format string, a []interface{}, parens bool) *expression {
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
		if val := fc.pkgCtx.Types[e.(ast.Expr)].Value; val != nil {
			continue
		}
		if !hasAssignments {
			hasAssignments = true
			out.WriteByte('(')
			parens = false
		}
		v := fc.newVariable("x")
		out.WriteString(v + " = " + fc.translateExpr(e.(ast.Expr)).String() + ", ")
		vars[i] = v
	}

	processFormat(func(b, k uint8, n int) {
		writeExpr := func(suffix string) {
			if vars[n] != "" {
				out.WriteString(vars[n] + suffix)
				return
			}
			out.WriteString(fc.translateExpr(a[n].(ast.Expr)).StringWithParens() + suffix)
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
			if val := fc.pkgCtx.Types[e].Value; val != nil {
				out.WriteString(fc.translateExpr(e).String())
				return
			}
			writeExpr("")
		case 'f':
			e := a[n].(ast.Expr)
			if val := fc.pkgCtx.Types[e].Value; val != nil {
				d, _ := constant.Int64Val(constant.ToInt(val))
				out.WriteString(strconv.FormatInt(d, 10))
				return
			}
			if is64Bit(fc.pkgCtx.TypeOf(e).Underlying().(*types.Basic)) {
				out.WriteString("$flatten64(")
				writeExpr("")
				out.WriteString(")")
				return
			}
			writeExpr("")
		case 'h':
			e := a[n].(ast.Expr)
			if val := fc.pkgCtx.Types[e].Value; val != nil {
				d, _ := constant.Uint64Val(constant.ToInt(val))
				if fc.pkgCtx.TypeOf(e).Underlying().(*types.Basic).Kind() == types.Int64 {
					out.WriteString(strconv.FormatInt(int64(d)>>32, 10))
					return
				}
				out.WriteString(strconv.FormatUint(d>>32, 10))
				return
			}
			writeExpr(".$high")
		case 'l':
			if val := fc.pkgCtx.Types[a[n].(ast.Expr)].Value; val != nil {
				d, _ := constant.Uint64Val(constant.ToInt(val))
				out.WriteString(strconv.FormatUint(d&(1<<32-1), 10))
				return
			}
			writeExpr(".$low")
		case 'r':
			if val := fc.pkgCtx.Types[a[n].(ast.Expr)].Value; val != nil {
				r, _ := constant.Float64Val(constant.Real(val))
				out.WriteString(strconv.FormatFloat(r, 'g', -1, 64))
				return
			}
			writeExpr(".$real")
		case 'i':
			if val := fc.pkgCtx.Types[a[n].(ast.Expr)].Value; val != nil {
				i, _ := constant.Float64Val(constant.Imag(val))
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
