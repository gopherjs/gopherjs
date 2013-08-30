package main

import (
	"bytes"
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

type Context struct {
	writer      io.Writer
	indentation int
	info        *types.Info
}

func (c *Context) Write(b []byte) (int, error) {
	return c.writer.Write(b)
}

func (c *Context) Print(format string, values ...interface{}) {
	c.Write([]byte(strings.Repeat("  ", c.indentation)))
	fmt.Fprintf(c, format, values...)
	c.Write([]byte{'\n'})
}

func (c *Context) Indent(f func()) {
	c.indentation += 1
	f()
	c.indentation -= 1
}

func (c *Context) CatchOutput(f func()) string {
	origWriter := c.writer
	b := bytes.NewBuffer(nil)
	c.writer = b
	f()
	c.writer = origWriter
	return b.String()
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

	c := &Context{
		writer: os.Stdout,
		info: &types.Info{
			Types:   make(map[ast.Expr]types.Type),
			Objects: make(map[*ast.Ident]types.Object),
		},
	}
	_, err = config.Check(files[0].Name.Name, fileSet, files, c.info)
	if err != nil {
		panic(err)
	}

	prelude, err := os.Open("prelude.js")
	if err != nil {
		panic(err)
	}
	io.Copy(c, prelude)
	prelude.Close()

	for _, file := range files {
		for _, decl := range file.Decls {
			c.translateDecl(decl)
		}
	}

	c.Print("main();")
}

func (c *Context) translateDecl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.VAR, token.CONST:
			for _, spec := range d.Specs {
				valueSpec := spec.(*ast.ValueSpec)

				defaultValue := "null"
				switch t := c.info.Types[valueSpec.Type].(type) {
				case *types.Basic:
					if t.Info()&types.IsInteger != 0 {
						defaultValue = "0"
					}
				case *types.Array:
					switch elt := t.Elem().(type) {
					case *types.Basic:
						defaultValue = fmt.Sprintf("newNumericArray(%d)", t.Len())
					// 	defaultValue = fmt.Sprintf("new %s(%d)", toTypedArray(elt), t.Len())
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
						value = c.translateExpr(valueSpec.Values[i])
					}
					c.Print("var %s = %s;", name, value)
				}
			}
		case token.TYPE:
			for _, spec := range d.Specs {
				nt := c.info.Objects[spec.(*ast.TypeSpec).Name].Type().(*types.Named)
				switch t := nt.Underlying().(type) {
				case *types.Basic:
					// skip
				case *types.Struct:
					params := make([]string, t.NumFields())
					for i := 0; i < t.NumFields(); i++ {
						params[i] = t.Field(i).Name()
					}
					c.Print("var %s = function(%s) {", nt.Obj().Name(), strings.Join(params, ", "))
					c.Indent(func() {
						for i := 0; i < t.NumFields(); i++ {
							c.Print("this.%s = %s;", t.Field(i).Name(), t.Field(i).Name())
						}
					})
					c.Print("};")
				case *types.Slice:
					// switch elt := t.Elem().(type) {
					// case *types.Basic:
					// 	// 	c.Print("var %s = %s;", nt.Obj().Name(), toTypedArray(elt))
					// case *types.Named:
					c.Print("var %s = function() { Slice.apply(this, arguments); };", nt.Obj().Name())
					c.Print("var _keys = Object.keys(Slice.prototype); for(var i = 0; i < _keys.length; i++) { %s.prototype[_keys[i]] = Slice.prototype[_keys[i]]; }", nt.Obj().Name())
					// default:
					// 	panic(fmt.Sprintf("Unhandled element type: %T\n", elt))
					// }
				case *types.Interface:
				default:
					panic(fmt.Sprintf("Unhandled type: %T\n", t))
				}
			}
		case token.IMPORT:
			// ignored
		default:
			panic("Unhandled declaration: " + d.Tok.String())
		}

	case *ast.FuncDecl:
		c.Print("var %s = function(%s) {", d.Name.Name, translateParams(c.info.Objects[d.Name].Type().(*types.Signature).Params()))
		c.Indent(func() {
			c.translateStmtList(d.Body.List)
		})
		c.Print("};")

	default:
		panic(fmt.Sprintf("Unhandled declaration: %T\n", d))

	}
}

