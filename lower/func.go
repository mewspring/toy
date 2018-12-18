package lower

import (
	"go/ast"
	gotypes "go/types"

	"github.com/llir/llvm/ir"
)

// funcGen is an LLVM IR generator for a given function.
type funcGen struct {
	// Module generator.
	gen *Generator
	// Function scope.
	scope *gotypes.Scope
	// LLVM IR function being generated.
	f *ir.Function
	// Current basic block being generated.
	cur *ir.BasicBlock
}

// newFuncGen returns a new LLVM IR function generator for the given module
// generator.
func (gen *Generator) newFuncGen() *funcGen {
	return &funcGen{
		gen: gen,
	}
}

// lowerFuncBody lowers the Go function body block statement to LLVM IR,
// emitting to f.
func (fgen *funcGen) lowerFuncBody(old *ast.BlockStmt) {
	fgen.cur = fgen.f.NewBlock("entry")
	for _, oldStmt := range old.List {
		fgen.lowerStmt(oldStmt)
	}
}
