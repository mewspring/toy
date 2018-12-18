package lower

import (
	"fmt"
	"go/ast"
	gotypes "go/types"

	"github.com/llir/llvm/ir/types"
)

// irTypeOf returns the LLVM IR type of the given Go expression. It is valid to
// pass a Go type expression (e.g. ast.FuncDecl.Type).
func (gen *Generator) irTypeOf(goExpr ast.Expr) (types.Type, error) {
	goType := gen.pkg.TypesInfo.TypeOf(goExpr)
	return gen.irType(goType)
}

// irType returns the LLVM IR type corresponding to the given Go type.
func (gen *Generator) irType(goType gotypes.Type) (types.Type, error) {
	switch goType := goType.(type) {
	case *gotypes.Basic:
		return gen.irBasicType(goType), nil
	default:
		panic(fmt.Errorf("support for Go type %T not yet implemented", goType))
	}
}

// CPU word size in number of bits.
const cpuWordSize = 64

// irBasicType returns the LLVM IR type corresponding to the given Go basic
// type.
func (gen *Generator) irBasicType(goType *gotypes.Basic) types.Type {
	switch goType.Kind() {
	// predeclared types
	case gotypes.Bool:
		return types.I1
	case gotypes.Int, gotypes.Uint:
		return types.NewInt(cpuWordSize)
	case gotypes.Int8, gotypes.Uint8:
		return types.I8
	case gotypes.Int16, gotypes.Uint16:
		return types.I16
	case gotypes.Int32, gotypes.Uint32:
		return types.I32
	case gotypes.Int64, gotypes.Uint64:
		return types.I64
	case gotypes.Uintptr:
		return types.NewInt(cpuWordSize)
	case gotypes.Float32:
		return types.Float
	case gotypes.Float64:
		return types.Double
	case gotypes.Complex64:
		return types.NewStruct(
			types.Float, // real
			types.Float, // imag
		)
	case gotypes.Complex128:
		return types.NewStruct(
			types.Double, // real
			types.Double, // imag
		)
	case gotypes.String:
		return types.NewStruct(
			types.NewPointer(types.I8), // data
			types.I64,                  // len
		)
	case gotypes.UnsafePointer:
		return types.NewInt(cpuWordSize)
	// types for untyped values
	case gotypes.UntypedBool:
		return types.I1
	case gotypes.UntypedInt:
		t := types.NewInt(64)
		t.SetName("untyped_int")
		gen.typeDefs["untyped_int"] = t
		return t
	case gotypes.UntypedRune:
		t := types.NewInt(32)
		t.SetName("untyped_rune")
		gen.typeDefs["untyped_rune"] = t
		return t
	case gotypes.UntypedFloat:
		t := &types.FloatType{Kind: types.FloatKindDouble}
		t.SetName("untyped_float")
		gen.typeDefs["untyped_float"] = t
		return t
	case gotypes.UntypedComplex:
		untypedFloat := &types.FloatType{Kind: types.FloatKindDouble}
		untypedFloat.SetName("untyped_float")
		t := types.NewStruct(
			untypedFloat, // real
			untypedFloat, // imag
		)
		t.SetName("untyped_complex")
		gen.typeDefs["untyped_complex"] = t
		return t
	case gotypes.UntypedString:
		t := types.NewStruct(
			types.NewPointer(types.I8), // data
			types.I64,                  // len
		)
		t.SetName("untyped_string")
		gen.typeDefs["untyped_string"] = t
		return t
	case gotypes.UntypedNil:
		t := types.NewPointer(types.I8)
		t.SetName("untyped_nil")
		gen.typeDefs["untyped_nil"] = t
		return t
	default:
		panic(fmt.Errorf("support for basic type of kind %v not yet implemented", goType.Kind()))
	}
}
