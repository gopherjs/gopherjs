package translator

import (
	"bytes"
	"code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/types"
	"encoding/asn1"
	"fmt"
	"go/ast"
	"io"
	"sort"
	"strconv"
	"strings"
)

var sizes32 = &types.StdSizes{WordSize: 4, MaxAlign: 8}
var typesPackages = map[string]*types.Package{"unsafe": types.Unsafe}

type Archive struct {
	ImportPath   string
	GcData       []byte
	Dependencies []string
	Imports      []Import
	Declarations []Decl
	Tests        []string
}

type Import struct {
	Path    string
	VarName string
}

type Decl struct {
	Var        string
	BodyCode   []byte
	InitCode   []byte
	DceFilters []string
	DceDeps    []Object
}

type Object struct {
	PkgPath string
	Name    string
}

func (a *Archive) AddDependency(path string) {
	for _, dep := range a.Dependencies {
		if dep == path {
			return
		}
	}
	a.Dependencies = append(a.Dependencies, path)
}

func (a *Archive) AddDependenciesOf(other *Archive) {
	for _, path := range other.Dependencies {
		a.AddDependency(path)
	}
}

func NewEmptyTypesPackage(path string) {
	typesPackages[path] = types.NewPackage(path, path)
}

func WriteProgramCode(pkgs []*Archive, mainPkgPath string, w io.Writer) {
	declsByObject := make(map[Object][]*Decl)
	var pendingDecls []*Decl
	for _, pkg := range pkgs {
		for i := range pkg.Declarations {
			d := &pkg.Declarations[i]
			if len(d.DceFilters) == 0 {
				pendingDecls = append(pendingDecls, d)
				continue
			}
			for _, f := range d.DceFilters {
				o := Object{pkg.ImportPath, f}
				declsByObject[o] = append(declsByObject[o], d)
			}
		}
	}

	for len(pendingDecls) != 0 {
		d := pendingDecls[len(pendingDecls)-1]
		pendingDecls = pendingDecls[:len(pendingDecls)-1]
		for _, o := range d.DceDeps {
			if decls, ok := declsByObject[o]; ok {
				delete(declsByObject, o)
				for _, d := range decls {
					for i, f := range d.DceFilters {
						if f == o.Name {
							d.DceFilters[i] = d.DceFilters[len(d.DceFilters)-1]
							d.DceFilters = d.DceFilters[:len(d.DceFilters)-1]
							break
						}
					}
					if len(d.DceFilters) == 0 {
						pendingDecls = append(pendingDecls, d)
					}
				}
			}
		}
	}

	w.Write([]byte("\"use strict\";\n(function() {\n\n"))
	w.Write([]byte(strings.TrimSpace(prelude)))
	w.Write([]byte("\n"))

	// write packages
	for _, pkg := range pkgs {
		WritePkgCode(pkg, w)
	}

	// write interfaces
	merge := false
	allTypeNames := []*types.TypeName{types.New("error").(*types.Named).Obj()}
	for _, pkg := range pkgs {
		scope := typesPackages[pkg.ImportPath].Scope()
		for _, name := range scope.Names() {
			if typeName, isTypeName := scope.Lookup(name).(*types.TypeName); isTypeName {
				if _, notUsed := declsByObject[Object{pkg.ImportPath, name}]; !notUsed {
					allTypeNames = append(allTypeNames, typeName)
				}
			}
		}
	}
	for _, t := range allTypeNames {
		if in, isInterface := t.Type().Underlying().(*types.Interface); isInterface {
			if in.Empty() {
				continue
			}
			implementedBy := make(map[string]bool, 0)
			for _, other := range allTypeNames {
				otherType := other.Type()
				switch otherType.Underlying().(type) {
				case *types.Interface:
					// skip
				case *types.Struct:
					if types.AssignableTo(otherType, in) {
						implementedBy[fmt.Sprintf("go$packages[\"%s\"].%s", other.Pkg().Path(), other.Name())] = true
					}
					if types.AssignableTo(types.NewPointer(otherType), in) {
						implementedBy[fmt.Sprintf("go$packages[\"%s\"].%s.Ptr", other.Pkg().Path(), other.Name())] = true
					}
				default:
					if types.AssignableTo(otherType, in) {
						implementedBy[fmt.Sprintf("go$packages[\"%s\"].%s", other.Pkg().Path(), other.Name())] = true
					}
					if types.AssignableTo(types.NewPointer(otherType), in) {
						implementedBy[fmt.Sprintf("go$ptrType(go$packages[\"%s\"].%s)", other.Pkg().Path(), other.Name())] = true
					}
				}
			}
			list := make([]string, 0, len(implementedBy))
			for ref := range implementedBy {
				list = append(list, ref)
			}
			sort.Strings(list)
			var target string
			switch t.Name() {
			case "error":
				target = "go$error"
			default:
				target = fmt.Sprintf("go$packages[\"%s\"].%s", t.Pkg().Path(), t.Name())
			}
			if merge {
				for _, entry := range list {
					fmt.Fprintf(w, "if (%s.implementedBy.indexOf(%s) === -1) { %s.implementedBy.push(%s); }\n", target, entry, target, entry)
				}
				continue
			}
			fmt.Fprintf(w, "%s.implementedBy = [%s];\n", target, strings.Join(list, ", "))
		}
	}

	for _, pkg := range pkgs {
		w.Write([]byte("go$packages[\"" + pkg.ImportPath + "\"].init();\n"))
	}

	w.Write([]byte("go$packages[\"" + mainPkgPath + "\"].main();\n\n})();"))
}

