package translator

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

func (c *PkgContext) translateExpr(expr ast.Expr) string {
	exprType := c.info.Types[expr].Type
	if value := c.info.Types[expr].Value; value != nil {
		basic := types.Typ[types.String]
		if value.Kind() != exact.String { // workaround for bug in go/types
			basic = exprType.Underlying().(*types.Basic)
		}
		switch {
		case basic.Info()&types.IsBoolean != 0:
			return strconv.FormatBool(exact.BoolVal(value))
		case basic.Info()&types.IsInteger != 0:
			if is64Bit(basic) {
				d, _ := exact.Uint64Val(value)
				if basic.Kind() == types.Int64 {
					return fmt.Sprintf("new %s(%d, %d)", c.typeName(exprType), int64(d)>>32, d&(1<<32-1))
				}
				return fmt.Sprintf("new %s(%d, %d)", c.typeName(exprType), d>>32, d&(1<<32-1))
			}
			d, _ := exact.Int64Val(value)
			return strconv.FormatInt(d, 10)
		case basic.Info()&types.IsFloat != 0:
			f, _ := exact.Float64Val(value)
			return strconv.FormatFloat(f, 'g', -1, 64)
		case basic.Info()&types.IsComplex != 0:
			r, _ := exact.Float64Val(exact.Real(value))
			i, _ := exact.Float64Val(exact.Imag(value))
			if basic.Kind() == types.UntypedComplex {
				exprType = types.Typ[types.Complex128]
			}
			return fmt.Sprintf("new %s(%s, %s)", c.typeName(exprType), strconv.FormatFloat(r, 'g', -1, 64), strconv.FormatFloat(i, 'g', -1, 64))
		case basic.Info()&types.IsString != 0:
			return encodeString(exact.StringVal(value))
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
					key, _ := exact.Int64Val(c.info.Types[kve.Key].Value)
					i = int(key)
					element = kve.Value
				}
				for len(elements) <= i {
					elements = append(elements, zero)
				}
				elements[i] = c.translateImplicitConversion(element, elementType)
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
				return fmt.Sprintf(`go$toNativeArray("%s", [%s])`, typeKind(t.Elem()), strings.Join(elements, ", "))
			}
			return fmt.Sprintf(`go$makeNativeArray("%s", %d, function() { return %s; })`, typeKind(t.Elem()), t.Len(), c.zeroValue(t.Elem()))
		case *types.Slice:
			return fmt.Sprintf("new %s([%s])", c.typeName(exprType), strings.Join(collectIndexedElements(t.Elem()), ", "))
		case *types.Map:
			mapVar := c.newVariable("_map")
			keyVar := c.newVariable("_key")
			assignments := ""
			for _, element := range e.Elts {
				kve := element.(*ast.KeyValueExpr)
				assignments += fmt.Sprintf(`%s = %s, %s[%s] = { k: %s, v: %s }, `, keyVar, c.translateImplicitConversion(kve.Key, t.Key()), mapVar, c.makeKey(c.newIdent(keyVar, t.Key()), t.Key()), keyVar, c.translateImplicitConversion(kve.Value, t.Elem()))
			}
			return fmt.Sprintf("(%s = new Go$Map(), %s%s)", mapVar, assignments, mapVar)
		case *types.Struct:
			elements := make([]string, t.NumFields())
			isKeyValue := true
			if len(e.Elts) != 0 {
				_, isKeyValue = e.Elts[0].(*ast.KeyValueExpr)
			}
			if !isKeyValue {
				for i, element := range e.Elts {
					elements[i] = c.translateImplicitConversion(element, t.Field(i).Type())
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
							elements[j] = c.translateImplicitConversion(kve.Value, t.Field(j).Type())
							break
						}
					}
				}
			}
			return fmt.Sprintf("new %s.Ptr(%s)", c.typeName(exprType), strings.Join(elements, ", "))
		default:
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", t))
		}

	case *ast.FuncLit:
		return strings.TrimSpace(string(c.CatchOutput(func() {
			c.newScope(func() {
				params := c.translateParams(e.Type)
				closurePrefix := "("
				closureSuffix := ")"
				if len(c.escapingVars) != 0 {
					list := strings.Join(c.escapingVars, ", ")
					closurePrefix = "(function(" + list + ") { return "
					closureSuffix = "; })(" + list + ")"
				}
				c.Printf("%sfunction(%s) {", closurePrefix, strings.Join(params, ", "))
				c.Indent(func() {
					c.translateFunctionBody(e.Body.List, exprType.(*types.Signature))
				})
				c.Printf("}%s", closureSuffix)
			})
		})))

	case *ast.UnaryExpr:
		switch e.Op {
		case token.AND:
			switch c.info.Types[e.X].Type.Underlying().(type) {
			case *types.Struct, *types.Array:
				return c.translateExpr(e.X)
			default:
				if _, isComposite := e.X.(*ast.CompositeLit); isComposite {
					return fmt.Sprintf("go$newDataPointer(%s, %s)", c.translateExpr(e.X), c.typeName(c.info.Types[e].Type))
				}
				closurePrefix := ""
				closureSuffix := ""
				if len(c.escapingVars) != 0 {
					list := strings.Join(c.escapingVars, ", ")
					closurePrefix = "(function(" + list + ") { return "
					closureSuffix = "; })(" + list + ")"
				}
				vVar := c.newVariable("v")
				return fmt.Sprintf("%snew %s(function() { return %s; }, function(%s) { %s; })%s", closurePrefix, c.typeName(exprType), c.translateExpr(e.X), vVar, c.translateAssign(e.X, vVar), closureSuffix)
			}
		case token.ARROW:
			return "undefined"
		}

		t := c.info.Types[e.X].Type
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
				return fixNumber(fmt.Sprintf("-%s", c.translateExpr(e.X)), basic)
			default:
				return fmt.Sprintf("-%s", c.translateExpr(e.X))
			}
		case token.XOR:
			if is64Bit(basic) {
				return c.formatExpr("new %1s(~%2h, ~%2l >>> 0)", c.typeName(t), e.X)
			}
			return fixNumber(fmt.Sprintf("~%s", c.translateExpr(e.X)), basic)
		case token.NOT:
			x := c.translateExpr(e.X)
			if x == "true" {
				return "false"
			}
			if x == "false" {
				return "true"
			}
			return "!" + x
		default:
			panic(e.Op)
		}

	case *ast.BinaryExpr:
		if e.Op == token.NEQ {
			return fmt.Sprintf("!(%s)", c.translateExpr(&ast.BinaryExpr{
				X:  e.X,
				Op: token.EQL,
				Y:  e.Y,
			}))
		}

		t := c.info.Types[e.X].Type
		t2 := c.info.Types[e.Y].Type
		_, isInterface := t2.Underlying().(*types.Interface)
		if isInterface {
			t = t2
		}

		if basic, isBasic := t.Underlying().(*types.Basic); isBasic && basic.Info()&types.IsNumeric != 0 {
			if is64Bit(basic) {
				switch e.Op {
				case token.MUL:
					return fmt.Sprintf("go$mul64(%s, %s)", c.translateExpr(e.X), c.translateExpr(e.Y))
				case token.QUO:
					return fmt.Sprintf("go$div64(%s, %s, false)", c.translateExpr(e.X), c.translateExpr(e.Y))
				case token.REM:
					return fmt.Sprintf("go$div64(%s, %s, true)", c.translateExpr(e.X), c.translateExpr(e.Y))
				case token.SHL:
					return fmt.Sprintf("go$shiftLeft64(%s, %s)", c.translateExpr(e.X), c.flatten64(e.Y))
				case token.SHR:
					return fmt.Sprintf("go$shiftRight%s(%s, %s)", toJavaScriptType(basic), c.translateExpr(e.X), c.flatten64(e.Y))
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
					return c.formatExpr("new %3s(%1h %4s %2h, %1l %4s %2l)", e.X, e.Y, c.typeName(t), e.Op.String())
				case token.AND, token.OR, token.XOR:
					return c.formatExpr("new %3s(%1h %4s %2h, (%1l %4s %2l) >>> 0)", e.X, e.Y, c.typeName(t), e.Op.String())
				case token.AND_NOT:
					return c.formatExpr("new %3s(%1h &~ %2h, (%1l &~ %2l) >>> 0)", e.X, e.Y, c.typeName(t))
				default:
					panic(e.Op)
				}
			}

			if basic.Info()&types.IsComplex != 0 {
				switch e.Op {
				case token.EQL:
					return c.formatExpr("(%1r === %2r && %1i === %2i)", e.X, e.Y)
				case token.ADD, token.SUB:
					return c.formatExpr("new %3s(%1r %4s %2r, %1i %4s %2i)", e.X, e.Y, c.typeName(t), e.Op.String())
				case token.MUL:
					return c.formatExpr("new %3s(%1r * %2r - %1i * %2i, %1r * %2i + %1i * %2r)", e.X, e.Y, c.typeName(t))
				case token.QUO:
					return fmt.Sprintf("go$divComplex(%s, %s)", c.translateExpr(e.X), c.translateExpr(e.Y))
				default:
					panic(e.Op)
				}
			}

			switch e.Op {
			case token.EQL:
				return fmt.Sprintf("%s === %s", c.translateExpr(e.X), c.translateExpr(e.Y))
			case token.LSS, token.LEQ, token.GTR, token.GEQ:
				return fmt.Sprintf("%s %s %s", c.translateExpr(e.X), e.Op, c.translateExpr(e.Y))
			case token.ADD, token.SUB:
				if basic.Info()&types.IsInteger != 0 {
					return fixNumber(fmt.Sprintf("%s %s %s", c.translateExpr(e.X), e.Op, c.translateExpr(e.Y)), basic)
				}
				return fmt.Sprintf("%s %s %s", c.translateExpr(e.X), e.Op, c.translateExpr(e.Y))
			case token.MUL:
				switch basic.Kind() {
				case types.Int32, types.Int:
					return c.formatExpr("((((%1e >>> 16 << 16) * %2e >> 0) + (%1e << 16 >>> 16) * %2e) >> 0)", e.X, e.Y)
				case types.Uint32, types.Uint, types.Uintptr:
					return c.formatExpr("((((%1e >>> 16 << 16) * %2e >>> 0) + (%1e << 16 >>> 16) * %2e) >>> 0)", e.X, e.Y)
				case types.Float32, types.Float64:
					return fmt.Sprintf("%s * %s", c.translateExpr(e.X), c.translateExpr(e.Y))
				default:
					return fixNumber(fmt.Sprintf("%s * %s", c.translateExpr(e.X), c.translateExpr(e.Y)), basic)
				}
			case token.QUO:
				if basic.Info()&types.IsInteger != 0 {
					// cut off decimals
					shift := ">>"
					if basic.Info()&types.IsUnsigned != 0 {
						shift = ">>>"
					}
					return fmt.Sprintf(`(%[1]s = %[2]s / %[3]s, (%[1]s === %[1]s && %[1]s !== 1/0 && %[1]s !== -1/0) ? %[1]s %[4]s 0 : go$throwRuntimeError("integer divide by zero"))`, c.newVariable("_q"), c.translateExpr(e.X), c.translateExpr(e.Y), shift)
				}
				return fmt.Sprintf("%s / %s", c.translateExpr(e.X), c.translateExpr(e.Y))
			case token.REM:
				return fmt.Sprintf(`(%[1]s = %[2]s %% %[3]s, %[1]s === %[1]s ? %[1]s : go$throwRuntimeError("integer divide by zero"))`, c.newVariable("_r"), c.translateExpr(e.X), c.translateExpr(e.Y))
			case token.SHL, token.SHR:
				op := e.Op.String()
				if e.Op == token.SHR && basic.Info()&types.IsUnsigned != 0 {
					op = ">>>"
				}
				if c.info.Types[e.Y].Value != nil {
					return fixNumber(fmt.Sprintf("%s %s %s", c.translateExpr(e.X), op, c.translateExpr(e.Y)), basic)
				}
				if e.Op == token.SHR && basic.Info()&types.IsUnsigned == 0 {
					return fixNumber(fmt.Sprintf("(%s >> go$min(%s, 31))", c.translateExpr(e.X), c.translateExpr(e.Y)), basic)
				}
				y := c.newVariable("y")
				return fixNumber(fmt.Sprintf("(%s = %s, %s < 32 ? (%s %s %s) : 0)", y, c.translateImplicitConversion(e.Y, types.Typ[types.Uint]), y, c.translateExpr(e.X), op, y), basic)
			case token.AND, token.OR:
				if basic.Info()&types.IsUnsigned != 0 {
					return fmt.Sprintf("((%s %s %s) >>> 0)", c.translateExpr(e.X), e.Op, c.translateExpr(e.Y))
				}
				return fmt.Sprintf("(%s %s %s)", c.translateExpr(e.X), e.Op, c.translateExpr(e.Y))
			case token.AND_NOT:
				return fmt.Sprintf("(%s & ~%s)", c.translateExpr(e.X), c.translateExpr(e.Y))
			case token.XOR:
				return fixNumber(fmt.Sprintf("(%s ^ %s)", c.translateExpr(e.X), c.translateExpr(e.Y)), basic)
			default:
				panic(e.Op)
			}
		}

		switch e.Op {
		case token.ADD, token.LSS, token.LEQ, token.GTR, token.GEQ:
			return fmt.Sprintf("%s %s %s", c.translateExpr(e.X), e.Op, c.translateExpr(e.Y))
		case token.LAND:
			x := c.translateExpr(e.X)
			y := c.translateExpr(e.Y)
			if x == "false" {
				return "false"
			}
			return x + " && " + y
		case token.LOR:
			x := c.translateExpr(e.X)
			y := c.translateExpr(e.Y)
			if x == "true" {
				return "true"
			}
			return x + " || " + y
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
					}))
				}
				if len(conds) == 0 {
					conds = []string{"true"}
				}
				return fmt.Sprintf("(%s = %s, %s = %s, %s)", x, c.translateExpr(e.X), y, c.translateExpr(e.Y), strings.Join(conds, " && "))
			case *types.Interface:
				return fmt.Sprintf("go$interfaceIsEqual(%s, %s)", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
			case *types.Array:
				return fmt.Sprintf("go$arrayIsEqual(%s, %s)", c.translateExpr(e.X), c.translateExpr(e.Y))
			case *types.Pointer:
				xUnary, xIsUnary := e.X.(*ast.UnaryExpr)
				yUnary, yIsUnary := e.Y.(*ast.UnaryExpr)
				if xIsUnary && xUnary.Op == token.AND && yIsUnary && yUnary.Op == token.AND {
					xIndex, xIsIndex := xUnary.X.(*ast.IndexExpr)
					yIndex, yIsIndex := yUnary.X.(*ast.IndexExpr)
					if xIsIndex && yIsIndex {
						return fmt.Sprintf("go$sliceIsEqual(%s, %s, %s, %s)", c.translateExpr(xIndex.X), c.flatten64(xIndex.Index), c.translateExpr(yIndex.X), c.flatten64(yIndex.Index))
					}
				}
				switch u.Elem().Underlying().(type) {
				case *types.Struct, *types.Interface:
					return c.translateImplicitConversion(e.X, t) + " === " + c.translateImplicitConversion(e.Y, t)
				case *types.Array:
					return fmt.Sprintf("go$arrayIsEqual(%s, %s)", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
				default:
					return fmt.Sprintf("go$pointerIsEqual(%s, %s)", c.translateImplicitConversion(e.X, t), c.translateImplicitConversion(e.Y, t))
				}
			default:
				return c.translateImplicitConversion(e.X, t) + " === " + c.translateImplicitConversion(e.Y, t)
			}
		default:
			panic(e.Op)
		}

	case *ast.ParenExpr:
		x := c.translateExpr(e.X)
		if x == "true" || x == "false" {
			return x
		}
		return "(" + x + ")"

	case *ast.IndexExpr:
		switch t := c.info.Types[e.X].Type.Underlying().(type) {
		case *types.Array, *types.Pointer:
			return fmt.Sprintf("%s[%s]", c.translateExpr(e.X), c.flatten64(e.Index))
		case *types.Slice:
			sliceVar := c.newVariable("_slice")
			indexVar := c.newVariable("_index")
			return fmt.Sprintf(`(%s = %s, %s = %s, (%s >= 0 && %s < %s.length) ? %s.array[%s.offset + %s] : go$throwRuntimeError("index out of range"))`, sliceVar, c.translateExpr(e.X), indexVar, c.flatten64(e.Index), indexVar, indexVar, sliceVar, sliceVar, sliceVar, indexVar)
		case *types.Map:
			key := c.makeKey(e.Index, t.Key())
			if _, isTuple := exprType.(*types.Tuple); isTuple {
				return fmt.Sprintf(`(%[1]s = %[2]s[%[3]s], %[1]s !== undefined ? [%[1]s.v, true] : [%[4]s, false])`, c.newVariable("_entry"), c.translateExpr(e.X), key, c.zeroValue(t.Elem()))
			}
			return fmt.Sprintf(`(%[1]s = %[2]s[%[3]s], %[1]s !== undefined ? %[1]s.v : %[4]s)`, c.newVariable("_entry"), c.translateExpr(e.X), key, c.zeroValue(t.Elem()))
		case *types.Basic:
			return fmt.Sprintf("%s.charCodeAt(%s)", c.translateExpr(e.X), c.flatten64(e.Index))
		default:
			panic(fmt.Sprintf("Unhandled IndexExpr: %T\n", t))
		}

	case *ast.SliceExpr:
		if b, isBasic := c.info.Types[e.X].Type.Underlying().(*types.Basic); isBasic && b.Info()&types.IsString != 0 {
			if e.High == nil {
				if e.Low == nil {
					return c.translateExpr(e.X)
				}
				return fmt.Sprintf("%s.substring(%s)", c.translateExpr(e.X), c.flatten64(e.Low))
			}
			low := "0"
			if e.Low != nil {
				low = c.flatten64(e.Low)
			}
			return fmt.Sprintf("%s.substring(%s, %s)", c.translateExpr(e.X), low, c.flatten64(e.High))
		}
		slice := c.translateConversionToSlice(e.X, exprType)
		if e.High == nil {
			if e.Low == nil {
				return slice
			}
			return fmt.Sprintf("go$subslice(%s, %s)", slice, c.flatten64(e.Low))
		}
		low := "0"
		if e.Low != nil {
			low = c.flatten64(e.Low)
		}
		if e.Max != nil {
			return fmt.Sprintf("go$subslice(%s, %s, %s, %s)", slice, low, c.flatten64(e.High), c.flatten64(e.Max))
		}
		return fmt.Sprintf("go$subslice(%s, %s, %s)", slice, low, c.flatten64(e.High))

	case *ast.SelectorExpr:
		sel := c.info.Selections[e]
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
					return c.formatExpr("go$internalize(%1e.%2s.%3s, %4s, %1e.%2s)", e.X, strings.Join(fields, "."), jsTag, c.typeName(sel.Type()))
				}
				return c.internalize(fmt.Sprintf("%s.%s.%s", c.translateExpr(e.X), strings.Join(fields, "."), jsTag), sel.Type())
			}
			return fmt.Sprintf("%s.%s", c.translateExpr(e.X), strings.Join(fields, "."))
		case types.MethodVal:
			parameters := makeParametersList()
			target := c.translateExpr(e.X)
			if isWrapped(sel.Recv()) {
				target = fmt.Sprintf("(new %s(%s))", c.typeName(sel.Recv()), target)
			}
			recv := c.newVariable("_recv")
			return fmt.Sprintf("(%s = %s, function(%s) { return %s.%s(%s); })", recv, target, strings.Join(parameters, ", "), recv, e.Sel.Name, strings.Join(parameters, ", "))
		case types.MethodExpr:
			recv := "recv"
			if isWrapped(sel.Recv()) {
				recv = fmt.Sprintf("(new %s(recv))", c.typeName(sel.Recv()))
			}
			parameters := makeParametersList()
			return fmt.Sprintf("(function(%s) { return %s.%s(%s); })", strings.Join(append([]string{"recv"}, parameters...), ", "), recv, sel.Obj().(*types.Func).Name(), strings.Join(parameters, ", "))
		case types.PackageObj:
			return c.translateExpr(e.X) + "." + e.Sel.Name
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
				_, ok := c.info.Objects[e].(*types.TypeName)
				return ok
			case *ast.SelectorExpr:
				_, ok := c.info.Objects[e.Sel].(*types.TypeName)
				return ok
			case *ast.ParenExpr:
				return isType(e.X)
			default:
				return false
			}
		}

		if isType(plainFun) {
			return c.translateConversion(e.Args[0], c.info.Types[plainFun].Type)
		}

		var fun string
		switch f := plainFun.(type) {
		case *ast.Ident:
			if o, ok := c.info.Objects[f].(*types.Builtin); ok {
				switch o.Name() {
				case "new":
					t := c.info.Types[e].Type.(*types.Pointer)
					if c.pkg.Path() == "syscall" && types.Identical(t.Elem().Underlying(), types.Typ[types.Uintptr]) {
						return "new Uint8Array(8)"
					}
					switch t.Elem().Underlying().(type) {
					case *types.Struct, *types.Array:
						return c.zeroValue(t.Elem())
					default:
						return fmt.Sprintf("go$newDataPointer(%s, %s)", c.zeroValue(t.Elem()), c.typeName(t))
					}
				case "make":
					switch argType := c.info.Types[e.Args[0]].Type.Underlying().(type) {
					case *types.Slice:
						length := c.flatten64(e.Args[1])
						capacity := "0"
						if len(e.Args) == 3 {
							capacity = c.flatten64(e.Args[2])
						}
						return fmt.Sprintf("%s.make(%s, %s, function() { return %s; })", c.typeName(c.info.Types[e.Args[0]].Type), length, capacity, c.zeroValue(argType.Elem()))
					case *types.Map:
						return "new Go$Map()"
					case *types.Chan:
						return fmt.Sprintf("new %s()", c.typeName(c.info.Types[e.Args[0]].Type))
					default:
						panic(fmt.Sprintf("Unhandled make type: %T\n", argType))
					}
				case "len":
					arg := c.translateExpr(e.Args[0])
					switch argType := c.info.Types[e.Args[0]].Type.Underlying().(type) {
					case *types.Basic, *types.Slice:
						return arg + ".length"
					case *types.Pointer:
						return fmt.Sprintf("(%s, %d)", arg, argType.Elem().(*types.Array).Len())
					case *types.Map:
						return fmt.Sprintf("go$keys(%s).length", arg)
					case *types.Chan:
						return "0"
					// length of array is constant
					default:
						panic(fmt.Sprintf("Unhandled len type: %T\n", argType))
					}
				case "cap":
					arg := c.translateExpr(e.Args[0])
					switch argType := c.info.Types[e.Args[0]].Type.Underlying().(type) {
					case *types.Slice:
						return arg + ".capacity"
					case *types.Pointer:
						return fmt.Sprintf("(%s, %d)", arg, argType.Elem().(*types.Array).Len())
					case *types.Chan:
						return "0"
					// capacity of array is constant
					default:
						panic(fmt.Sprintf("Unhandled cap type: %T\n", argType))
					}
				case "panic":
					return fmt.Sprintf("throw go$panic(%s)", c.translateImplicitConversion(e.Args[0], types.NewInterface(nil, nil)))
				case "append":
					if len(e.Args) == 1 {
						return c.translateExpr(e.Args[0])
					}
					if e.Ellipsis.IsValid() {
						return fmt.Sprintf("go$appendSlice(%s, %s)", c.translateExpr(e.Args[0]), c.translateConversionToSlice(e.Args[1], exprType))
					}
					sliceType := exprType.Underlying().(*types.Slice)
					return fmt.Sprintf("go$append(%s, %s)", c.translateExpr(e.Args[0]), strings.Join(c.translateExprSlice(e.Args[1:], sliceType.Elem()), ", "))
				case "delete":
					return fmt.Sprintf(`delete %s[%s]`, c.translateExpr(e.Args[0]), c.makeKey(e.Args[1], c.info.Types[e.Args[0]].Type.Underlying().(*types.Map).Key()))
				case "copy":
					if basic, isBasic := c.info.Types[e.Args[1]].Type.Underlying().(*types.Basic); isBasic && basic.Info()&types.IsString != 0 {
						return fmt.Sprintf("go$copyString(%s, %s)", c.translateExpr(e.Args[0]), c.translateExpr(e.Args[1]))
					}
					return fmt.Sprintf("go$copySlice(%s, %s)", c.translateExpr(e.Args[0]), c.translateExpr(e.Args[1]))
				case "print", "println":
					return fmt.Sprintf("console.log(%s)", strings.Join(c.translateExprSlice(e.Args, nil), ", "))
				case "complex":
					return fmt.Sprintf("new %s(%s, %s)", c.typeName(c.info.Types[e].Type), c.translateExpr(e.Args[0]), c.translateExpr(e.Args[1]))
				case "real":
					return c.translateExpr(e.Args[0]) + ".real"
				case "imag":
					return c.translateExpr(e.Args[0]) + ".imag"
				case "recover":
					return "go$recover()"
				case "close":
					return `go$notSupported("close")`
				default:
					panic(fmt.Sprintf("Unhandled builtin: %s\n", o.Name()))
				}
			}
			fun = c.translateExpr(plainFun)

		case *ast.SelectorExpr:
			sel := c.info.Selections[f]
			o := sel.Obj()

			externalizeExpr := func(e ast.Expr) string {
				t := c.info.Types[e].Type
				if types.Identical(t, types.Typ[types.UntypedNil]) {
					return "null"
				}
				return c.externalize(c.translateExpr(e), t)
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
				methodName := o.Name()
				if ReservedKeywords[methodName] {
					methodName += "$"
				}

				fun = c.translateExpr(f.X)
				t := sel.Recv()
				for _, index := range sel.Index()[:len(sel.Index())-1] {
					if ptr, isPtr := t.(*types.Pointer); isPtr {
						t = ptr.Elem()
					}
					s := t.Underlying().(*types.Struct)
					fun += "." + fieldName(s, index)
					t = s.Field(index).Type()
				}

				if o.Pkg() != nil && o.Pkg().Path() == "github.com/neelance/gopherjs/js" {
					switch o.Name() {
					case "Get":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							return fmt.Sprintf("%s.%s", fun, id)
						}
						return fmt.Sprintf("%s[go$externalize(%s, Go$String)]", fun, c.translateExpr(e.Args[0]))
					case "Set":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							return fmt.Sprintf("%s.%s = %s", fun, id, externalizeExpr(e.Args[1]))
						}
						return fmt.Sprintf("%s[go$externalize(%s, Go$String)] = %s", fun, c.translateExpr(e.Args[0]), externalizeExpr(e.Args[1]))
					case "Length":
						return fmt.Sprintf("go$parseInt(%s.length)", fun)
					case "Index":
						return fmt.Sprintf("%s[%s]", fun, c.translateExpr(e.Args[0]))
					case "SetIndex":
						return fmt.Sprintf("%s[%s] = %s", fun, c.translateExpr(e.Args[0]), externalizeExpr(e.Args[1]))
					case "Call":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							if e.Ellipsis.IsValid() {
								objVar := c.newVariable("obj")
								return fmt.Sprintf("(%s = %s, %s.%s.apply(%s, %s))", objVar, fun, objVar, id, objVar, externalizeExpr(e.Args[1]))
							}
							return fmt.Sprintf("%s.%s(%s)", fun, id, externalizeArgs(e.Args[1:]))
						}
						if e.Ellipsis.IsValid() {
							objVar := c.newVariable("obj")
							return fmt.Sprintf("(%s = %s, %s[go$externalize(%s, Go$String)].apply(%s, %s))", objVar, fun, objVar, c.translateExpr(e.Args[0]), objVar, externalizeExpr(e.Args[1]))
						}
						return fmt.Sprintf("%s[go$externalize(%s, Go$String)](%s)", fun, c.translateExpr(e.Args[0]), externalizeArgs(e.Args[1:]))
					case "Invoke":
						if e.Ellipsis.IsValid() {
							return fmt.Sprintf("%s.apply(undefined, %s)", fun, externalizeExpr(e.Args[0]))
						}
						return fmt.Sprintf("%s(%s)", fun, externalizeArgs(e.Args))
					case "New":
						if e.Ellipsis.IsValid() {
							return fmt.Sprintf("new (go$global.Function.prototype.bind.apply(%s, [undefined].concat(%s)))", fun, externalizeExpr(e.Args[0]))
						}
						return fmt.Sprintf("new %s(%s)", fun, externalizeArgs(e.Args))
					case "Bool":
						return c.internalize(fun, types.Typ[types.Bool])
					case "String":
						return c.internalize(fun, types.Typ[types.String])
					case "Int":
						return c.internalize(fun, types.Typ[types.Int])
					case "Float":
						return c.internalize(fun, types.Typ[types.Float64])
					case "Interface":
						return c.internalize(fun, types.NewInterface(nil, nil))
					case "IsUndefined":
						return fmt.Sprintf("(%s === undefined)", fun)
					case "IsNull":
						return fmt.Sprintf("(%s === null)", fun)
					default:
						panic("Invalid js package method: " + o.Name())
					}
				}

				methodsRecvType := o.Type().(*types.Signature).Recv().Type()
				_, pointerExpected := methodsRecvType.(*types.Pointer)
				_, isPointer := t.Underlying().(*types.Pointer)
				_, isStruct := t.Underlying().(*types.Struct)
				_, isArray := t.Underlying().(*types.Array)
				if pointerExpected && !isPointer && !isStruct && !isArray {
					vVar := c.newVariable("v")
					fun = fmt.Sprintf("(new %s(function() { return %s; }, function(%s) { %s = %s; })).%s", c.typeName(methodsRecvType), fun, vVar, fun, vVar, methodName)
					break
				}

				if isWrapped(t) {
					fun = fmt.Sprintf("(new %s(%s)).%s", c.typeName(t), fun, methodName)
					break
				}
				fun += "." + methodName

			case types.PackageObj:
				if o.Pkg() != nil && o.Pkg().Path() == "github.com/neelance/gopherjs/js" {
					switch o.Name() {
					case "Global":
						if id, ok := c.identifierConstant(e.Args[0]); ok {
							return fmt.Sprintf("go$global.%s", id)
						}
						return fmt.Sprintf("go$global[go$externalize(%s, Go$String)]", c.translateExpr(e.Args[0]))
					case "This":
						return "this"
					default:
						panic("Invalid js package method: " + o.Name())
					}
				}
				fun = c.translateExpr(f)

			case types.FieldVal:
				fields, jsTag := c.translateSelection(sel)
				if jsTag != "" {
					sig := sel.Type().(*types.Signature)
					return c.internalize(fmt.Sprintf("%s.%s.%s(%s)", c.translateExpr(f.X), strings.Join(fields, "."), jsTag, externalizeArgs(e.Args)), sig.Results().At(0).Type())
				}
				fun = fmt.Sprintf("%s.%s", c.translateExpr(f.X), strings.Join(fields, "."))

			case types.MethodExpr:
				fun = c.translateExpr(f)

			default:
				panic("")
			}
		default:
			fun = c.translateExpr(plainFun)
		}

		sig := c.info.Types[plainFun].Type.Underlying().(*types.Signature)
		if len(e.Args) == 1 {
			if tuple, isTuple := c.info.Types[e.Args[0]].Type.(*types.Tuple); isTuple {
				tupleVar := c.newVariable("_tuple")
				args := make([]ast.Expr, tuple.Len())
				for i := range args {
					args[i] = c.newIdent(fmt.Sprintf("%s[%d]", tupleVar, i), tuple.At(i).Type())
				}
				return fmt.Sprintf("(%s = %s, %s(%s))", tupleVar, c.translateExpr(e.Args[0]), fun, c.translateArgs(sig, args, false))
			}
		}
		return fmt.Sprintf("%s(%s)", fun, c.translateArgs(sig, e.Args, e.Ellipsis.IsValid()))

	case *ast.StarExpr:
		if c1, isCall := e.X.(*ast.CallExpr); isCall && len(c1.Args) == 1 {
			if c2, isCall := c1.Args[0].(*ast.CallExpr); isCall && len(c2.Args) == 1 && types.Identical(c.info.Types[c2.Fun].Type, types.Typ[types.UnsafePointer]) {
				if unary, isUnary := c2.Args[0].(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
					return c.translateExpr(unary.X) // unsafe conversion
				}
			}
		}
		switch exprType.Underlying().(type) {
		case *types.Struct, *types.Array:
			return c.translateExpr(e.X)
		}
		return c.translateExpr(e.X) + ".go$get()"

	case *ast.TypeAssertExpr:
		if e.Type == nil {
			return c.translateExpr(e.X)
		}
		t := c.info.Types[e.Type].Type
		check := "%1e !== null && " + c.typeCheck("%1e.constructor", t)
		valueSuffix := ""
		if _, isInterface := t.Underlying().(*types.Interface); !isInterface {
			valueSuffix = ".go$val"
		}
		if _, isTuple := exprType.(*types.Tuple); isTuple {
			return c.formatExpr("("+check+" ? [%1e%2s, true] : [%3s, false])", e.X, valueSuffix, c.zeroValue(c.info.Types[e.Type].Type))
		}
		return c.formatExpr("("+check+" ? %1e%2s : go$typeAssertionFailed(%1e, %3s))", e.X, valueSuffix, c.typeName(t))

	case *ast.Ident:
		if e.Name == "_" {
			panic("Tried to translate underscore identifier.")
		}
		switch o := c.info.Objects[e].(type) {
		case *types.PkgName:
			return c.pkgVars[o.Pkg().Path()]
		case *types.Var, *types.Const:
			return c.objectName(o)
		case *types.Func:
			return c.objectName(o)
		case *types.TypeName:
			return c.typeName(o.Type())
		case *types.Nil:
			return c.zeroValue(c.info.Types[e].Type)
		case nil:
			return e.Name
		default:
			panic(fmt.Sprintf("Unhandled object: %T\n", o))
		}

	case nil:
		return ""

	default:
		panic(fmt.Sprintf("Unhandled expression: %T\n", e))

	}
}

