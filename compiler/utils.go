package compiler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/exact"
	"golang.org/x/tools/go/types"
)

func (c *funcContext) Write(b []byte) (int, error) {
	c.output = append(c.output, b...)
	return len(b), nil
}

func (c *funcContext) Printf(format string, values ...interface{}) {
	c.Write([]byte(strings.Repeat("\t", c.p.indentation)))
	fmt.Fprintf(c, format, values...)
	c.Write([]byte{'\n'})
	c.Write(c.delayedOutput)
	c.delayedOutput = nil
}

func (c *funcContext) PrintCond(cond bool, onTrue, onFalse string) {
	if !cond {
		c.Printf("/* %s */ %s", strings.Replace(onTrue, "*/", "<star>/", -1), onFalse)
		return
	}
	c.Printf("%s", onTrue)
}

func (c *funcContext) WritePos(pos token.Pos) {
	c.Write([]byte{'\b'})
	binary.Write(c, binary.BigEndian, uint32(pos))
}

func (c *funcContext) Indent(f func()) {
	c.p.indentation++
	f()
	c.p.indentation--
}

func (c *funcContext) CatchOutput(indent int, f func()) []byte {
	origoutput := c.output
	c.output = nil
	c.p.indentation += indent
	f()
	catched := c.output
	c.output = origoutput
	c.p.indentation -= indent
	return catched
}

func (c *funcContext) Delayed(f func()) {
	c.delayedOutput = c.CatchOutput(0, f)
}

func (c *funcContext) translateArgs(sig *types.Signature, args []ast.Expr, ellipsis bool) []string {
	if len(args) == 1 {
		if tuple, isTuple := c.p.info.Types[args[0]].Type.(*types.Tuple); isTuple {
			tupleVar := c.newVariable("_tuple")
			c.Printf("%s = %s;", tupleVar, c.translateExpr(args[0]))
			args = make([]ast.Expr, tuple.Len())
			for i := range args {
				args[i] = c.newIdent(c.formatExpr("%s[%d]", tupleVar, i).String(), tuple.At(i).Type())
			}
		}
	}

	params := make([]string, sig.Params().Len())
	for i := range params {
		if sig.Variadic() && i == len(params)-1 && !ellipsis {
			varargType := sig.Params().At(i).Type().(*types.Slice)
			varargs := make([]string, len(args)-i)
			for j, arg := range args[i:] {
				varargs[j] = c.translateImplicitConversionWithCloning(arg, varargType.Elem()).String()
			}
			params[i] = fmt.Sprintf("new %s([%s])", c.typeName(varargType), strings.Join(varargs, ", "))
			break
		}
		argType := sig.Params().At(i).Type()
		params[i] = c.translateImplicitConversionWithCloning(args[i], argType).String()
	}
	return params
}

func (c *funcContext) translateSelection(sel *types.Selection) ([]string, string) {
	var fields []string
	t := sel.Recv()
	for _, index := range sel.Index() {
		if ptr, isPtr := t.(*types.Pointer); isPtr {
			t = ptr.Elem()
		}
		s := t.Underlying().(*types.Struct)
		if jsTag := getJsTag(s.Tag(index)); jsTag != "" {
			var searchJsObject func(*types.Struct) []string
			searchJsObject = func(s *types.Struct) []string {
				for i := 0; i < s.NumFields(); i++ {
					ft := s.Field(i).Type()
					if isJsObject(ft) {
						return []string{fieldName(s, i)}
					}
					ft = ft.Underlying()
					if ptr, ok := ft.(*types.Pointer); ok {
						ft = ptr.Elem().Underlying()
					}
					if s2, ok := ft.(*types.Struct); ok {
						if f := searchJsObject(s2); f != nil {
							return append([]string{fieldName(s, i)}, f...)
						}
					}
				}
				return nil
			}
			if jsObjectFields := searchJsObject(s); jsObjectFields != nil {
				return append(fields, jsObjectFields...), jsTag
			}
		}
		fields = append(fields, fieldName(s, index))
		t = s.Field(index).Type()
	}
	return fields, ""
}

func (c *funcContext) zeroValue(ty types.Type) string {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		switch {
		case is64Bit(t) || t.Info()&types.IsComplex != 0:
			return fmt.Sprintf("new %s(0, 0)", c.typeName(ty))
		case t.Info()&types.IsBoolean != 0:
			return "false"
		case t.Info()&types.IsNumeric != 0, t.Kind() == types.UnsafePointer:
			return "0"
		case t.Info()&types.IsString != 0:
			return `""`
		case t.Kind() == types.UntypedNil:
			panic("Zero value for untyped nil.")
		default:
			panic("Unhandled type")
		}
	case *types.Array:
		return fmt.Sprintf("%s.zero()", c.typeName(ty))
	case *types.Signature:
		return "$throwNilPointerError"
	case *types.Slice:
		return fmt.Sprintf("%s.nil", c.typeName(ty))
	case *types.Struct:
		return fmt.Sprintf("new %s.Ptr()", c.typeName(ty))
	case *types.Map:
		return "false"
	case *types.Interface:
		return "$ifaceNil"
	}
	return fmt.Sprintf("%s.nil", c.typeName(ty))
}

