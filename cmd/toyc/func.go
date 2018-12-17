package main

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
	"github.com/mewspring/toy/irgen"
)

// funcGen is a generator for a given IR function.
type funcGen struct {
	// Module generator.
	gen *generator
	// LLVM IR function being generated.
	f *ir.Function
	// Current basic block being generated.
	cur *ir.BasicBlock
}

// newFuncGen returns a new generator for the given IR function.
func (gen *generator) newFuncGen(f *ir.Function) *funcGen {
	return &funcGen{
		gen: gen,
		f:   f,
	}
}

// lowerBlockStmt lowers the Go block statement to LLVM IR, emitting to f.
func (fgen *funcGen) lowerBlockStmt(old *ast.BlockStmt) {
	// TODO: handle scope?
	for _, oldStmt := range old.List {
		fgen.lowerStmt(oldStmt)
	}
}

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
			fgen.gen.c.errs = append(fgen.gen.c.errs, err)
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

// ### [ Helper functions ] ####################################################

// irParams returns the LLVM IR parameters based on the given Go field list.
func (gen *generator) irParams(old *ast.FieldList) []*ir.Param {
	if old == nil {
		return nil
	}
	var params []*ir.Param
	for _, oldParam := range old.List {
		typ, err := gen.irASTType(oldParam.Type)
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
