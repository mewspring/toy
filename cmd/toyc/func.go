package main

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir"
)

// funcGen is a generator for a given IR function.
type funcGen struct {
	// Module generator.
	gen *generator
	// LLVM IR function being generated.
	f *ir.Function
}

// newFuncGen returns a new generator for the given IR function.
func (gen *generator) newFuncGen(f *ir.Function) *funcGen {
	return &funcGen{
		gen: gen,
		f:   f,
	}
}

// compileBlockStmt compiles the Go block statement to LLVM IR, emitting to f.
func (fgen *funcGen) compileBlockStmt(old *ast.BlockStmt) {
	// TODO: handle scope?
	for _, oldStmt := range old.List {
		fgen.compileStmt(oldStmt)
	}
}

// compileStmt compiles the Go statement to LLVM IR, emitting to f.
func (fgen *funcGen) compileStmt(old ast.Stmt) {
	switch old := old.(type) {
	default:
		panic(fmt.Errorf("support for statement %T not yet implemented", old))
	}
}

// ### [ Helper functions ] ####################################################

// irParams returns the LLVM IR parameters based on the given Go field list.
func (gen *generator) irParams(old *ast.FieldList) []*ir.Param {
	if old == nil {
		return nil
	}
	var params []*ir.Param
	for _, oldParam := range old.List {
		typ, err := gen.irType(oldParam.Type)
		if err != nil {
			gen.c.errs = append(gen.c.errs, err)
			break
		}
		for _, name := range oldParam.Names {
			param := ir.NewParam(name.String(), typ)
			params = append(params, param)
		}
	}
	return params
}