func (c *funcContext) newVariable(name string) string {
	return c.newVariableWithLevel(name, false, "")
}

func (c *funcContext) newVariableWithLevel(name string, pkgLevel bool, initializer string) string {
	if name == "" {
		panic("newVariable: empty name")
	}
	for _, b := range []byte(name) {
		if b < '0' || b > 'z' {
			name = "nonAsciiName"
			break
		}
	}
	if strings.HasPrefix(name, "dollar_") {
		return "$" + name[7:]
	}
	if c.p.minify {
		i := 0
		for {
			offset := int('a')
			if pkgLevel {
				offset = int('A')
			}
			j := i
			name = ""
			for {
				name = string(offset+(j%26)) + name
				j = j/26 - 1
				if j == -1 {
					break
				}
			}
			if c.allVars[name] == 0 {
				break
			}
			i++
		}
	}
	n := c.allVars[name]
	c.allVars[name] = n + 1
	if n > 0 {
		name = fmt.Sprintf("%s$%d", name, n)
	}
	if initializer != "" {
		c.localVars = append(c.localVars, name+" = "+initializer)
		return name
	}
	c.localVars = append(c.localVars, name)
	return name
}

func (c *funcContext) newIdent(name string, t types.Type) *ast.Ident {
	ident := ast.NewIdent(name)
	c.setType(ident, t)
	obj := types.NewVar(0, c.p.pkg, name, t)
	c.p.info.Uses[ident] = obj
	c.p.objectVars[obj] = name
	return ident
}

func (c *funcContext) newInt(i int, t types.Type) *ast.BasicLit {
	lit := &ast.BasicLit{Kind: token.INT}
	c.p.info.Types[lit] = types.TypeAndValue{Type: t, Value: exact.MakeInt64(int64(i))}
	return lit
}

func (c *funcContext) setType(e ast.Expr, t types.Type) ast.Expr {
	c.p.info.Types[e] = types.TypeAndValue{Type: t}
	return e
}

func (c *funcContext) objectName(o types.Object) string {
	if o.Pkg() != c.p.pkg || o.Parent() == c.p.pkg.Scope() {
		c.p.dependencies[o] = true
	}

	if o.Pkg() != c.p.pkg {
		pkgVar, found := c.p.pkgVars[o.Pkg().Path()]
		if !found {
			pkgVar = fmt.Sprintf(`$packages["%s"]`, o.Pkg().Path())
		}
		return pkgVar + "." + o.Name()
	}

	switch o.(type) {
	case *types.Var, *types.Const:
		if o.Exported() && o.Parent() == c.p.pkg.Scope() {
			return "$pkg." + o.Name()
		}
	}

	name, found := c.p.objectVars[o]
	if !found {
		name = c.newVariableWithLevel(o.Name(), o.Parent() == c.p.pkg.Scope(), "")
		c.p.objectVars[o] = name
	}

	if c.p.escapingVars[o] {
		return name + "[0]"
	}
	return name
}

func (c *funcContext) typeName(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.UnsafePointer:
			return "$UnsafePointer"
		default:
			return "$" + toJavaScriptType(t)
		}
	case *types.Named:
		if t.Obj().Name() == "error" {
			return "$error"
		}
		return c.objectName(t.Obj())
	case *types.Pointer:
		return fmt.Sprintf("($ptrType(%s))", c.initArgs(t))
	case *types.Interface:
		if t.Empty() {
			return "$emptyInterface"
		}
		return fmt.Sprintf("($interfaceType(%s))", c.initArgs(t))
	case *types.Array, *types.Chan, *types.Slice, *types.Map, *types.Signature, *types.Struct:
		return fmt.Sprintf("($%sType(%s))", strings.ToLower(typeKind(t)[5:]), c.initArgs(t))
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
}

func (c *funcContext) makeKey(expr ast.Expr, keyType types.Type) string {
	switch t := keyType.Underlying().(type) {
	case *types.Array, *types.Struct:
		return fmt.Sprintf("(new %s(%s)).$key()", c.typeName(keyType), c.translateExpr(expr))
	case *types.Basic:
		if is64Bit(t) {
			return fmt.Sprintf("%s.$key()", c.translateExpr(expr))
		}
		if t.Info()&types.IsFloat != 0 {
			return fmt.Sprintf("$floatKey(%s)", c.translateExpr(expr))
		}
		return c.translateImplicitConversion(expr, keyType).String()
	case *types.Chan, *types.Pointer, *types.Interface:
		return fmt.Sprintf("%s.$key()", c.translateImplicitConversion(expr, keyType))
	default:
		return c.translateImplicitConversion(expr, keyType).String()
	}
}

