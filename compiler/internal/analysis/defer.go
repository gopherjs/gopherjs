package analysis

import (
	"go/ast"
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

// deferStmt represents a defer statement that is blocking or not.
//
// A blocking defer statement will cause a return statement to be blocking
// since the defer is called and potentially blocked while leaving the method.
// We try to determine which defers affect which returns so that we only
// mark returns as blocking if they are affected by a blocking defer.
// In general we know that a defer will affect all returns that have been
// declared after the defer statement.
//
// Since analysis doesn't create [CFG] basic blocks for full control
// flow analysis we can't easily determine several cases:
//
//   - Terminating if-statements(i.e. does the body of the if-statement always
//     return from the method) are difficult to determine. Any defer that is
//     added whilst inside a terminating if-statement body can only affect the
//     returns inside that if-statement body.
//     Otherwise, the defer may affect returns after the if-statement block has
//     rejoined the flow that it branched from. Since terminating if-statements
//     are difficult to determine without [CFG] blocks, we treat all
//     if-statements as if they are not terminating.
//     That means there may be some false positives, since returns declared
//     after a terminating branch will be marked as affected by a defer
//     declared in that branch, when in reality they are not.
//
//   - Same as above but for else blocks, switch cases, and any branching.
//
//   - Loops (i.e. for-statements and for-range-statements) can cause return
//     statements declared earlier in the loop to be affected by defers
//     declared after it in the loop. We can't determine which branches in a
//     loop may return to the start of the loop so we assume anywhere inside
//     of a loop can return to the start of the loop.
//     To handle this, all defers defined anywhere within a loop are assumed
//     to affect any return also defined in that loop.
//     We only need to track the top-level loop since nested loops will be
//     superseded by the top-level loop.
//
//   - Labels and goto's are similar to loops in [CFG] blocks but without those
//     blocks it's harder to determine which defers will affect which returns.
//     To be safe, for any function with any blocking defers, returns, and
//     goto's, all the returns are defaulted to blocking.
//
// [CFG]: https://en.wikipedia.org/wiki/Control-flow_graph
type deferStmt struct {
	obj      types.Object
	lit      *ast.FuncLit
	typeArgs typesutil.TypeList
}

// newBlockingDefer creates a new defer statement that is blocking.
//
// If the defer is calling a js.Object method then the defer is non-blocking.
// If the defers calling an interface method or function pointer in a var
// then the defer is blocking.
func newBlockingDefer() *deferStmt {
	return &deferStmt{}
}

// newInstDefer creates a new defer statement for an instances of a method.
// The instance is used to look up the blocking information later.
func newInstDefer(inst typeparams.Instance) *deferStmt {
	return &deferStmt{obj: inst.Object, typeArgs: inst.TArgs}
}

// newLitDefer creates a new defer statement for a function literal.
// The literal is used to look up the blocking information later.
func newLitDefer(lit *ast.FuncLit, typeArgs typesutil.TypeList) *deferStmt {
	return &deferStmt{lit: lit, typeArgs: typeArgs}
}

// IsBlocking determines if the defer statement is blocking or not.
func (d *deferStmt) IsBlocking(info *Info) bool {
	// If the instance or the literal is set then we can look up the blocking,
	// otherwise assume blocking because otherwise the defer wouldn't
	// have been recorded.
	if d.obj != nil {
		return info.IsBlocking(typeparams.Instance{Object: d.obj, TArgs: d.typeArgs})
	}
	if d.lit != nil {
		return info.FuncLitInfo(d.lit, d.typeArgs).IsBlocking()
	}
	return true
}

func isAnyDeferBlocking(deferStmts []*deferStmt, info *Info) bool {
	for _, def := range deferStmts {
		if def.IsBlocking(info) {
			return true
		}
	}
	return false
}
