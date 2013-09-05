package main

import (
	"bytes"
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"sort"
	"strings"
)

type Context struct {
	writer       io.Writer
	indentation  int
	info         *types.Info
	varToName    map[*types.Var]string
	varNameUsed  map[string]bool
	hasInit      bool
	namedResults []string
}

func (c *Context) newVarName(prefix string) string {
	n := 0
	for {
		name := prefix
		if n != 0 {
			name += fmt.Sprintf("%d", n)
		}
		if !c.varNameUsed[name] {
			c.varNameUsed[name] = true
			return name
		}
		n += 1
	}
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

// TODO start new variable naming scope
func (c *Context) NewScope(namedResults []string, f func()) {
	origNamedResults := c.namedResults
	c.namedResults = namedResults
	f()
	c.namedResults = origNamedResults
}

func (c *Context) CatchOutput(f func()) string {
	origWriter := c.writer
	b := bytes.NewBuffer(nil)
	c.writer = b
	f()
	c.writer = origWriter
	return b.String()
}

type This struct {
	ast.Ident
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
			Values:  make(map[ast.Expr]exact.Value),
			Objects: make(map[*ast.Ident]types.Object),
		},
		varToName: make(map[*types.Var]string),
		varNameUsed: map[string]bool{
			"try": true,
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

	if c.hasInit {
		c.Print("init();")
	}
	c.Print("main();")
}

func (c *Context) translateDecl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.VAR:
			for _, spec := range d.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				defaultValue := zeroValue(c.info.Types[valueSpec.Type])
				for i, name := range valueSpec.Names {
					value := defaultValue
					if len(valueSpec.Values) != 0 {
						value = c.translateExpr(valueSpec.Values[i])
					}
					c.Print("var %s = %s;", c.translateExpr(name), value)
				}
			}
		case token.TYPE:
			for _, spec := range d.Specs {
				nt := c.info.Objects[spec.(*ast.TypeSpec).Name].Type().(*types.Named)
				switch t := nt.Underlying().(type) {
				case *types.Basic:
					c.Print("var %s = function(v) { this.v = v; };", nt.Obj().Name())
					if t.Info()&types.IsString != 0 {
						c.Print("%s.prototype.len = function() { return this.v.length; };", nt.Obj().Name())
					}
				case *types.Struct:
					params := make([]string, t.NumFields())
					for i := 0; i < t.NumFields(); i++ {
						params[i] = t.Field(i).Name()
					}
					c.Print("var %s = function(%s) {", nt.Obj().Name(), strings.Join(params, ", "))
					c.Indent(func() {
						for i := 0; i < t.NumFields(); i++ {
							field := t.Field(i)
							c.Print("this.%s = %s;", field.Name(), field.Name())
						}
					})
					c.Print("};")
					for i := 0; i < t.NumFields(); i++ {
						field := t.Field(i)
						if field.Anonymous() {
							fieldType := field.Type()
							_, isPointer := fieldType.(*types.Pointer)
							_, isUnderlyingInterface := fieldType.Underlying().(*types.Interface)
							if !isPointer && !isUnderlyingInterface {
								fieldType = types.NewPointer(fieldType) // strange, seems like a bug in go/types
							}
							methods := fieldType.MethodSet()
							for j := 0; j < methods.Len(); j++ {
								name := methods.At(j).Obj().Name()
								sig := methods.At(j).Type().(*types.Signature)
								params := make([]string, sig.Params().Len())
								for k := range params {
									params[k] = sig.Params().At(k).Name()
								}
								c.Print("%s.prototype.%s = function(%s) { return this.%s.%s(%s); };", nt.Obj().Name(), name, strings.Join(params, ", "), field.Name(), name, strings.Join(params, ", "))
							}
						}
					}
				case *types.Slice:
					c.Print("var %s = function() { Slice.apply(this, arguments); };", nt.Obj().Name())
					c.Print("var _keys = Object.keys(Slice.prototype); for(var i = 0; i < _keys.length; i++) { %s.prototype[_keys[i]] = Slice.prototype[_keys[i]]; }", nt.Obj().Name())
				case *types.Interface:
					if t.MethodSet().Len() == 0 {
						c.Print("var %s = function(t) { return true };", nt.Obj().Name())
						continue
					}
					implementedBy := make([]string, 0)
					for _, other := range c.info.Objects {
						if otherTypeName, isTypeName := other.(*types.TypeName); isTypeName {
							index := sort.SearchStrings(implementedBy, otherTypeName.Name())
							if (index == len(implementedBy) || implementedBy[index] != otherTypeName.Name()) && types.IsAssignableTo(otherTypeName.Type(), t) {
								implementedBy = append(implementedBy, otherTypeName.Name())
								sort.Strings(implementedBy)
							}
						}
					}
					conditions := make([]string, len(implementedBy))
					for i, other := range implementedBy {
						conditions[i] = "t === " + other
					}
					if len(conditions) == 0 {
						conditions = []string{"false"}
					}
					c.Print("var %s = function(t) { return %s };", nt.Obj().Name(), strings.Join(conditions, " || "))
				default:
					panic(fmt.Sprintf("Unhandled type: %T\n", t))
				}
			}
		case token.IMPORT, token.CONST:
			// ignored
		default:
			panic("Unhandled declaration: " + d.Tok.String())
		}

	case *ast.FuncDecl:
		if d.Name.Name == "init" {
			c.hasInit = true
		}
		var lhs ast.Expr = d.Name
		tok := token.DEFINE
		body := d.Body.List
		if d.Recv != nil {
			recv := d.Recv.List[0].Type
			lhs = &ast.SelectorExpr{
				X: &ast.SelectorExpr{
					X:   recv,
					Sel: ast.NewIdent("prototype"),
				},
				Sel: d.Name,
			}
			tok = token.ASSIGN
			var this ast.Expr = &This{}
			thisType := c.info.Objects[d.Recv.List[0].Names[0]].Type()
			if _, isUnderlyingStruct := thisType.Underlying().(*types.Struct); isUnderlyingStruct {
				this = &ast.StarExpr{X: this}
			}
			c.info.Types[this] = thisType
			body = append([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{d.Recv.List[0].Names[0]},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{this},
				},
			}, body...)
		}
		c.translateStmt(&ast.AssignStmt{
			Tok: tok,
			Lhs: []ast.Expr{lhs},
			Rhs: []ast.Expr{&ast.FuncLit{
				Type: d.Type,
				Body: &ast.BlockStmt{
					List: body,
				},
			}},
		})

	default:
		panic(fmt.Sprintf("Unhandled declaration: %T\n", d))

	}
}

