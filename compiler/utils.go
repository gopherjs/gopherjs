package compiler

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"net/url"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/gopherjs/gopherjs/compiler/analysis"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

func (fc *funcContext) Write(b []byte) (int, error) {
	fc.writePos()
	fc.output = append(fc.output, b...)
	return len(b), nil
}

func (fc *funcContext) Printf(format string, values ...interface{}) {
	fc.Write([]byte(strings.Repeat("\t", fc.pkgCtx.indentation)))
	fmt.Fprintf(fc, format, values...)
	fc.Write([]byte{'\n'})
	fc.Write(fc.delayedOutput)
	fc.delayedOutput = nil
}

func (fc *funcContext) PrintCond(cond bool, onTrue, onFalse string) {
	if !cond {
		fc.Printf("/* %s */ %s", strings.Replace(onTrue, "*/", "<star>/", -1), onFalse)
		return
	}
	fc.Printf("%s", onTrue)
}

func (fc *funcContext) SetPos(pos token.Pos) {
	fc.posAvailable = true
	fc.pos = pos
}

func (fc *funcContext) writePos() {
	if fc.posAvailable {
		fc.posAvailable = false
		fc.Write([]byte{'\b'})
		binary.Write(fc, binary.BigEndian, uint32(fc.pos))
	}
}

func (fc *funcContext) Indent(f func()) {
	fc.pkgCtx.indentation++
	f()
	fc.pkgCtx.indentation--
}

func (fc *funcContext) CatchOutput(indent int, f func()) []byte {
	origoutput := fc.output
	fc.output = nil
	fc.pkgCtx.indentation += indent
	f()
	fc.writePos()
	caught := fc.output
	fc.output = origoutput
	fc.pkgCtx.indentation -= indent
	return caught
}

func (fc *funcContext) Delayed(f func()) {
	fc.delayedOutput = fc.CatchOutput(0, f)
}

// expandTupleArgs converts a function call which argument is a tuple returned
// by another function into a set of individual call arguments corresponding to
// tuple elements.
//
// For example, for functions defined as:
//   func a() (int, string) {return 42, "foo"}
//   func b(a1 int, a2 string) {}
// ...the following statement:
//     b(a())
// ...will be transformed into:
//     _tuple := a()
//     b(_tuple[0], _tuple[1])
func (fc *funcContext) expandTupleArgs(argExprs []ast.Expr) []ast.Expr {
	if len(argExprs) != 1 {
		return argExprs
	}

	tuple, isTuple := fc.pkgCtx.TypeOf(argExprs[0]).(*types.Tuple)
	if !isTuple {
		return argExprs
	}

	tupleVar := fc.newVariable("_tuple")
	fc.Printf("%s = %s;", tupleVar, fc.translateExpr(argExprs[0]))
	argExprs = make([]ast.Expr, tuple.Len())
	for i := range argExprs {
		argExprs[i] = fc.newIdent(fc.formatExpr("%s[%d]", tupleVar, i).String(), tuple.At(i).Type())
	}
	return argExprs
}

func (fc *funcContext) translateArgs(sig *types.Signature, argExprs []ast.Expr, ellipsis bool) []string {
	argExprs = fc.expandTupleArgs(argExprs)

	sigTypes := signatureTypes{Sig: sig}

	preserveOrder := false
	for i := 1; i < len(argExprs); i++ {
		preserveOrder = preserveOrder || fc.Blocking[argExprs[i]]
	}

	args := make([]string, len(argExprs))
	for i, argExpr := range argExprs {
		arg := fc.translateImplicitConversionWithCloning(argExpr, sigTypes.Param(i, ellipsis)).String()

		if preserveOrder && fc.pkgCtx.Types[argExpr].Value == nil {
			argVar := fc.newVariable("_arg")
			fc.Printf("%s = %s;", argVar, arg)
			arg = argVar
		}

		args[i] = arg
	}

	// If variadic arguments were passed in as individual elements, regroup them
	// into a slice and pass it as a single argument.
	if sig.Variadic() && !ellipsis {
		return append(args[:sigTypes.RequiredParams()],
			fmt.Sprintf("new %s([%s])", fc.typeName(sigTypes.VariadicType()), strings.Join(args[sigTypes.RequiredParams():], ", ")))
	}
	return args
}

