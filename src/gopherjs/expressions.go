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
			elements = make([]string, len(e.Elts))
			for i, element := range e.Elts {
				kve := element.(*ast.KeyValueExpr)
				elements[i] = fmt.Sprintf("%s: %s", c.translateExpr(kve.Key), c.translateExpr(kve.Value))
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
			return fmt.Sprintf("new Slice(%s)", createListComposite(t.Elem(), elements))
		case *types.Map:
			return fmt.Sprintf("new Map({ %s })", strings.Join(elements, ", "))
		case *types.Struct:
			for i, element := range elements {
				elements[i] = fmt.Sprintf("%s: %s", t.Field(i).Name(), element)
			}
			return fmt.Sprintf("{ %s }", strings.Join(elements, ", "))
		case *types.Named:
			if s, isSlice := t.Underlying().(*types.Slice); isSlice {
				return fmt.Sprintf("new %s(%s)", c.typeName(t), createListComposite(s.Elem(), elements))
			}
			return fmt.Sprintf("new %s(%s)", c.typeName(t), strings.Join(elements, ", "))
		default:
			fmt.Println(e.Type, elements)
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", c.info.Types[e]))
		}

	case *ast.FuncLit:
		n := c.usedVarNames
		defer func() { c.usedVarNames = n }()
		body := c.CatchOutput(func() {
			c.Indent(func() {
				var namedResults []string
				if e.Type.Results != nil && e.Type.Results.List[0].Names != nil {
					for _, result := range e.Type.Results.List {
						for _, name := range result.Names {
							if isUnderscore(name) {
								namedResults = append(namedResults, c.newVarName("result"))
								continue
							}
							namedResults = append(namedResults, c.translateExpr(name))
						}
					}
				}
				r := c.namedResults
				defer func() { c.namedResults = r }()
				c.namedResults = namedResults
				c.Printf("var _obj, _tuple;")
				if namedResults != nil {
					c.Printf("var %s;", strings.Join(namedResults, ", "))
				}

				v := HasDeferVisitor{}
				ast.Walk(&v, e.Body)
				if v.hasDefer {
					c.Printf("var _deferred = [];")
					c.Printf("try {")
					c.Indent(func() {
						c.translateStmtList(e.Body.List)
					})
					c.Printf("} catch(err) {")
					c.Indent(func() {
						c.Printf("_error_stack.push({ frame: getStackDepth(), error: err });")
					})
					c.Printf("} finally {")
					c.Indent(func() {
						c.Printf("callDeferred(_deferred);")
						if namedResults != nil {
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
		return fmt.Sprintf("%s %s %s", ex, op, ey)

	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", c.translateExpr(e.X))

	case *ast.IndexExpr:
		x := c.translateExpr(e.X)
		index := c.translateExpr(e.Index)
		switch t := c.info.Types[e.X].Underlying().(type) {
		case *types.Basic:
			if t.Info()&types.IsString != 0 {
				return fmt.Sprintf("%s.charCodeAt(%s)", x, index)
			}
		case *types.Slice:
			return fmt.Sprintf("%s.get(%s)", x, index)
		}
		return fmt.Sprintf("%s[%s]", x, index)

	case *ast.SliceExpr:
		method := "subslice"
		if b, ok := c.info.Types[e.X].(*types.Basic); ok && b.Info()&types.IsString != 0 {
			method = "substring"
		}
		slice := c.translateExpr(e.X)
		if _, ok := c.info.Types[e.X].(*types.Array); ok {
			slice = fmt.Sprintf("new Slice(%s)", slice)
		}
		if e.High == nil {
			return fmt.Sprintf("%s.%s(%s)", slice, method, c.translateExpr(e.Low))
		}
		low := "0"
		if e.Low != nil {
			low = c.translateExpr(e.Low)
		}
		return fmt.Sprintf("%s.%s(%s, %s)", slice, method, low, c.translateExpr(e.High))

	case *ast.SelectorExpr:
		sel := c.info.Selections[e]
		if sel != nil {
			switch sel.Kind() {
			case types.MethodVal:
				methodsRecvType := sel.Obj().(*types.Func).Type().(*types.Signature).Recv().Type()
				_, pointerExpected := methodsRecvType.(*types.Pointer)
				_, isStruct := sel.Recv().Underlying().(*types.Struct)
				if pointerExpected && !isStruct && !types.IsIdentical(sel.Recv(), methodsRecvType) {
					target := c.translateExpr(e.X)
					return fmt.Sprintf("(new %s(function() { return %s; }, function(v) { %s = v; })).%s", c.typeName(methodsRecvType), target, target, e.Sel.Name)
				}
			case types.MethodExpr:
				return fmt.Sprintf("%s.prototype.%s.call", c.typeName(sel.Recv()), sel.Obj().(*types.Func).Name())
			}
		}

		return fmt.Sprintf("%s.%s", c.translateExprToType(e.X, types.NewInterface(nil)), e.Sel.Name)

	case *ast.CallExpr:
		funType := c.info.Types[e.Fun]
		switch t := funType.Underlying().(type) {
		case *types.Builtin:
			switch t.Name() {
			case "new":
				return c.zeroValue(c.info.Types[e].(*types.Pointer).Elem())
			case "make":
				args := []string{"undefined"}
				for _, arg := range e.Args[1:] {
					args = append(args, c.translateExpr(arg))
				}
				return fmt.Sprintf("new %s(%s)", c.typeName(c.info.Types[e.Args[0]]), strings.Join(args, ", "))
			case "len":
				arg := c.translateExpr(e.Args[0])
				argType := c.info.Types[e.Args[0]]
				switch argt := argType.Underlying().(type) {
				case *types.Basic, *types.Array, *types.Slice:
					return fmt.Sprintf("%s.length", arg)
				case *types.Map:
					return fmt.Sprintf("Object.keys(%s.data).length", arg)
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
			case "append", "copy", "delete", "real", "imag", "recover", "complex", "print", "println":
				return fmt.Sprintf("%s(%s)", t.Name(), strings.Join(c.translateArgs(e), ", "))
			default:
				panic(fmt.Sprintf("Unhandled builtin: %s\n", t.Name()))
			}
		case *types.Basic:
			jsValue := func() string {
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
					return src
				default:
					panic(fmt.Sprintf("Unhandled conversion: %v\n", t))
				}
			}
			if named, isNamed := funType.(*types.Named); isNamed {
				return fmt.Sprintf("new %s(%s)", named.Obj().Name(), jsValue())
			}
			return jsValue()
		case *types.Slice:
			return fmt.Sprintf("%s.toSlice()", c.translateExpr(e.Args[0]))
		case *types.Chan, *types.Interface, *types.Map, *types.Pointer:
			return c.translateExpr(e.Args[0])
		case *types.Signature:
			if t.Params().Len() > 1 && len(e.Args) == 1 {
				argRefs := make([]string, t.Params().Len())
				for i := range argRefs {
					argRefs[i] = fmt.Sprintf("_tuple[%d]", i)
				}
				return fmt.Sprintf("(_tuple = %s, %s(%s))", c.translateExpr(e.Args[0]), c.translateExpr(e.Fun), strings.Join(argRefs, ", "))
			}
			return fmt.Sprintf("%s(%s)", c.translateExpr(e.Fun), strings.Join(c.translateArgs(e), ", "))
		default:
			panic(fmt.Sprintf("Unhandled call: %T\n", t))
		}

	case *ast.StarExpr:
		t := c.info.Types[e]
		if _, isStruct := t.Underlying().(*types.Struct); isStruct {
			return fmt.Sprintf("(_obj = %s, %s)", c.translateExpr(e.X), c.cloneStruct([]string{"_obj"}, t.(*types.Named)))
		}
		return c.translateExpr(e.X) + ".get()"

	case *ast.TypeAssertExpr:
		if e.Type == nil {
			return c.translateExpr(e.X)
		}
		t := c.info.Types[e.Type]
		check := fmt.Sprintf("typeOf(_obj) === %s", c.typeName(t))
		if e.Type != nil {
			if _, isInterface := t.Underlying().(*types.Interface); isInterface {
				check = fmt.Sprintf("%s(typeOf(_obj))", c.typeName(t))
			}
		}
		value := "_obj"
		_, isNamed := t.(*types.Named)
		_, isUnderlyingBasic := t.Underlying().(*types.Basic)
		if isNamed && isUnderlyingBasic {
			value += ".v"
		}
		if _, isTuple := c.info.Types[e].(*types.Tuple); isTuple {
			return fmt.Sprintf("(_obj = %s, %s ? [%s, true] : [%s, false])", c.translateExpr(e.X), check, value, c.zeroValue(c.info.Types[e.Type]))
		}
		return fmt.Sprintf("(_obj = %s, %s ? %s : typeAssertionFailed())", c.translateExpr(e.X), check, value)

	case *ast.Ident:
		if e.Name == "_" {
			panic("Tried to translate underscore identifier.")
		}
		switch o := c.info.Objects[e].(type) {
		case *types.PkgName:
			return c.pkgVars[o.Pkg().Path()]
		case *types.Var, *types.Const, *types.Func:
			if _, isBuiltin := o.Type().(*types.Builtin); isBuiltin {
				return e.Name
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

	case *This:
		return "this"

	case nil:
		return ""

	default:
		panic(fmt.Sprintf("Unhandled expression: %T\n", e))

	}
}

func (c *PkgContext) translateExprToType(expr ast.Expr, desiredType types.Type) string {
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
	case *ast.FuncLit, *This:
		return nil
	}
	return v
}
