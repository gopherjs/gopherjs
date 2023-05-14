package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/analysis"
	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

// functions.go contains logic responsible for translating top-level functions
// and function literals.

// newFunctionContext creates a new nested context for a function corresponding
// to the provided info.
func (fc *funcContext) nestedFunctionContext(info *analysis.FuncInfo, o *types.Func) *funcContext {
	if info == nil {
		panic(errors.New("missing *analysis.FuncInfo"))
	}
	if o == nil {
		panic(errors.New("missing *types.Func"))
	}

	c := &funcContext{
		FuncInfo:    info,
		funcObject:  o,
		pkgCtx:      fc.pkgCtx,
		genericCtx:  fc.genericCtx,
		parent:      fc,
		sigTypes:    &typesutil.Signature{Sig: o.Type().(*types.Signature)},
		allVars:     make(map[string]int, len(fc.allVars)),
		localVars:   []string{},
		flowDatas:   map[*types.Label]*flowData{nil: {}},
		caseCounter: 1,
		labelCases:  make(map[*types.Label]int),
	}
	// Register all variables from the parent context to avoid shadowing.
	for k, v := range fc.allVars {
		c.allVars[k] = v
	}

	// Synthesize an identifier by which the function may reference itself. Since
	// it appears in the stack trace, it's useful to include the receiver type in
	// it.
	funcRef := o.Name()
	if typeName := c.sigTypes.RecvTypeName(); typeName != "" {
		funcRef = typeName + midDot + funcRef
	}
	c.funcRef = c.newVariable(funcRef, varPackage)

	// If the function has type parameters, create a new generic context for it.
	if c.sigTypes.IsGeneric() {
		c.genericCtx = &genericCtx{}
	}

	return c
}

// namedFuncContext creates a new funcContext for a named Go function
// (standalone or method).
func (fc *funcContext) namedFuncContext(fun *ast.FuncDecl) *funcContext {
	o := fc.pkgCtx.Defs[fun.Name].(*types.Func)
	info := fc.pkgCtx.FuncDeclInfos[o]
	c := fc.nestedFunctionContext(info, o)

	return c
}

// literalFuncContext creates a new funcContext for a function literal. Since
// go/types doesn't generate *types.Func objects for function literals, we
// generate a synthetic one for it.
func (fc *funcContext) literalFuncContext(fun *ast.FuncLit) *funcContext {
	info := fc.pkgCtx.FuncLitInfos[fun]
	sig := fc.pkgCtx.TypeOf(fun).(*types.Signature)
	o := types.NewFunc(fun.Pos(), fc.pkgCtx.Pkg, fc.newLitFuncName(), sig)

	c := fc.nestedFunctionContext(info, o)
	return c
}

// translateTopLevelFunction translates a top-level function declaration
// (standalone function or method) into a corresponding JS function.
//
// Returns a string with JavaScript statements that define the function or
// method. For generic functions it returns a generic factory function, which
// instantiates the actual function at runtime given type parameters. For
// methods it returns declarations for both value- and pointer-receiver (if
// appropriate).
func (fc *funcContext) translateTopLevelFunction(fun *ast.FuncDecl) []byte {
	if fun.Recv == nil {
		return fc.translateStandaloneFunction(fun)
	}
	return fc.translateMethod(fun)
}

// translateStandaloneFunction translates a package-level function.
//
// It returns JS statements which define the corresponding function in a
// package context. Exported functions are also assigned to the `$pkg` object.
func (fc *funcContext) translateStandaloneFunction(fun *ast.FuncDecl) []byte {
	o := fc.pkgCtx.Defs[fun.Name].(*types.Func)
	if fun.Recv != nil {
		panic(fmt.Errorf("expected standalone function, got method: %s", o))
	}

	lvalue := fc.objectName(o)
	if fun.Body == nil {
		return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fc.unimplementedFunction(o)))
	}
	body := fc.namedFuncContext(fun).translateFunctionBody(fun.Type, nil, fun.Body)

	code := &bytes.Buffer{}
	fmt.Fprintf(code, "\t%s = %s;\n", lvalue, body)
	if fun.Name.IsExported() {
		fmt.Fprintf(code, "\t$pkg.%s = %s;\n", encodeIdent(fun.Name.Name), lvalue)
	}
	return code.Bytes()
}

