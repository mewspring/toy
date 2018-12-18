package lower

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir"
	"github.com/mewspring/toy/irgen"
)

// lowerStmt lowers the Go statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerStmt(goStmt ast.Stmt) {
	switch goStmt := goStmt.(type) {
	case *ast.BlockStmt:
		fgen.lowerBlockStmt(goStmt)
	case *ast.IfStmt:
		fgen.lowerIfStmt(goStmt)
	case *ast.ReturnStmt:
		fgen.lowerReturnStmt(goStmt)
	default:
		panic(fmt.Errorf("support for statement %T not yet implemented", goStmt))
	}
}

// lowerBlockStmt lowers the Go block statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerBlockStmt(goBlockStmt *ast.BlockStmt) {
	// TODO: handle scope?
	for _, goStmt := range goBlockStmt.List {
		fgen.lowerStmt(goStmt)
	}
}

// lowerIfStmt lowers the Go if-statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerIfStmt(goStmt *ast.IfStmt) {
	// Initialization statement.
	if goStmt.Init != nil {
		fgen.lowerStmt(goStmt.Init)
	}
	// Condition.
	cond, err := fgen.lowerExprUse(goStmt.Cond)
	if err != nil {
		fgen.gen.eh(err)
		return
	}
	// Record condition basic block.
	//
	// We will later add a terminator to conditionally branch to either the if-
	// or the else-branch.
	condBlock := fgen.cur
	// Follow basic block, target of both if- and else-branch.
	followBlock := ir.NewBlock("")
	// True branch (if-branch).
	targetTrue := fgen.f.NewBlock("")
	fgen.cur = targetTrue
	fgen.lowerStmt(goStmt.Body)
	fgen.cur.NewBr(followBlock)
	// The follow branch is used as the false branch when no else-branch is
	// present.
	targetFalse := followBlock
	// False branch (else-branch).
	if goStmt.Else != nil {
		targetFalse = fgen.f.NewBlock("")
		fgen.cur = targetFalse
		fgen.lowerStmt(goStmt.Else)
		fgen.cur.NewBr(followBlock)
	}
	// Add terminator to condition basic block.
	condBlock.NewCondBr(cond, targetTrue, targetFalse)
	// Set follow as the current basic block used for generation.
	fgen.cur = followBlock
	// Append follow basic block to the function.
	fgen.f.Blocks = append(fgen.f.Blocks, followBlock)
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