func (c *Context) hasDefer(stmts []ast.Stmt) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.BlockStmt:
			if c.hasDefer(s.List) {
				return true
			}
		case *ast.IfStmt:
			if c.hasDefer(s.Body.List) {
				return true
			}
		case *ast.SwitchStmt:
			if c.hasDefer(s.Body.List) {
				return true
			}
		case *ast.CaseClause:
			if c.hasDefer(s.Body) {
				return true
			}
		case *ast.ForStmt:
			if c.hasDefer(s.Body.List) {
				return true
			}
		case *ast.RangeStmt:
			if c.hasDefer(s.Body.List) {
				return true
			}
		case *ast.DeferStmt:
			return true
		}
	}
	return false
}

func (c *Context) translateStmtList(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		c.translateStmt(stmt)
	}
}

func (c *Context) translateStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		c.Print("{")
		c.Indent(func() {
			c.translateStmtList(s.List)
		})
		c.Print("}")

	case *ast.IfStmt:
		c.translateStmt(s.Init)
		c.Print("if (%s) {", c.translateExpr(s.Cond))
		c.Indent(func() {
			c.translateStmtList(s.Body.List)
		})
		if s.Else != nil {
			c.Print("} else")
			c.translateStmt(s.Else)
			return
		}
		c.Print("}")

	case *ast.SwitchStmt:
		c.translateStmt(s.Init)

		if s.Tag == nil {
			if s.Body.List == nil {
				return
			}
			if len(s.Body.List) == 1 && s.Body.List[0].(*ast.CaseClause).List == nil {
				c.translateStmtList(s.Body.List[0].(*ast.CaseClause).Body)
				return
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
			return
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
		init := strings.TrimSuffix(c.CatchOutput(func() { c.translateStmt(s.Init) }), ";\n")
		post := strings.TrimSuffix(c.CatchOutput(func() { c.translateStmt(s.Post) }), ";\n")
		c.Print("for (%s; %s; %s) {", init, c.translateExpr(s.Cond), post)
		c.Indent(func() {
			c.translateStmtList(s.Body.List)
		})
		c.Print("}")

	case *ast.RangeStmt:
		refVar := c.newVarName("_ref")
		lenVar := c.newVarName("_len")
		iVar := c.newVarName("_i")
		vars := []string{refVar, lenVar, iVar}

		key := c.translateExpr(s.Key)
		value := c.translateExpr(s.Value)
		keyAssign := ""
		if key != "" {
			keyAssign = fmt.Sprintf(", %s = %s", key, iVar)
			if s.Tok == token.DEFINE {
				vars = append(vars, key)
			}
		}
		if value != "" {
			if s.Tok == token.DEFINE {
				vars = append(vars, value)
			}
		}

		c.Print("var %s;", strings.Join(vars, ", "))
		forParams := "" +
			fmt.Sprintf("%s = %s", refVar, c.translateExpr(s.X)) +
			fmt.Sprintf(", %s = %s.length", lenVar, refVar) +
			fmt.Sprintf(", %s = 0", iVar) +
			keyAssign +
			fmt.Sprintf("; %s < %s", iVar, lenVar) +
			fmt.Sprintf("; %s++", iVar) +
			keyAssign
		c.Print("for (%s) {", forParams)
		c.Indent(func() {
			if value != "" {
				switch t := c.info.Types[s.X].Underlying().(type) {
				case *types.Array:
					c.Print("var %s = %s[%s];", value, refVar, iVar)
				case *types.Slice:
					c.Print("var %s = %s.get(%s);", value, refVar, iVar)
				case *types.Basic:
					c.Print("var %s = %s.charCodeAt(%s);", value, refVar, iVar)
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
		results := make([]string, len(s.Results))
		for i, result := range s.Results {
			results[i] = c.translateExpr(result)
			if c.namedResults != nil {
				c.Print("%s = %s;", c.namedResults[i], results[i])
			}
		}
		if c.namedResults != nil {
			results = c.namedResults
		}
		switch len(results) {
		case 0:
			c.Print("return;")
		case 1:
			c.Print("return %s;", results[0])
		default:
			c.Print("return [%s];", strings.Join(results, ", "))
		}

	case *ast.DeferStmt:
		args := c.translateArgs(s.Call)
		c.Print("_deferred.push({ fun: %s, recv: %s, args: [%s] });", c.translateExpr(s.Call.Fun), "this", strings.Join(args, ", ")) // TODO fix receiver

	case *ast.ExprStmt:
		c.Print("%s;", c.translateExpr(s.X))

	case *ast.DeclStmt:
		c.translateDecl(s.Decl)

	case *ast.LabeledStmt:
		c.Print("// label: %s", s.Label.Name)
		c.translateStmt(s.Stmt)

	case *ast.AssignStmt:
		rhsExprs := make([]string, len(s.Rhs))
		// rhsTypes := make([]types.Type, len(s.Rhs))
		for i, rhs := range s.Rhs {
			rhsExprs[i] = c.translateExpr(rhs)
			// rhsTypes[i] = c.info.Types[rhs]
		}
		rhs := rhsExprs[0]
		// completeRhsType := rhsTypes[0]
		if len(rhsExprs) > 1 {
			rhs = "[" + strings.Join(rhsExprs, ", ") + "]"
			// completeRhsType = types.NewTuple(rhsTypes...)
		}

		if len(s.Lhs) > 1 {
			c.Print("_tuple = %s;", rhs)
		}

		for i, l := range s.Lhs {
			lhs := c.translateExpr(l)
			// lhsType := c.info.Types[l]

			// rhsType := completeRhsType
			if len(s.Lhs) > 1 {
				if lhs == "" {
					continue
				}
				rhs = fmt.Sprintf("_tuple[%d]", i)
				// rhsType = completeRhsType.(*types.Tuple).At(i)
			}

			if lhs == "" {
				c.Print("%s;", rhs)
				continue
			}

			if s.Tok == token.DEFINE {
				c.Print("var %s = %s;", lhs, rhs)
				continue
			}

			if iExpr, ok := s.Lhs[0].(*ast.IndexExpr); ok && s.Tok == token.ASSIGN {
				if _, isSlice := c.info.Types[iExpr.X].Underlying().(*types.Slice); isSlice {
					c.Print("%s.set(%s, %s);", c.translateExpr(iExpr.X), c.translateExpr(iExpr.Index), rhs)
					continue
				}
			}

			c.Print("%s %s %s;", lhs, s.Tok, rhs)
		}

	case *ast.IncDecStmt:
		c.Print("%s%s;", c.translateExpr(s.X), s.Tok)

	case nil:
		// skip

	default:
		panic(fmt.Sprintf("Unhandled statement: %T\n", s))

	}
}

func (c *Context) translateExpr(expr ast.Expr) string {
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
			panic("Unhandled BasicLit: " + value.String())
		}

		if named, isNamed := c.info.Types[expr].(*types.Named); isNamed {
			return fmt.Sprintf("(new %s(%s))", named.Obj().Name(), jsValue)
		}
		return jsValue
	}

	switch e := expr.(type) {
	case *ast.CompositeLit:
		compType := c.info.Types[e]
		if ptrType, isPointer := compType.(*types.Pointer); isPointer {
			compType = ptrType.Elem()
		}
		structType, isStruct := compType.Underlying().(*types.Struct)

		elements := make([]string, len(e.Elts))
		if isStruct {
			elements = make([]string, structType.NumFields())
			for i := range elements {
				elements[i] = zeroValue(structType.Field(i).Type())
			}
		}

		for i, element := range e.Elts {
			if kve, isKve := element.(*ast.KeyValueExpr); isKve {
				if isStruct {
					for j := range elements {
						if kve.Key.(*ast.Ident).Name == structType.Field(j).Name() {
							elements[j] = c.translateExpr(kve.Value)
							break
						}
					}
					continue
				}
				elements[i] = fmt.Sprintf("%s: %s", c.translateExpr(kve.Key), c.translateExpr(kve.Value))
				continue
			}
			elements[i] = c.translateExpr(element)
		}

		switch t := compType.(type) {
		case *types.Array:
			for i := int64(len(elements)); i < t.Len(); i++ {
				elements = append(elements, zeroValue(t.Elem()))
			}
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
		case *types.Map:
			return fmt.Sprintf("new Map({ %s })", strings.Join(elements, ", "))
		default:
			fmt.Println(e.Type, elements)
			panic(fmt.Sprintf("Unhandled CompositeLit type: %T\n", c.info.Types[e]))
		}

	case *ast.FuncLit:
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
				c.NewScope(namedResults, func() {
					c.Print("var _obj, _tuple;")
					if namedResults != nil {
						c.Print("var %s;", strings.Join(namedResults, ", "))
					}
					if c.hasDefer(e.Body.List) {
						c.Print("var _deferred = [];")
						c.Print("try {")
						c.Indent(func() {
							c.translateStmtList(e.Body.List)
						})
						c.Print("} catch(err) {")
						c.Indent(func() {
							c.Print("_error_stack.push({ frame: getStackDepth(), error: err });")
						})
						c.Print("} finally {")
						c.Indent(func() {
							c.Print("callDeferred(_deferred);")
							if namedResults != nil {
								c.translateStmt(&ast.ReturnStmt{})
							}
						})
						c.Print("}")
						return
					}
					c.translateStmtList(e.Body.List)
				})
			})
			c.Print("")
		})
		return fmt.Sprintf("(function(%s) {\n%s})", c.translateParams(e.Type), body[:len(body)-1])

	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return c.translateExpr(e.X)
		}
		return fmt.Sprintf("%s%s", e.Op.String(), c.translateExpr(e.X))

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
		return fmt.Sprintf("%s.%s", c.translateExpr(e.X), e.Sel.Name)

	case *ast.CallExpr:
		fun := c.translateExpr(e.Fun)
		args := c.translateArgs(e)
		funType := c.info.Types[e.Fun]

		if _, isSliceType := funType.(*types.Slice); isSliceType {
			return fmt.Sprintf("%s.toSlice()", args[0])
		}
		if sig, isSignature := funType.(*types.Signature); isSignature && sig.Params().Len() > 1 && len(args) == 1 {
			argRefs := make([]string, sig.Params().Len())
			for i := range argRefs {
				argRefs[i] = fmt.Sprintf("_tuple[%d]", i)
			}
			return fmt.Sprintf("(_tuple = %s, %s(%s))", args[0], fun, strings.Join(argRefs, ", "))
		}
		_, isSignature := funType.(*types.Signature)
		_, isBuiltin := funType.(*types.Builtin)
		ident, isIdent := e.Fun.(*ast.Ident)
		if !isSignature && !isBuiltin && isIdent {
			return fmt.Sprintf("cast(%s, %s)", c.translateExpr(ident), args[0])
		}
		return fmt.Sprintf("%s(%s)", fun, strings.Join(args, ", "))

	case *ast.StarExpr:
		t := c.info.Types[e]
		if _, isStruct := t.Underlying().(*types.Struct); isStruct {
			return fmt.Sprintf("(_obj = %s, %s)", c.translateExpr(e.X), cloneStruct([]string{"_obj"}, t.(*types.Named)))
		}
		return c.translateExpr(e.X)

	case *ast.TypeAssertExpr:
		check := fmt.Sprintf("_obj.constructor === %s", c.translateExpr(e.Type))
		if _, isInterface := c.info.Types[e.Type].Underlying().(*types.Interface); isInterface {
			check = fmt.Sprintf("%s(_obj.constructor)", c.translateExpr(e.Type))
		}
		if _, isTuple := c.info.Types[e].(*types.Tuple); isTuple {
			return fmt.Sprintf("(_obj = %s, %s ? [_obj, true] : [%s, false])", c.translateExpr(e.X), check, zeroValue(c.info.Types[e.Type]))
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

	case *ast.Ident:
		if e.Name == "_" {
			return ""
		}
		if tn, isTypeName := c.info.Objects[e].(*types.TypeName); isTypeName {
			switch tn.Name() {
			case "int", "int64", "float64":
				return "Number"
			case "string":
				return "String"
			}
		}
		switch o := c.info.Objects[e].(type) {
		case *types.Var:
			name, found := c.varToName[o]
			if !found {
				name = c.newVarName(e.Name)
				c.varToName[o] = name
			}
			return name
		case *types.Func:
			switch o.Name() {
			case "true", "false", "try":
				return o.Name() + "_"
			}
			return o.Name()
		}
		return e.Name

	case *This:
		return "this"

	case nil:
		return ""

	default:
		panic(fmt.Sprintf("Unhandled expression: %T\n", e))

	}
	return ""
}