func (fc *funcContext) translateSelection(sel selection, pos token.Pos) ([]string, string) {
	var fields []string
	t := sel.Recv()
	for _, index := range sel.Index() {
		if ptr, isPtr := t.Underlying().(*types.Pointer); isPtr {
			t = ptr.Elem()
		}
		s := t.Underlying().(*types.Struct)
		if jsTag := getJsTag(s.Tag(index)); jsTag != "" {
			jsFieldName := s.Field(index).Name()
			for {
				fields = append(fields, fieldName(s, 0))
				ft := s.Field(0).Type()
				if typesutil.IsJsObject(ft) {
					return fields, jsTag
				}
				ft = ft.Underlying()
				if ptr, ok := ft.(*types.Pointer); ok {
					ft = ptr.Elem().Underlying()
				}
				var ok bool
				s, ok = ft.(*types.Struct)
				if !ok || s.NumFields() == 0 {
					fc.pkgCtx.errList = append(fc.pkgCtx.errList, types.Error{Fset: fc.pkgCtx.fileSet, Pos: pos, Msg: fmt.Sprintf("could not find field with type *js.Object for 'js' tag of field '%s'", jsFieldName), Soft: true})
					return nil, ""
				}
			}
		}
		fields = append(fields, fieldName(s, index))
		t = s.Field(index).Type()
	}
	return fields, ""
}

var nilObj = types.Universe.Lookup("nil")

func (fc *funcContext) zeroValue(ty types.Type) ast.Expr {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		switch {
		case isBoolean(t):
			return fc.newConst(ty, constant.MakeBool(false))
		case isNumeric(t):
			return fc.newConst(ty, constant.MakeInt64(0))
		case isString(t):
			return fc.newConst(ty, constant.MakeString(""))
		case t.Kind() == types.UnsafePointer:
			// fall through to "nil"
		case t.Kind() == types.UntypedNil:
			panic("Zero value for untyped nil.")
		default:
			panic(fmt.Sprintf("Unhandled basic type: %v\n", t))
		}
	case *types.Array, *types.Struct:
		return fc.setType(&ast.CompositeLit{}, ty)
	case *types.Chan, *types.Interface, *types.Map, *types.Signature, *types.Slice, *types.Pointer:
		// fall through to "nil"
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
	id := fc.newIdent("nil", ty)
	fc.pkgCtx.Uses[id] = nilObj
	return id
}

func (fc *funcContext) newConst(t types.Type, value constant.Value) ast.Expr {
	id := &ast.Ident{}
	fc.pkgCtx.Types[id] = types.TypeAndValue{Type: t, Value: value}
	return id
}

func (fc *funcContext) newVariable(name string) string {
	return fc.newVariableWithLevel(name, false)
}

func (fc *funcContext) newVariableWithLevel(name string, pkgLevel bool) string {
	if name == "" {
		panic("newVariable: empty name")
	}
	name = encodeIdent(name)
	if fc.pkgCtx.minify {
		i := 0
		for {
			offset := int('a')
			if pkgLevel {
				offset = int('A')
			}
			j := i
			name = ""
			for {
				name = string(rune(offset+(j%26))) + name
				j = j/26 - 1
				if j == -1 {
					break
				}
			}
			if fc.allVars[name] == 0 {
				break
			}
			i++
		}
	}
	n := fc.allVars[name]
	fc.allVars[name] = n + 1
	varName := name
	if n > 0 {
		varName = fmt.Sprintf("%s$%d", name, n)
	}

	if pkgLevel {
		for c2 := fc.parent; c2 != nil; c2 = c2.parent {
			c2.allVars[name] = n + 1
		}
		return varName
	}

	fc.localVars = append(fc.localVars, varName)
	return varName
}

