package lower

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
	"github.com/pkg/errors"
)

// lowerBasicLit lowers the Go literal of basic type to LLVM IR.
func (gen *Generator) lowerBasicLit(old *ast.BasicLit) constant.Constant {
	typ, err := gen.irTypeOf(old)
	if err != nil {
		panic(fmt.Errorf("unable to locate type of expresion `%v`; %v", old, err))
	}
	switch old.Kind {
	case token.INT:
		t, ok := typ.(*types.IntType)
		if !ok {
			panic(fmt.Errorf("invalid type of integer literal; expected *types.IntType, got %T", t))
		}
		x, err := constant.NewIntFromString(t, old.Value)
		if err != nil {
			panic(fmt.Errorf("unable to parse integer literal %q; %v", old.Value, err))
		}
		return x
	case token.FLOAT:
		t, ok := typ.(*types.FloatType)
		if !ok {
			panic(fmt.Errorf("invalid type of integer literal; expected *types.FloatType, got %T", t))
		}
		x, err := constant.NewFloatFromString(t, old.Value)
		if err != nil {
			panic(fmt.Errorf("unable to parse floating-point literal %q; %v", old.Value, err))
		}
		return x
	//case token.IMAG:
	case token.CHAR:
		t, ok := typ.(*types.IntType)
		if !ok {
			panic(fmt.Errorf("invalid type of integer literal; expected *types.IntType, got %T", t))
		}
		s := old.Value
		if len(s) >= 2 && strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
			s = s[len("'") : len(s)-len("'")]
		}
		val, _, _, err := strconv.UnquoteChar(s, '\'')
		if err != nil {
			panic(fmt.Errorf("unable to parse character literal %s; %v", s, err))
		}
		return constant.NewInt(t, int64(val))
	case token.STRING:
		s, err := strconv.Unquote(old.Value)
		if err != nil {
			panic(fmt.Errorf("unable to parse string literal %s; %v", s, err))
		}
		return constant.NewCharArrayFromString(s)
	default:
		panic(fmt.Errorf("support for literal of basic type %v not yet implemented", old.Kind))
	}
}

// lowerExpr lowers the Go expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerExpr(old ast.Expr) (value.Value, error) {
	switch old := old.(type) {
	case *ast.BasicLit:
		return fgen.gen.lowerBasicLit(old), nil
	case *ast.BinaryExpr:
		return fgen.lowerBinaryExpr(old)
	case *ast.CallExpr:
		return fgen.lowerCallExpr(old)
	case *ast.Ident:
		return fgen.lowerIdentExpr(old)
	case *ast.UnaryExpr:
		return fgen.lowerUnaryExpr(old)
	default:
		panic(fmt.Errorf("support for expression %T not yet implemented", old))
	}
}

// lowerBinaryExpr lowers the Go binary expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerBinaryExpr(old *ast.BinaryExpr) (value.Value, error) {
	x, err := fgen.lowerExpr(old.X)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	y, err := fgen.lowerExpr(old.Y)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	t := x.Type()
	switch old.Op {
	// Binary operations.
	case token.ADD: // +
		switch {
		case isIntOrIntVectorType(t):
			return fgen.cur.NewAdd(x, y), nil
		case isFloatOrFloatVectorType(t):
			return fgen.cur.NewFAdd(x, y), nil
		default:
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", old.Op, t)
		}
	case token.SUB: // -
		switch {
		case isIntOrIntVectorType(t):
			return fgen.cur.NewSub(x, y), nil
		case isFloatOrFloatVectorType(t):
			return fgen.cur.NewFSub(x, y), nil
		default:
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", old.Op, t)
		}
	case token.MUL: // *
		switch {
		case isIntOrIntVectorType(t):
			return fgen.cur.NewMul(x, y), nil
		case isFloatOrFloatVectorType(t):
			return fgen.cur.NewFMul(x, y), nil
		default:
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", old.Op, t)
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
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", old.Op, t)
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
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar, integer vector, floating-point scalar or floating-point vector type, got %T", old.Op, t)
		}
	// Bitwise operations.
	case token.SHL: // <<
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", old.Op, t)
		}
		return fgen.cur.NewShl(x, y), nil
	case token.SHR: // >>
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", old.Op, t)
		}
		return fgen.cur.NewLShr(x, y), nil
	case token.AND: // &
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", old.Op, t)
		}
		return fgen.cur.NewAnd(x, y), nil
	case token.OR: // |
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", old.Op, t)
		}
		return fgen.cur.NewOr(x, y), nil
	case token.XOR: // ^
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", old.Op, t)
		}
		return fgen.cur.NewXor(x, y), nil
	case token.AND_NOT: // &^
		if !isIntOrIntVectorType(t) {
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected integer scalar or integer vector type, got %T", old.Op, t)
		}
		// Mask.
		mask, err := allOnesMask(y.Type())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		tmp := fgen.cur.NewXor(y, mask)
		return fgen.cur.NewAnd(x, tmp), nil
	// Logical operations.
	case token.LAND: // &&
		switch {
		case !types.Equal(x.Type(), types.I1):
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected boolean type, got %T", old.Op, x.Type())
		case !types.Equal(y.Type(), types.I1):
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected boolean type, got %T", old.Op, y.Type())
		}
		return fgen.cur.NewAnd(x, y), nil
	case token.LOR: // ||
		switch {
		case !types.Equal(x.Type(), types.I1):
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected boolean type, got %T", old.Op, x.Type())
		case !types.Equal(y.Type(), types.I1):
			return nil, errors.Errorf("invalid operand type to '%s' binary expression; expected boolean type, got %T", old.Op, y.Type())
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
		panic(fmt.Errorf("support for '%s' binary expression not yet implemented", old.Op))
	}
}

