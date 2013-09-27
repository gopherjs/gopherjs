package main

import (
	"bytes"
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func (c *PkgContext) translateExpr(expr ast.Expr) string {
	if value, valueFound := c.info.Values[expr]; valueFound {
		switch value.Kind() {
		case exact.Nil:
			return "null"
		case exact.Bool:
			return fmt.Sprintf("%t", exact.BoolVal(value))
		case exact.Int:
			if c.info.Types[expr].Underlying().(*types.Basic).Kind() == types.Uint64 {
				d, _ := exact.Uint64Val(value)
				return fmt.Sprintf("new Go$Uint64(%d, %d)", d>>32, d&(1<<32-1))
			}
			d, _ := exact.Int64Val(value)
			return fmt.Sprintf("%d", d)
		case exact.Float:
			f, _ := exact.Float64Val(value)
			return fmt.Sprintf("%f", f)
		case exact.Complex:
			f, _ := exact.Float64Val(exact.Real(value))
			return fmt.Sprintf("%f", f)
		case exact.String:
			buffer := bytes.NewBuffer(nil)
			for _, r := range exact.StringVal(value) {
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
					if r > 0xFFFF {
						panic("Too big unicode character in string.")
					}
					if r < 0x20 || r > 0x7E {
						fmt.Fprintf(buffer, `\u%04x`, r)
						continue
					}
					buffer.WriteRune(r)
				}
			}
			return `"` + buffer.String() + `"`
		default:
			panic("Unhandled value: " + value.String())
		}
	}

	switch e := expr.(type) {
	case *ast.CompositeLit:
		compType := c.info.Types[e]
		if ptrType, isPointer := compType.(*types.Pointer); isPointer {
			compType = ptrType.Elem()
		}

		var elements []string
		collectIndexedElements := func(elementType types.Type, length int64) {
			elements = make([]string, 0, length)
			var i int64 = 0
			zero := c.zeroValue(elementType)
			for _, element := range e.Elts {
				if kve, isKve := element.(*ast.KeyValueExpr); isKve {
					key, _ := exact.Int64Val(c.info.Values[kve.Key])
					for i < key {
						elements = append(elements, zero)
						i += 1
					}
					element = kve.Value
				}
				elements = append(elements, c.translateExpr(element))
				i += 1
			}
			for i < length {
				elements = append(elements, zero)
				i += 1
			}
		}
		switch t := compType.Underlying().(type) {
		case *types.Array:
			collectIndexedElements(t.Elem(), t.Len())
		case *types.Slice:
			collectIndexedElements(t.Elem(), 0)
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
					elements[i] = c.translateExpr(element)
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
							elements[j] = c.translateExpr(kve.Value)
							break
						}
					}
				}
			}
		}

		switch t := compType.(type) {
		case *types.Array:
			return createListComposite(t.Elem(), elements)
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
				return fmt.Sprintf("new %s(%s)", c.typeName(t), strings.Join(elements, ", "))
			default:
				panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", u))
			}
		default:
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", t))
		}

	case *ast.FuncLit:
		n := c.usedVarNames
		defer func() { c.usedVarNames = n }()
		body := c.CatchOutput(func() {
			c.Indent(func() {
				t := c.info.Types[e].(*types.Signature)
				var resultNames []ast.Expr
				if t.Results().Len() != 0 && t.Results().At(0).Name() != "" {
					resultNames = make([]ast.Expr, t.Results().Len())
					for i := 0; i < t.Results().Len(); i++ {
						result := t.Results().At(i)
						name := result.Name()
						if result.Name() == "_" {
							name = "result"
						}
						id := ast.NewIdent(name)
						c.info.Types[id] = result.Type()
						c.info.Objects[id] = result
						c.Printf("var %s = %s;", c.translateExpr(id), c.zeroValue(result.Type()))
						resultNames[i] = id
					}
				}
				r := c.resultNames
				defer func() { c.resultNames = r }()
				c.resultNames = resultNames

				v := HasDeferVisitor{}
				ast.Walk(&v, e.Body)
				if v.hasDefer {
					c.Printf("var Go$deferred = [];")
					c.Printf("try {")
					c.Indent(func() {
						c.translateStmtList(e.Body.List)
					})
					c.Printf("} catch(err) {")
					c.Indent(func() {
						c.Printf("Go$errorStack.push({ frame: Go$getStackDepth(), error: err });")
					})
					c.Printf("} finally {")
					c.Indent(func() {
						c.Printf("Go$callDeferred(Go$deferred);")
						if resultNames != nil {
							c.translateStmt(&ast.ReturnStmt{}, "")
						}
					})
					c.Printf("}")
					return
				}
				c.translateStmtList(e.Body.List)
			})
			c.Printf("")
		})
		return fmt.Sprintf("(function(%s) {\n%s})", c.translateParams(e.Type), body[:len(body)-1])

	case *ast.UnaryExpr:
		op := e.Op.String()
		switch e.Op {
		case token.AND:
			target := c.translateExpr(e.X)
			if _, isStruct := c.info.Types[e.X].Underlying().(*types.Struct); !isStruct {
				return fmt.Sprintf("new %s(function() { return %s; }, function(v) { %s = v; })", c.typeName(c.info.Types[e]), target, target)
			}
			return target
		case token.XOR:
			op = "~"
			basic := c.info.Types[e.X].Underlying().(*types.Basic)
			if basic.Kind() == types.Uint64 {
				return fmt.Sprintf("(Go$x = %s, new Go$Uint64(~Go$x.high, ~Go$x.low))", c.translateExpr(e.X))
			}
		}
		return fmt.Sprintf("%s%s", op, c.translateExpr(e.X))

	case *ast.BinaryExpr:
		ex := c.translateExpr(e.X)
		ey := c.translateExpr(e.Y)
		op := e.Op.String()
		if e.Op == token.AND_NOT {
			op = "&~"
		}

		if basic, isBasic := c.info.Types[e.X].Underlying().(*types.Basic); isBasic && basic.Kind() == types.Uint64 {
			var expr string = "0"
			switch e.Op {
			case token.ADD:
				return fmt.Sprintf("Go$addUint64(%s, %s)", ex, ey)
			case token.SUB:
				return fmt.Sprintf("Go$subUint64(%s, %s)", ex, ey)
			case token.MUL:
				return fmt.Sprintf("Go$mulUint64(%s, %s)", ex, ey)
			case token.QUO:
				return fmt.Sprintf("Go$divUint64(%s, %s, false)", ex, ey)
			case token.REM:
				return fmt.Sprintf("Go$divUint64(%s, %s, true)", ex, ey)
			case token.EQL:
				expr = "Go$x.high === Go$y.high && Go$x.low === Go$y.low"
			case token.NEQ:
				expr = "Go$x.high !== Go$y.high || Go$x.low !== Go$y.low"
			case token.LSS:
				expr = "Go$x.high < Go$y.high || (Go$x.high === Go$y.high && Go$x.low < Go$y.low)"
			case token.LEQ:
				expr = "Go$x.high < Go$y.high || (Go$x.high === Go$y.high && Go$x.low <= Go$y.low)"
			case token.GTR:
				expr = "Go$x.high > Go$y.high || (Go$x.high === Go$y.high && Go$x.low > Go$y.low)"
			case token.GEQ:
				expr = "Go$x.high > Go$y.high || (Go$x.high === Go$y.high && Go$x.low >= Go$y.low)"
			case token.AND, token.OR, token.XOR, token.AND_NOT:
				expr = fmt.Sprintf("new Go$Uint64(((Go$x.high %s Go$y.high) + 4294967296) %% 4294967296, ((Go$x.low %s Go$y.low) + 4294967296) %% 4294967296)", op, op)
			case token.SHL:
				return fmt.Sprintf("Go$shiftUint64(%s, %s)", ex, ey)
			case token.SHR:
				return fmt.Sprintf("Go$shiftUint64(%s, -%s)", ex, ey)
			default:
				panic(e.Op)
			}
			return fmt.Sprintf("(Go$x = %s, Go$y = %s, %s)", ex, ey, expr)
		}

		switch e.Op {
		case token.QUO:
			if c.info.Types[e].Underlying().(*types.Basic).Info()&types.IsInteger != 0 {
				return fmt.Sprintf("Math.floor(%s / %s)", ex, ey)
			}
		case token.EQL:
			ix, xIsI := c.info.Types[e.X].(*types.Interface)
			iy, yIsI := c.info.Types[e.Y].(*types.Interface)
			if xIsI && ix.MethodSet().Len() == 0 && yIsI && iy.MethodSet().Len() == 0 {
				return fmt.Sprintf("_isEqual(%s, %s)", ex, ey)
			}
			op = "==="
		case token.NEQ:
			op = "!=="
		}

		switch e.Op {
		case token.AND, token.OR, token.XOR, token.AND_NOT, token.SHL, token.SHR:
			return fmt.Sprintf("(%s %s %s)", ex, op, ey)
		default:
			return fmt.Sprintf("%s %s %s", ex, op, ey)
		}

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
			return fmt.Sprintf("%s.Go$get(%s)", x, c.translateExprToType(e.Index, types.Typ[types.Int]))
		case *types.Map:
			index := c.translateExpr(e.Index)
			if hasId(t.Key()) {
				index = fmt.Sprintf("(%s || Go$nil).Go$id", index)
			}
			if _, isTuple := c.info.Types[e].(*types.Tuple); isTuple {
				return fmt.Sprintf("(Go$obj = (%s || Go$nil)[%s], Go$obj !== undefined ? [Go$obj.v, true] : [%s, false])", x, index, c.zeroValue(t.Elem()))
			}
			return fmt.Sprintf("(Go$obj = (%s || Go$nil)[%s], Go$obj !== undefined ? Go$obj.v : %s)", x, index, c.zeroValue(t.Elem()))
		case *types.Basic:
			return fmt.Sprintf("%s.charCodeAt(%s)", x, c.translateExprToType(e.Index, types.Typ[types.Int]))
		default:
			panic(fmt.Sprintf("Unhandled IndexExpr: %T\n", t))
		}

	case *ast.SliceExpr:
		method := "Go$subslice"
		if b, ok := c.info.Types[e.X].(*types.Basic); ok && b.Info()&types.IsString != 0 {
			method = "substring"
		}
		slice := c.translateExpr(e.X)
		if _, ok := c.info.Types[e.X].(*types.Array); ok {
			slice = fmt.Sprintf("new Go$Slice(%s)", slice)
		}
		if e.High == nil {
			if e.Low == nil {
				return slice
			}
			return fmt.Sprintf("(%s || Go$nil).%s(%s)", slice, method, c.translateExpr(e.Low))
		}
		low := "0"
		if e.Low != nil {
			low = c.translateExpr(e.Low)
		}
		return fmt.Sprintf("(%s || Go$nil).%s(%s, %s)", slice, method, low, c.translateExpr(e.High))

	case *ast.SelectorExpr:
		sel := c.info.Selections[e]
		switch sel.Kind() {
		case types.FieldVal:
			val := c.translateExprToType(e.X, types.NewInterface(nil))
			t := sel.Recv()
			for _, index := range sel.Index() {
				if ptr, isPtr := t.(*types.Pointer); isPtr {
					t = ptr.Elem()
				}
				field := t.Underlying().(*types.Struct).Field(index)
				val += "." + field.Name()
				t = field.Type()
			}
			return val
		case types.MethodVal:
			params := sel.Obj().Type().(*types.Signature).Params()
			names := make([]string, params.Len())
			for i := 0; i < params.Len(); i++ {
				if params.At(i).Anonymous() {
					names[i] = c.newVarName("param")
					continue
				}
				names[i] = c.newVarName(params.At(i).Name())
			}
			nameList := strings.Join(names, ", ")
			return fmt.Sprintf("function(%s) { return %s.%s(%s); }", nameList, c.translateExprToType(e.X, types.NewInterface(nil)), e.Sel.Name, nameList)
		case types.MethodExpr:
			return fmt.Sprintf("%s.prototype.%s.call", c.typeName(sel.Recv()), sel.Obj().(*types.Func).Name())
		case types.PackageObj:
			return fmt.Sprintf("%s.%s", c.translateExpr(e.X), e.Sel.Name)
		}
		panic("")

	case *ast.CallExpr:
		funType := c.info.Types[e.Fun]
		switch t := funType.Underlying().(type) {
		case *types.Signature:
			var fun string
			switch f := e.Fun.(type) {
			case *ast.SelectorExpr:
				sel := c.info.Selections[f]
				if sel.Kind() == types.MethodVal {
					methodsRecvType := sel.Obj().(*types.Func).Type().(*types.Signature).Recv().Type()
					_, pointerExpected := methodsRecvType.(*types.Pointer)
					_, isStruct := sel.Recv().Underlying().(*types.Struct)
					if pointerExpected && !isStruct && !types.IsIdentical(sel.Recv(), methodsRecvType) {
						target := c.translateExpr(f.X)
						fun = fmt.Sprintf("(new %s(function() { return %s; }, function(v) { %s = v; })).%s", c.typeName(methodsRecvType), target, target, f.Sel.Name)
						break
					}
				}
				if sel.Kind() == types.PackageObj {
					fun = fmt.Sprintf("%s.%s", c.translateExpr(f.X), f.Sel.Name)
					break
				}
				fun = fmt.Sprintf("%s.%s", c.translateExprToType(f.X, types.NewInterface(nil)), f.Sel.Name)
			default:
				fun = c.translateExpr(e.Fun)
			}
			if t.Params().Len() > 1 && len(e.Args) == 1 {
				argRefs := make([]string, t.Params().Len())
				for i := range argRefs {
					argRefs[i] = fmt.Sprintf("Go$tuple[%d]", i)
				}
				return fmt.Sprintf("(Go$tuple = %s, %s(%s))", c.translateExpr(e.Args[0]), fun, strings.Join(argRefs, ", "))
			}
			return fmt.Sprintf("%s(%s)", fun, strings.Join(c.translateArgs(e), ", "))
		case *types.Builtin:
			switch t.Name() {
			case "new":
				return c.zeroValue(c.info.Types[e].(*types.Pointer).Elem())
			case "make":
				switch t2 := c.info.Types[e.Args[0]].(type) {
				case *types.Slice:
					if len(e.Args) == 3 {
						return fmt.Sprintf("new %s(Go$clear(new %s(%s)), %s)", c.typeName(c.info.Types[e.Args[0]]), toTypedArray(t2.Elem()), c.translateExpr(e.Args[2]), c.translateExpr(e.Args[1]))
					}
					return fmt.Sprintf("new %s(Go$clear(new %s(%s)))", c.typeName(c.info.Types[e.Args[0]]), toTypedArray(t2.Elem()), c.translateExpr(e.Args[1]))
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
				case *types.Basic, *types.Array:
					return fmt.Sprintf("%s.length", arg)
				case *types.Slice:
					return fmt.Sprintf("(Go$obj = %s, Go$obj !== null ? Go$obj.length : 0)", arg)
				case *types.Map:
					return fmt.Sprintf("Go$keys(%s).length", arg)
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
				return fmt.Sprintf("throw new GoError(%s)", c.translateExpr(e.Args[0]))
			case "append":
				sliceType := c.info.Types[e]
				if e.Ellipsis.IsValid() {
					return fmt.Sprintf("Go$append(%s, %s)", c.translateExpr(e.Args[0]), c.translateExprToType(e.Args[1], sliceType))
				}
				toAppend := createListComposite(sliceType.Underlying().(*types.Slice).Elem(), c.translateExprSlice(e.Args[1:]))
				return fmt.Sprintf("Go$append(%s, new %s(%s))", c.translateExpr(e.Args[0]), c.typeName(sliceType), toAppend)
			case "copy":
				return fmt.Sprintf("Go$copy(%s, %s)", c.translateExprToType(e.Args[0], types.NewSlice(nil)), c.translateExprToType(e.Args[1], types.NewSlice(nil)))
			case "print", "println":
				return fmt.Sprintf("Go$%s(%s)", t.Name(), strings.Join(c.translateExprSlice(e.Args), ", "))
			case "delete", "real", "imag", "recover", "complex":
				return fmt.Sprintf("Go$%s(%s)", t.Name(), strings.Join(c.translateExprSlice(e.Args), ", "))
			default:
				panic(fmt.Sprintf("Unhandled builtin: %s\n", t.Name()))
			}
		case *types.Basic, *types.Slice, *types.Chan, *types.Interface, *types.Map, *types.Pointer:
			return c.translateExprToType(e.Args[0], t)
		default:
			panic(fmt.Sprintf("Unhandled call: %T\n", t))
		}

	case *ast.StarExpr:
		t := c.info.Types[e]
		if _, isStruct := t.Underlying().(*types.Struct); isStruct {
			return fmt.Sprintf("(Go$obj = %s, %s)", c.translateExpr(e.X), c.cloneStruct([]string{"Go$obj"}, t.(*types.Named)))
		}
		return c.translateExpr(e.X) + ".Go$get()"

	case *ast.TypeAssertExpr:
		if e.Type == nil {
			return c.translateExpr(e.X)
		}
		t := c.info.Types[e.Type]
		check := fmt.Sprintf("Go$typeOf(Go$obj) === %s", c.typeName(t))
		if e.Type != nil {
			if _, isInterface := t.Underlying().(*types.Interface); isInterface {
				check = fmt.Sprintf("%s(Go$typeOf(Go$obj))", c.typeName(t))
			}
		}
		value := "Go$obj"
		_, isNamed := t.(*types.Named)
		_, isUnderlyingBasic := t.Underlying().(*types.Basic)
		if isNamed && isUnderlyingBasic {
			value += ".v"
		}
		if _, isTuple := c.info.Types[e].(*types.Tuple); isTuple {
			return fmt.Sprintf("(Go$obj = %s, %s ? [%s, true] : [%s, false])", c.translateExpr(e.X), check, value, c.zeroValue(c.info.Types[e.Type]))
		}
		return fmt.Sprintf("(Go$obj = %s, %s ? %s : typeAssertionFailed(Go$obj))", c.translateExpr(e.X), check, value)

	case *ast.Ident:
		if e.Name == "_" {
			panic("Tried to translate underscore identifier.")
		}
		switch o := c.info.Objects[e].(type) {
		case *types.PkgName:
			return c.pkgVars[o.Pkg().Path()]
		case *types.Var, *types.Const, *types.Func:
			if _, isBuiltin := o.Type().(*types.Builtin); isBuiltin {
				return "Go$" + e.Name
			}
			name, found := c.objectVars[o]
			if !found {
				name = c.newVarName(o.Name())
				c.objectVars[o] = name
			}
			return name
		case *types.TypeName:
			return c.typeName(o.Type())
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

func (c *PkgContext) translateExprSlice(exprs []ast.Expr) []string {
	parts := make([]string, len(exprs))
	for i, expr := range exprs {
		parts[i] = c.translateExpr(expr)
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

	value := c.translateExpr(expr)
	exprType := c.info.Types[expr]
	if basicLit, isBasicLit := expr.(*ast.BasicLit); isBasicLit && basicLit.Kind == token.STRING {
		exprType = types.Typ[types.String]
	}

	switch t := desiredType.Underlying().(type) {
	case *types.Basic:
		switch {
		case t.Info()&types.IsInteger != 0:
			basicExprType := exprType.Underlying().(*types.Basic)
			if basicExprType.Info()&types.IsFloat != 0 {
				value = fmt.Sprintf("Math.floor(%s)", value)
			}
			if t.Kind() == types.Uint64 && basicExprType.Kind() != types.Uint64 {
				value = fmt.Sprintf("new Go$Uint64(0, %s)", value)
			}
			if t.Kind() != types.Uint64 && basicExprType.Kind() == types.Uint64 {
				value += ".low"
			}
			return value
		case t.Info()&types.IsFloat != 0:
			return value
		case t.Info()&types.IsString != 0:
			switch st := exprType.Underlying().(type) {
			case *types.Basic:
				if st.Info()&types.IsNumeric != 0 {
					return fmt.Sprintf("Go$String.fromCharCode(%s)", value)
				}
				return value
			case *types.Slice:
				return fmt.Sprintf("Go$String.fromCharCode.apply(null, %s.Go$toArray())", value)
			default:
				panic(fmt.Sprintf("Unhandled conversion: %v\n", t))
			}
		case t.Info()&types.IsComplex != 0:
			return value
		case t.Info()&types.IsBoolean != 0:
			return value
		case t.Kind() == types.UnsafePointer:
			if unary, isUnary := expr.(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
				if indexExpr, isIndexExpr := unary.X.(*ast.IndexExpr); isIndexExpr {
					return fmt.Sprintf("%s.Go$toArray()", c.translateExpr(indexExpr.X))
				}
				if ident, isIdent := unary.X.(*ast.Ident); isIdent && ident.Name == "_zero" {
					return "new Uint8Array(0)"
				}
			}
			if ptr, isPtr := c.info.Types[expr].(*types.Pointer); isPtr {
				if s, isStruct := ptr.Elem().Underlying().(*types.Struct); isStruct {
					array := c.newVarName("_array")
					view := c.newVarName("_view")
					target := c.newVarName("_struct")
					c.Printf("var %s = new Uint8Array(%d);", array, types.DefaultSizeof(s))
					c.PrintfDelayed("var %s = new DataView(%s.buffer);", view, array)
					c.PrintfDelayed("var %s = %s;", target, c.translateExpr(expr))
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
					offsets := types.DefaultOffsetsof(fields)
					for i, field := range fields {
						if basic, isBasic := field.Type().Underlying().(*types.Basic); isBasic && basic.Info()&types.IsNumeric != 0 {
							c.PrintfDelayed("%s = %s.get%s(%d, true);", field.Name(), view, toJavaScriptType(basic), offsets[i])
						}
					}
					return array
				}
			}
		default:
			panic(fmt.Sprintf("Unhandled conversion: %v\n", t))
		}

	case *types.Interface:
		named, isNamed := exprType.(*types.Named)
		_, isUnderlyingBasic := exprType.Underlying().(*types.Basic)
		if isNamed && isUnderlyingBasic {
			value = fmt.Sprintf("new %s(%s)", c.typeName(named), value)
		}

	case *types.Slice:
		if basic, isBasic := exprType.Underlying().(*types.Basic); isBasic && basic.Info()&types.IsString != 0 {
			value = fmt.Sprintf("%s.Go$toSlice()", value)
		}

	case *types.Array, *types.Struct, *types.Chan, *types.Map, *types.Pointer, *types.Signature:
		// no converion

	default:
		panic(fmt.Sprintf("Unhandled conversion: %v\n", t))
	}

	return value
}

func (c *PkgContext) cloneStruct(srcPath []string, t *types.Named) string {
	s := t.Underlying().(*types.Struct)
	fields := make([]string, s.NumFields())
	for i := range fields {
		field := s.Field(i)
		fieldPath := append(srcPath, field.Name())
		if _, isStruct := field.Type().Underlying().(*types.Struct); isStruct {
			fields[i] = c.cloneStruct(fieldPath, field.Type().(*types.Named))
			continue
		}
		fields[i] = strings.Join(fieldPath, ".")
	}
	return fmt.Sprintf("new %s(%s)", c.typeName(t), strings.Join(fields, ", "))
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
