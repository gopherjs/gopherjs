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
	exprType := c.info.Types[expr]
	if value, valueFound := c.info.Values[expr]; valueFound {
		// workaround
		if id, isIdent := expr.(*ast.Ident); isIdent {
			switch id.Name {
			case "MaxFloat32":
				return "3.40282346638528859811704183484516925440e+38"
			case "SmallestNonzeroFloat32":
				return "1.401298464324817070923729583289916131280e-45"
			case "MaxFloat64":
				return "1.797693134862315708145274237317043567981e+308"
			case "SmallestNonzeroFloat64":
				return "4.940656458412465441765687928682213723651e-324"
			}
		}
		v := HasEvilConstantVisitor{}
		ast.Walk(&v, expr)
		if !v.hasEvilConstant {
			switch value.Kind() {
			case exact.Bool:
				return strconv.FormatBool(exact.BoolVal(value))
			case exact.Int:
				basic := exprType.Underlying().(*types.Basic)
				if is64Bit(basic) {
					d, _ := exact.Uint64Val(value)
					return fmt.Sprintf("new %s(%d, %d)", c.typeName(exprType), d>>32, d&(1<<32-1))
				}
				d, _ := exact.Int64Val(value)
				return strconv.FormatInt(d, 10)
			case exact.Float:
				f, _ := exact.Float64Val(value)
				return strconv.FormatFloat(f, 'g', -1, int(sizes32.Sizeof(exprType))*8)
			case exact.Complex:
				f, _ := exact.Float64Val(exact.Real(value))
				return strconv.FormatFloat(f, 'g', -1, int(sizes32.Sizeof(exprType))*8/2)
			case exact.String:
				buffer := bytes.NewBuffer(nil)
				for _, r := range []byte(exact.StringVal(value)) {
					switch r {
					case '\b':
						buffer.WriteString(`\b`)
					case '\f':
						buffer.WriteString(`\f`)
					case '\n':
						buffer.WriteString(`\n`)
					case '\r':
						buffer.WriteString(`\r`)
					case '\t':
						buffer.WriteString(`\t`)
					case '\v':
						buffer.WriteString(`\v`)
					case 0:
						buffer.WriteString(`\0`)
					case '"':
						buffer.WriteString(`\"`)
					case '\\':
						buffer.WriteString(`\\`)
					default:
						if r < 0x20 || r > 0x7E {
							fmt.Fprintf(buffer, `\x%02X`, r)
							continue
						}
						buffer.WriteByte(r)
					}
				}
				return `"` + buffer.String() + `"`
			default:
				panic("Unhandled value: " + value.String())
			}
		}
	}

	switch e := expr.(type) {
	case *ast.CompositeLit:
		if ptrType, isPointer := exprType.(*types.Pointer); isPointer {
			exprType = ptrType.Elem()
		}

		var elements []string
		collectIndexedElements := func(elementType types.Type) {
			elements = make([]string, 0)
			zero := c.zeroValue(elementType)
			for _, element := range e.Elts {
				if kve, isKve := element.(*ast.KeyValueExpr); isKve {
					key, _ := exact.Int64Val(c.info.Values[kve.Key])
					for len(elements) < int(key) {
						elements = append(elements, zero)
					}
					element = kve.Value
				}
				elements = append(elements, c.translateExpr(element))
			}
		}
		switch t := exprType.Underlying().(type) {
		case *types.Array:
			collectIndexedElements(t.Elem())
		case *types.Slice:
			collectIndexedElements(t.Elem())
		case *types.Map:
			elements = make([]string, len(e.Elts)*2)
			for i, element := range e.Elts {
				kve := element.(*ast.KeyValueExpr)
				elements[i*2] = c.translateExpr(kve.Key)
				elements[i*2+1] = c.translateExpr(kve.Value)
			}
		case *types.Struct:
			elements = make([]string, t.NumFields())
			isKeyValue := true
			if len(e.Elts) != 0 {
				_, isKeyValue = e.Elts[0].(*ast.KeyValueExpr)
			}
			if !isKeyValue {
				for i, element := range e.Elts {
					elements[i] = c.translateExprToType(element, t.Field(i).Type())
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
							elements[j] = c.translateExprToType(kve.Value, t.Field(j).Type())
							break
						}
					}
				}
			}
		}

		switch t := exprType.(type) {
		case *types.Array:
			if len(elements) != 0 {
				zero := c.zeroValue(t.Elem())
				for len(elements) < int(t.Len()) {
					elements = append(elements, zero)
				}
				return createListComposite(t.Elem(), elements)
			}
			return fmt.Sprintf("Go$makeArray(%s, %d, function() { return %s; })", toArrayType(t.Elem()), t.Len(), c.zeroValue(t.Elem()))
		case *types.Slice:
			return fmt.Sprintf("new Go$Slice(%s)", createListComposite(t.Elem(), elements))
		case *types.Map:
			return fmt.Sprintf("new Go$Map([%s])", strings.Join(elements, ", "))
		case *types.Struct:
			for i, element := range elements {
				elements[i] = fmt.Sprintf("%s: %s", t.Field(i).Name(), element)
			}
			return fmt.Sprintf("{ %s }", strings.Join(elements, ", "))
		case *types.Named:
			switch u := t.Underlying().(type) {
			case *types.Array:
				return fmt.Sprintf("new %s(%s)", c.typeName(t), createListComposite(u.Elem(), elements))
			case *types.Slice:
				return fmt.Sprintf("new %s(%s)", c.typeName(t), createListComposite(u.Elem(), elements))
			case *types.Map:
				return fmt.Sprintf("new %s([%s])", c.typeName(t), strings.Join(elements, ", "))
			case *types.Struct:
				return fmt.Sprintf("new %s(%s)", c.objectName(t.Obj()), strings.Join(elements, ", "))
			default:
				panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", u))
			}
		default:
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", t))
		}

	case *ast.FuncLit:
		return strings.TrimSpace(string(c.CatchOutput(func() {
			params := c.translateParams(e.Type)
			c.Printf("(function(%s) {", strings.Join(params, ", "))
			c.Indent(func() {
				c.translateFunctionBody(e.Body.List, exprType.(*types.Signature), params)
			})
			c.Printf("})")
		})))

	case *ast.UnaryExpr:
		op := e.Op.String()
		switch e.Op {
		case token.AND:
			if _, isStruct := c.info.Types[e.X].Underlying().(*types.Struct); !isStruct {
				v := ast.NewIdent("v")
				c.info.Types[v] = c.info.Types[e.X]
				assignStmt := &ast.AssignStmt{
					Lhs: []ast.Expr{e.X},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{v},
				}
				assign := strings.TrimSpace(string(c.CatchOutput(func() { c.translateStmt(assignStmt, "") })))
				return fmt.Sprintf("new %s(function() { return %s; }, function(v) { %s })", c.typeName(exprType), c.translateExpr(e.X), assign)
			}
			return c.translateExpr(e.X)
		case token.XOR:
			op = "~"
		case token.ARROW:
			return "undefined"
		}
		t := c.info.Types[e.X]
		basic := t.Underlying().(*types.Basic)
		if is64Bit(basic) {
			x := c.newVariable("x")
			return fmt.Sprintf("(%s = %s, new %s(%s%s.high, %s%s.low))", x, c.translateExpr(e.X), c.typeName(t), op, x, op, x)
		}
		value := fmt.Sprintf("%s%s", op, c.translateExpr(e.X))
		value = fixNumber(value, basic)
		return value

	case *ast.BinaryExpr:
		t := c.info.Types[e.X]
		if basic, isBasic := t.(*types.Basic); isBasic && basic.Kind() == types.UntypedNil {
			t = c.info.Types[e.Y]
		}
		ex := c.translateExprToType(e.X, t)
		ey := c.translateExprToType(e.Y, t)
		op := e.Op.String()
		if e.Op == token.AND_NOT {
			op = "&~"
		}

		basic, isBasic := t.Underlying().(*types.Basic)
		if isBasic && is64Bit(basic) {
			var expr string = "0"
			switch e.Op {
			case token.MUL:
				return fmt.Sprintf("Go$mul64(%s, %s)", ex, ey)
			case token.QUO:
				return fmt.Sprintf("Go$div64(%s, %s, false)", ex, ey)
			case token.REM:
				return fmt.Sprintf("Go$div64(%s, %s, true)", ex, ey)
			case token.SHL:
				return fmt.Sprintf("Go$shiftLeft64(%s, %s)", ex, c.translateExprToType(e.Y, types.Typ[types.Uint]))
			case token.SHR:
				return fmt.Sprintf("Go$shiftRight%s(%s, %s)", toJavaScriptType(basic), ex, c.translateExprToType(e.Y, types.Typ[types.Uint]))
			case token.EQL:
				expr = "x.high === y.high && x.low === y.low"
			case token.NEQ:
				expr = "x.high !== y.high || x.low !== y.low"
			case token.LSS:
				expr = "x.high < y.high || (x.high === y.high && x.low < y.low)"
			case token.LEQ:
				expr = "x.high < y.high || (x.high === y.high && x.low <= y.low)"
			case token.GTR:
				expr = "x.high > y.high || (x.high === y.high && x.low > y.low)"
			case token.GEQ:
				expr = "x.high > y.high || (x.high === y.high && x.low >= y.low)"
			case token.ADD, token.SUB:
				expr = fmt.Sprintf("new %s(x.high %s y.high, x.low %s y.low)", c.typeName(t), op, op)
			case token.AND, token.OR, token.XOR, token.AND_NOT:
				expr = fmt.Sprintf("new %s(x.high %s y.high, (x.low %s y.low) >>> 0)", c.typeName(t), op, op)
			default:
				panic(e.Op)
			}
			x := c.newVariable("x")
			y := c.newVariable("y")
			expr = strings.Replace(expr, "x", x, -1)
			expr = strings.Replace(expr, "y", y, -1)
			return fmt.Sprintf("(%s = %s, %s = %s, %s)", x, ex, y, ey, expr)
		}

		var value string
		switch e.Op {
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
					conds = append(conds, fmt.Sprintf("%s.%s === %s.%s", x, field.Name(), y, field.Name()))
				}
				if len(conds) == 0 {
					conds = []string{"true"}
				}
				return fmt.Sprintf("(%s = %s, %s = %s, %s)", x, ex, y, ey, strings.Join(conds, " && "))
			case *types.Interface:
				return fmt.Sprintf("Go$interfaceIsEqual(%s, %s)", ex, ey)
			case *types.Pointer:
				xUnary, xIsUnary := e.X.(*ast.UnaryExpr)
				yUnary, yIsUnary := e.Y.(*ast.UnaryExpr)
				if xIsUnary && xUnary.Op == token.AND && yIsUnary && yUnary.Op == token.AND {
					xIndex, xIsIndex := xUnary.X.(*ast.IndexExpr)
					yIndex, yIsIndex := yUnary.X.(*ast.IndexExpr)
					if xIsIndex && yIsIndex {
						return fmt.Sprintf("Go$sliceIsEqual(%s, %s, %s, %s)", c.translateExpr(xIndex.X), c.translateExpr(xIndex.Index), c.translateExpr(yIndex.X), c.translateExpr(yIndex.Index))
					}
				}
			}
			return ex + " === " + ey
		case token.NEQ:
			return "!" + c.translateExpr(&ast.BinaryExpr{
				X:  e.X,
				Op: token.EQL,
				Y:  e.Y,
			})
		case token.MUL:
			if basic.Kind() == types.Int32 {
				x := c.newVariable("x")
				y := c.newVariable("y")
				return fmt.Sprintf("(%s = %s, %s = %s, (((%s >>> 16 << 16) * %s | 0) + (%s << 16 >>> 16) * %s) | 0)", x, ex, y, ey, x, y, x, y)
			}
			if basic.Kind() == types.Uint32 {
				x := c.newVariable("x")
				y := c.newVariable("y")
				return fmt.Sprintf("(%s = %s, %s = %s, (((%s >>> 16 << 16) * %s >>> 0) + (%s << 16 >>> 16) * %s) >>> 0)", x, ex, y, ey, x, y, x, y)
			}
			value = ex + " * " + ey
		case token.QUO:
			if basic.Info()&types.IsInteger != 0 {
				value = fmt.Sprintf("Math.floor(%s / %s)", ex, ey)
				break
			}
			value = ex + " / " + ey
		case token.SHL, token.SHR:
			if e.Op == token.SHR && basic.Info()&types.IsUnsigned != 0 {
				op = ">>>"
			}
			switch basic.Kind() {
			case types.Int32, types.Uint32, types.Uintptr:
				y := c.newVariable("y")
				value = fmt.Sprintf("(%s = %s, %s < 32 ? (%s %s %s) : 0)", y, c.translateExprToType(e.Y, types.Typ[types.Uint]), y, ex, op, y)
			default:
				value = "(" + ex + " " + op + " " + ey + ")"
			}
		case token.AND, token.OR, token.XOR, token.AND_NOT:
			value = "(" + ex + " " + op + " " + ey + ")"
		default:
			value = ex + " " + op + " " + ey
		}

		if isBasic {
			switch e.Op {
			case token.ADD, token.SUB, token.MUL, token.QUO, token.AND, token.OR, token.XOR, token.AND_NOT, token.SHL, token.SHR:
				value = fixNumber(value, basic)
			}
		}

		return value

	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", c.translateExpr(e.X))

	case *ast.IndexExpr:
		x := c.translateExpr(e.X)
		xType := c.info.Types[e.X]
		if ptr, isPointer := xType.(*types.Pointer); isPointer {
			xType = ptr.Elem()
		}
		switch t := xType.Underlying().(type) {
		case *types.Array:
			return fmt.Sprintf("%s[%s]", x, c.translateExprToType(e.Index, types.Typ[types.Int]))
		case *types.Slice:
			return fmt.Sprintf("(Go$obj = %s, Go$obj.array[Go$obj.offset + %s])", x, c.translateExprToType(e.Index, types.Typ[types.Int]))
		case *types.Map:
			index := c.translateExprToType(e.Index, t.Key())
			if hasId(t.Key()) {
				index = fmt.Sprintf("(%s || Go$Map.Go$nil).Go$key()", index)
			}
			if _, isTuple := exprType.(*types.Tuple); isTuple {
				return fmt.Sprintf(`(Go$obj = (%s || false)[%s], Go$obj !== undefined ? [Go$obj.v, true] : [%s, false])`, x, index, c.zeroValue(t.Elem()))
			}
			return fmt.Sprintf(`(Go$obj = (%s || false)[%s], Go$obj !== undefined ? Go$obj.v : %s)`, x, index, c.zeroValue(t.Elem()))
		case *types.Basic:
			return fmt.Sprintf("%s.charCodeAt(%s)", x, c.translateExprToType(e.Index, types.Typ[types.Int]))
		default:
			panic(fmt.Sprintf("Unhandled IndexExpr: %T\n", t))
		}

	case *ast.SliceExpr:
		b, isBasic := c.info.Types[e.X].(*types.Basic)
		isString := isBasic && b.Info()&types.IsString != 0
		slice := c.translateExprToType(e.X, exprType)
		if e.High == nil {
			if e.Low == nil {
				return slice
			}
			if isString {
				return fmt.Sprintf("%s.substring(%s)", slice, c.translateExpr(e.Low))
			}
			return fmt.Sprintf("Go$subslice(%s, %s)", slice, c.translateExpr(e.Low))
		}
		low := "0"
		if e.Low != nil {
			low = c.translateExpr(e.Low)
		}
		if isString {
			return fmt.Sprintf("%s.substring(%s, %s)", slice, low, c.translateExpr(e.High))
		}
		return fmt.Sprintf("Go$subslice(%s, %s, %s)", slice, low, c.translateExpr(e.High))

	case *ast.SelectorExpr:
		sel := c.info.Selections[e]
		parameterName := func(v *types.Var) string {
			if v.Anonymous() {
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
			return c.translateExpr(e.X) + "." + translateSelection(sel)
		case types.MethodVal:
			parameters := makeParametersList()
			recv := c.newVariable("_recv")
			return fmt.Sprintf("(%s = %s, function(%s) { return %s.%s(%s); })", recv, c.translateExpr(e.X), strings.Join(parameters, ", "), recv, e.Sel.Name, strings.Join(parameters, ", "))
		case types.MethodExpr:
			parameters := append([]string{parameterName(sel.Obj().Type().(*types.Signature).Recv())}, makeParametersList()...)
			return fmt.Sprintf("(function(%s) { return %s.prototype.%s.call(%s); })", strings.Join(parameters, ", "), c.typeName(sel.Recv()), sel.Obj().(*types.Func).Name(), strings.Join(parameters, ", "))
		case types.PackageObj:
			return fmt.Sprintf("%s.%s", c.translateExpr(e.X), e.Sel.Name)
		}
		panic("")

	case *ast.CallExpr:
		switch f := e.Fun.(type) {
		case *ast.Ident:
			switch o := c.info.Objects[f].(type) {
			case *types.Builtin:
				switch o.Name() {
				case "new":
					elemType := c.info.Types[e.Args[0]]
					if types.IsIdentical(elemType, types.Typ[types.Uintptr]) {
						return "new Uint8Array(8)"
					}
					return c.zeroValue(elemType)
				case "make":
					switch t2 := c.info.Types[e.Args[0]].Underlying().(type) {
					case *types.Slice:
						if len(e.Args) == 3 {
							return fmt.Sprintf("Go$subslice(new %s(Go$makeArray(%s, %s, function() { return %s; })), 0, %s)", c.typeName(c.info.Types[e.Args[0]]), toArrayType(t2.Elem()), c.translateExpr(e.Args[2]), c.zeroValue(t2.Elem()), c.translateExpr(e.Args[1]))
						}
						return fmt.Sprintf("new %s(Go$makeArray(%s, %s, function() { return %s; }))", c.typeName(c.info.Types[e.Args[0]]), toArrayType(t2.Elem()), c.translateExpr(e.Args[1]), c.zeroValue(t2.Elem()))
					default:
						args := []string{"undefined"}
						for _, arg := range e.Args[1:] {
							args = append(args, c.translateExpr(arg))
						}
						return fmt.Sprintf("new %s(%s)", c.typeName(c.info.Types[e.Args[0]]), strings.Join(args, ", "))
					}
				case "len":
					arg := c.translateExpr(e.Args[0])
					argType := c.info.Types[e.Args[0]]
					switch argt := argType.Underlying().(type) {
					case *types.Basic, *types.Array, *types.Slice:
						return fmt.Sprintf("%s.length", arg)
					case *types.Map:
						return fmt.Sprintf("(Go$obj = %s, Go$obj !== null ? Go$keys(Go$obj).length : 0)", arg)
					default:
						panic(fmt.Sprintf("Unhandled len type: %T\n", argt))
					}
				case "cap":
					arg := c.translateExpr(e.Args[0])
					argType := c.info.Types[e.Args[0]]
					switch argt := argType.Underlying().(type) {
					case *types.Array:
						return fmt.Sprintf("%s.length", arg)
					case *types.Slice:
						return fmt.Sprintf("(Go$obj = %s, Go$obj !== null ? Go$obj.array.length : 0)", arg)
					default:
						panic(fmt.Sprintf("Unhandled cap type: %T\n", argt))
					}
				case "panic":
					return fmt.Sprintf("throw new Go$Panic(%s)", c.translateExprToType(e.Args[0], types.NewInterface(nil)))
				case "append":
					if e.Ellipsis.IsValid() {
						return fmt.Sprintf("Go$append(%s, %s)", c.translateExpr(e.Args[0]), c.translateExprToType(e.Args[1], exprType))
					}
					sliceType := exprType.Underlying().(*types.Slice)
					toAppend := createListComposite(sliceType.Elem(), c.translateExprSlice(e.Args[1:], sliceType.Elem()))
					return fmt.Sprintf("Go$append(%s, new %s(%s))", c.translateExpr(e.Args[0]), c.typeName(exprType), toAppend)
				case "delete":
					index := c.translateExpr(e.Args[1])
					if hasId(c.info.Types[e.Args[0]].Underlying().(*types.Map).Key()) {
						index = fmt.Sprintf("(%s || Go$Map.Go$nil).Go$key()", index)
					}
					return fmt.Sprintf(`delete %s[%s]`, c.translateExpr(e.Args[0]), index)
				case "copy":
					return fmt.Sprintf("Go$copy(%s, %s)", c.translateExprToType(e.Args[0], types.NewSlice(types.Typ[types.Byte])), c.translateExprToType(e.Args[1], types.NewSlice(types.Typ[types.Byte])))
				case "print", "println":
					return fmt.Sprintf("console.log(%s)", strings.Join(c.translateExprSlice(e.Args, nil), ", "))
				case "recover", "complex", "real", "imag":
					return fmt.Sprintf("Go$%s(%s)", o.Name(), strings.Join(c.translateExprSlice(e.Args, nil), ", "))
				default:
					panic(fmt.Sprintf("Unhandled builtin: %s\n", o.Name()))
				}
			case *types.TypeName: // conversion
				return c.translateExprToType(e.Args[0], o.Type())
			}
		}

		funType := c.info.Types[e.Fun]
		sig, isSig := funType.Underlying().(*types.Signature)
		if !isSig { // conversion
			if call, isCall := e.Args[0].(*ast.CallExpr); isCall {
				if types.IsIdentical(c.info.Types[call.Fun], types.Typ[types.UnsafePointer]) {
					return c.translateExpr(call.Args[0]) + "." + c.typeName(funType) // unsafe conversion
				}
			}
			return c.translateExprToType(e.Args[0], funType)
		}

		var fun string
		switch f := e.Fun.(type) {
		case *ast.SelectorExpr:
			sel := c.info.Selections[f]

			switch sel.Kind() {
			case types.FieldVal:
				fun = fmt.Sprintf("%s.%s", c.translateExpr(f.X), translateSelection(sel))
			case types.MethodVal:
				methodsRecvType := sel.Obj().(*types.Func).Type().(*types.Signature).Recv().Type()
				_, pointerExpected := methodsRecvType.(*types.Pointer)
				_, isPointer := sel.Recv().Underlying().(*types.Pointer)
				_, isStruct := sel.Recv().Underlying().(*types.Struct)
				if pointerExpected && !isPointer && !isStruct {
					target := c.translateExpr(f.X)
					fun = fmt.Sprintf("(new %s(function() { return %s; }, function(v) { %s = v; })).%s", c.typeName(methodsRecvType), target, target, f.Sel.Name)
					break
				}
				if isWrapped(sel.Recv()) {
					fun = fmt.Sprintf("(new %s(%s)).%s", c.typeName(sel.Recv()), c.translateExpr(f.X), f.Sel.Name)
					break
				}
				fun = fmt.Sprintf("%s.%s", c.translateExpr(f.X), f.Sel.Name)
			case types.PackageObj:
				fun = fmt.Sprintf("%s.%s", c.translateExpr(f.X), f.Sel.Name)
			default:
				panic("")
			}
		case *ast.Ident, *ast.FuncLit, *ast.ParenExpr:
			fun = c.translateExpr(e.Fun)
		default:
			panic(fmt.Sprintf("Unhandled expression: %T\n", f))
		}
		if sig.Params().Len() > 1 && len(e.Args) == 1 && !sig.IsVariadic() {
			argRefs := make([]string, sig.Params().Len())
			for i := range argRefs {
				argRefs[i] = fmt.Sprintf("Go$tuple[%d]", i)
			}
			return fmt.Sprintf("(Go$tuple = %s, %s(%s))", c.translateExpr(e.Args[0]), fun, strings.Join(argRefs, ", "))
		}
		return fmt.Sprintf("%s(%s)", fun, strings.Join(c.translateArgs(e), ", "))

	case *ast.StarExpr:
		if c1, isCall := e.X.(*ast.CallExpr); isCall && len(c1.Args) == 1 {
			if c2, isCall := c1.Args[0].(*ast.CallExpr); isCall && len(c2.Args) == 1 && types.IsIdentical(c.info.Types[c2.Fun], types.Typ[types.UnsafePointer]) {
				if unary, isUnary := c2.Args[0].(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
					return c.translateExpr(unary.X) // unsafe conversion
				}
			}
		}
		switch exprType.Underlying().(type) {
		case *types.Struct, *types.Array:
			return c.clone(c.translateExpr(e.X), exprType)
		}
		return c.translateExpr(e.X) + ".Go$get()"

	case *ast.TypeAssertExpr:
		if e.Type == nil {
			return c.translateExpr(e.X)
		}
		t := c.info.Types[e.Type]
		check := "Go$obj !== null && " + c.typeCheck("Go$obj.constructor", t)
		value := "Go$obj"
		if _, isInterface := t.Underlying().(*types.Interface); !isInterface {
			value += ".Go$val"
		}
		if _, isTuple := exprType.(*types.Tuple); isTuple {
			return fmt.Sprintf("(Go$obj = %s, %s ? [%s, true] : [%s, false])", c.translateExpr(e.X), check, value, c.zeroValue(c.info.Types[e.Type]))
		}
		return fmt.Sprintf("(Go$obj = %s, %s ? %s : Go$typeAssertionFailed(Go$obj))", c.translateExpr(e.X), check, value)

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
		case *types.Builtin:
			if e.Name == "recover" {
				return "Go$recover"
			}
			panic(fmt.Sprintf("Unhandled object: %T\n", o))
		case *types.Nil:
			return c.zeroValue(c.info.Types[e])
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

func (c *PkgContext) translateExprSlice(exprs []ast.Expr, desiredType types.Type) []string {
	parts := make([]string, len(exprs))
	for i, expr := range exprs {
		parts[i] = c.translateExprToType(expr, desiredType)
	}
	return parts
}

func (c *PkgContext) translateExprToType(expr ast.Expr, desiredType types.Type) string {
	if desiredType == nil {
		return c.translateExpr(expr)
	}
	if expr == nil {
		return c.zeroValue(desiredType)
	}

	exprType := c.info.Types[expr]
	if basicLit, isBasicLit := expr.(*ast.BasicLit); isBasicLit && basicLit.Kind == token.STRING {
		exprType = types.Typ[types.String]
	}
	basicExprType, isBasicExpr := exprType.Underlying().(*types.Basic)
	if isBasicExpr && basicExprType.Kind() == types.UntypedNil {
		return c.zeroValue(desiredType)
	}

	switch t := desiredType.Underlying().(type) {
	case *types.Basic:
		switch {
		case t.Info()&types.IsInteger != 0:
			value := c.translateExpr(expr)

			if basicExprType.Info()&types.IsFloat != 0 {
				value = fmt.Sprintf("Math.floor(%s)", value)
			}

			switch {
			case is64Bit(t):
				switch {
				case !is64Bit(basicExprType):
					value = fmt.Sprintf("new %s(0, %s)", c.typeName(desiredType), value)
				case !types.IsIdentical(exprType, desiredType):
					value = fmt.Sprintf("(Go$obj = %s, new %s(Go$obj.high, Go$obj.low))", value, c.typeName(desiredType))
				}
			case is64Bit(basicExprType):
				switch t.Info()&types.IsUnsigned != 0 {
				case true:
					value += ".low"
				case false:
					value = fmt.Sprintf("(%s.low | 0)", value)
				}
			}

			return value
		case t.Info()&types.IsFloat != 0:
			if is64Bit(exprType.Underlying().(*types.Basic)) {
				return fmt.Sprintf("(Go$obj = %s, Go$obj.high * 4294967296 + Go$obj.low)", c.translateExpr(expr))
			}
		case t.Info()&types.IsString != 0:
			value := c.translateExpr(expr)
			switch et := exprType.Underlying().(type) {
			case *types.Basic:
				if is64Bit(et) {
					value += ".low"
				}
				if et.Info()&types.IsNumeric != 0 {
					return fmt.Sprintf("Go$charToString(%s)", value)
				}
				return value
			case *types.Slice:
				if types.IsIdentical(et.Elem().Underlying(), types.Typ[types.Rune]) {
					return fmt.Sprintf("Go$runesToString(%s)", value)
				}
				return fmt.Sprintf("Go$bytesToString(%s)", value)
			default:
				panic(fmt.Sprintf("Unhandled conversion: %v\n", et))
			}
		case t.Kind() == types.UnsafePointer:
			if unary, isUnary := expr.(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
				if indexExpr, isIndexExpr := unary.X.(*ast.IndexExpr); isIndexExpr {
					return fmt.Sprintf("Go$sliceToArray(%s)", c.translateExprToType(indexExpr.X, types.NewSlice(nil)))
				}
				if ident, isIdent := unary.X.(*ast.Ident); isIdent && ident.Name == "_zero" {
					return "new Uint8Array(0)"
				}
			}
			if ptr, isPtr := c.info.Types[expr].(*types.Pointer); isPtr {
				if s, isStruct := ptr.Elem().Underlying().(*types.Struct); isStruct {
					array := c.newVariable("_array")
					target := c.newVariable("_struct")
					c.Printf("%s = new Uint8Array(%d);", array, sizes32.Sizeof(s))
					c.Delayed(func() {
						c.Printf("%s = %s;", target, c.translateExpr(expr))
						c.loadStruct(array, target, s)
					})
					return array
				}
			}
		}

	case *types.Slice:
		switch et := exprType.Underlying().(type) {
		case *types.Basic:
			if et.Info()&types.IsString != 0 {
				if types.IsIdentical(t.Elem().Underlying(), types.Typ[types.Rune]) {
					return fmt.Sprintf("new %s(Go$stringToRunes(%s))", c.typeName(desiredType), c.translateExpr(expr))
				}
				return fmt.Sprintf("new %s(Go$stringToBytes(%s))", c.typeName(desiredType), c.translateExpr(expr))
			}
		case *types.Array, *types.Pointer:
			return fmt.Sprintf("new Go$Slice(%s)", c.translateExpr(expr))
		}
		_, desiredIsNamed := desiredType.(*types.Named)
		if desiredIsNamed && !types.IsIdentical(exprType, desiredType) {
			return fmt.Sprintf("(Go$obj = %s, Go$subslice(new %s(Go$obj.array), Go$obj.offset, Go$obj.offset + Go$obj.length))", c.translateExpr(expr), c.typeName(desiredType))
		}
		return c.translateExpr(expr)

	case *types.Interface:
		if isWrapped(exprType) {
			return fmt.Sprintf("new %s(%s)", c.typeName(exprType), c.translateExpr(expr))
		}
		if named, isNamed := exprType.(*types.Named); isNamed {
			if _, isStruct := exprType.Underlying().(*types.Struct); isStruct {
				return fmt.Sprintf("new %s.Go$NonPointer(%s)", c.objectName(named.Obj()), c.translateExpr(expr))
			}
		}

	case *types.Pointer:
		s, isStruct := t.Elem().Underlying().(*types.Struct)
		if isStruct && types.IsIdentical(exprType, types.Typ[types.UnsafePointer]) {
			array := c.newVariable("_array")
			target := c.newVariable("_struct")
			c.Printf("%s = %s;", array, c.translateExpr(expr))
			c.Printf("%s = %s;", target, c.zeroValue(t.Elem()))
			c.loadStruct(array, target, s)
			return target
		}

	case *types.Struct, *types.Array:
		if _, isComposite := expr.(*ast.CompositeLit); !isComposite {
			return c.clone(c.translateExpr(expr), desiredType)
		}

	case *types.Chan, *types.Map, *types.Signature:
		// no converion

	default:
		panic(fmt.Sprintf("Unhandled conversion: %v\n", t))
	}

	return c.translateExpr(expr)
}

func (c *PkgContext) clone(src string, ty types.Type) string {
	named, isNamed := ty.(*types.Named)
	switch t := ty.Underlying().(type) {
	case *types.Struct:
		structVar := c.newVariable("_struct")
		fields := make([]string, t.NumFields())
		for i := range fields {
			field := t.Field(i)
			if !isNamed {
				fields[i] = field.Name() + ": "
			}
			fields[i] += c.clone(structVar+"."+field.Name(), field.Type())
		}
		if isNamed {
			return fmt.Sprintf("(%s = %s, new %s(%s))", structVar, src, c.objectName(named.Obj()), strings.Join(fields, ", "))
		}
		return fmt.Sprintf("(%s = %s, {%s})", structVar, src, strings.Join(fields, ", "))
	case *types.Array:
		return fmt.Sprintf("Go$mapArray(%s, function(entry) { return %s; })", src, c.clone("entry", t.Elem()))
	default:
		return src
	}
}

func (c *PkgContext) loadStruct(array, target string, s *types.Struct) {
	view := c.newVariable("_view")
	c.Printf("%s = new DataView(%s.buffer, %s.byteOffset);", view, array, array)
	var fields []*types.Var
	var collectFields func(s *types.Struct, path string)
	collectFields = func(s *types.Struct, path string) {
		for i := 0; i < s.NumFields(); i++ {
			field := s.Field(i)
			if fs, isStruct := field.Type().Underlying().(*types.Struct); isStruct {
				collectFields(fs, path+"."+field.Name())
				continue
			}
			fields = append(fields, types.NewVar(0, nil, path+"."+field.Name(), field.Type()))
		}
	}
	collectFields(s, target)
	offsets := sizes32.Offsetsof(fields)
	for i, field := range fields {
		switch t := field.Type().Underlying().(type) {
		case *types.Basic:
			if t.Info()&types.IsNumeric != 0 {
				if is64Bit(t) {
					c.Printf("%s = new %s(%s.getUint32(%d, true), %s.getUint32(%d, true));", field.Name(), c.typeName(field.Type()), view, offsets[i]+4, view, offsets[i])
					continue
				}
				c.Printf("%s = %s.get%s(%d, true);", field.Name(), view, toJavaScriptType(t), offsets[i])
			}
			continue
		case *types.Array:
			c.Printf("%s = new %s(%s.buffer, Go$min(%s.byteOffset + %d, %s.buffer.byteLength));", field.Name(), toArrayType(t.Elem()), array, array, offsets[i], array)
			continue
		}
		c.Printf("// skipped: %s %s", field.Name(), field.Type().String())
	}
}

func (c *PkgContext) typeCheck(of string, to types.Type) string {
	if in, isInterface := to.Underlying().(*types.Interface); isInterface {
		if in.MethodSet().Len() == 0 {
			return "true"
		}
		return fmt.Sprintf("%s.Go$implementedBy.indexOf(%s) !== -1", c.typeName(to), of)
	}
	return of + " === " + c.typeName(to)
}

func translateSelection(sel *types.Selection) string {
	var selectors []string
	t := sel.Recv()
	for _, index := range sel.Index() {
		if ptr, isPtr := t.(*types.Pointer); isPtr {
			t = ptr.Elem()
		}
		s := t.Underlying().(*types.Struct)
		field := s.Field(index)
		selectors = append(selectors, field.Name())
		t = field.Type()
	}
	return strings.Join(selectors, ".")
}

func fixNumber(value string, basic *types.Basic) string {
	switch basic.Kind() {
	case types.Int32:
		return "(" + value + " | 0)"
	case types.Uint32, types.Uintptr:
		return "(" + value + " >>> 0)"
	}
	return value
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

type HasEvilConstantVisitor struct {
	hasEvilConstant bool
}

func (v *HasEvilConstantVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if v.hasEvilConstant {
		return nil
	}
	switch n := node.(type) {
	case *ast.Ident:
		switch n.Name {
		case "MaxFloat32", "SmallestNonzeroFloat32", "MaxFloat64", "SmallestNonzeroFloat64":
			v.hasEvilConstant = true
			return nil
		}
	}
	return v
}
