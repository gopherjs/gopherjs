package analysis

// returnStmt represents a return statement that is blocking or not.
type returnStmt struct {
	analyzeStack astPath
	deferStmts   []*deferStmt
}

func newReturnStmt(stack astPath, deferStmts []*deferStmt) returnStmt {
	return returnStmt{
		analyzeStack: stack.copy(),
		deferStmts:   deferStmts,
	}
}

// IsBlocking determines if the return statement is blocking or not
// based on the defer statements that affect the return.
// The return may still be blocking if the function has labels and goto's.
func (r returnStmt) IsBlocking(info *FuncInfo) bool {
	return isAnyDeferBlocking(r.deferStmts, info.pkgInfo)
}
