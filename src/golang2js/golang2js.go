package main

import (
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"strings"
)

type Printer struct {
	writer      io.Writer
	indentation int
}

func (p *Printer) Write(b []byte) (int, error) {
	return p.writer.Write(b)
}

func (p *Printer) Print(format string, values ...interface{}) {
	p.Write([]byte(strings.Repeat("  ", p.indentation)))
	fmt.Fprintf(p, format, values...)
	p.Write([]byte{'\n'})
}

func (p *Printer) Indent(f func()) {
	p.indentation += 1
	f()
	p.indentation -= 1
}

func main() {
	fi, err := os.Stat(os.Args[1])
	if err != nil {
		panic(err)
	}

	dir := path.Dir(os.Args[1])
	fileNames := []string{path.Base(os.Args[1])}
	if fi.IsDir() {
		pkg, err := build.ImportDir(os.Args[1], 0)
		if err != nil {
			panic(err)
		}
		dir = pkg.Dir
		fileNames = pkg.GoFiles
	}

	files := make([]*ast.File, 0)
	fileSet := token.NewFileSet()
	for _, name := range fileNames {
		file, err := parser.ParseFile(fileSet, dir+"/"+name, nil, 0)
		if err != nil {
			panic(err)
		}
		files = append(files, file)
	}

	config := &types.Config{
		Error: func(err error) {
			panic(err)
		},
	}

	info := &types.Info{Types: make(map[ast.Expr]types.Type), Objects: make(map[*ast.Ident]types.Object)}
	_, err = config.Check(files[0].Name.Name, fileSet, files, info)
	if err != nil {
		panic(err)
	}

	out := &Printer{writer: os.Stdout}

	prelude, err := os.Open("prelude.js")
	if err != nil {
		panic(err)
	}
	io.Copy(out, prelude)
	prelude.Close()

	for _, file := range files {
		for _, decl := range file.Decls {
			translateDecl(decl, out, info)
		}
	}

	out.Print("main();")
}

func translateDecl(decl ast.Decl, out *Printer, info *types.Info) {
	switch d := decl.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.VAR, token.CONST:
			for _, spec := range d.Specs {
				valueSpec := spec.(*ast.ValueSpec)

				defaultValue := "null"
				switch t := info.Types[valueSpec.Type].(type) {
				case *types.Basic:
					if t.Info()&types.IsInteger != 0 {
						defaultValue = "0"
					}
				case *types.Array:
					switch elt := t.Elem().(type) {
					case *types.Basic:
						defaultValue = fmt.Sprintf("new %s(%d)", toTypedArray(elt), t.Len())
					default:
						panic(fmt.Sprintf("Unhandled element type: %T\n", elt))
					}
				case nil:
					// skip
				default:
					panic(fmt.Sprintf("Unhandled type: %T\n", t))
				}
				for i, name := range valueSpec.Names {
					value := defaultValue
					if len(valueSpec.Values) != 0 {
						value = translateExpr(valueSpec.Values[i], info)
					}
					out.Print("var %s = %s;", name, value)
				}
			}
		}

	case *ast.FuncDecl:
		t := info.Objects[d.Name].Type().(*types.Signature)
		params := make([]string, t.Params().Len())
		for i := 0; i < t.Params().Len(); i++ {
			params[i] = t.Params().At(i).Name()
		}
		out.Print("var %s = function(%s) {", d.Name.Name, strings.Join(params, ", "))
		out.Indent(func() {
			translateStmtList(d.Body.List, out, info)
		})
		out.Print("};")

	default:
		panic(fmt.Sprintf("Unhandled declaration: %T\n", d))

	}
}