func WritePkgCode(pkg *Archive, w io.Writer) {
	fmt.Fprintf(w, "go$packages[\"%s\"] = (function() {\n", pkg.ImportPath)
	vars := []string{"go$pkg = {}"}
	for _, imp := range pkg.Imports {
		vars = append(vars, fmt.Sprintf("%s = go$packages[\"%s\"]", imp.VarName, imp.Path))
	}
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 && d.Var != "" {
			vars = append(vars, d.Var)
		}
	}
	if len(vars) != 0 {
		fmt.Fprintf(w, "\tvar %s;\n", strings.Join(vars, ", "))
	}
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 {
			w.Write(d.BodyCode)
		}
	}
	w.Write([]byte("\tgo$pkg.init = function() {\n"))
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 {
			w.Write(d.InitCode)
		}
	}
	w.Write([]byte("\t}\n\treturn go$pkg;\n})();\n"))
}

func ReadArchive(filename, id string, data []byte) (*Archive, error) {
	var a Archive
	_, err := asn1.Unmarshal(data, &a)
	if err != nil {
		return nil, err
	}

	pkg, err := gcimporter.ImportData(typesPackages, filename, id, bytes.NewReader(a.GcData))
	if err != nil {
		return nil, err
	}
	typesPackages[pkg.Path()] = pkg

	return &a, nil
}

func WriteArchive(a *Archive) ([]byte, error) {
	return asn1.Marshal(*a)
}

func (c *pkgContext) translateArgs(sig *types.Signature, args []ast.Expr, ellipsis bool) string {
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
	return strings.Join(params, ", ")
}

func (c *pkgContext) translateSelection(sel *types.Selection) (fields []string, jsTag string) {
	t := sel.Recv()
	for _, index := range sel.Index() {
		if ptr, isPtr := t.(*types.Pointer); isPtr {
			t = ptr.Elem()
		}
		s := t.Underlying().(*types.Struct)
		if jsTag = getJsTag(s.Tag(index)); jsTag != "" {
			for i := 0; i < s.NumFields(); i++ {
				if isJsObject(s.Field(i).Type()) {
					fields = append(fields, fieldName(s, i))
					return
				}
			}
		}
		fields = append(fields, fieldName(s, index))
		t = s.Field(index).Type()
	}
	return
}

