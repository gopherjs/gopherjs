package compiler

// functions.go contains logic responsible for translating top-level functions
// and function literals.

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/internal/analysis"
	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

// nestedFunctionContext creates a new nested context for a function corresponding
// to the provided info and instance.
func (fc *funcContext) nestedFunctionContext(info *analysis.FuncInfo, inst typeparams.Instance) *funcContext {
	if info == nil {
		panic(errors.New("missing *analysis.FuncInfo"))
	}
	if inst.Object == nil {
		panic(errors.New("missing inst.Object"))
	}
	o := inst.Object.(*types.Func)
	sig := o.Type().(*types.Signature)

	c := &funcContext{
		FuncInfo:     info,
		instance:     inst,
		pkgCtx:       fc.pkgCtx,
		parent:       fc,
		allVars:      make(map[string]int, len(fc.allVars)),
		localVars:    []string{},
		flowDatas:    map[*types.Label]*flowData{nil: {}},
		caseCounter:  1,
		labelCases:   make(map[*types.Label]int),
		typeResolver: fc.typeResolver,
		objectNames:  map[types.Object]string{},
		sig:          &typesutil.Signature{Sig: sig},
	}
	for k, v := range fc.allVars {
		c.allVars[k] = v
	}

	if sig.TypeParams().Len() > 0 {
		c.typeResolver = typeparams.NewResolver(c.pkgCtx.typesCtx, typeparams.ToSlice(sig.TypeParams()), inst.TArgs)
	} else if sig.RecvTypeParams().Len() > 0 {
		c.typeResolver = typeparams.NewResolver(c.pkgCtx.typesCtx, typeparams.ToSlice(sig.RecvTypeParams()), inst.TArgs)
	}
	if c.objectNames == nil {
		c.objectNames = map[types.Object]string{}
	}

	// Synthesize an identifier by which the function may reference itself. Since
	// it appears in the stack trace, it's useful to include the receiver type in
	// it.
	funcRef := o.Name()
	if recvType := typesutil.RecvType(sig); recvType != nil {
		funcRef = recvType.Obj().Name() + midDot + funcRef
	}
	c.funcRef = c.newVariable(funcRef, true /*pkgLevel*/)

	return c
}

// namedFuncContext creates a new funcContext for a named Go function
// (standalone or method).
func (fc *funcContext) namedFuncContext(inst typeparams.Instance) *funcContext {
	info := fc.pkgCtx.FuncInfo(inst)
	c := fc.nestedFunctionContext(info, inst)

	return c
}

// literalFuncContext creates a new funcContext for a function literal. Since
// go/types doesn't generate *types.Func objects for function literals, we
// generate a synthetic one for it.
func (fc *funcContext) literalFuncContext(fun *ast.FuncLit) *funcContext {
	info := fc.pkgCtx.FuncLitInfo(fun)
	sig := fc.pkgCtx.TypeOf(fun).(*types.Signature)
	o := types.NewFunc(fun.Pos(), fc.pkgCtx.Pkg, fc.newLitFuncName(), sig)
	inst := typeparams.Instance{Object: o}

	c := fc.nestedFunctionContext(info, inst)
	return c
}

// translateTopLevelFunction translates a top-level function declaration
// (standalone function or method) into a corresponding JS function. Must be
// called on the function context created for the function corresponding instance.
//
// Returns a string with JavaScript statements that define the function or
// method. For methods it returns declarations for both value- and
// pointer-receiver (if appropriate).
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
	o := fc.instance.Object.(*types.Func)

	if fun.Recv != nil {
		panic(fmt.Errorf("expected standalone function, got method: %s", o))
	}

	lvalue := fc.instName(fc.instance)

	if fun.Body == nil {
		return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fc.unimplementedFunction(o)))
	}

	body := fc.translateFunctionBody(fun.Type, nil, fun.Body)
	code := bytes.NewBuffer(nil)
	fmt.Fprintf(code, "\t%s = %s;\n", lvalue, body)
	if fun.Name.IsExported() {
		fmt.Fprintf(code, "\t$pkg.%s = %s;\n", encodeIdent(fun.Name.Name), lvalue)
	}
	return code.Bytes()
}

