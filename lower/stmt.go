package lower

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir/value"
	"github.com/mewspring/toy/irgen"
)

// lowerStmt lowers the Go statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerStmt(old ast.Stmt) {
	switch old := old.(type) {
	case *ast.ReturnStmt:
		fgen.lowerReturnStmt(old)
	default:
		panic(fmt.Errorf("support for statement %T not yet implemented", old))
	}
}

// lowerReturnStmt lowers the Go return statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerReturnStmt(old *ast.ReturnStmt) {
	var results []value.Value
	for _, oldResult := range old.Results {
		result, err := fgen.lowerExpr(oldResult)
		if err != nil {
			fgen.gen.eh(err)
			return
		}
		results = append(results, result)
	}
	switch len(results) {
	case 0:
		// void return.
		fgen.cur.NewRet(nil)
	case 1:
		// single return value.
		x := results[0]
		fgen.cur.NewRet(x)
	default:
		// multiple return values.
		irgen.NewAggregateRet(fgen.cur, results...)
	}
}
