package lower

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
	"github.com/mewspring/toy/irgen"
	"github.com/pkg/errors"
)

// lowerStmt lowers the Go statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerStmt(goStmt ast.Stmt) {
	switch goStmt := goStmt.(type) {
	case *ast.BlockStmt:
		fgen.lowerBlockStmt(goStmt)
	case *ast.ExprStmt:
		fgen.lowerExprStmt(goStmt)
	case *ast.IfStmt:
		fgen.lowerIfStmt(goStmt)
	case *ast.ReturnStmt:
		fgen.lowerReturnStmt(goStmt)
	case *ast.SwitchStmt:
		fgen.lowerSwitchStmt(goStmt)
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

// lowerExprStmt lowers the Go expression statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerExprStmt(goExprStmt *ast.ExprStmt) {
	if _, err := fgen.lowerExpr(goExprStmt.X); err != nil {
		fgen.gen.eh(err)
		return
	}
}

// lowerIfStmt lowers the Go if-statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerIfStmt(goIfStmt *ast.IfStmt) {
	// Initialization statement.
	if goIfStmt.Init != nil {
		fgen.lowerStmt(goIfStmt.Init)
	}
	// Condition.
	cond, err := fgen.lowerExprUse(goIfStmt.Cond)
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
	fgen.lowerStmt(goIfStmt.Body)
	if fgen.cur.Term == nil {
		fgen.cur.NewBr(followBlock)
	}
	// The follow branch is used as the false branch when no else-branch is
	// present.
	targetFalse := followBlock
	// False branch (else-branch).
	if goIfStmt.Else != nil {
		targetFalse = fgen.f.NewBlock("")
		fgen.cur = targetFalse
		fgen.lowerStmt(goIfStmt.Else)
		if fgen.cur.Term == nil {
			fgen.cur.NewBr(followBlock)
		}
	}
	// Add terminator to condition basic block.
	condBlock.NewCondBr(cond, targetTrue, targetFalse)
	// Set follow as the current basic block used for generation.
	fgen.cur = followBlock
	// Append follow basic block to the function.
	fgen.f.Blocks = append(fgen.f.Blocks, followBlock)
}

// lowerReturnStmt lowers the Go return statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerReturnStmt(goRetStmt *ast.ReturnStmt) {
	results, err := fgen.lowerExprs(goRetStmt.Results)
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

// lowerSwitchStmt lowers the Go switch-statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerSwitchStmt(goSwitchStmt *ast.SwitchStmt) {
	// Initialization statement.
	if goSwitchStmt.Init != nil {
		fgen.lowerStmt(goSwitchStmt.Init)
	}
	var goCases []*ast.CaseClause
	for _, goStmt := range goSwitchStmt.Body.List {
		goCase, ok := goStmt.(*ast.CaseClause)
		if !ok {
			panic(fmt.Errorf("invalid case clause type; expected *ast.CaseClause, got %T", goStmt))
		}
		goCases = append(goCases, goCase)
	}
	// Tag.
	var tag value.Value
	if goSwitchStmt.Tag != nil {
		var err error
		tag, err = fgen.lowerExprUse(goSwitchStmt.Tag)
		if err != nil {
			fgen.gen.eh(err)
			return
		}
	}
	var caseBlocks []*ir.BasicBlock
	nextBlock := ir.NewBlock("")
	//followBlock := ir.NewBlock("follow")
	followBlock := ir.NewBlock("")
	for _, goCase := range goCases {
		if goCase.List != nil {
			// case branches.
			//caseBlock := ir.NewBlock(fmt.Sprintf("case_%d", i))
			caseBlock := ir.NewBlock("")
			caseBlocks = append(caseBlocks, caseBlock)
			if tag != nil {
				// Tag.
				for _, goExpr := range goCase.List {
					x, err := fgen.lowerExprUse(goExpr)
					if err != nil {
						fgen.gen.eh(err)
						continue
					}
					cond, err := fgen.lowerEqual(tag, x)
					if err != nil {
						fgen.gen.eh(err)
						continue
					}
					fgen.cur.NewCondBr(cond, caseBlock, nextBlock)
					fgen.cur = nextBlock
					fgen.f.Blocks = append(fgen.f.Blocks, nextBlock)
					nextBlock = ir.NewBlock("")
				}
			} else {
				// No tag.
				var cond value.Value
				for _, goExpr := range goCase.List {
					x, err := fgen.lowerExprUse(goExpr)
					if err != nil {
						fgen.gen.eh(err)
						continue
					}
					if cond != nil {
						cond = fgen.cur.NewOr(cond, x)
					} else {
						cond = x
					}
				}
				fgen.cur.NewCondBr(cond, caseBlock, nextBlock)
				fgen.cur = nextBlock
				fgen.f.Blocks = append(fgen.f.Blocks, nextBlock)
				nextBlock = ir.NewBlock("")
			}
		} else {
			// default branch.
			//caseBlock := ir.NewBlock("default")
			caseBlock := ir.NewBlock("")
			caseBlocks = append(caseBlocks, caseBlock)
			fgen.cur.NewBr(caseBlock)
		}
	}
	// Case bodies.
	for i, goCase := range goCases {
		caseBlock := caseBlocks[i]
		fgen.cur = caseBlock
		for _, goStmt := range goCase.Body {
			fgen.lowerStmt(goStmt)
		}
		if fgen.cur.Term == nil {
			fgen.cur.NewBr(followBlock)
		}
		fgen.f.Blocks = append(fgen.f.Blocks, caseBlock)
	}
	// Follow basic block.
	fgen.cur = followBlock
	fgen.f.Blocks = append(fgen.f.Blocks, followBlock)
}

// ### [ Helper functions ] ####################################################

// lowerEqual lowers a Go equality comparison between a and b to LLVM IR,
// emitting to f.
func (fgen *funcGen) lowerEqual(a, b value.Value) (value.Value, error) {
	if !types.Equal(a.Type(), b.Type()) {
		return nil, errors.Errorf("type mismatch between `%s` and `%s`", a.Type(), a.Type())
	}
	t := a.Type()
	switch {
	case types.IsInt(t):
		return fgen.cur.NewICmp(enum.IPredEQ, a, b), nil
	case types.IsFloat(t):
		// TODO: figure out when to use enum.FPredUEQ.
		return fgen.cur.NewFCmp(enum.FPredOEQ, a, b), nil
	default:
		panic(fmt.Errorf("support for equality comparison of type %v not yet implemented", t))
	}
}
