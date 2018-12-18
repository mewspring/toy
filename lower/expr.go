package lower

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
	"github.com/pkg/errors"
)

// --- [ Lower expression with function generator ] ----------------------------

// lowerExpr lowers the Go expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerExpr(goExpr ast.Expr) (value.Value, error) {
	switch goExpr := goExpr.(type) {
	case *ast.BasicLit:
		return fgen.gen.lowerBasicLit(goExpr), nil
	case *ast.BinaryExpr:
		return fgen.lowerBinaryExpr(goExpr)
	case *ast.CallExpr:
		return fgen.lowerCallExpr(goExpr)
	case *ast.Ident:
		return fgen.lowerIdentExpr(goExpr)
	case *ast.UnaryExpr:
		return fgen.lowerUnaryExpr(goExpr)
	default:
		panic(fmt.Errorf("support for expression %T not yet implemented", goExpr))
	}
}

// lowerBinaryExpr lowers the Go binary expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerBinaryExpr(goExpr *ast.BinaryExpr) (value.Value, error) {
	x, err := fgen.lowerExprUse(goExpr.X)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	y, err := fgen.lowerExprUse(goExpr.Y)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	t := x.Type()
	switch goExpr.Op {
	// Binary operations.
	case token.ADD: // +
		switch {
		case isIntOrIntVectorType(t):
			return fgen.cur.NewAdd(x, y), nil
		case isFloatOrFloatVectorType(t):
			return fgen.cur.NewFAdd(x, y), nil
		default:
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", goExpr.Op, t)
		}
	case token.SUB: // -
		switch {
		case isIntOrIntVectorType(t):
			return fgen.cur.NewSub(x, y), nil
		case isFloatOrFloatVectorType(t):
			return fgen.cur.NewFSub(x, y), nil
		default:
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", goExpr.Op, t)
		}
	case token.MUL: // *
		switch {
		case isIntOrIntVectorType(t):
			return fgen.cur.NewMul(x, y), nil
		case isFloatOrFloatVectorType(t):
			return fgen.cur.NewFMul(x, y), nil
		default:
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", goExpr.Op, t)
		}
	case token.QUO: // /
		switch {
		case isIntOrIntVectorType(t):
			// TODO: figure out how to distinguish signed vs. unsigned values. Use
			// SDiv for signed and UDiv for unsigned.
			return fgen.cur.NewSDiv(x, y), nil
		case isFloatOrFloatVectorType(t):
			return fgen.cur.NewFDiv(x, y), nil
		default:
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", goExpr.Op, t)
		}
	case token.REM: // %
		switch {
		case isIntOrIntVectorType(t):
			// TODO: figure out how to distinguish signed vs. unsigned values. Use
			// SRem for signed and URem for unsigned.
			return fgen.cur.NewSRem(x, y), nil
		case isFloatOrFloatVectorType(t):
			return fgen.cur.NewFRem(x, y), nil
		default:
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", goExpr.Op, t)
		}
	// Bitwise operations.
	case token.SHL: // <<
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", goExpr.Op, t)
		}
		return fgen.cur.NewShl(x, y), nil
	case token.SHR: // >>
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", goExpr.Op, t)
		}
		return fgen.cur.NewLShr(x, y), nil
	case token.AND: // &
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", goExpr.Op, t)
		}
		return fgen.cur.NewAnd(x, y), nil
	case token.OR: // |
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", goExpr.Op, t)
		}
		return fgen.cur.NewOr(x, y), nil
	case token.XOR: // ^
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", goExpr.Op, t)
		}
		return fgen.cur.NewXor(x, y), nil
	case token.AND_NOT: // &^
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", goExpr.Op, t)
		}
		// Mask.
		mask, err := allOnes(y.Type())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		tmp := fgen.cur.NewXor(y, mask)
		return fgen.cur.NewAnd(x, tmp), nil
	// Logical operations.
	case token.LAND: // &&
		switch {
		case !types.Equal(x.Type(), types.I1):
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected boolean type, got %T", goExpr.Op, x.Type())
		case !types.Equal(y.Type(), types.I1):
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected boolean type, got %T", goExpr.Op, y.Type())
		}
		return fgen.cur.NewAnd(x, y), nil
	case token.LOR: // ||
		switch {
		case !types.Equal(x.Type(), types.I1):
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected boolean type, got %T", goExpr.Op, x.Type())
		case !types.Equal(y.Type(), types.I1):
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected boolean type, got %T", goExpr.Op, y.Type())
		}
		return fgen.cur.NewOr(x, y), nil
	// Relational operations.
	case token.EQL: // ==
		return fgen.cur.NewICmp(enum.IPredEQ, x, y), nil
	case token.NEQ: // !=
		return fgen.cur.NewICmp(enum.IPredNE, x, y), nil
	case token.LSS: // <
		// TODO: figure out how to distinguish signed vs. unsigned values. Use
		// IPredSLT for signed and IPredULT for unsigned.
		return fgen.cur.NewICmp(enum.IPredSLT, x, y), nil
	case token.LEQ: // <=
		// TODO: figure out how to distinguish signed vs. unsigned values. Use
		// IPredSLE for signed and IPredULE for unsigned.
		return fgen.cur.NewICmp(enum.IPredSLE, x, y), nil
	case token.GTR: // >
		// TODO: figure out how to distinguish signed vs. unsigned values. Use
		// IPredSGT for signed and IPredUGT for unsigned.
		return fgen.cur.NewICmp(enum.IPredSGT, x, y), nil
	case token.GEQ: // >=
		// TODO: figure out how to distinguish signed vs. unsigned values. Use
		// IPredSGE for signed and IPredUGE for unsigned.
		return fgen.cur.NewICmp(enum.IPredSGE, x, y), nil
	default:
		panic(fmt.Errorf("support for '%s' binary expression not yet implemented", goExpr.Op))
	}
}