func (fc *funcContext) newIdent(name string, t types.Type) *ast.Ident {
	ident := ast.NewIdent(name)
	fc.setType(ident, t)
	obj := types.NewVar(0, fc.pkgCtx.Pkg, name, t)
	fc.pkgCtx.Uses[ident] = obj
	fc.pkgCtx.objectNames[obj] = name
	return ident
}

func (fc *funcContext) setType(e ast.Expr, t types.Type) ast.Expr {
	fc.pkgCtx.Types[e] = types.TypeAndValue{Type: t}
	return e
}

func (fc *funcContext) pkgVar(pkg *types.Package) string {
	if pkg == fc.pkgCtx.Pkg {
		return "$pkg"
	}

	pkgVar, found := fc.pkgCtx.pkgVars[pkg.Path()]
	if !found {
		pkgVar = fmt.Sprintf(`$packages["%s"]`, pkg.Path())
	}
	return pkgVar
}

func isVarOrConst(o types.Object) bool {
	switch o.(type) {
	case *types.Var, *types.Const:
		return true
	}
	return false
}

func isPkgLevel(o types.Object) bool {
	return o.Parent() != nil && o.Parent().Parent() == types.Universe
}

func (fc *funcContext) objectName(o types.Object) string {
	if isPkgLevel(o) {
		fc.pkgCtx.dependencies[o] = true

		if o.Pkg() != fc.pkgCtx.Pkg || (isVarOrConst(o) && o.Exported()) {
			return fc.pkgVar(o.Pkg()) + "." + o.Name()
		}
	}

	name, ok := fc.pkgCtx.objectNames[o]
	if !ok {
		name = fc.newVariableWithLevel(o.Name(), isPkgLevel(o))
		fc.pkgCtx.objectNames[o] = name
	}

	if v, ok := o.(*types.Var); ok && fc.pkgCtx.escapingVars[v] {
		return name + "[0]"
	}
	return name
}

func (fc *funcContext) varPtrName(o *types.Var) string {
	if isPkgLevel(o) && o.Exported() {
		return fc.pkgVar(o.Pkg()) + "." + o.Name() + "$ptr"
	}

	name, ok := fc.pkgCtx.varPtrNames[o]
	if !ok {
		name = fc.newVariableWithLevel(o.Name()+"$ptr", isPkgLevel(o))
		fc.pkgCtx.varPtrNames[o] = name
	}
	return name
}

func (fc *funcContext) typeName(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		return "$" + toJavaScriptType(t)
	case *types.Named:
		if t.Obj().Name() == "error" {
			return "$error"
		}
		return fc.objectName(t.Obj())
	case *types.Interface:
		if t.Empty() {
			return "$emptyInterface"
		}
	}

	anonType, ok := fc.pkgCtx.anonTypeMap.At(ty).(*types.TypeName)
	if !ok {
		fc.initArgs(ty) // cause all embedded types to be registered
		varName := fc.newVariableWithLevel(strings.ToLower(typeKind(ty)[5:])+"Type", true)
		anonType = types.NewTypeName(token.NoPos, fc.pkgCtx.Pkg, varName, ty) // fake types.TypeName
		fc.pkgCtx.anonTypes = append(fc.pkgCtx.anonTypes, anonType)
		fc.pkgCtx.anonTypeMap.Set(ty, anonType)
	}
	fc.pkgCtx.dependencies[anonType] = true
	return anonType.Name()
}

func (fc *funcContext) externalize(s string, t types.Type) string {
	if typesutil.IsJsObject(t) {
		return s
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if isNumeric(u) && !is64Bit(u) && !isComplex(u) {
			return s
		}
		if u.Kind() == types.UntypedNil {
			return "null"
		}
	}
	return fmt.Sprintf("$externalize(%s, %s)", s, fc.typeName(t))
}

