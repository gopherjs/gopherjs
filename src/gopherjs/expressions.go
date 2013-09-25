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
			d, _ := exact.Int64Val(value)
			if d == 1<<63-1 { // max bitwise mask
				d = 1<<53 - 1
			}
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
				var resultNames []ast.Expr
				if e.Type.Results != nil && e.Type.Results.List[0].Names != nil {
					for _, result := range e.Type.Results.List {
						for _, name := range result.Names {
							if isUnderscore(name) {
								name = ast.NewIdent("result")
								c.info.Objects[name] = types.NewVar(0, nil, "result", c.info.Types[result.Type])
							}
							c.Printf("var %s = %s;", c.translateExpr(name), c.zeroValue(c.info.Types[name]))
							resultNames = append(resultNames, name)
						}
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
		}
		return fmt.Sprintf("%s%s", op, c.translateExpr(e.X))

	case *ast.BinaryExpr:
		ex := c.translateExpr(e.X)
		ey := c.translateExpr(e.Y)
		op := e.Op.String()
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
		case token.AND_NOT:
			op = "&~"
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
		index := c.translateExpr(e.Index)
		xType := c.info.Types[e.X]
		if ptr, isPointer := xType.(*types.Pointer); isPointer {
			xType = ptr.Elem()
		}
		switch t := xType.Underlying().(type) {
		case *types.Array:
			return fmt.Sprintf("%s[%s]", x, index)
		case *types.Slice:
			return fmt.Sprintf("%s.get(%s)", x, index)
		case *types.Map:
			if _, isPointer := t.Key().Underlying().(*types.Pointer); isPointer {
				index = fmt.Sprintf("(%s || Go$nil).Go$id", index)
			}
			if _, isTuple := c.info.Types[e].(*types.Tuple); isTuple {
				return fmt.Sprintf("(Go$obj = (%s || Go$nil)[%s], Go$obj !== undefined ? [Go$obj.v, true] : [%s, false])", x, index, c.zeroValue(t.Elem()))
			}
			return fmt.Sprintf("(Go$obj = (%s || Go$nil)[%s], Go$obj !== undefined ? Go$obj.v : %s)", x, index, c.zeroValue(t.Elem()))
		case *types.Basic:
			return fmt.Sprintf("%s.charCodeAt(%s)", x, index)
		default:
			panic(fmt.Sprintf("Unhandled IndexExpr: %T\n", t))
		}

	case *ast.SliceExpr:
		method := "subslice"
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
			return fmt.Sprintf("%s.%s(%s)", slice, method, c.translateExpr(e.Low))
		}
		low := "0"
		if e.Low != nil {
			low = c.translateExpr(e.Low)
		}
		return fmt.Sprintf("%s.%s(%s, %s)", slice, method, low, c.translateExpr(e.High))

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
				names[i] = params.At(i).Name()
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
				if sel := c.info.Selections[f]; sel.Kind() == types.MethodVal {
					methodsRecvType := sel.Obj().(*types.Func).Type().(*types.Signature).Recv().Type()
					_, pointerExpected := methodsRecvType.(*types.Pointer)
					_, isStruct := sel.Recv().Underlying().(*types.Struct)
					if pointerExpected && !isStruct && !types.IsIdentical(sel.Recv(), methodsRecvType) {
						target := c.translateExpr(f.X)
						fun = fmt.Sprintf("(new %s(function() { return %s; }, function(v) { %s = v; })).%s", c.typeName(methodsRecvType), target, target, f.Sel.Name)
						break
					}
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
					return fmt.Sprintf("Object.keys(%s).length", arg)
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
					return fmt.Sprintf("%s.array.length", arg)
				default:
					panic(fmt.Sprintf("Unhandled cap type: %T\n", argt))
				}
			case "panic":
				return fmt.Sprintf("throw new GoError(%s)", c.translateExpr(e.Args[0]))
			case "append":
				if e.Ellipsis.IsValid() {
					return fmt.Sprintf("Go$append(%s, %s)", c.translateExpr(e.Args[0]), c.translateExprToSlice(e.Args[1]))
				}
				sliceType := c.info.Types[e]
				toAppend := createListComposite(sliceType.Underlying().(*types.Slice).Elem(), c.translateExprSlice(e.Args[1:]))
				return fmt.Sprintf("Go$append(%s, new %s(%s))", c.translateExpr(e.Args[0]), c.typeName(sliceType), toAppend)
			case "copy":
				return fmt.Sprintf("Go$copy(%s, %s)", c.translateExprToSlice(e.Args[0]), c.translateExprToSlice(e.Args[1]))
			case "print", "println":
				return fmt.Sprintf("Go$%s(%s)", t.Name(), strings.Join(c.translateExprSlice(e.Args), ", "))
			case "delete", "real", "imag", "recover", "complex":
				return fmt.Sprintf("Go$%s(%s)", t.Name(), strings.Join(c.translateExprSlice(e.Args), ", "))
			default:
				panic(fmt.Sprintf("Unhandled builtin: %s\n", t.Name()))
			}
		case *types.Basic:
			src := c.translateExpr(e.Args[0])
			srcType := c.info.Types[e.Args[0]]
			switch {
			case t.Info()&types.IsInteger != 0:
				if srcType.Underlying().(*types.Basic).Info()&types.IsFloat != 0 {
					return fmt.Sprintf("Math.floor(%s)", src)
				}
				return src
			case t.Info()&types.IsFloat != 0:
				return src
			case t.Info()&types.IsString != 0:
				switch st := srcType.Underlying().(type) {
				case *types.Basic:
					if st.Info()&types.IsNumeric != 0 {
						return fmt.Sprintf("String.fromCharCode(%s)", src)
					}
					return src
				case *types.Slice:
					return fmt.Sprintf("String.fromCharCode.apply(null, %s.toArray())", src)
				default:
					panic(fmt.Sprintf("Unhandled conversion: %v\n", t))
				}
			case t.Info()&types.IsComplex != 0:
				return src
			case t.Info()&types.IsBoolean != 0:
				return src
			case t.Kind() == types.UnsafePointer:
				if unary, isUnary := e.Args[0].(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
					if indexExpr, isIndexExpr := unary.X.(*ast.IndexExpr); isIndexExpr {
						return fmt.Sprintf("%s.toArray()", c.translateExpr(indexExpr.X))
					}
					if ident, isIdent := unary.X.(*ast.Ident); isIdent && ident.Name == "_zero" {
						return "new Uint8Array(0)"
					}
				}
				if ptr, isPtr := c.info.Types[e.Args[0]].(*types.Pointer); isPtr {
					if s, isStruct := ptr.Elem().Underlying().(*types.Struct); isStruct {
						array := c.newVarName("_array")
						view := c.newVarName("_view")
						target := c.newVarName("_struct")
						c.Printf("var %s = new Uint8Array(%d);", array, types.DefaultSizeof(s))
						c.PrintfDelayed("var %s = new DataView(%s.buffer);", view, array)
						c.PrintfDelayed("var %s = %s;", target, c.translateExpr(e.Args[0]))
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
				return src
			default:
				panic(fmt.Sprintf("Unhandled conversion: %v\n", t))
			}
		case *types.Slice:
			return fmt.Sprintf("%s.toSlice()", c.translateExpr(e.Args[0]))
		case *types.Chan, *types.Interface, *types.Map, *types.Pointer:
			return c.translateExpr(e.Args[0])
		default:
			panic(fmt.Sprintf("Unhandled call: %T\n", t))
		}

	case *ast.StarExpr:
		t := c.info.Types[e]
		if _, isStruct := t.Underlying().(*types.Struct); isStruct {
			return fmt.Sprintf("(Go$obj = %s, %s)", c.translateExpr(e.X), c.cloneStruct([]string{"Go$obj"}, t.(*types.Named)))
		}
		return c.translateExpr(e.X) + ".get()"

	case *ast.TypeAssertExpr:
		if e.Type == nil {
			return c.translateExpr(e.X)
		}
		t := c.info.Types[e.Type]
		check := fmt.Sprintf("typeOf(Go$obj) === %s", c.typeName(t))
		if e.Type != nil {
			if _, isInterface := t.Underlying().(*types.Interface); isInterface {
				check = fmt.Sprintf("%s(typeOf(Go$obj))", c.typeName(t))
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
	if expr == nil {
		return c.zeroValue(desiredType)
	}
	exprType := c.info.Types[expr]
	if exprType != nil && desiredType != nil {
		named, isNamed := exprType.(*types.Named)
		_, isUnderlyingBasic := exprType.Underlying().(*types.Basic)
		_, interfaceIsDesired := desiredType.Underlying().(*types.Interface)
		if isNamed && isUnderlyingBasic && interfaceIsDesired {
			return fmt.Sprintf("new %s(%s)", c.typeName(named), c.translateExpr(expr))
		}
	}
	return c.translateExpr(expr)
}

func (c *PkgContext) translateExprToSlice(expr ast.Expr) string {
	if t, isBasic := c.info.Types[expr].Underlying().(*types.Basic); isBasic && t.Info()&types.IsString != 0 {
		return c.translateExpr(expr) + ".toSlice()"
	}
	return c.translateExpr(expr)
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