func (c *PkgContext) formatExpr(format string, a ...interface{}) string {
	var vars = make([]string, len(a))
	var assignments []string
	varFor := func(i int) string {
		v := vars[i]
		if v != "" {
			return v
		}
		e := a[i].(ast.Expr)
		if ident, isIdent := e.(*ast.Ident); isIdent {
			v = c.translateExpr(ident)
			vars[i] = v
			return v
		}
		v = c.newVariable("x")
		assignments = append(assignments, v+" = "+c.translateExpr(e))
		vars[i] = v
		return v
	}
	out := bytes.NewBuffer(nil)
	for i := 0; i < len(format); i++ {
		b := format[i]
		if b == '%' {
			n := int(format[i+1] - '0' - 1)
			k := format[i+2]
			i += 2
			switch k {
			case 's':
				out.WriteString(a[n].(string))
			case 'e':
				out.WriteString(varFor(n))
			case 'h':
				if val := c.info.Types[a[n].(ast.Expr)].Value; val != nil {
					d, _ := exact.Uint64Val(val)
					if c.info.Types[a[n].(ast.Expr)].Type.Underlying().(*types.Basic).Kind() == types.Int64 {
						out.WriteString(strconv.FormatInt(int64(d)>>32, 10))
						continue
					}
					out.WriteString(strconv.FormatUint(d>>32, 10))
					continue
				}
				out.WriteString(varFor(n) + ".high")
			case 'l':
				if val := c.info.Types[a[n].(ast.Expr)].Value; val != nil {
					d, _ := exact.Uint64Val(val)
					out.WriteString(strconv.FormatUint(d&(1<<32-1), 10))
					continue
				}
				out.WriteString(varFor(n) + ".low")
			case 'r':
				if val := c.info.Types[a[n].(ast.Expr)].Value; val != nil {
					r, _ := exact.Float64Val(exact.Real(val))
					out.WriteString(strconv.FormatFloat(r, 'g', -1, 64))
					continue
				}
				out.WriteString(varFor(n) + ".real")
			case 'i':
				if val := c.info.Types[a[n].(ast.Expr)].Value; val != nil {
					i, _ := exact.Float64Val(exact.Imag(val))
					out.WriteString(strconv.FormatFloat(i, 'g', -1, 64))
					continue
				}
				out.WriteString(varFor(n) + ".imag")
			default:
				panic("formatExpr: " + format[i:i+3])
			}
			continue
		}
		out.WriteByte(b)
	}
	if len(assignments) == 0 {
		return out.String()
	}
	return "(" + strings.Join(assignments, ", ") + ", " + out.String() + ")"
}