func (c *pkgContext) zeroValue(ty types.Type) string {
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
		return fmt.Sprintf(`go$makeNativeArray("%s", %d, function() { return %s; })`, typeKind(t.Elem()), t.Len(), c.zeroValue(t.Elem()))
	case *types.Signature:
		return "go$throwNilPointerError"
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

func (c *pkgContext) newVariable(name string) string {
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
		name = "$" + name[7:]
	}
	n := c.f.allVars[name]
	c.f.allVars[name] = n + 1
	if n > 0 {
		name = fmt.Sprintf("%s$%d", name, n)
	}
	c.f.localVars = append(c.f.localVars, name)
	return name
}

func (c *pkgContext) newIdent(name string, t types.Type) *ast.Ident {
	ident := ast.NewIdent(name)
	c.info.Types[ident] = types.TypeAndValue{Type: t}
	obj := types.NewVar(0, c.pkg, name, t)
	c.info.Uses[ident] = obj
	c.objectVars[obj] = name
	return ident
}

func (c *pkgContext) objectName(o types.Object) string {
	if o.Pkg() != c.pkg || o.Parent() == c.pkg.Scope() {
		c.dependencies[o] = true
	}

	if o.Pkg() != c.pkg {
		pkgVar, found := c.pkgVars[o.Pkg().Path()]
		if !found {
			pkgVar = fmt.Sprintf(`go$packages["%s"]`, o.Pkg().Path())
		}
		return pkgVar + "." + o.Name()
	}

	name, found := c.objectVars[o]
	if !found {
		name = c.newVariable(o.Name())
		c.objectVars[o] = name
	}

	switch o.(type) {
	case *types.Var, *types.Const:
		if o.Exported() && o.Parent() == c.pkg.Scope() {
			return "go$pkg." + name
		}
	}
	return name
}

func (c *pkgContext) typeName(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.UntypedNil:
			return "null"
		case types.UnsafePointer:
			return "Go$UnsafePointer"
		default:
			return "Go$" + toJavaScriptType(t)
		}
	case *types.Named:
		if t.Obj().Name() == "error" {
			return "go$error"
		}
		return c.objectName(t.Obj())
	case *types.Pointer:
		return fmt.Sprintf("(go$ptrType(%s))", c.initArgs(t))
	case *types.Interface:
		if t.Empty() {
			return "go$emptyInterface"
		}
		return fmt.Sprintf("(go$interfaceType(%s))", c.initArgs(t))
	case *types.Array, *types.Chan, *types.Slice, *types.Map, *types.Signature, *types.Struct:
		return fmt.Sprintf("(go$%sType(%s))", strings.ToLower(typeKind(t)), c.initArgs(t))
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
}

func (c *pkgContext) makeKey(expr ast.Expr, keyType types.Type) string {
	switch t := keyType.Underlying().(type) {
	case *types.Array, *types.Struct:
		return fmt.Sprintf("(new %s(%s)).go$key()", c.typeName(keyType), c.translateExpr(expr))
	case *types.Basic:
		if is64Bit(t) {
			return fmt.Sprintf("%s.go$key()", c.translateExpr(expr))
		}
		if t.Info()&types.IsFloat != 0 {
			return fmt.Sprintf("go$floatKey(%s)", c.translateExpr(expr))
		}
		return c.translateImplicitConversion(expr, keyType).String()
	case *types.Chan, *types.Pointer:
		return fmt.Sprintf("%s.go$key()", c.translateImplicitConversion(expr, keyType))
	case *types.Interface:
		return fmt.Sprintf("(%s || go$interfaceNil).go$key()", c.translateImplicitConversion(expr, keyType))
	default:
		return c.translateImplicitConversion(expr, keyType).String()
	}
}

func (c *pkgContext) typeArray(t *types.Tuple) string {
	s := make([]string, t.Len())
	for i := range s {
		s[i] = c.typeName(t.At(i).Type())
	}
	return "[" + strings.Join(s, ", ") + "]"
}

func (c *pkgContext) externalize(s string, t types.Type) string {
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
	return fmt.Sprintf("go$externalize(%s, %s)", s, c.typeName(t))
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

func isJsObject(t types.Type) bool {
	named, isNamed := t.(*types.Named)
	return isNamed && named.Obj().Pkg().Path() == "github.com/gopherjs/gopherjs/js" && named.Obj().Name() == "Object"
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