// lowerCallExpr lowers the Go call expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerCallExpr(goCallExpr *ast.CallExpr) (value.Value, error) {
	callee, err := fgen.lowerExprUse(goCallExpr.Fun)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	args, err := fgen.lowerExprs(goCallExpr.Args)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// TODO: handle goCallExpr.Ellipsis.
	return fgen.cur.NewCall(callee, args...), nil
}

// lowerIdentExpr lowers the Go identifier expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerIdentExpr(goIdent *ast.Ident) (value.Value, error) {
	name := goIdent.String()
	if f, ok := fgen.gen.funcs[name]; ok {
		return f, nil
	}
	if v, ok := fgen.gen.globals[name]; ok {
		return v, nil
	}
	return nil, errors.Errorf("unable to locate top-level definition of identifier %q", name)
}

// lowerBinaryExpr lowers the Go binary expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerUnaryExpr(goExpr *ast.UnaryExpr) (value.Value, error) {
	x, err := fgen.lowerExprUse(goExpr.X)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	t := x.Type()
	switch goExpr.Op {
	// Unary operations.
	case token.ADD: // +
		// Plus prefix is optional and has no effect.
		return x, nil
	case token.SUB: // -
		zero, err := allZeros(t)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// 0 - x
		return fgen.cur.NewSub(zero, x), nil
	case token.NOT: // !
		one := constant.True
		// x ^ 1
		return fgen.cur.NewXor(x, one), nil
	case token.XOR: // ^
		mask, err := allOnes(t)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return fgen.cur.NewXor(x, mask), nil
	//case token.MUL: // *
	//case token.AND: // &
	//case token.ARROW: // <-
	default:
		panic(fmt.Errorf("support for '%s' unary expression not yet implemented", goExpr.Op))
	}
}

// --- [ Lower expression with module generator ] ------------------------------

// lowerGlobalInitExpr lowers the given Go global definition initialization
// expression to LLVM IR, emitting to m.
func (gen *Generator) lowerGlobalInitExpr(goExpr ast.Expr) (constant.Constant, error) {
	switch goExpr := goExpr.(type) {
	// Constant.
	case *ast.BasicLit:
		return gen.lowerBasicLit(goExpr), nil
	// Non-constant.
	// TODO: generate init functions for non-constant initializers (e.g. call
	// expressions)
	default:
		panic(fmt.Errorf("support for global initialization expression %T not yet implemented", goExpr))
	}
}

