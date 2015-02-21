package filter

import "go/ast"

func ElseIf(stmt ast.Stmt) ast.Stmt {
	if ifStmt, ok := stmt.(*ast.IfStmt); ok {
		if elseStmt, ok := ifStmt.Else.(*ast.IfStmt); ok {
			return &ast.IfStmt{
				If:   ifStmt.If,
				Init: ifStmt.Init,
				Cond: ifStmt.Cond,
				Body: ifStmt.Body,
				Else: &ast.BlockStmt{
					List: []ast.Stmt{elseStmt},
				},
			}
		}
	}
	return stmt
}