// lowerCallExpr lowers the Go call expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerCallExpr(old *ast.CallExpr) (value.Value, error) {
	args, err := fgen.lowerExprs(old.Args)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// TODO: handle old.Ellipsis.
	callee, err := fgen.lowerExpr(old.Fun)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return fgen.cur.NewCall(callee, args...), nil
}

// lowerIdentExpr lowers the Go identifier expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerIdentExpr(old *ast.Ident) (value.Value, error) {
	name := old.String()
	if f, ok := fgen.gen.new.funcs[name]; ok {
		return f, nil
	}
	if mem, ok := fgen.gen.new.globals[name]; ok {
		v := fgen.cur.NewLoad(mem)
		return v, nil
	}
	return nil, errors.Errorf("unable to locate top-level definition of identifier %q", name)
}

// lowerBinaryExpr lowers the Go binary expression to LLVM IR, emitting to f.
func (fgen *funcGen) lowerUnaryExpr(old *ast.UnaryExpr) (value.Value, error) {
	x, err := fgen.lowerExpr(old.X)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	t := x.Type()
	switch old.Op {
	// Unary operations.
	case token.ADD: // +
		// Plus prefix is optional and has no effect.
		return x, nil
	case token.SUB: // -
		zero, err := allZero(t)
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
		oneMask, err := allOnesMask(t)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return fgen.cur.NewXor(x, oneMask), nil
	//case token.MUL: // *
	//case token.AND: // &
	//case token.ARROW: // <-
	default:
		panic(fmt.Errorf("support for '%s' unary expression not yet implemented", old.Op))
	}
}

// ### [ Helper functions ] ####################################################

// lowerExprs lowers the given Go expressions to LLVM IR, emitting to f.
func (fgen *funcGen) lowerExprs(oldExprs []ast.Expr) ([]value.Value, error) {
	var vs []value.Value
	for _, oldExpr := range oldExprs {
		v, err := fgen.lowerExpr(oldExpr)
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

// allZero returns an integer scalar or integer vector with every bit set to 0,
// based on the bit size of the given integer scalar or integer vector type.
func allZero(t types.Type) (constant.Constant, error) {
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

// allOnesMask returns an integer scalar or integer vector mask with every bit
// set to 1, based on the bit size of the given integer scalar or integer vector
// type.
func allOnesMask(t types.Type) (constant.Constant, error) {
	size, ok := bitSize(t)
	if !ok {
		return nil, errors.Errorf("invalid shift operand type; expected integer scalar or integer vector type, got %T", t)
	}
	maskType := types.NewInt(size)
	var maskValue int64
	for i := int64(0); i < int64(size); i++ {
		if i != 0 {
			maskValue <<= 1
		}
		maskValue |= 1
	}
	mask := constant.NewInt(maskType, maskValue)
	if t, ok := t.(*types.VectorType); ok {
		elems := make([]constant.Constant, t.Len)
		for i := range elems {
			elems[i] = mask
		}
		return constant.NewVector(elems...), nil
	}
	return mask, nil
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