func translateStmtList(stmts []ast.Stmt, out *Printer, info *types.Info) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.BlockStmt:
			out.Print("{")
			out.Indent(func() {
				translateStmtList(s.List, out, info)
			})
			out.Print("}")
		case *ast.IfStmt:
			out.Print("if (%s) {", translateExpr(s.Cond, info))
			out.Indent(func() {
				translateStmtList(s.Body.List, out, info)
			})
			if s.Else != nil {
				out.Print("} else")
				translateStmtList([]ast.Stmt{s.Else}, out, info)
				continue
			}
			out.Print("}")
		case *ast.SwitchStmt:
			out.Print("switch (%s) {", translateExpr(s.Tag, info))
			translateStmtList(s.Body.List, out, info)
			out.Print("}")
		case *ast.CaseClause:
			c := "default:"
			if len(s.List) > 0 {
				c = fmt.Sprintf("case %s:", translateExpr(s.List[0], info))
			}
			out.Print(c)
			out.Indent(func() {
				translateStmtList(s.Body, out, info)
				out.Print("break;")
			})
		case *ast.ForStmt:
			out.Print("for (%s; %s; %s) {", translateStmt(s.Init, info), translateExpr(s.Cond, info), translateStmt(s.Post, info))
			out.Indent(func() {
				translateStmtList(s.Body.List, out, info)
			})
			out.Print("}")
		case *ast.RangeStmt:
			keyAssign := ""
			if s.Key != nil && s.Key.(*ast.Ident).Name != "_" {
				keyAssign = s.Key.(*ast.Ident).Name + " = "
			}
			out.Print("var _ref = %s;", translateExpr(s.X, info))
			out.Print("var _i, _len;")
			out.Print("for (%s_i = 0, _len = _ref.length; _i < _len; %s++_i) {", keyAssign, keyAssign)
			out.Indent(func() {
				if s.Value != nil && s.Value.(*ast.Ident).Name != "_" {
					switch info.Types[s.X].(type) {
					case *types.Array:
						out.Print("var %s = _ref[_i];", s.Value.(*ast.Ident).Name)
					case *types.Slice:
						out.Print("var %s = _ref.get(_i);", s.Value.(*ast.Ident).Name)
					}
				}
				translateStmtList(s.Body.List, out, info)
			})
			out.Print("}")
		case *ast.BranchStmt:
			switch s.Tok {
			case token.BREAK:
				out.Print("break;")
			case token.CONTINUE:
				out.Print("continue;")
			case token.GOTO:
				out.Print(`throw "goto not implemented";`)
			default:
				panic("Unhandled branch statment: " + s.Tok.String())
			}
		case *ast.ReturnStmt:
			switch len(s.Results) {
			case 0:
				out.Print("return;")
			case 1:
				out.Print("return %s;", translateExpr(s.Results[0], info))
			default:
				results := make([]string, len(s.Results))
				for i, result := range s.Results {
					results[i] = translateExpr(result, info)
				}
				out.Print("return [%s];", strings.Join(results, ", "))
			}
		case *ast.ExprStmt:
			out.Print("%s;", translateExpr(s.X, info))
		case *ast.DeclStmt:
			translateDecl(s.Decl, out, info)
		case *ast.LabeledStmt:
			out.Print("// label: %s", s.Label.Name)
		default:
			out.Print("%s;", translateStmt(s, info))
		}
	}

}

func translateStmt(stmt ast.Stmt, info *types.Info) string {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if s.Tok == token.DEFINE {
			return fmt.Sprintf("var %s = %s", translateExpr(s.Lhs[0], info), translateExpr(s.Rhs[0], info))
		}
		if iExpr, ok := s.Lhs[0].(*ast.IndexExpr); ok && s.Tok == token.ASSIGN {
			return fmt.Sprintf("%s.set(%s, %s)", translateExpr(iExpr.X, info), translateExpr(iExpr.Index, info), translateExpr(s.Rhs[0], info))
		}
		return fmt.Sprintf("%s %s %s", translateExpr(s.Lhs[0], info), s.Tok, translateExpr(s.Rhs[0], info))
	case *ast.IncDecStmt:
		return fmt.Sprintf("%s%s", translateExpr(s.X, info), s.Tok)
	case nil:
		return ""
	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))
	}
	return ""
}