// translateMethod translates a named type method.
//
// It returns one or more JS statements which define the method. Methods with
// non-pointer receiver are automatically defined for the pointer-receiver type.
func (fc *funcContext) translateMethod(fun *ast.FuncDecl) []byte {
	o := fc.instance.Object.(*types.Func)
	funName := fc.methodName(o)

	// primaryFunction generates a JS function equivalent of the current Go function
	// and assigns it to the JS expression defined by lvalue.
	primaryFunction := func(lvalue string) []byte {
		if fun.Body == nil {
			return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fc.unimplementedFunction(o)))
		}

		var recv *ast.Ident
		if fun.Recv != nil && fun.Recv.List[0].Names != nil {
			recv = fun.Recv.List[0].Names[0]
		}
		fun := fc.translateFunctionBody(fun.Type, recv, fun.Body)
		return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fun))
	}

	recvInst := fc.instance.Recv()
	recvInstName := fc.instName(recvInst)
	recvType := recvInst.Object.Type().(*types.Named)

	// Objects the method should be assigned to for the plain and pointer type
	// of the receiver.
	prototypeVar := fmt.Sprintf("%s.prototype.%s", recvInstName, funName)
	ptrPrototypeVar := fmt.Sprintf("$ptrType(%s).prototype.%s", recvInstName, funName)

	// Methods with pointer-receiver are only attached to the pointer-receiver type.
	if _, isPointer := fc.sig.Sig.Recv().Type().(*types.Pointer); isPointer {
		return primaryFunction(ptrPrototypeVar)
	}

	// Methods with non-pointer receivers must be defined both for the pointer
	// and non-pointer types. To minimize generated code size, we generate a
	// complete implementation for only one receiver (non-pointer for most types)
	// and define a proxy function on the other, which converts the receiver type
	// and forwards the call to the primary implementation.
	proxyFunction := func(lvalue, receiver string) []byte {
		fun := fmt.Sprintf("function(...$args) { return %s.%s(...$args); }", receiver, funName)
		return []byte(fmt.Sprintf("\t%s = %s;\n", lvalue, fun))
	}

	// Structs are a special case: they are represented by JS objects and their
	// methods are the underlying object's methods. Due to reference semantics of
	// the JS variables, the actual backing object is considered to represent the
	// pointer-to-struct type, and methods are attacher to it first and foremost.
	if _, isStruct := recvType.Underlying().(*types.Struct); isStruct {
		code := bytes.Buffer{}
		code.Write(primaryFunction(ptrPrototypeVar))
		code.Write(proxyFunction(prototypeVar, "this.$val"))
		return code.Bytes()
	}

	// Methods defined for non-pointer receiver are attached to both pointer- and
	// non-pointer-receiver types.
	proxyRecvExpr := "this.$get()"
	if isWrapped(recvType) {
		proxyRecvExpr = fmt.Sprintf("new %s(%s)", recvInstName, proxyRecvExpr)
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

// translateFunctionBody translates body of a top-level or literal function.
//
// It returns a JS function expression that represents the given Go function.
// Function receiver must have been created with nestedFunctionContext() to have
// required metadata set up.
func (fc *funcContext) translateFunctionBody(typ *ast.FuncType, recv *ast.Ident, body *ast.BlockStmt) string {
	prevEV := fc.pkgCtx.escapingVars

	// Generate a list of function argument variables. Since Go allows nameless
	// arguments, we have to generate synthetic names for their JS counterparts.
	var args []string
	for _, param := range typ.Params.List {
		if len(param.Names) == 0 {
			args = append(args, fc.newLocalVariable("param"))
			continue
		}
		for _, ident := range param.Names {
			if isBlank(ident) {
				args = append(args, fc.newLocalVariable("param"))
				continue
			}
			args = append(args, fc.objectName(fc.pkgCtx.Defs[ident]))
		}
	}

	bodyOutput := string(fc.CatchOutput(1, func() {
		if fc.IsBlocking() {
			fc.pkgCtx.Scopes[body] = fc.pkgCtx.Scopes[typ]
			fc.handleEscapingVars(body)
		}

		if fc.sig != nil && fc.sig.HasNamedResults() {
			fc.resultNames = make([]ast.Expr, fc.sig.Sig.Results().Len())
			for i := 0; i < fc.sig.Sig.Results().Len(); i++ {
				result := fc.sig.Sig.Results().At(i)
				typ := fc.typeResolver.Substitute(result.Type())
				fc.Printf("%s = %s;", fc.objectName(result), fc.translateExpr(fc.zeroValue(typ)).String())
				id := ast.NewIdent("")
				fc.pkgCtx.Uses[id] = result
				fc.resultNames[i] = fc.setType(id, typ)
			}
		}

		if recv != nil && !isBlank(recv) {
			this := "this"
			if isWrapped(fc.typeOf(recv)) {
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
		// $s contains an index of the switch case a blocking function reached
		// before getting blocked. When execution resumes, it will allow to continue
		// from where we left off.
		fc.localVars = append(fc.localVars, "$s")
		prefix = prefix + " $s = $s || 0;"
	}

	if fc.HasDefer {
		fc.localVars = append(fc.localVars, "$deferred")
		suffix = " }" + suffix
		if fc.IsBlocking() {
			suffix = " }" + suffix
		}
	}

	localVarDefs := "" // Function-local var declaration at the top.

	if fc.IsBlocking() {
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
		localVarDefs = fmt.Sprintf("var {%s, $c} = $restore(this, {%s});\n", strings.Join(localVars, ", "), strings.Join(args, ", "))
		// If the function gets blocked, save local variables for future.
		saveContext := fmt.Sprintf("var $f = {$blk: "+fc.funcRef+", $c: true, $r, %s};", strings.Join(fc.localVars, ", "))

		suffix = " " + saveContext + "return $f;" + suffix
	} else if len(fc.localVars) > 0 {
		// Non-blocking functions simply declare local variables with no need for restore support.
		localVarDefs = fmt.Sprintf("var %s;\n", strings.Join(fc.localVars, ", "))
	}

	if fc.HasDefer {
		prefix = prefix + " var $err = null; try {"
		deferSuffix := " } catch(err) { $err = err;"
		if fc.IsBlocking() {
			deferSuffix += " $s = -1;"
		}
		if fc.resultNames == nil && fc.sig.HasResults() {
			deferSuffix += fmt.Sprintf(" return%s;", fc.translateResults(nil))
		}
		deferSuffix += " } finally { $callDeferred($deferred, $err);"
		if fc.resultNames != nil {
			deferSuffix += fmt.Sprintf(" if (!$curGoroutine.asleep) { return %s; }", fc.translateResults(fc.resultNames))
		}
		if fc.IsBlocking() {
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
		bodyOutput = fc.Indentation(1) + "/* */" + prefix + "\n" + bodyOutput
	}
	if suffix != "" {
		bodyOutput = bodyOutput + fc.Indentation(1) + "/* */" + suffix + "\n"
	}
	if localVarDefs != "" {
		bodyOutput = fc.Indentation(1) + localVarDefs + bodyOutput
	}

	fc.pkgCtx.escapingVars = prevEV

	return fmt.Sprintf("function %s(%s) {\n%s%s}", fc.funcRef, strings.Join(args, ", "), bodyOutput, fc.Indentation(0))
}