// lowerBasicLit lowers the Go literal of basic type to LLVM IR.
func (gen *Generator) lowerBasicLit(goLit *ast.BasicLit) constant.Constant {
	typ, err := gen.irTypeOf(goLit)
	if err != nil {
		panic(fmt.Errorf("unable to locate type of expresion `%v`; %v", goLit, err))
	}
	switch goLit.Kind {
	case token.INT:
		t, ok := typ.(*types.IntType)
		if !ok {
			panic(fmt.Errorf("invalid type of integer literal; expected *types.IntType, got %T", t))
		}
		x, err := constant.NewIntFromString(t, goLit.Value)
		if err != nil {
			panic(fmt.Errorf("unable to parse integer literal %q; %v", goLit.Value, err))
		}
		return x
	case token.FLOAT:
		t, ok := typ.(*types.FloatType)
		if !ok {
			panic(fmt.Errorf("invalid type of integer literal; expected *types.FloatType, got %T", t))
		}
		x, err := constant.NewFloatFromString(t, goLit.Value)
		if err != nil {
			panic(fmt.Errorf("unable to parse floating-point literal %q; %v", goLit.Value, err))
		}
		return x
	//case token.IMAG:
	case token.CHAR:
		t, ok := typ.(*types.IntType)
		if !ok {
			panic(fmt.Errorf("invalid type of integer literal; expected *types.IntType, got %T", t))
		}
		s := goLit.Value
		if len(s) >= 2 && strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
			s = s[len("'") : len(s)-len("'")]
		}
		val, _, _, err := strconv.UnquoteChar(s, '\'')
		if err != nil {
			panic(fmt.Errorf("unable to parse character literal %s; %v", s, err))
		}
		return constant.NewInt(t, int64(val))
	case token.STRING:
		s, err := strconv.Unquote(goLit.Value)
		if err != nil {
			panic(fmt.Errorf("unable to parse string literal %s; %v", s, err))
		}
		return constant.NewCharArrayFromString(s)
	default:
		panic(fmt.Errorf("support for literal of basic type %v not yet implemented", goLit.Kind))
	}
}

// ### [ Helper functions ] ####################################################

// lowerExprUse lowers the Go expression to LLVM IR, emitting to f. The value
// stored at global variables is loaded to be ready for use.
func (fgen *funcGen) lowerExprUse(goExpr ast.Expr) (value.Value, error) {
	v, err := fgen.lowerExpr(goExpr)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if v, ok := v.(*ir.Global); ok {
		return fgen.cur.NewLoad(v), nil
	}
	return v, nil
}

// lowerExprs lowers the given Go expressions to LLVM IR, emitting to f.
func (fgen *funcGen) lowerExprs(goExprs []ast.Expr) ([]value.Value, error) {
	var vs []value.Value
	for _, goExpr := range goExprs {
		v, err := fgen.lowerExprUse(goExpr)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		vs = append(vs, v)
	}
	return vs, nil
}

// isIntOrIntVectorType reports whether the given type is an integer scalar or
// integer vector type.
func isIntOrIntVectorType(t types.Type) bool {
	switch t := t.(type) {
	case *types.IntType:
		return true
	case *types.VectorType:
		_, ok := t.ElemType.(*types.IntType)
		return ok
	default:
		return false
	}
}

// isFloatOrFloatVectorType reports whether the given type is a floating-point
// scalar or floating-point vector type.
func isFloatOrFloatVectorType(t types.Type) bool {
	switch t := t.(type) {
	case *types.FloatType:
		return true
	case *types.VectorType:
		_, ok := t.ElemType.(*types.FloatType)
		return ok
	default:
		return false
	}
}

// allZeros returns an integer scalar or integer vector with every bit set to 0,
// based on the bit size of the given integer scalar or integer vector type.
func allZeros(t types.Type) (constant.Constant, error) {
	size, ok := bitSize(t)
	if !ok {
		return nil, errors.Errorf("invalid operand type; expected integer scalar or integer vector type, got %T", t)
	}
	elemType := types.NewInt(size)
	zero := constant.NewInt(elemType, 0)
	if t, ok := t.(*types.VectorType); ok {
		elems := make([]constant.Constant, t.Len)
		for i := range elems {
			elems[i] = zero
		}
		return constant.NewVector(elems...), nil
	}
	return zero, nil
}

// allOnes returns an integer scalar or integer vector mask with every bit set
// to 1, based on the bit size of the given integer scalar or integer vector
// type.
func allOnes(t types.Type) (constant.Constant, error) {
	size, ok := bitSize(t)
	if !ok {
		return nil, errors.Errorf("invalid operand type; expected integer scalar or integer vector type, got %T", t)
	}
	elemType := types.NewInt(size)
	var x int64
	for i := int64(0); i < int64(size); i++ {
		if i != 0 {
			x <<= 1
		}
		x |= 1
	}
	elem := constant.NewInt(elemType, x)
	if t, ok := t.(*types.VectorType); ok {
		elems := make([]constant.Constant, t.Len)
		for i := range elems {
			elems[i] = elem
		}
		return constant.NewVector(elems...), nil
	}
	return elem, nil
}

// bitSize returns the bit size of the given integer scalar or integer vector
// type.
func bitSize(t types.Type) (uint64, bool) {
	switch t := t.(type) {
	case *types.IntType:
		return t.BitSize, true
	case *types.VectorType:
		if e, ok := t.ElemType.(*types.IntType); ok {
			return e.BitSize, true
		}
	}
	return 0, false
}