func (c *Context) translateStmtList(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.BlockStmt:
			c.Print("{")
			c.Indent(func() {
				c.translateStmtList(s.List)
			})
			c.Print("}")

		case *ast.IfStmt:
			c.Print("if (%s) {", c.translateExpr(s.Cond))
			c.Indent(func() {
				c.translateStmtList(s.Body.List)
			})
			if s.Else != nil {
				c.Print("} else")
				c.translateStmtList([]ast.Stmt{s.Else})
				continue
			}
			c.Print("}")

		case *ast.SwitchStmt:
			if s.Init != nil {
				c.Print(c.translateStmt(s.Init) + ";")
			}

			if s.Tag == nil {
				if s.Body.List == nil {
					continue
				}
				if len(s.Body.List) == 1 && s.Body.List[0].(*ast.CaseClause).List == nil {
					c.translateStmtList(s.Body.List[0].(*ast.CaseClause).Body)
					continue
				}

				clauseStmts := make([][]ast.Stmt, len(s.Body.List))
				openClauses := make([]int, 0)
				for i, child := range s.Body.List {
					caseClause := child.(*ast.CaseClause)
					openClauses = append(openClauses, i)
					for _, j := range openClauses {
						clauseStmts[j] = append(clauseStmts[j], caseClause.Body...)
					}
					if !hasFallthrough(caseClause) {
						openClauses = nil
					}
				}

				var defaultClause []ast.Stmt
				for i, child := range s.Body.List {
					caseClause := child.(*ast.CaseClause)
					if len(caseClause.List) == 0 {
						defaultClause = clauseStmts[i]
						continue
					}
					conds := make([]string, len(caseClause.List))
					for i, cond := range caseClause.List {
						conds[i] = c.translateExpr(cond)
					}
					c.Print("if (%s) {", strings.Join(conds, " || "))
					c.Indent(func() {
						c.translateStmtList(clauseStmts[i])
					})
					if i < len(s.Body.List)-1 || defaultClause != nil {
						c.Print("} else")
						continue
					}
					c.Print("}")
				}
				if defaultClause != nil {
					c.Print("{")
					c.Indent(func() {
						c.translateStmtList(defaultClause)
					})
					c.Print("}")
				}
				continue
			}

			c.Print("switch (%s) {", c.translateExpr(s.Tag))
			hasDefault := false
			for _, child := range s.Body.List {
				caseClause := child.(*ast.CaseClause)
				for _, cond := range caseClause.List {
					c.Print("case %s:", c.translateExpr(cond))
				}
				if len(caseClause.List) == 0 {
					c.Print("default:")
					hasDefault = true
				}
				c.Indent(func() {
					c.translateStmtList(caseClause.Body)
					if !hasFallthrough(caseClause) {
						c.Print("break;")
					}
				})
			}
			if !hasDefault {
				c.Print("default:")
				c.Print("  // empty")
				c.Print("  break;")
			}
			c.Print("}")

		case *ast.ForStmt:
			c.Print("for (%s; %s; %s) {", c.translateStmt(s.Init), c.translateExpr(s.Cond), c.translateStmt(s.Post))
			c.Indent(func() {
				c.translateStmtList(s.Body.List)
			})
			c.Print("}")

		case *ast.RangeStmt:
			keyAssign := ""
			if s.Key != nil && s.Key.(*ast.Ident).Name != "_" {
				keyAssign = fmt.Sprintf(", %s = _i", s.Key.(*ast.Ident).Name)
			}
			c.Print("var _ref = %s;", c.translateExpr(s.X))
			c.Print("var _i, _len;")
			c.Print("for (_i = 0%s, _len = _ref.length; _i < _len; _i++%s) {", keyAssign, keyAssign)
			c.Indent(func() {
				if s.Value != nil && s.Value.(*ast.Ident).Name != "_" {
					switch t := c.info.Types[s.X].Underlying().(type) {
					case *types.Array:
						c.Print("var %s = _ref[_i];", s.Value.(*ast.Ident).Name)
					case *types.Slice:
						c.Print("var %s = _ref.get(_i);", s.Value.(*ast.Ident).Name)
					default:
						panic(fmt.Sprintf("Unhandled range type: %T\n", t))
					}
				}
				c.translateStmtList(s.Body.List)
			})
			c.Print("}")

		case *ast.BranchStmt:
			switch s.Tok {
			case token.BREAK:
				c.Print("break;")
			case token.CONTINUE:
				c.Print("continue;")
			case token.GOTO:
				c.Print(`throw "goto not implemented";`)
			case token.FALLTHROUGH:
				// handled in CaseClause
			default:
				panic("Unhandled branch statment: " + s.Tok.String())
			}

		case *ast.ReturnStmt:
			switch len(s.Results) {
			case 0:
				c.Print("return;")
			case 1:
				c.Print("return %s;", c.translateExpr(s.Results[0]))
			default:
				results := make([]string, len(s.Results))
				for i, result := range s.Results {
					results[i] = c.translateExpr(result)
				}
				c.Print("return [%s];", strings.Join(results, ", "))
			}

		case *ast.ExprStmt:
			c.Print("%s;", c.translateExpr(s.X))

		case *ast.DeclStmt:
			c.translateDecl(s.Decl)

		case *ast.LabeledStmt:
			c.Print("// label: %s", s.Label.Name)

		default:
			c.Print("%s;", c.translateStmt(s))

		}
	}

}

