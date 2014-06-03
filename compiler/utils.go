package compiler

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
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
	params := make([]string, sig.Params().Len())
	for i := range params {
		if sig.Variadic() && i == len(params)-1 && !ellipsis {
			varargType := sig.Params().At(i).Type().(*types.Slice)
			varargs := make([]string, len(args)-i)
			for j, arg := range args[i:] {
				varargs[j] = c.translateImplicitConversion(arg, varargType.Elem()).String()
			}
			params[i] = fmt.Sprintf("new %s([%s])", c.typeName(varargType), strings.Join(varargs, ", "))
			break
		}
		argType := sig.Params().At(i).Type()
		params[i] = c.translateImplicitConversion(args[i], argType).String()
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
			for i := 0; i < s.NumFields(); i++ {
				if isJsObject(s.Field(i).Type()) {
					fields = append(fields, fieldName(s, i))
					return fields, jsTag
				}
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
		return fmt.Sprintf(`$makeNativeArray("%s", %d, function() { return %s; })`, typeKind(t.Elem()), t.Len(), c.zeroValue(t.Elem()))
	case *types.Signature:
		return "$throwNilPointerError"
	case *types.Slice:
		return fmt.Sprintf("%s.nil", c.typeName(ty))
	case *types.Struct:
		if named, isNamed := ty.(*types.Named); isNamed {
			return fmt.Sprintf("new %s.Ptr()", c.objectName(named.Obj()))
		}
		fields := make([]string, t.NumFields())
		for i := range fields {
			fields[i] = c.zeroValue(t.Field(i).Type())
		}
		return fmt.Sprintf("new %s.Ptr(%s)", c.typeName(ty), strings.Join(fields, ", "))
	case *types.Map:
		return "false"
	case *types.Interface:
		return "null"
	}
	return fmt.Sprintf("%s.nil", c.typeName(ty))
}

func (c *funcContext) newVariable(name string) string {
	return c.newVariableWithLevel(name, false)
}

func (c *funcContext) newVariableWithLevel(name string, pkgLevel bool) string {
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
	c.localVars = append(c.localVars, name)
	return name
}

func (c *funcContext) newIdent(name string, t types.Type) *ast.Ident {
	ident := ast.NewIdent(name)
	c.p.info.Types[ident] = types.TypeAndValue{Type: t}
	obj := types.NewVar(0, c.p.pkg, name, t)
	c.p.info.Uses[ident] = obj
	c.p.objectVars[obj] = name
	return ident
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
		name = c.newVariableWithLevel(o.Name(), o.Parent() == c.p.pkg.Scope())
		c.p.objectVars[o] = name
	}

	return name
}