func (c *PkgContext) identifierConstant(expr ast.Expr) (string, bool) {
	val := c.info.Types[expr].Value
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

func (c *PkgContext) translateExprSlice(exprs []ast.Expr, desiredType types.Type) []string {
	parts := make([]string, len(exprs))
	for i, expr := range exprs {
		parts[i] = c.translateImplicitConversion(expr, desiredType)
	}
	return parts
}

func (c *PkgContext) translateConversion(expr ast.Expr, desiredType types.Type) string {
	exprType := c.info.Types[expr].Type
	if types.Identical(exprType, desiredType) {
		return c.translateExpr(expr)
	}

	if c.pkg.Path() == "reflect" {
		if call, isCall := expr.(*ast.CallExpr); isCall && types.Identical(c.info.Types[call.Fun].Type, types.Typ[types.UnsafePointer]) {
			if ptr, isPtr := desiredType.(*types.Pointer); isPtr {
				if named, isNamed := ptr.Elem().(*types.Named); isNamed {
					return c.translateExpr(call.Args[0]) + "." + named.Obj().Name() // unsafe conversion
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
					return fmt.Sprintf("new %s(0, %s)", c.typeName(desiredType), c.translateExpr(expr))
				}
				return c.formatExpr("new %1s(%2h, %2l)", c.typeName(desiredType), expr)
			case is64Bit(basicExprType):
				if t.Info()&types.IsUnsigned == 0 && basicExprType.Info()&types.IsUnsigned == 0 {
					return fixNumber(c.formatExpr("(%1l + ((%1h >> 31) * 4294967296))", expr), t)
				}
				return fixNumber(fmt.Sprintf("%s.low", c.translateExpr(expr)), t)
			case basicExprType.Info()&types.IsFloat != 0:
				return fmt.Sprintf("(%s >> 0)", c.translateExpr(expr))
			case types.Identical(exprType, types.Typ[types.UnsafePointer]):
				return c.translateExpr(expr)
			default:
				return fixNumber(c.translateExpr(expr), t)
			}
		case t.Info()&types.IsFloat != 0:
			return c.flatten64(expr)
		case t.Info()&types.IsComplex != 0:
			return c.formatExpr("new %1s(%2r, %2i)", c.typeName(desiredType), expr)
		case t.Info()&types.IsString != 0:
			value := c.translateExpr(expr)
			switch et := exprType.Underlying().(type) {
			case *types.Basic:
				if is64Bit(et) {
					value = fmt.Sprintf("%s.low", value)
				}
				if et.Info()&types.IsNumeric != 0 {
					return fmt.Sprintf("go$encodeRune(%s)", value)
				}
				return value
			case *types.Slice:
				if types.Identical(et.Elem().Underlying(), types.Typ[types.Rune]) {
					return fmt.Sprintf("go$runesToString(%s)", value)
				}
				return fmt.Sprintf("go$bytesToString(%s)", value)
			default:
				panic(fmt.Sprintf("Unhandled conversion: %v\n", et))
			}
		case t.Kind() == types.UnsafePointer:
			if unary, isUnary := expr.(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
				if indexExpr, isIndexExpr := unary.X.(*ast.IndexExpr); isIndexExpr {
					return fmt.Sprintf("go$sliceToArray(%s)", c.translateConversionToSlice(indexExpr.X, types.NewSlice(types.Typ[types.Uint8])))
				}
				if ident, isIdent := unary.X.(*ast.Ident); isIdent && ident.Name == "_zero" {
					return "new Uint8Array(0)"
				}
			}
			if ptr, isPtr := c.info.Types[expr].Type.(*types.Pointer); c.pkg.Path() == "syscall" && isPtr {
				if s, isStruct := ptr.Elem().Underlying().(*types.Struct); isStruct {
					array := c.newVariable("_array")
					target := c.newVariable("_struct")
					c.Printf("%s = new Uint8Array(%d);", array, sizes32.Sizeof(s))
					c.Delayed(func() {
						c.Printf("%s = %s, %s;", target, c.translateExpr(expr), c.loadStruct(array, target, s))
					})
					return array
				}
			}
		}

	case *types.Slice:
		switch et := exprType.Underlying().(type) {
		case *types.Basic:
			if et.Info()&types.IsString != 0 {
				if types.Identical(t.Elem().Underlying(), types.Typ[types.Rune]) {
					return fmt.Sprintf("new %s(go$stringToRunes(%s))", c.typeName(desiredType), c.translateExpr(expr))
				}
				return fmt.Sprintf("new %s(go$stringToBytes(%s))", c.typeName(desiredType), c.translateExpr(expr))
			}
		case *types.Array, *types.Pointer:
			return fmt.Sprintf("new %s(%s)", c.typeName(desiredType), c.translateExpr(expr))
		}

	case *types.Pointer:
		if s, isStruct := t.Elem().Underlying().(*types.Struct); isStruct {
			if c.pkg.Path() == "syscall" && types.Identical(exprType, types.Typ[types.UnsafePointer]) {
				array := c.newVariable("_array")
				target := c.newVariable("_struct")
				return fmt.Sprintf("(%s = %s, %s = %s, %s, %s)", array, c.translateExpr(expr), target, c.zeroValue(t.Elem()), c.loadStruct(array, target, s), target)
			}
			return c.clone(c.translateExpr(expr), t.Elem())
		}

		if !types.Identical(exprType, types.Typ[types.UnsafePointer]) {
			return c.formatExpr("new %1s(%2e.go$get, %2e.go$set)", c.typeName(desiredType), expr)
		}
	}

	return c.translateImplicitConversion(expr, desiredType)
}

func (c *PkgContext) translateImplicitConversion(expr ast.Expr, desiredType types.Type) string {
	if desiredType == nil {
		return c.translateExpr(expr)
	}
	if expr == nil {
		return c.zeroValue(desiredType)
	}

	switch desiredType.Underlying().(type) {
	case *types.Struct, *types.Array:
		if _, isComposite := expr.(*ast.CompositeLit); !isComposite {
			return c.clone(c.translateExpr(expr), desiredType)
		}
	}

	exprType := c.info.Types[expr].Type
	if types.Identical(exprType, desiredType) {
		return c.translateExpr(expr)
	}

	basicExprType, isBasicExpr := exprType.Underlying().(*types.Basic)
	if isBasicExpr && basicExprType.Kind() == types.UntypedNil {
		return c.zeroValue(desiredType)
	}

	switch desiredType.Underlying().(type) {
	case *types.Slice:
		return c.formatExpr("go$subslice(new %1s(%2e.array), %2e.offset, %2e.offset + %2e.length)", c.typeName(desiredType), expr)

	case *types.Interface:
		if isWrapped(exprType) {
			return fmt.Sprintf("new %s(%s)", c.typeName(exprType), c.translateExpr(expr))
		}
		if _, isStruct := exprType.Underlying().(*types.Struct); isStruct {
			return c.formatExpr("new %1e.constructor.Struct(%1e)", expr)
		}
	}

	return c.translateExpr(expr)
}

func (c *PkgContext) translateConversionToSlice(expr ast.Expr, desiredType types.Type) string {
	switch c.info.Types[expr].Type.Underlying().(type) {
	case *types.Basic:
		return fmt.Sprintf("new %s(go$stringToBytes(%s))", c.typeName(desiredType), c.translateExpr(expr))
	case *types.Array, *types.Pointer:
		return fmt.Sprintf("new %s(%s)", c.typeName(desiredType), c.translateExpr(expr))
	}
	return c.translateExpr(expr)
}

func (c *PkgContext) clone(src string, ty types.Type) string {
	switch t := ty.Underlying().(type) {
	case *types.Struct:
		structVar := c.newVariable("_struct")
		fields := make([]string, t.NumFields())
		for i := range fields {
			fields[i] = c.clone(structVar+"."+fieldName(t, i), t.Field(i).Type())
		}
		constructor := structVar + ".constructor"
		if named, isNamed := ty.(*types.Named); isNamed {
			constructor = c.objectName(named.Obj()) + ".Ptr"
		}
		return fmt.Sprintf("(%s = %s, new %s(%s))", structVar, src, constructor, strings.Join(fields, ", "))
	case *types.Array:
		return fmt.Sprintf("go$mapArray(%s, function(entry) { return %s; })", src, c.clone("entry", t.Elem()))
	default:
		return src
	}
}

func (c *PkgContext) loadStruct(array, target string, s *types.Struct) string {
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
			code += fmt.Sprintf(`, %s = new (go$nativeArray("%s"))(%s.buffer, go$min(%s.byteOffset + %d, %s.buffer.byteLength))`, field.Name(), typeKind(t.Elem()), array, array, offsets[i], array)
		}
	}
	return code
}