// translateMethod translates a named type method.
//
// It returns one or more JS statements, which define the methods. Methods with
// non-pointer receiver are automatically defined for the pointer-receiver type.
func (fc *funcContext) translateMethod(fun *ast.FuncDecl) []byte {
	if fun.Recv == nil {
		panic(fmt.Errorf("expected a method, got %v", fun))
	}

	o := fc.pkgCtx.Defs[fun.Name].(*types.Func)
	var recv *ast.Ident
	if fun.Recv.List[0].Names != nil {
		recv = fun.Recv.List[0].Names[0]
	}
	nestedFC := fc.namedFuncContext(fun)

	// primaryFunction generates a JS function equivalent of the current Go function
	// and assigns it to the JS expression defined by lvalue.
	primaryFunction := func(lvalue string) []byte {
		if fun.Body == nil {
			return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fc.unimplementedFunction(o)))
		}

		funDef := nestedFC.translateFunctionBody(fun.Type, recv, fun.Body)
		return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, funDef))
	}

	recvType := nestedFC.sigTypes.Sig.Recv().Type()
	ptr, isPointer := recvType.(*types.Pointer)
	namedRecvType, _ := recvType.(*types.Named)
	if isPointer {
		namedRecvType = ptr.Elem().(*types.Named)
	}
	typeName := fc.objectName(namedRecvType.Obj())
	funName := fc.methodName(o)

	// Objects the method should be assigned to.
	prototypeVar := fmt.Sprintf("%s.prototype.%s", typeName, funName)
	ptrPrototypeVar := fmt.Sprintf("$ptrType(%s).prototype.%s", typeName, funName)
	isGeneric := nestedFC.sigTypes.IsGeneric()
	if isGeneric {
		// Generic method factories are assigned to the generic type factory
		// properties, to be invoked at type construction time rather than method
		// call time.
		prototypeVar = fmt.Sprintf("%s.methods.%s", typeName, funName)
		ptrPrototypeVar = fmt.Sprintf("%s.ptrMethods.%s", typeName, funName)
	}

	// Methods with pointer receivers are defined only on the pointer type.
	if isPointer {
		return primaryFunction(ptrPrototypeVar)
	}

	// Methods with non-pointer receivers must be defined both for the pointer
	// and non-pointer types. To minimize generated code size, we generate a
	// complete implementation for only one receiver (non-pointer for most types)
	// and define a proxy function on the other, which converts the receiver type
	// and forwards the call to the primary implementation.
	proxyFunction := func(lvalue, receiver string) []byte {
		params := strings.Join(nestedFC.funcParamVars(fun.Type), ", ")
		fun := fmt.Sprintf("function(%s) { return %s.%s(%s); }", params, receiver, funName, params)
		if isGeneric {
			// For a generic function, we wrap the proxy function in a trivial generic
			// factory function for consistency. It is the same for any possible type
			// arguments, so we simply ignore them.
			fun = fmt.Sprintf("function() { return %s; }", fun)
		}
		return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fun))
	}

	// Structs are a special case: because of the JS's reference semantics, the
	// actual JS objects correspond to pointer-to-struct types and struct value
	// types are emulated via cloning. Because of that, the real method
	// implementation is defined on the pointer type.
	if _, isStruct := namedRecvType.Underlying().(*types.Struct); isStruct {
		code := bytes.Buffer{}
		code.Write(primaryFunction(ptrPrototypeVar))
		code.Write(proxyFunction(prototypeVar, "this.$val"))
		return code.Bytes()
	}

	proxyRecvExpr := "this.$get()"
	if typesutil.IsGeneric(recvType) {
		proxyRecvExpr = fmt.Sprintf("%s.wrap(%s)", typeName, proxyRecvExpr)
	} else if isWrapped(recvType) {
		proxyRecvExpr = fmt.Sprintf("new %s(%s)", typeName, proxyRecvExpr)
	}
	code := bytes.Buffer{}
	code.Write(primaryFunction(prototypeVar))
	code.Write(proxyFunction(ptrPrototypeVar, proxyRecvExpr))
	return code.Bytes()
}