func (c *funcContext) externalize(s string, t types.Type) string {
	if isJsObject(t) {
		return s
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Info()&types.IsNumeric != 0 && !is64Bit(u) && u.Info()&types.IsComplex == 0 {
			return s
		}
		if u.Kind() == types.UntypedNil {
			return "null"
		}
	}
	return fmt.Sprintf("$externalize(%s, %s)", s, c.typeName(t))
}

func fieldName(t *types.Struct, i int) string {
	name := t.Field(i).Name()
	if name == "_" || reservedKeywords[name] {
		return fmt.Sprintf("%s$%d", name, i)
	}
	return name
}

func typeKind(ty types.Type) string {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		return "$kind" + toJavaScriptType(t)
	case *types.Array:
		return "$kindArray"
	case *types.Chan:
		return "$kindChan"
	case *types.Interface:
		return "$kindInterface"
	case *types.Map:
		return "$kindMap"
	case *types.Signature:
		return "$kindFunc"
	case *types.Slice:
		return "$kindSlice"
	case *types.Struct:
		return "$kindStruct"
	case *types.Pointer:
		return "$kindPtr"
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
}

func toJavaScriptType(t *types.Basic) string {
	switch t.Kind() {
	case types.UntypedInt:
		return "Int"
	case types.Byte:
		return "Uint8"
	case types.Rune:
		return "Int32"
	case types.UnsafePointer:
		return "UnsafePointer"
	default:
		name := t.String()
		return strings.ToUpper(name[:1]) + name[1:]
	}
}

func is64Bit(t *types.Basic) bool {
	return t.Kind() == types.Int64 || t.Kind() == types.Uint64
}

func isComplex(t *types.Basic) bool {
	return t.Kind() == types.Complex64 || t.Kind() == types.Complex128
}

func isBlank(expr ast.Expr) bool {
	if expr == nil {
		return true
	}
	if id, isIdent := expr.(*ast.Ident); isIdent {
		return id.Name == "_"
	}
	return false
}

func isWrapped(ty types.Type) bool {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		return !is64Bit(t) && t.Info()&types.IsComplex == 0 && t.Kind() != types.UntypedNil
	case *types.Array, *types.Map, *types.Signature:
		return true
	case *types.Pointer:
		_, isArray := t.Elem().Underlying().(*types.Array)
		return isArray
	}
	return false
}

func removeParens(e ast.Expr) ast.Expr {
	for {
		p, isParen := e.(*ast.ParenExpr)
		if !isParen {
			return e
		}
		e = p.X
	}
}

func encodeString(s string) string {
	buffer := bytes.NewBuffer(nil)
	for _, r := range []byte(s) {
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
}

func isJsPackage(pkg *types.Package) bool {
	return pkg != nil && pkg.Path() == "github.com/gopherjs/gopherjs/js"
}

func isJsObject(t types.Type) bool {
	named, isNamed := t.(*types.Named)
	return isNamed && isJsPackage(named.Obj().Pkg()) && named.Obj().Name() == "Object"
}

func getJsTag(tag string) string {
	for tag != "" {
		// skip leading space
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// scan to colon.
		// a space or a quote is a syntax error
		i = 0
		for i < len(tag) && tag[i] != ' ' && tag[i] != ':' && tag[i] != '"' {
			i++
		}
		if i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// scan quoted string to find value
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if name == "js" {
			value, _ := strconv.Unquote(qvalue)
			return value
		}
	}
	return ""
}

func needsSpace(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

func removeWhitespace(b []byte, minify bool) []byte {
	if !minify {
		return b
	}

	var out []byte
	var previous byte
	for len(b) > 0 {
		switch b[0] {
		case '\b':
			out = append(out, b[:5]...)
			b = b[5:]
			continue
		case ' ', '\t', '\n':
			if (!needsSpace(previous) || !needsSpace(b[1])) && !(previous == '-' && b[1] == '-') {
				b = b[1:]
				continue
			}
		case '"':
			out = append(out, '"')
			b = b[1:]
			for {
				i := bytes.IndexAny(b, "\"\\")
				out = append(out, b[:i]...)
				b = b[i:]
				if b[0] == '"' {
					break
				}
				// backslash
				out = append(out, b[:2]...)
				b = b[2:]
			}
		case '/':
			if b[1] == '*' {
				i := bytes.Index(b[2:], []byte("*/"))
				b = b[i+4:]
				continue
			}
		}
		out = append(out, b[0])
		previous = b[0]
		b = b[1:]
	}
	return out
}
