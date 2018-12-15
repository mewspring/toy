package main

import (
	"go/ast"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
)

// generator keeps track of top-level entities when translating from AST to IR
// representation.
type generator struct {
	// Compiler for tracking errors.
	c *compiler
	// LLVM IR module being generated.
	m *ir.Module
	// index of AST top-level entities.
	old oldIndex
	// index of IR top-level entities.
	new newIndex
}

// newGenerator returns a new generator for translating an LLVM IR module from
// AST to IR representation.
func (c *compiler) newGenerator() *generator {
	gen := &generator{
		m: &ir.Module{},
		old: oldIndex{
			typeDefs: make(map[string]ast.Expr),
			globals:  make(map[string]*ast.GenDecl),
			funcs:    make(map[string]*ast.FuncDecl),
		},
		new: newIndex{
			typeDefs: make(map[string]types.Type),
			globals:  make(map[string]*ir.Global),
			funcs:    make(map[string]*ir.Function),
		},
	}
	// Add builtin types.
	stringType := types.NewStruct(
		// data
		types.NewPointer(types.I8),
		// len
		types.I64,
	)
	stringType.SetName("string")
	gen.new.typeDefs["string"] = stringType
	return gen
}

// oldIndex is an index of AST top-level entities.
type oldIndex struct {
	// typeDefs maps from type identifier to the underlying type definition.
	typeDefs map[string]ast.Expr // type
	// globals maps from global identifier to global declarations and defintions.
	globals map[string]*ast.GenDecl
	// funcs maps from global identifier to function declarations and defintions.
	funcs map[string]*ast.FuncDecl
}

// newIndex is an index of IR top-level entities.
type newIndex struct {
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
