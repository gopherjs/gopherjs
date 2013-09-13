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
		jsValue := ""
		switch value.Kind() {
		case exact.Nil:
			jsValue = "null"
		case exact.Bool:
			jsValue = fmt.Sprintf("%t", exact.BoolVal(value))
		case exact.Int:
			d, _ := exact.Int64Val(value)
			jsValue = fmt.Sprintf("%d", d)
		case exact.Float:
			f, _ := exact.Float64Val(value)
			jsValue = fmt.Sprintf("%f", f)
		case exact.Complex:
			f, _ := exact.Float64Val(exact.Real(value))
			jsValue = fmt.Sprintf("%f", f)
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
			jsValue = `"` + buffer.String() + `"`
		default:
			panic("Unhandled value: " + value.String())
		}

		if named, isNamed := c.info.Types[expr].(*types.Named); isNamed {
			return fmt.Sprintf("(new %s(%s))", c.TypeName(named), jsValue)
		}
		return jsValue
	}

	switch e := expr.(type) {
	case *ast.CompositeLit:
		compType := c.info.Types[e]
		if ptrType, isPointer := compType.(*types.Pointer); isPointer {
			compType = ptrType.Elem()
		}

		var elements []string
		switch t := compType.Underlying().(type) {
		case *types.Array:
			elements = make([]string, t.Len())
			var i int64 = 0
			zero := c.zeroValue(t.Elem())
			for _, element := range e.Elts {
				if kve, isKve := element.(*ast.KeyValueExpr); isKve {
					key, _ := exact.Int64Val(c.info.Values[kve.Key])
					for i < key {
						elements[i] = zero
						i += 1
					}
					element = kve.Value
				}
				elements[i] = c.translateExpr(element)
				i += 1
			}
			for i < t.Len() {
				elements[i] = zero
				i += 1
			}
		case *types.Slice:
			elements = make([]string, len(e.Elts))
			for i, element := range e.Elts {
				elements[i] = c.translateExpr(element)
			}
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
				return fmt.Sprintf("new %s(%s)", c.TypeName(t), createListComposite(s.Elem(), elements))
			}
			return fmt.Sprintf("new %s(%s)", c.TypeName(t), strings.Join(elements, ", "))
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
							c.translateStmt(&ast.ReturnStmt{})
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
			if c.savedAsPointer(e) {
				pointerType := "_Pointer"
				if named, isNamed := c.info.Types[e.X].(*types.Named); isNamed {
					pointerType = named.Obj().Name() + "._Pointer"
				}
				return fmt.Sprintf("new %s(function() { return %s; }, function(v) { %s = v; })", pointerType, target, target)
			}
			return target
		case token.XOR:
			op = "~"
		}
		return fmt.Sprintf("%s%s", op, c.translateExpr(e.X))

	case *ast.BinaryExpr:
		ex := c.translateExpressionToBasic(e.X)
		ey := c.translateExpressionToBasic(e.Y)
		op := e.Op.String()
		switch e.Op {
		case token.QUO:
			if c.info.Types[e].(*types.Basic).Info()&types.IsInteger != 0 {
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
			slice = fmt.Sprintf("(new Slice(%s))", slice)
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
		if sel != nil && sel.Kind() == types.MethodVal {
			methodsRecvType := sel.Obj().(*types.Func).Type().(*types.Signature).Recv().Type()
			_, isStruct := sel.Recv().Underlying().(*types.Struct)
			if !isStruct && !types.IsIdentical(sel.Recv(), methodsRecvType) {
				fakeExpr := &ast.UnaryExpr{
					Op: token.AND,
					X:  e.X,
				}
				c.info.Types[fakeExpr] = methodsRecvType
				return c.translateExpr(&ast.SelectorExpr{
					X:   fakeExpr,
					Sel: e.Sel,
				})
			}
		}

		return fmt.Sprintf("%s.%s", c.translateExpr(e.X), e.Sel.Name)

	case *ast.CallExpr:
		funType := c.info.Types[e.Fun]
		builtin, isBuiltin := funType.(*types.Builtin)
		if isBuiltin && builtin.Name() == "new" {
			return c.zeroValue(c.info.Types[e].(*types.Pointer).Elem())
		}

		fun := c.translateExpr(e.Fun)
		args := c.translateArgs(e)
		if _, isSliceType := funType.(*types.Slice); isSliceType {
			return fmt.Sprintf("%s.toSlice()", args[0])
		}
		sig, isSignature := funType.(*types.Signature)
		if isSignature && sig.Params().Len() > 1 && len(args) == 1 {
			argRefs := make([]string, sig.Params().Len())
			for i := range argRefs {
				argRefs[i] = fmt.Sprintf("_tuple[%d]", i)
			}
			return fmt.Sprintf("(_tuple = %s, %s(%s))", args[0], fun, strings.Join(argRefs, ", "))
		}
		ident, isIdent := e.Fun.(*ast.Ident)
		if !isSignature && !isBuiltin && isIdent {
			return fmt.Sprintf("cast(%s, %s)", c.translateExpr(ident), args[0])
		}
		return fmt.Sprintf("%s(%s)", fun, strings.Join(args, ", "))

	case *ast.StarExpr:
		t := c.info.Types[e]
		if _, isStruct := t.Underlying().(*types.Struct); isStruct {
			return fmt.Sprintf("(_obj = %s, %s)", c.translateExpr(e.X), c.cloneStruct([]string{"_obj"}, t.(*types.Named)))
		}
		if c.savedAsPointer(e.X) {
			return c.translateExpr(e.X) + ".get()"
		}
		return c.translateExpr(e.X)

	case *ast.TypeAssertExpr:
		if e.Type == nil {
			return c.translateExpr(e.X)
		}
		check := fmt.Sprintf("typeOf(_obj) === %s", c.translateExpr(e.Type))
		if e.Type != nil {
			if _, isInterface := c.info.Types[e.Type].Underlying().(*types.Interface); isInterface {
				check = fmt.Sprintf("%s(typeOf(_obj))", c.translateExpr(e.Type))
			}
		}
		if _, isTuple := c.info.Types[e].(*types.Tuple); isTuple {
			return fmt.Sprintf("(_obj = %s, %s ? [_obj, true] : [%s, false])", c.translateExpr(e.X), check, c.zeroValue(c.info.Types[e.Type]))
		}
		return fmt.Sprintf("(_obj = %s, %s ? _obj : typeAssertionFailed())", c.translateExpr(e.X), check)

	case *ast.ArrayType:
		return "Slice"

	case *ast.MapType:
		return "Map"

	case *ast.InterfaceType:
		return "Interface"

	case *ast.ChanType:
		return "Channel"

	case *ast.FuncType:
		return "Function"

	case *ast.Ident:
		if e.Name == "_" {
			panic("Tried to translate underscore identifier.")
		}
		if tn, isTypeName := c.info.Objects[e].(*types.TypeName); isTypeName {
			switch tn.Name() {
			case "bool":
				return "Boolean"
			case "int", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "uintptr":
				return "Integer"
			case "float32", "float64":
				return "Float"
			case "complex64", "complex128":
				return "Complex"
			case "string":
				return "String"
			}
		}
		switch o := c.info.Objects[e].(type) {
		case *types.Package:
			return c.pkgVars[o.Path()]
		case *types.Var, *types.Const, *types.TypeName, *types.Func:
			if _, isBuiltin := o.Type().(*types.Builtin); isBuiltin {
				return e.Name
			}
			name, found := c.objectVars[o]
			if !found {
				name = c.newVarName(o.Name())
				c.objectVars[o] = name
			}
			return name
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

func (c *PkgContext) translateExpressionToBasic(expr ast.Expr) string {
	t := c.info.Types[expr]
	_, isNamed := t.(*types.Named)
	_, iUnderlyingBasic := t.Underlying().(*types.Basic)
	if isNamed && iUnderlyingBasic {
		return c.translateExpr(expr) + ".v"
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
	return fmt.Sprintf("new %s(%s)", c.TypeName(t), strings.Join(fields, ", "))
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