func (fc *funcContext) handleEscapingVars(n ast.Node) {
	newEscapingVars := make(map[*types.Var]bool)
	for escaping := range fc.pkgCtx.escapingVars {
		newEscapingVars[escaping] = true
	}
	fc.pkgCtx.escapingVars = newEscapingVars

	var names []string
	objs := analysis.EscapingObjects(n, fc.pkgCtx.Info.Info)
	sort.Slice(objs, func(i, j int) bool {
		if objs[i].Name() == objs[j].Name() {
			return objs[i].Pos() < objs[j].Pos()
		}
		return objs[i].Name() < objs[j].Name()
	})
	for _, obj := range objs {
		names = append(names, fc.objectName(obj))
		fc.pkgCtx.escapingVars[obj] = true
	}
	sort.Strings(names)
	for _, name := range names {
		fc.Printf("%s = [%s];", name, name)
	}
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

func isBoolean(t *types.Basic) bool {
	return t.Info()&types.IsBoolean != 0
}

func isComplex(t *types.Basic) bool {
	return t.Info()&types.IsComplex != 0
}

func isFloat(t *types.Basic) bool {
	return t.Info()&types.IsFloat != 0
}

func isInteger(t *types.Basic) bool {
	return t.Info()&types.IsInteger != 0
}

func isNumeric(t *types.Basic) bool {
	return t.Info()&types.IsNumeric != 0
}

func isString(t *types.Basic) bool {
	return t.Info()&types.IsString != 0
}

func isUnsigned(t *types.Basic) bool {
	return t.Info()&types.IsUnsigned != 0
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

// isWrapped returns true for types that may need to be boxed to access full
// functionality of the Go type.
//
// For efficiency or interoperability reasons certain Go types can be represented
// by JavaScript values that weren't constructed by the corresponding Go type
// constructor.
//
// For example, consider a Go type:
//
// 		 type SecretInt int
//     func (_ SecretInt) String() string { return "<secret>" }
//
//     func main() {
//       var i SecretInt = 1
//       println(i.String())
//     }
//
// For this example the compiler will generate code similar to the snippet below:
//
//     SecretInt = $pkg.SecretInt = $newType(4, $kindInt, "main.SecretInt", true, "main", true, null);
//     SecretInt.prototype.String = function() {
//       return "<secret>";
//     };
//     main = function() {
//       var i = 1;
//       console.log(new SecretInt(i).String());
//     };
//
// Note that the generated code assigns a primitive "number" value into i, and
// only boxes it into an object when it's necessary to access its methods.
func isWrapped(ty types.Type) bool {
	switch t := ty.Underlying().(type) {
	case *types.Basic:
		return !is64Bit(t) && !isComplex(t) && t.Kind() != types.UntypedNil
	case *types.Array, *types.Chan, *types.Map, *types.Signature:
		return true
	case *types.Pointer:
		_, isArray := t.Elem().Underlying().(*types.Array)
		return isArray
	}
	return false
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

func rangeCheck(pattern string, constantIndex, array bool) string {
	if constantIndex && array {
		return pattern
	}
	lengthProp := "$length"
	if array {
		lengthProp = "length"
	}
	check := "%2f >= %1e." + lengthProp
	if !constantIndex {
		check = "(%2f < 0 || " + check + ")"
	}
	return "(" + check + ` ? ($throwRuntimeError("index out of range"), undefined) : ` + pattern + ")"
}

func encodeIdent(name string) string {
	return strings.Replace(url.QueryEscape(name), "%", "$", -1)
}

// formatJSStructTagVal returns JavaScript code for accessing an object's property
// identified by jsTag. It prefers the dot notation over the bracket notation when
// possible, since the dot notation produces slightly smaller output.
//
// For example:
//
// 	"my_name" -> ".my_name"
// 	"my name" -> `["my name"]`
//
// For more information about JavaScript property accessors and identifiers, see
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Property_Accessors and
// https://developer.mozilla.org/en-US/docs/Glossary/Identifier.
//
func formatJSStructTagVal(jsTag string) string {
	for i, r := range jsTag {
		ok := unicode.IsLetter(r) || (i != 0 && unicode.IsNumber(r)) || r == '$' || r == '_'
		if !ok {
			// Saw an invalid JavaScript identifier character,
			// so use bracket notation.
			return `["` + template.JSEscapeString(jsTag) + `"]`
		}
	}
	// Safe to use dot notation without any escaping.
	return "." + jsTag
}

// signatureTypes is a helper that provides convenient access to function
// signature type information.
type signatureTypes struct {
	Sig *types.Signature
}

// RequiredParams returns the number of required parameters in the function signature.
func (st signatureTypes) RequiredParams() int {
	l := st.Sig.Params().Len()
	if st.Sig.Variadic() {
		return l - 1 // Last parameter is a slice of variadic params.
	}
	return l
}

// VariadicType returns the slice-type corresponding to the signature's variadic
// parameter, or nil of the signature is not variadic. With the exception of
// the special-case `append([]byte{}, "string"...)`, the returned type is
// `*types.Slice` and `.Elem()` method can be used to get the type of individual
// arguments.
func (st signatureTypes) VariadicType() types.Type {
	if !st.Sig.Variadic() {
		return nil
	}
	return st.Sig.Params().At(st.Sig.Params().Len() - 1).Type()
}

// Returns the expected argument type for the i'th argument position.
//
// This function is able to return correct expected types for variadic calls
// both when ellipsis syntax (e.g. myFunc(requiredArg, optionalArgSlice...))
// is used and when optional args are passed individually.
//
// The returned types may differ from the actual argument expression types if
// there is an implicit type conversion involved (e.g. passing a struct into a
// function that expects an interface).
func (st signatureTypes) Param(i int, ellipsis bool) types.Type {
	if i < st.RequiredParams() {
		return st.Sig.Params().At(i).Type()
	}
	if !st.Sig.Variadic() {
		// This should never happen if the code was type-checked successfully.
		panic(fmt.Errorf("Tried to access parameter %d of a non-variadic signature %s", i, st.Sig))
	}
	if ellipsis {
		return st.VariadicType()
	}
	return st.VariadicType().(*types.Slice).Elem()
}

// ErrorAt annotates an error with a position in the source code.
func ErrorAt(err error, fset *token.FileSet, pos token.Pos) error {
	return fmt.Errorf("%s: %w", fset.Position(pos), err)
}

// FatalError is an error compiler panics with when it encountered a fatal error.
//
// FatalError implements io.Writer, which can be used to record any free-form
// debugging details for human consumption. This information will be included
// into String() result along with the rest.
type FatalError struct {
	cause interface{}
	stack []byte
	clues strings.Builder
}

func (b FatalError) Unwrap() error {
	if b.cause == nil {
		return nil
	}
	if err, ok := b.cause.(error); ok {
		return err
	}
	if s, ok := b.cause.(string); ok {
		return errors.New(s)
	}
	return fmt.Errorf("[%T]: %v", b.cause, b.cause)
}

// Write implements io.Writer and can be used to store free-form debugging clues.
func (b *FatalError) Write(p []byte) (n int, err error) { return b.clues.Write(p) }

func (b FatalError) Error() string {
	buf := &strings.Builder{}
	fmt.Fprintln(buf, "[compiler panic] ", strings.TrimSpace(b.Unwrap().Error()))
	if b.clues.Len() > 0 {
		fmt.Fprintln(buf, "\n"+b.clues.String())
	}
	if len(b.stack) > 0 {
		// Shift stack track by 2 spaces for better readability.
		stack := regexp.MustCompile("(?m)^").ReplaceAll(b.stack, []byte("  "))
		fmt.Fprintln(buf, "\nOriginal stack trace:\n", string(stack))
	}
	return buf.String()
}

func bailout(cause interface{}) *FatalError {
	b := &FatalError{
		cause: cause,
		stack: debug.Stack(),
	}
	return b
}

func bailingOut(err interface{}) (*FatalError, bool) {
	fe, ok := err.(*FatalError)
	return fe, ok
}