func translateExpr(expr ast.Expr, info *types.Info) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.CHAR {
			return fmt.Sprintf("%s.charCodeAt(0)", e.Value)
		}
		return e.Value
	case *ast.CompositeLit:
		elements := make([]string, len(e.Elts))
		for i, element := range e.Elts {
			elements[i] = translateExpr(element, info)
		}
		switch t := info.Types[e].(type) {
		case *types.Array:
			return createListComposite(t.Elem(), elements)
		case *types.Slice:
			return fmt.Sprintf("new Slice(%s)", createListComposite(t.Elem(), elements))
		case *types.Struct:
			for i, element := range elements {
				elements[i] = fmt.Sprintf("%s: %s", t.Field(i).Name(), element)
			}
			return fmt.Sprintf("{ %s }", strings.Join(elements, ", "))
		case *types.Named:
			return fmt.Sprintf("new %s(%s)", t.Obj().Name(), strings.Join(elements, ", "))
		default:
			fmt.Println(e.Type, elements)
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", info.Types[e]))
		}
	case *ast.Ident:
		return e.Name
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return translateExpr(e.X, info)
		}
		return fmt.Sprintf("%s%s", e.Op.String(), translateExpr(e.X, info))
	case *ast.BinaryExpr:
		op := e.Op.String()
		if e.Op == token.EQL {
			op = "==="
		}
		if e.Op == token.NEQ {
			op = "!=="
		}
		return fmt.Sprintf("%s %s %s", translateExpr(e.X, info), op, translateExpr(e.Y, info))
	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", translateExpr(e.X, info))
	case *ast.IndexExpr:
		if _, ok := info.Types[e.X].(*types.Slice); ok {
			return fmt.Sprintf("%s.get(%s)", translateExpr(e.X, info), translateExpr(e.Index, info))
		}
		return fmt.Sprintf("%s[%s]", translateExpr(e.X, info), translateExpr(e.Index, info))
	case *ast.SliceExpr:
		method := "subslice"
		if b, ok := info.Types[e.X].(*types.Basic); ok && b.Kind() == types.String {
			method = "substring"
		}
		slice := translateExpr(e.X, info)
		if _, ok := info.Types[e.X].(*types.Array); ok {
			slice = fmt.Sprintf("(new Slice(%s))", slice)
		}
		if e.High == nil {
			return fmt.Sprintf("%s.%s(%s)", slice, method, translateExpr(e.Low, info))
		}
		low := "0"
		if e.Low != nil {
			low = translateExpr(e.Low, info)
		}
		return fmt.Sprintf("%s.%s(%s, %s)", slice, method, low, translateExpr(e.High, info))
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", translateExpr(e.X, info), e.Sel.Name)
	case *ast.CallExpr:
		funType := info.Types[e.Fun]
		args := make([]string, len(e.Args))
		for i, arg := range e.Args {
			args[i] = translateExpr(arg, info)
		}
		isVariadic, numParams, variadicType := getVariadicInfo(funType)
		if isVariadic && !e.Ellipsis.IsValid() {
			args = append(args[:numParams-1], fmt.Sprintf("new Slice(%s)", createListComposite(variadicType, args[numParams-1:])))
		}
		if e.Ellipsis.IsValid() && len(e.Args) > 0 {
			l := len(e.Args)
			if t, isBasic := info.Types[e.Args[l-1]].(*types.Basic); isBasic && t.Kind() == types.UntypedString {
				args[l-1] = fmt.Sprintf("%s.toSlice()", args[l-1])
			}
		}
		if _, isSliceType := funType.(*types.Slice); isSliceType {
			return fmt.Sprintf("(%s).toSlice()", args[0])
		}
		return fmt.Sprintf("%s(%s)", translateExpr(e.Fun, info), strings.Join(args, ", "))
	case *ast.TypeAssertExpr:
		return translateExpr(e.X, info)
	case *ast.ArrayType:
		return toTypedArray(info.Types[e].(*types.Slice).Elem().(*types.Basic))
	case nil:
		return ""
	default:
		panic(fmt.Sprintf("Unhandled expression: %T\n", e))
	}
	return ""
}

func toTypedArray(t *types.Basic) string {
	switch t.Kind() {
	case types.Int8:
		return "Int8Array"
	case types.Uint8:
		return "Uint8Array"
	case types.Int16:
		return "Int16Array"
	case types.Uint16:
		return "Uint16Array"
	case types.Int32, types.Int:
		return "Int32Array"
	case types.Uint32:
		return "Uint32Array"
	case types.Float32:
		return "Float32Array"
	case types.Float64, types.Complex64, types.Complex128:
		return "Float64Array"
	default:
		panic("Unhandled typed array: " + t.String())
	}
	return ""
}

func createListComposite(elementType types.Type, elements []string) string {
	switch elt := elementType.(type) {
	case *types.Basic:
		switch elt.Kind() {
		case types.Bool, types.String:
			return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
		default:
			return fmt.Sprintf("new %s([%s])", toTypedArray(elt), strings.Join(elements, ", "))
		}
	default:
		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
		// panic(fmt.Sprintf("Unhandled element type: %T\n", elt))
	}
}

func getVariadicInfo(funType types.Type) (bool, int, types.Type) {
	switch t := funType.(type) {
	case *types.Signature:
		if t.IsVariadic() {
			return true, t.Params().Len(), t.Params().At(t.Params().Len() - 1).Type()
		}
	case *types.Builtin:
		if t.Name() == "append" {
			return true, 2, types.NewInterface(nil)
		}
	}
	return false, 0, nil
}