func (c *Context) translateStmt(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if len(s.Lhs) > 1 {
			exprs := make([]string, len(s.Rhs))
			for i, rhs := range s.Rhs {
				exprs[i] = c.translateExpr(rhs)
			}
			rhs := exprs[0]
			if len(exprs) > 1 {
				rhs = "[" + strings.Join(exprs, ", ") + "]"
			}

			assignments := make([]string, len(s.Lhs))
			for i, lhs := range s.Lhs {
				assignments[i] = fmt.Sprintf("%s = _ref[%d]", c.translateExpr(lhs), i)
			}

			return fmt.Sprintf("var _ref = %s, %s", rhs, strings.Join(assignments, ", "))
		}

		if s.Tok == token.DEFINE {
			return fmt.Sprintf("var %s = %s", c.translateExpr(s.Lhs[0]), c.translateExpr(s.Rhs[0]))
		}

		if iExpr, ok := s.Lhs[0].(*ast.IndexExpr); ok && s.Tok == token.ASSIGN {
			return fmt.Sprintf("%s.set(%s, %s)", c.translateExpr(iExpr.X), c.translateExpr(iExpr.Index), c.translateExpr(s.Rhs[0]))
		}

		if id, isIdent := s.Lhs[0].(*ast.Ident); isIdent && id.Name == "_" {
			return c.translateExpr(s.Rhs[0])
		}

		return fmt.Sprintf("%s %s %s", c.translateExpr(s.Lhs[0]), s.Tok, c.translateExpr(s.Rhs[0]))

	case *ast.IncDecStmt:
		return fmt.Sprintf("%s%s", c.translateExpr(s.X), s.Tok)

	case nil:
		return ""

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
	return ""
}