// unimplementedFunction returns a JS function expression for a Go function
// without a body, which would throw an exception if called.
//
// In Go such functions are either used with a //go:linkname directive or with
// assembler intrinsics, only former of which is supported by GopherJS.
func (fc *funcContext) unimplementedFunction(o *types.Func) string {
	return fmt.Sprintf("function() {\n\t\t$throwRuntimeError(\"native function not implemented: %s\");\n\t}", o.FullName())
}

func (fc *funcContext) translateFunctionBody(typ *ast.FuncType, recv *ast.Ident, body *ast.BlockStmt) string {
	prevEV := fc.pkgCtx.escapingVars

	params := fc.funcParamVars(typ)

	bodyOutput := string(fc.CatchOutput(fc.bodyIndent(), func() {
		if len(fc.Blocking) != 0 {
			fc.pkgCtx.Scopes[body] = fc.pkgCtx.Scopes[typ]
			fc.handleEscapingVars(body)
		}

		if fc.sigTypes != nil && fc.sigTypes.HasNamedResults() {
			fc.resultNames = make([]ast.Expr, fc.sigTypes.Sig.Results().Len())
			for i := 0; i < fc.sigTypes.Sig.Results().Len(); i++ {
				result := fc.sigTypes.Sig.Results().At(i)
				fc.Printf("%s = %s;", fc.objectName(result), fc.translateExpr(fc.zeroValue(result.Type())).String())
				id := ast.NewIdent("")
				fc.pkgCtx.Uses[id] = result
				fc.resultNames[i] = fc.setType(id, result.Type())
			}
		}

		if recv != nil && !isBlank(recv) {
			this := "this"
			if isWrapped(fc.pkgCtx.TypeOf(recv)) {
				this = "this.$val" // Unwrap receiver value.
			}
			fc.Printf("%s = %s;", fc.translateExpr(recv), this)
		}

		fc.translateStmtList(body.List)
		if len(fc.Flattened) != 0 && !astutil.EndsWithReturn(body.List) {
			fc.translateStmt(&ast.ReturnStmt{}, nil)
		}
	}))

	sort.Strings(fc.localVars)

	var prefix, suffix string

	if len(fc.Flattened) != 0 {
		fc.localVars = append(fc.localVars, "$s")
		prefix = prefix + " $s = $s || 0;"
	}

	if fc.HasDefer {
		fc.localVars = append(fc.localVars, "$deferred")
		suffix = " }" + suffix
		if len(fc.Blocking) != 0 {
			suffix = " }" + suffix
		}
	}

	localVarDefs := "" // Function-local var declaration at the top.

	if len(fc.Blocking) != 0 {
		localVars := append([]string{}, fc.localVars...)
		// There are several special variables involved in handling blocking functions:
		// $r is sometimes used as a temporary variable to store blocking call result.
		// $c indicates that a function is being resumed after a blocking call when set to true.
		// $f is an object used to save and restore function context for blocking calls.
		localVars = append(localVars, "$r")
		// funcRef identifies the function object itself, so it doesn't need to be saved
		// or restored.
		localVars = removeMatching(localVars, fc.funcRef)
		// If a blocking function is being resumed, initialize local variables from the saved context.
		localVarDefs = fmt.Sprintf("var {%s, $c} = $restore(this, {%s});\n", strings.Join(localVars, ", "), strings.Join(params, ", "))
		// If the function gets blocked, save local variables for future.
		saveContext := fmt.Sprintf("var $f = {$blk: %s, $c: true, %s};", fc.funcRef, strings.Join(localVars, ", "))

		suffix = " " + saveContext + "return $f;" + suffix
	} else if len(fc.localVars) > 0 {
		// Non-blocking functions simply declare local variables with no need for restore support.
		localVarDefs = fmt.Sprintf("var %s;\n", strings.Join(fc.localVars, ", "))
	}

	if fc.HasDefer {
		prefix = prefix + " var $err = null; try {"
		deferSuffix := " } catch(err) { $err = err;"
		if len(fc.Blocking) != 0 {
			deferSuffix += " $s = -1;"
		}
		if fc.resultNames == nil && fc.sigTypes.HasResults() {
			deferSuffix += fmt.Sprintf(" return%s;", fc.translateResults(nil))
		}
		deferSuffix += " } finally { $callDeferred($deferred, $err);"
		if fc.resultNames != nil {
			deferSuffix += fmt.Sprintf(" if (!$curGoroutine.asleep) { return %s; }", fc.translateResults(fc.resultNames))
		}
		if len(fc.Blocking) != 0 {
			deferSuffix += " if($curGoroutine.asleep) {"
		}
		suffix = deferSuffix + suffix
	}

	if len(fc.Flattened) != 0 {
		prefix = prefix + " s: while (true) { switch ($s) { case 0:"
		suffix = " } return; }" + suffix
	}

	if fc.HasDefer {
		prefix = prefix + " $deferred = []; $curGoroutine.deferStack.push($deferred);"
	}

	if prefix != "" {
		bodyOutput = fc.Indentation(fc.bodyIndent()) + "/* */" + prefix + "\n" + bodyOutput
	}
	if suffix != "" {
		bodyOutput = bodyOutput + fc.Indentation(fc.bodyIndent()) + "/* */" + suffix + "\n"
	}
	if localVarDefs != "" {
		bodyOutput = fc.Indentation(fc.bodyIndent()) + localVarDefs + bodyOutput
	}

	fc.pkgCtx.escapingVars = prevEV

	if !fc.sigTypes.IsGeneric() {
		return fmt.Sprintf("function %s(%s) {\n%s%s}", fc.funcRef, strings.Join(params, ", "), bodyOutput, fc.Indentation(0))
	}

	// For generic function, funcRef refers to the generic factory function,
	// allocate a separate variable for a function instance.
	instanceVar := fc.newVariable("instance", varGenericFactory)

	// Generic functions are generated as factories to allow passing type parameters
	// from the call site.
	// TODO(nevkontakte): Cache function instances for a given combination of type
	// parameters.
	typeParams := fc.typeParamVars(fc.sigTypes.Sig.TypeParams())
	typeParams = append(typeParams, fc.typeParamVars(fc.sigTypes.Sig.RecvTypeParams())...)

	// anonymous types
	typesInit := strings.Builder{}
	for _, t := range fc.genericCtx.anonTypes.Ordered() {
		fmt.Fprintf(&typesInit, "%svar %s = $%sType(%s);\n", fc.Indentation(1), t.Name(), strings.ToLower(typeKind(t.Type())[5:]), fc.initArgs(t.Type()))
	}

	code := &strings.Builder{}
	fmt.Fprintf(code, "function(%s){\n", strings.Join(typeParams, ", "))
	fmt.Fprintf(code, "%s", typesInit.String())
	fmt.Fprintf(code, "%sconst %s = function %s(%s) {\n", fc.Indentation(1), instanceVar, fc.funcRef, strings.Join(params, ", "))
	fmt.Fprintf(code, "%s", bodyOutput)
	fmt.Fprintf(code, "%s};\n", fc.Indentation(1))
	fmt.Fprintf(code, "%sreturn %s;\n", fc.Indentation(1), instanceVar)
	fmt.Fprintf(code, "%s}", fc.Indentation(0))
	return code.String()
}

// funcParamVars returns a list of JS variables corresponding to the function
// parameters in the order they are defined in the signature. Unnamed or blank
// parameters are assigned unique synthetic names.
//
// Note that JS doesn't allow blank or repeating function argument names, so
// we must assign unique names to all such blank variables.
func (fc *funcContext) funcParamVars(typ *ast.FuncType) []string {
	var params []string
	for _, param := range typ.Params.List {
		if len(param.Names) == 0 {
			params = append(params, fc.newBlankVariable(param.Pos()))
			continue
		}
		for _, ident := range param.Names {
			if isBlank(ident) {
				params = append(params, fc.newBlankVariable(ident.Pos()))
				continue
			}
			params = append(params, fc.objectName(fc.pkgCtx.Defs[ident]))
		}
	}

	return params
}
