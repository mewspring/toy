package main

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// lowerExpr lowers the Go expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerExpr(old ast.Expr) (value.Value, error) {
	switch old := old.(type) {
	case *ast.BasicLit:
		return fgen.lowerBasicLit(old), nil
	default:
		panic(fmt.Errorf("support for expression %T not yet implemented", old))
	}
}

// lowerBasicLit lowers the Go literal of basic type to LLVM IR.
func (fgen *funcGen) lowerBasicLit(old *ast.BasicLit) value.Value {
	switch old.Kind {
	case token.INT:
		typ, err := fgen.gen.irTypeOf(old)
		if err != nil {
			panic(fmt.Errorf("unable to locate type of expresion `%v`; %v", old, err))
		}
		t, ok := typ.(*types.IntType)
		if !ok {
			panic(fmt.Errorf("support for type %T not yet implemented", old))
		}
		x, err := constant.NewIntFromString(t, old.Value)
		if err != nil {
			panic(fmt.Errorf("unable to parse integer literal %q; %v", old.Value, err))
		}
		return x
	//case token.FLOAT:
	//case token.IMAG:
	//case token.CHAR:
	//case token.STRING:
	default:
		panic(fmt.Errorf("support for literal of basic type %v not yet implemented", old.Kind))
	}
}
