package lower

import (
	"fmt"
	"go/ast"

	"github.com/mewspring/toy/irgen"
)

// lowerStmt lowers the Go statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerStmt(goStmt ast.Stmt) {
	switch goStmt := goStmt.(type) {
	case *ast.ReturnStmt:
		fgen.lowerReturnStmt(goStmt)
	default:
		panic(fmt.Errorf("support for statement %T not yet implemented", goStmt))
	}
}

// lowerReturnStmt lowers the Go return statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerReturnStmt(goStmt *ast.ReturnStmt) {
	results, err := fgen.lowerExprs(goStmt.Results)
	if err != nil {
		fgen.gen.eh(err)
		return
	}
	switch len(results) {
	case 0:
		// void return.
		fgen.cur.NewRet(nil)
	case 1:
		// single return value.
		fgen.cur.NewRet(results[0])
	default:
		// multiple return values.
		irgen.NewAggregateRet(fgen.cur, results...)
	}
}