func (c *Context) translateExpressionToBasic(expr ast.Expr) string {
	t := c.info.Types[expr]
	_, isNamed := t.(*types.Named)
	_, iUnderlyingBasic := t.Underlying().(*types.Basic)
	if isNamed && iUnderlyingBasic {
		return c.translateExpr(expr) + ".v"
	}
	return c.translateExpr(expr)
}

func (c *Context) translateParams(t *ast.FuncType) string {
	params := make([]string, 0)
	for _, param := range t.Params.List {
		for _, ident := range param.Names {
			params = append(params, c.translateExpr(ident))
		}
	}
	return strings.Join(params, ", ")
}

func (c *Context) translateArgs(call *ast.CallExpr) []string {
	funType := c.info.Types[call.Fun]
	args := make([]string, len(call.Args))
	for i, arg := range call.Args {
		args[i] = c.translateExpr(arg)
	}
	isVariadic, numParams, variadicType := getVariadicInfo(funType)
	if isVariadic && !call.Ellipsis.IsValid() {
		args = append(args[:numParams-1], fmt.Sprintf("new Slice(%s)", createListComposite(variadicType, args[numParams-1:])))
	}
	if call.Ellipsis.IsValid() && len(call.Args) > 0 {
		l := len(call.Args)
		if t, isBasic := c.info.Types[call.Args[l-1]].(*types.Basic); isBasic && t.Info()&types.IsString != 0 {
			args[l-1] = fmt.Sprintf("%s.toSlice()", args[l-1])
		}
	}
	return args
}

