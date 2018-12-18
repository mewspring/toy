package lower

import (
	gotypes "go/types"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"golang.org/x/tools/go/packages"
)

// Generator keeps track of top-level entities when translating from Go AST to
// LLVM IR representation.
type Generator struct {
	// Error handler used to report errors encountered during compilation.
	eh func(error)
	// Go package being compiled.
	pkg *packages.Package
	// Package scope.
	scope *gotypes.Scope
	// LLVM IR module being generated.
	m *ir.Module

	// Index of IR top-level entities.

	// typeDefs maps from type identifier (without '%' prefix) to type
	// definition.
	typeDefs map[string]types.Type
	// globals maps from global identifier (without '@' prefix) to global
	// declarations and defintions.
	globals map[string]*ir.Global
	// funcs maps from global identifier (without '@' prefix) to function
	// declarations and defintions.
	funcs map[string]*ir.Function
}

// NewGenerator returns a new generator for lowering the source code of the Go
// package to LLVM IR assembly. The error handler eh is invoked when an error is
// encountered during compilation.
func NewGenerator(eh func(error), pkg *packages.Package) *Generator {
	gen := &Generator{
		eh:       eh,
		pkg:      pkg,
		scope:    pkg.Types.Scope(),
		m:        ir.NewModule(),
		typeDefs: make(map[string]types.Type),
		globals:  make(map[string]*ir.Global),
		funcs:    make(map[string]*ir.Function),
	}
	return gen
}