func (c *funcContext) typeName(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.UntypedNil:
			return "null"
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
		return fmt.Sprintf("($%sType(%s))", strings.ToLower(typeKind(t)), c.initArgs(t))
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
	case *types.Chan, *types.Pointer:
		return fmt.Sprintf("%s.$key()", c.translateImplicitConversion(expr, keyType))
	case *types.Interface:
		return fmt.Sprintf("(%s || $interfaceNil).$key()", c.translateImplicitConversion(expr, keyType))
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
		return toJavaScriptType(t)
	case *types.Array:
		return "Array"
	case *types.Chan:
		return "Chan"
	case *types.Interface:
		return "Interface"
	case *types.Map:
		return "Map"
	case *types.Signature:
		return "Func"
	case *types.Slice:
		return "Slice"
	case *types.Struct:
		return "Struct"
	case *types.Pointer:
		return "Ptr"
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

func elemType(ty types.Type) types.Type {
	switch t := ty.Underlying().(type) {
	case *types.Array:
		return t.Elem()
	case *types.Slice:
		return t.Elem()
	case *types.Pointer:
		return t.Elem().Underlying().(*types.Array).Elem()
	default:
		panic("")
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

func applyPatches(file *ast.File, fileSet *token.FileSet, importPath string) *ast.File {
	removeImport := func(path string) {
		for _, decl := range file.Decls {
			if d, ok := decl.(*ast.GenDecl); ok && d.Tok == token.IMPORT {
				for j, spec := range d.Specs {
					value := spec.(*ast.ImportSpec).Path.Value
					if value[1:len(value)-1] == path {
						d.Specs[j] = d.Specs[len(d.Specs)-1]
						d.Specs = d.Specs[:len(d.Specs)-1]
						return
					}
				}
			}
		}
	}

	removeFunction := func(name string) {
		for i, decl := range file.Decls {
			if d, ok := decl.(*ast.FuncDecl); ok {
				if d.Name.String() == name {
					file.Decls[i] = file.Decls[len(file.Decls)-1]
					file.Decls = file.Decls[:len(file.Decls)-1]
					return
				}
			}
		}
	}

	basename := filepath.Base(fileSet.Position(file.Pos()).Filename)
	switch {
	case importPath == "bytes_test" && basename == "equal_test.go":
		file, _ = parser.ParseFile(fileSet, basename, "package bytes_test", 0)

	case importPath == "crypto/rc4" && basename == "rc4_ref.go": // see https://codereview.appspot.com/40540049/
		file, _ = parser.ParseFile(fileSet, basename, "// Copyright 2013 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\n\n// +build !amd64,!arm,!386\n\npackage rc4\n\n// XORKeyStream sets dst to the result of XORing src with the key stream.\n// Dst and src may be the same slice but otherwise should not overlap.\nfunc (c *Cipher) XORKeyStream(dst, src []byte) {\n  i, j := c.i, c.j\n  for k, v := range src {\n    i += 1\n    j += uint8(c.s[i])\n    c.s[i], c.s[j] = c.s[j], c.s[i]\n    dst[k] = v ^ uint8(c.s[uint8(c.s[i]+c.s[j])])\n  }\n  c.i, c.j = i, j\n}", 0)

	case importPath == "encoding/json" && basename == "stream_test.go":
		removeImport("net")
		removeFunction("TestBlocking")

	case importPath == "math/big" && basename == "arith.go": // see https://codereview.appspot.com/55470046/
		file, _ = parser.ParseFile(fileSet, basename, "// Copyright 2009 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\n\n// This file provides Go implementations of elementary multi-precision\n// arithmetic operations on word vectors. Needed for platforms without\n// assembly implementations of these routines.\n\npackage big\n\n// A Word represents a single digit of a multi-precision unsigned integer.\ntype Word uintptr\n\nconst (\n	// Compute the size _S of a Word in bytes.\n	_m    = ^Word(0)\n	_logS = _m>>8&1 + _m>>16&1 + _m>>32&1\n	_S    = 1 << _logS\n\n	_W = _S << 3 // word size in bits\n	_B = 1 << _W // digit base\n	_M = _B - 1  // digit mask\n\n	_W2 = _W / 2   // half word size in bits\n	_B2 = 1 << _W2 // half digit base\n	_M2 = _B2 - 1  // half digit mask\n)\n\n// ----------------------------------------------------------------------------\n// Elementary operations on words\n//\n// These operations are used by the vector operations below.\n\n// z1<<_W + z0 = x+y+c, with c == 0 or 1\nfunc addWW_g(x, y, c Word) (z1, z0 Word) {\n	yc := y + c\n	z0 = x + yc\n	if z0 < x || yc < y {\n		z1 = 1\n	}\n	return\n}\n\n// z1<<_W + z0 = x-y-c, with c == 0 or 1\nfunc subWW_g(x, y, c Word) (z1, z0 Word) {\n	yc := y + c\n	z0 = x - yc\n	if z0 > x || yc < y {\n		z1 = 1\n	}\n	return\n}\n\n// z1<<_W + z0 = x*y\n// Adapted from Warren, Hacker's Delight, p. 132.\nfunc mulWW_g(x, y Word) (z1, z0 Word) {\n	x0 := x & _M2\n	x1 := x >> _W2\n	y0 := y & _M2\n	y1 := y >> _W2\n	w0 := x0 * y0\n	t := x1*y0 + w0>>_W2\n	w1 := t & _M2\n	w2 := t >> _W2\n	w1 += x0 * y1\n	z1 = x1*y1 + w2 + w1>>_W2\n	z0 = x * y\n	return\n}\n\n// z1<<_W + z0 = x*y + c\nfunc mulAddWWW_g(x, y, c Word) (z1, z0 Word) {\n	z1, zz0 := mulWW(x, y)\n	if z0 = zz0 + c; z0 < zz0 {\n		z1++\n	}\n	return\n}\n\n// Length of x in bits.\nfunc bitLen_g(x Word) (n int) {\n	for ; x >= 0x8000; x >>= 16 {\n		n += 16\n	}\n	if x >= 0x80 {\n		x >>= 8\n		n += 8\n	}\n	if x >= 0x8 {\n		x >>= 4\n		n += 4\n	}\n	if x >= 0x2 {\n		x >>= 2\n		n += 2\n	}\n	if x >= 0x1 {\n		n++\n	}\n	return\n}\n\n// log2 computes the integer binary logarithm of x.\n// The result is the integer n for which 2^n <= x < 2^(n+1).\n// If x == 0, the result is -1.\nfunc log2(x Word) int {\n	return bitLen(x) - 1\n}\n\n// Number of leading zeros in x.\nfunc leadingZeros(x Word) uint {\n	return uint(_W - bitLen(x))\n}\n\n// q = (u1<<_W + u0 - r)/y\n// Adapted from Warren, Hacker's Delight, p. 152.\nfunc divWW_g(u1, u0, v Word) (q, r Word) {\n	if u1 >= v {\n		return 1<<_W - 1, 1<<_W - 1\n	}\n\n	s := leadingZeros(v)\n	v <<= s\n\n	vn1 := v >> _W2\n	vn0 := v & _M2\n	un32 := u1<<s | u0>>(_W-s)\n	un10 := u0 << s\n	un1 := un10 >> _W2\n	un0 := un10 & _M2\n	q1 := un32 / vn1\n	rhat := un32 - q1*vn1\n\n	for q1 >= _B2 || q1*vn0 > _B2*rhat+un1 {\n		q1--\n		rhat += vn1\n		if rhat >= _B2 {\n			break\n		}\n	}\n\n	un21 := un32*_B2 + un1 - q1*v\n	q0 := un21 / vn1\n	rhat = un21 - q0*vn1\n\n	for q0 >= _B2 || q0*vn0 > _B2*rhat+un0 {\n		q0--\n		rhat += vn1\n		if rhat >= _B2 {\n			break\n		}\n	}\n\n	return q1*_B2 + q0, (un21*_B2 + un0 - q0*v) >> s\n}\n\nfunc addVV_g(z, x, y []Word) (c Word) {\n	for i := range z {\n		c, z[i] = addWW_g(x[i], y[i], c)\n	}\n	return\n}\n\nfunc subVV_g(z, x, y []Word) (c Word) {\n	for i := range z {\n		c, z[i] = subWW_g(x[i], y[i], c)\n	}\n	return\n}\n\nfunc addVW_g(z, x []Word, y Word) (c Word) {\n	c = y\n	for i := range z {\n		c, z[i] = addWW_g(x[i], c, 0)\n	}\n	return\n}\n\nfunc subVW_g(z, x []Word, y Word) (c Word) {\n	c = y\n	for i := range z {\n		c, z[i] = subWW_g(x[i], c, 0)\n	}\n	return\n}\n\nfunc shlVU_g(z, x []Word, s uint) (c Word) {\n	if n := len(z); n > 0 {\n		ŝ := _W - s\n		w1 := x[n-1]\n		c = w1 >> ŝ\n		for i := n - 1; i > 0; i-- {\n			w := w1\n			w1 = x[i-1]\n			z[i] = w<<s | w1>>ŝ\n		}\n		z[0] = w1 << s\n	}\n	return\n}\n\nfunc shrVU_g(z, x []Word, s uint) (c Word) {\n	if n := len(z); n > 0 {\n		ŝ := _W - s\n		w1 := x[0]\n		c = w1 << ŝ\n		for i := 0; i < n-1; i++ {\n			w := w1\n			w1 = x[i+1]\n			z[i] = w>>s | w1<<ŝ\n		}\n		z[n-1] = w1 >> s\n	}\n	return\n}\n\nfunc mulAddVWW_g(z, x []Word, y, r Word) (c Word) {\n	c = r\n	for i := range z {\n		c, z[i] = mulAddWWW_g(x[i], y, c)\n	}\n	return\n}\n\nfunc addMulVVW_g(z, x []Word, y Word) (c Word) {\n	for i := range z {\n		z1, z0 := mulAddWWW_g(x[i], y, z[i])\n		c, z[i] = addWW_g(z0, c, 0)\n		c += z1\n	}\n	return\n}\n\nfunc divWVW_g(z []Word, xn Word, x []Word, y Word) (r Word) {\n	r = xn\n	for i := len(z) - 1; i >= 0; i-- {\n		z[i], r = divWW_g(r, x[i], y)\n	}\n	return\n}", 0)

	case importPath == "reflect_test" && basename == "all_test.go":
		removeImport("unsafe")
		removeFunction("TestAlignment")
		removeFunction("TestSliceOverflow")

	case importPath == "runtime" && strings.HasPrefix(basename, "zgoarch_"):
		file, _ = parser.ParseFile(fileSet, basename, "package runtime\nconst theGoarch = `js`\n", 0)

	case importPath == "sync/atomic_test" && basename == "atomic_test.go":
		removeFunction("TestUnaligned64")

	case importPath == "text/template/parse" && basename == "lex.go": // this patch will be removed as soon as goroutines are supported
		file, _ = parser.ParseFile(fileSet, basename, "// Copyright 2011 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\npackage parse\nimport (\n	\"container/list\"\n	\"fmt\"\n	\"strings\"\n	\"unicode\"\n	\"unicode/utf8\"\n)\n// item represents a token or text string returned from the scanner.\ntype item struct {\n	typ itemType // The type of this item.\n	pos Pos      // The starting position, in bytes, of this item in the input string.\n	val string   // The value of this item.\n}\nfunc (i item) String() string {\n	switch {\n	case i.typ == itemEOF:\n		return \"EOF\"\n	case i.typ == itemError:\n		return i.val\n	case i.typ > itemKeyword:\n		return fmt.Sprintf(\"<%s>\", i.val)\n	case len(i.val) > 10:\n		return fmt.Sprintf(\"%.10q...\", i.val)\n	}\n	return fmt.Sprintf(\"%q\", i.val)\n}\n// itemType identifies the type of lex items.\ntype itemType int\nconst (\n	itemError        itemType = iota // error occurred; value is text of error\n	itemBool                         // boolean constant\n	itemChar                         // printable ASCII character; grab bag for comma etc.\n	itemCharConstant                 // character constant\n	itemComplex                      // complex constant (1+2i); imaginary is just a number\n	itemColonEquals                  // colon-equals (':=') introducing a declaration\n	itemEOF\n	itemField      // alphanumeric identifier starting with '.'\n	itemIdentifier // alphanumeric identifier not starting with '.'\n	itemLeftDelim  // left action delimiter\n	itemLeftParen  // '(' inside action\n	itemNumber     // simple number, including imaginary\n	itemPipe       // pipe symbol\n	itemRawString  // raw quoted string (includes quotes)\n	itemRightDelim // right action delimiter\n	itemRightParen // ')' inside action\n	itemSpace      // run of spaces separating arguments\n	itemString     // quoted string (includes quotes)\n	itemText       // plain text\n	itemVariable   // variable starting with '$', such as '$' or  '$1' or '$hello'\n	// Keywords appear after all the rest.\n	itemKeyword  // used only to delimit the keywords\n	itemDot      // the cursor, spelled '.'\n	itemDefine   // define keyword\n	itemElse     // else keyword\n	itemEnd      // end keyword\n	itemIf       // if keyword\n	itemNil      // the untyped nil constant, easiest to treat as a keyword\n	itemRange    // range keyword\n	itemTemplate // template keyword\n	itemWith     // with keyword\n)\nvar key = map[string]itemType{\n	\".\":        itemDot,\n	\"define\":   itemDefine,\n	\"else\":     itemElse,\n	\"end\":      itemEnd,\n	\"if\":       itemIf,\n	\"range\":    itemRange,\n	\"nil\":      itemNil,\n	\"template\": itemTemplate,\n	\"with\":     itemWith,\n}\nconst eof = -1\n// stateFn represents the state of the scanner as a function that returns the next state.\ntype stateFn func(*lexer) stateFn\n// lexer holds the state of the scanner.\ntype lexer struct {\n	name       string     // the name of the input; used only for error reports\n	input      string     // the string being scanned\n	leftDelim  string     // start of action\n	rightDelim string     // end of action\n	state      stateFn    // the next lexing function to enter\n	pos        Pos        // current position in the input\n	start      Pos        // start position of this item\n	width      Pos        // width of last rune read from input\n	lastPos    Pos        // position of most recent item returned by nextItem\n	items      *list.List // scanned items\n	parenDepth int        // nesting depth of ( ) exprs\n}\n// next returns the next rune in the input.\nfunc (l *lexer) next() rune {\n	if int(l.pos) >= len(l.input) {\n		l.width = 0\n		return eof\n	}\n	r, w := utf8.DecodeRuneInString(l.input[l.pos:])\n	l.width = Pos(w)\n	l.pos += l.width\n	return r\n}\n// peek returns but does not consume the next rune in the input.\nfunc (l *lexer) peek() rune {\n	r := l.next()\n	l.backup()\n	return r\n}\n// backup steps back one rune. Can only be called once per call of next.\nfunc (l *lexer) backup() {\n	l.pos -= l.width\n}\n// emit passes an item back to the client.\nfunc (l *lexer) emit(t itemType) {\n	l.items.PushBack(item{t, l.start, l.input[l.start:l.pos]})\n	l.start = l.pos\n}\n// ignore skips over the pending input before this point.\nfunc (l *lexer) ignore() {\n	l.start = l.pos\n}\n// accept consumes the next rune if it's from the valid set.\nfunc (l *lexer) accept(valid string) bool {\n	if strings.IndexRune(valid, l.next()) >= 0 {\n		return true\n	}\n	l.backup()\n	return false\n}\n// acceptRun consumes a run of runes from the valid set.\nfunc (l *lexer) acceptRun(valid string) {\n	for strings.IndexRune(valid, l.next()) >= 0 {\n	}\n	l.backup()\n}\n// lineNumber reports which line we're on, based on the position of\n// the previous item returned by nextItem. Doing it this way\n// means we don't have to worry about peek double counting.\nfunc (l *lexer) lineNumber() int {\n	return 1 + strings.Count(l.input[:l.lastPos], \"\\n\")\n}\n// errorf returns an error token and terminates the scan by passing\n// back a nil pointer that will be the next state, terminating l.nextItem.\nfunc (l *lexer) errorf(format string, args ...interface{}) stateFn {\n	l.items.PushBack(item{itemError, l.start, fmt.Sprintf(format, args...)})\n	return nil\n}\n// nextItem returns the next item from the input.\nfunc (l *lexer) nextItem() item {\n	element := l.items.Front()\n	for element == nil {\n		l.state = l.state(l)\n		element = l.items.Front()\n	}\n	l.items.Remove(element)\n	item := element.Value.(item)\n	l.lastPos = item.pos\n	return item\n}\n// lex creates a new scanner for the input string.\nfunc lex(name, input, left, right string) *lexer {\n	if left == \"\" {\n		left = leftDelim\n	}\n	if right == \"\" {\n		right = rightDelim\n	}\n	l := &lexer{\n		name:       name,\n		input:      input,\n		leftDelim:  left,\n		rightDelim: right,\n		items:      list.New(),\n	}\n	l.state = lexText\n	return l\n}\n// state functions\nconst (\n	leftDelim    = \"{{\"\n	rightDelim   = \"}}\"\n	leftComment  = \"/*\"\n	rightComment = \"*/\"\n)\n// lexText scans until an opening action delimiter, \"{{\".\nfunc lexText(l *lexer) stateFn {\n	for {\n		if strings.HasPrefix(l.input[l.pos:], l.leftDelim) {\n			if l.pos > l.start {\n				l.emit(itemText)\n			}\n			return lexLeftDelim\n		}\n		if l.next() == eof {\n			break\n		}\n	}\n	// Correctly reached EOF.\n	if l.pos > l.start {\n		l.emit(itemText)\n	}\n	l.emit(itemEOF)\n	return nil\n}\n// lexLeftDelim scans the left delimiter, which is known to be present.\nfunc lexLeftDelim(l *lexer) stateFn {\n	l.pos += Pos(len(l.leftDelim))\n	if strings.HasPrefix(l.input[l.pos:], leftComment) {\n		return lexComment\n	}\n	l.emit(itemLeftDelim)\n	l.parenDepth = 0\n	return lexInsideAction\n}\n// lexComment scans a comment. The left comment marker is known to be present.\nfunc lexComment(l *lexer) stateFn {\n	l.pos += Pos(len(leftComment))\n	i := strings.Index(l.input[l.pos:], rightComment)\n	if i < 0 {\n		return l.errorf(\"unclosed comment\")\n	}\n	l.pos += Pos(i + len(rightComment))\n	if !strings.HasPrefix(l.input[l.pos:], l.rightDelim) {\n		return l.errorf(\"comment ends before closing delimiter\")\n	}\n	l.pos += Pos(len(l.rightDelim))\n	l.ignore()\n	return lexText\n}\n// lexRightDelim scans the right delimiter, which is known to be present.\nfunc lexRightDelim(l *lexer) stateFn {\n	l.pos += Pos(len(l.rightDelim))\n	l.emit(itemRightDelim)\n	return lexText\n}\n// lexInsideAction scans the elements inside action delimiters.\nfunc lexInsideAction(l *lexer) stateFn {\n	// Either number, quoted string, or identifier.\n	// Spaces separate arguments; runs of spaces turn into itemSpace.\n	// Pipe symbols separate and are emitted.\n	if strings.HasPrefix(l.input[l.pos:], l.rightDelim) {\n		if l.parenDepth == 0 {\n			return lexRightDelim\n		}\n		return l.errorf(\"unclosed left paren\")\n	}\n	switch r := l.next(); {\n	case r == eof || isEndOfLine(r):\n		return l.errorf(\"unclosed action\")\n	case isSpace(r):\n		return lexSpace\n	case r == ':':\n		if l.next() != '=' {\n			return l.errorf(\"expected :=\")\n		}\n		l.emit(itemColonEquals)\n	case r == '|':\n		l.emit(itemPipe)\n	case r == '\"':\n		return lexQuote\n	case r == '`':\n		return lexRawQuote\n	case r == '$':\n		return lexVariable\n	case r == '\\'':\n		return lexChar\n	case r == '.':\n		// special look-ahead for \".field\" so we don't break l.backup().\n		if l.pos < Pos(len(l.input)) {\n			r := l.input[l.pos]\n			if r < '0' || '9' < r {\n				return lexField\n			}\n		}\n		fallthrough // '.' can start a number.\n	case r == '+' || r == '-' || ('0' <= r && r <= '9'):\n		l.backup()\n		return lexNumber\n	case isAlphaNumeric(r):\n		l.backup()\n		return lexIdentifier\n	case r == '(':\n		l.emit(itemLeftParen)\n		l.parenDepth++\n		return lexInsideAction\n	case r == ')':\n		l.emit(itemRightParen)\n		l.parenDepth--\n		if l.parenDepth < 0 {\n			return l.errorf(\"unexpected right paren %#U\", r)\n		}\n		return lexInsideAction\n	case r <= unicode.MaxASCII && unicode.IsPrint(r):\n		l.emit(itemChar)\n		return lexInsideAction\n	default:\n		return l.errorf(\"unrecognized character in action: %#U\", r)\n	}\n	return lexInsideAction\n}\n// lexSpace scans a run of space characters.\n// One space has already been seen.\nfunc lexSpace(l *lexer) stateFn {\n	for isSpace(l.peek()) {\n		l.next()\n	}\n	l.emit(itemSpace)\n	return lexInsideAction\n}\n// lexIdentifier scans an alphanumeric.\nfunc lexIdentifier(l *lexer) stateFn {\nLoop:\n	for {\n		switch r := l.next(); {\n		case isAlphaNumeric(r):\n			// absorb.\n		default:\n			l.backup()\n			word := l.input[l.start:l.pos]\n			if !l.atTerminator() {\n				return l.errorf(\"bad character %#U\", r)\n			}\n			switch {\n			case key[word] > itemKeyword:\n				l.emit(key[word])\n			case word[0] == '.':\n				l.emit(itemField)\n			case word == \"true\", word == \"false\":\n				l.emit(itemBool)\n			default:\n				l.emit(itemIdentifier)\n			}\n			break Loop\n		}\n	}\n	return lexInsideAction\n}\n// lexField scans a field: .Alphanumeric.\n// The . has been scanned.\nfunc lexField(l *lexer) stateFn {\n	return lexFieldOrVariable(l, itemField)\n}\n// lexVariable scans a Variable: $Alphanumeric.\n// The $ has been scanned.\nfunc lexVariable(l *lexer) stateFn {\n	if l.atTerminator() { // Nothing interesting follows -> \"$\".\n		l.emit(itemVariable)\n		return lexInsideAction\n	}\n	return lexFieldOrVariable(l, itemVariable)\n}\n// lexVariable scans a field or variable: [.$]Alphanumeric.\n// The . or $ has been scanned.\nfunc lexFieldOrVariable(l *lexer, typ itemType) stateFn {\n	if l.atTerminator() { // Nothing interesting follows -> \".\" or \"$\".\n		if typ == itemVariable {\n			l.emit(itemVariable)\n		} else {\n			l.emit(itemDot)\n		}\n		return lexInsideAction\n	}\n	var r rune\n	for {\n		r = l.next()\n		if !isAlphaNumeric(r) {\n			l.backup()\n			break\n		}\n	}\n	if !l.atTerminator() {\n		return l.errorf(\"bad character %#U\", r)\n	}\n	l.emit(typ)\n	return lexInsideAction\n}\n// atTerminator reports whether the input is at valid termination character to\n// appear after an identifier. Breaks .X.Y into two pieces. Also catches cases\n// like \"$x+2\" not being acceptable without a space, in case we decide one\n// day to implement arithmetic.\nfunc (l *lexer) atTerminator() bool {\n	r := l.peek()\n	if isSpace(r) || isEndOfLine(r) {\n		return true\n	}\n	switch r {\n	case eof, '.', ',', '|', ':', ')', '(':\n		return true\n	}\n	// Does r start the delimiter? This can be ambiguous (with delim==\"//\", $x/2 will\n	// succeed but should fail) but only in extremely rare cases caused by willfully\n	// bad choice of delimiter.\n	if rd, _ := utf8.DecodeRuneInString(l.rightDelim); rd == r {\n		return true\n	}\n	return false\n}\n// lexChar scans a character constant. The initial quote is already\n// scanned. Syntax checking is done by the parser.\nfunc lexChar(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case '\\\\':\n			if r := l.next(); r != eof && r != '\\n' {\n				break\n			}\n			fallthrough\n		case eof, '\\n':\n			return l.errorf(\"unterminated character constant\")\n		case '\\'':\n			break Loop\n		}\n	}\n	l.emit(itemCharConstant)\n	return lexInsideAction\n}\n// lexNumber scans a number: decimal, octal, hex, float, or imaginary. This\n// isn't a perfect number scanner - for instance it accepts \".\" and \"0x0.2\"\n// and \"089\" - but when it's wrong the input is invalid and the parser (via\n// strconv) will notice.\nfunc lexNumber(l *lexer) stateFn {\n	if !l.scanNumber() {\n		return l.errorf(\"bad number syntax: %q\", l.input[l.start:l.pos])\n	}\n	if sign := l.peek(); sign == '+' || sign == '-' {\n		// Complex: 1+2i. No spaces, must end in 'i'.\n		if !l.scanNumber() || l.input[l.pos-1] != 'i' {\n			return l.errorf(\"bad number syntax: %q\", l.input[l.start:l.pos])\n		}\n		l.emit(itemComplex)\n	} else {\n		l.emit(itemNumber)\n	}\n	return lexInsideAction\n}\nfunc (l *lexer) scanNumber() bool {\n	// Optional leading sign.\n	l.accept(\"+-\")\n	// Is it hex?\n	digits := \"0123456789\"\n	if l.accept(\"0\") && l.accept(\"xX\") {\n		digits = \"0123456789abcdefABCDEF\"\n	}\n	l.acceptRun(digits)\n	if l.accept(\".\") {\n		l.acceptRun(digits)\n	}\n	if l.accept(\"eE\") {\n		l.accept(\"+-\")\n		l.acceptRun(\"0123456789\")\n	}\n	// Is it imaginary?\n	l.accept(\"i\")\n	// Next thing mustn't be alphanumeric.\n	if isAlphaNumeric(l.peek()) {\n		l.next()\n		return false\n	}\n	return true\n}\n// lexQuote scans a quoted string.\nfunc lexQuote(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case '\\\\':\n			if r := l.next(); r != eof && r != '\\n' {\n				break\n			}\n			fallthrough\n		case eof, '\\n':\n			return l.errorf(\"unterminated quoted string\")\n		case '\"':\n			break Loop\n		}\n	}\n	l.emit(itemString)\n	return lexInsideAction\n}\n// lexRawQuote scans a raw quoted string.\nfunc lexRawQuote(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case eof, '\\n':\n			return l.errorf(\"unterminated raw quoted string\")\n		case '`':\n			break Loop\n		}\n	}\n	l.emit(itemRawString)\n	return lexInsideAction\n}\n// isSpace reports whether r is a space character.\nfunc isSpace(r rune) bool {\n	return r == ' ' || r == '\\t'\n}\n// isEndOfLine reports whether r is an end-of-line character.\nfunc isEndOfLine(r rune) bool {\n	return r == '\\r' || r == '\\n'\n}\n// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.\nfunc isAlphaNumeric(r rune) bool {\n	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)\n}", 0)
	}

	return file
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