func zeroValue(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		if t.Info()&types.IsNumeric != 0 {
			return "0"
		}
		if t.Info()&types.IsString != 0 {
			return `""`
		}
	case *types.Array:
		switch elt := t.Elem().(type) {
		case *types.Basic:
			return fmt.Sprintf("newNumericArray(%d)", t.Len())
			// return fmt.Sprintf("new %s(%d)", toTypedArray(elt), t.Len())
		default:
			panic(fmt.Sprintf("Unhandled element type: %T\n", elt))
		}
	case *types.Named:
		if s, isStruct := t.Underlying().(*types.Struct); isStruct {
			zeros := make([]string, s.NumFields())
			for i := range zeros {
				zeros[i] = zeroValue(s.Field(i).Type())
			}
			return fmt.Sprintf("new %s(%s)", t.Obj().Name(), strings.Join(zeros, ", "))
		}
		return fmt.Sprintf("new %s(%s)", t.Obj().Name(), zeroValue(t.Underlying()))
	}
	return "null"
}

func cloneStruct(srcPath []string, t *types.Named) string {
	s := t.Underlying().(*types.Struct)
	fields := make([]string, s.NumFields())
	for i := range fields {
		field := s.Field(i)
		fieldPath := append(srcPath, field.Name())
		if _, isStruct := field.Type().Underlying().(*types.Struct); isStruct {
			fields[i] = cloneStruct(fieldPath, field.Type().(*types.Named))
			continue
		}
		fields[i] = strings.Join(fieldPath, ".")
	}
	return fmt.Sprintf("new %s(%s)", t.Obj().Name(), strings.Join(fields, ", "))
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
		case "print", "println":
			return true, 1, types.NewInterface(nil)
		}
	}
	return false, 0, nil
}

func hasFallthrough(caseClause *ast.CaseClause) bool {
	if len(caseClause.Body) == 0 {
		return false
	}
	b, isBranchStmt := caseClause.Body[len(caseClause.Body)-1].(*ast.BranchStmt)
	return isBranchStmt && b.Tok == token.FALLTHROUGH
}