func (c *Context) translateExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.CHAR {
			return fmt.Sprintf("%s.charCodeAt(0)", e.Value)
		}
		if e.Kind == token.STRING && e.Value[0] == '`' {
			return `"` + strings.Replace(e.Value[1:len(e.Value)-1], `"`, `\"`, -1) + `"`
		}
		return e.Value

	case *ast.CompositeLit:
		elements := make([]string, len(e.Elts))
		for i, element := range e.Elts {
			elements[i] = c.translateExpr(element)
		}
		switch t := c.info.Types[e].(type) {
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
			if s, isSlice := t.Underlying().(*types.Slice); isSlice {
				return fmt.Sprintf("new %s(%s)", t.Obj().Name(), createListComposite(s.Elem(), elements))
			}
			return fmt.Sprintf("new %s(%s)", t.Obj().Name(), strings.Join(elements, ", "))
		default:
			fmt.Println(e.Type, elements)
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", c.info.Types[e]))
		}

	case *ast.FuncLit:
		params := translateParams(c.info.Types[e].(*types.Signature).Params())
		body := c.CatchOutput(func() {
			c.Indent(func() {
				c.translateStmtList(e.Body.List)
			})
			c.Print("")
		})
		return fmt.Sprintf("function (%s) {\n%s}", params, body[:len(body)-1])

	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return c.translateExpr(e.X)
		}
		return fmt.Sprintf("%s%s", e.Op.String(), c.translateExpr(e.X))

	case *ast.BinaryExpr:
		op := e.Op.String()
		if e.Op == token.EQL {
			op = "==="
		}
		if e.Op == token.NEQ {
			op = "!=="
		}
		return fmt.Sprintf("%s %s %s", c.translateExpr(e.X), op, c.translateExpr(e.Y))

	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", c.translateExpr(e.X))

	case *ast.IndexExpr:
		x := c.translateExpr(e.X)
		index := c.translateExpr(e.Index)
		switch t := c.info.Types[e.X].Underlying().(type) {
		case *types.Basic:
			if t.Kind() == types.UntypedString {
				return fmt.Sprintf("%s.charCodeAt(%s)", x, index)
			}
		case *types.Slice:
			return fmt.Sprintf("%s.get(%s)", x, index)
		}
		return fmt.Sprintf("%s[%s]", x, index)

	case *ast.SliceExpr:
		method := "subslice"
		if b, ok := c.info.Types[e.X].(*types.Basic); ok && b.Kind() == types.String {
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
		return fmt.Sprintf("%s.%s", c.translateExpr(e.X), e.Sel.Name)

	case *ast.CallExpr:
		funType := c.info.Types[e.Fun]
		args := make([]string, len(e.Args))
		for i, arg := range e.Args {
			args[i] = c.translateExpr(arg)
		}
		isVariadic, numParams, variadicType := getVariadicInfo(funType)
		if isVariadic && !e.Ellipsis.IsValid() {
			args = append(args[:numParams-1], fmt.Sprintf("new Slice(%s)", createListComposite(variadicType, args[numParams-1:])))
		}
		if e.Ellipsis.IsValid() && len(e.Args) > 0 {
			l := len(e.Args)
			if t, isBasic := c.info.Types[e.Args[l-1]].(*types.Basic); isBasic && t.Kind() == types.UntypedString {
				args[l-1] = fmt.Sprintf("%s.toSlice()", args[l-1])
			}
		}
		if _, isSliceType := funType.(*types.Slice); isSliceType {
			return fmt.Sprintf("(%s).toSlice()", args[0])
		}
		return fmt.Sprintf("%s(%s)", c.translateExpr(e.Fun), strings.Join(args, ", "))

	case *ast.StarExpr:
		return "starExpr"

	case *ast.TypeAssertExpr:
		return c.translateExpr(e.X)

	case *ast.ArrayType:
		return "Slice"
	// 	return toTypedArray(c.info.Types[e].(*types.Slice).Elem().(*types.Basic))

	case *ast.MapType:
		return "Map"

	case *ast.InterfaceType:
		return "Interface"

	case *ast.ChanType:
		return "Channel"

	case *ast.Ident:
		if e.Name == "nil" {
			return "null"
		}
		// if tn, isTypeName := c.info.Objects[e].(*types.TypeName); isTypeName {
		// 	if _, isSlice := tn.Type().Underlying().(*types.Slice); isSlice {
		// 		return "Array"
		// 	}
		// }
		return e.Name

	case nil:
		return ""

	default:
		panic(fmt.Sprintf("Unhandled expression: %T\n", e))

	}
	return ""
}

// func toTypedArray(t *types.Basic) string {
// 	switch t.Kind() {
// 	case types.Int8:
// 		return "Int8Array"
// 	case types.Uint8:
// 		return "Uint8Array"
// 	case types.Int16:
// 		return "Int16Array"
// 	case types.Uint16:
// 		return "Uint16Array"
// 	case types.Int32, types.Int:
// 		return "Int32Array"
// 	case types.Uint32:
// 		return "Uint32Array"
// 	case types.Float32:
// 		return "Float32Array"
// 	case types.Float64, types.Complex64, types.Complex128:
// 		return "Float64Array"
// 	default:
// 		panic("Unhandled typed array: " + t.String())
// 	}
// 	return ""
// }

func createListComposite(elementType types.Type, elements []string) string {
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	// switch elt := elementType.(type) {
	// case *types.Basic:
	// 	switch elt.Kind() {
	// 	case types.Bool, types.String:
	// 		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	// 	default:
	// 		return fmt.Sprintf("new %s([%s])", toTypedArray(elt), strings.Join(elements, ", "))
	// 	}
	// default:
	// 	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	// 	// panic(fmt.Sprintf("Unhandled element type: %T\n", elt))
	// }
}

func getVariadicInfo(funType types.Type) (bool, int, types.Type) {
	switch t := funType.(type) {
	case *types.Signature:
		if t.IsVariadic() {
			return true, t.Params().Len(), t.Params().At(t.Params().Len() - 1).Type()
		}
	case *types.Builtin:
		switch t.Name() {
		case "append":
			return true, 2, types.NewInterface(nil)
		case "print":
			return true, 1, types.NewInterface(nil)
		}
	}
	return false, 0, nil
}

func translateParams(t *types.Tuple) string {
	params := make([]string, t.Len())
	for i := 0; i < t.Len(); i++ {
		params[i] = t.At(i).Name()
	}
	return strings.Join(params, ", ")
}

func hasFallthrough(caseClause *ast.CaseClause) bool {
	if len(caseClause.Body) == 0 {
		return false
	}
	b, isBranchStmt := caseClause.Body[len(caseClause.Body)-1].(*ast.BranchStmt)
	return isBranchStmt && b.Tok == token.FALLTHROUGH
}