func (c *PkgContext) typeCheck(of string, to types.Type) string {
	if in, isInterface := to.Underlying().(*types.Interface); isInterface {
		if in.MethodSet().Len() == 0 {
			return "true"
		}
		return fmt.Sprintf("%s.implementedBy.indexOf(%s) !== -1", c.typeName(to), of)
	}
	return of + " === " + c.typeName(to)
}

func (c *PkgContext) flatten64(expr ast.Expr) string {
	if is64Bit(c.info.Types[expr].Type.Underlying().(*types.Basic)) {
		return fmt.Sprintf("go$flatten64(%s)", c.translateExpr(expr))
	}
	return c.translateExpr(expr)
}

func fixNumber(value string, basic *types.Basic) string {
	switch basic.Kind() {
	case types.Int8:
		return "(" + value + " << 24 >> 24)"
	case types.Uint8:
		return "(" + value + " << 24 >>> 24)"
	case types.Int16:
		return "(" + value + " << 16 >> 16)"
	case types.Uint16:
		return "(" + value + " << 16 >>> 16)"
	case types.Int32, types.Int:
		return "(" + value + " >> 0)"
	case types.Uint32, types.Uint, types.Uintptr:
		return "(" + value + " >>> 0)"
	default:
		panic(int(basic.Kind()))
	}
}

type HasDeferVisitor struct {
	hasDefer bool
}

func (v *HasDeferVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if v.hasDefer {
		return nil
	}
	switch node.(type) {
	case *ast.DeferStmt:
		v.hasDefer = true
		return nil
	case *ast.FuncLit:
		return nil
	}
	return v
}
