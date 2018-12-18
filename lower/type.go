package lower

import (
	"fmt"
	"go/ast"
	gotypes "go/types"

	"github.com/llir/llvm/ir/types"
)

// irTypeOf returns the LLVM IR type of the given Go expression.
func (gen *Generator) irTypeOf(expr ast.Expr) (types.Type, error) {
	goType := gen.pkg.TypesInfo.TypeOf(expr)
	return gen.irType(goType)
}

// irType returns the IR type of the given Go expression.
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

// irBasicType returns the IR type of the given Go basic type.
func (gen *Generator) irBasicType(goType *gotypes.Basic) types.Type {
	// predeclared types
	switch goType.Kind() {
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
		var (
			realType    = types.Float
			complexType = types.Float
		)
		return types.NewStruct(realType, complexType)
	case gotypes.Complex128:
		var (
			realType    = types.Double
			complexType = types.Double
		)
		return types.NewStruct(realType, complexType)
	case gotypes.String:
		var (
			dataType = types.NewPointer(types.I8)
			lenType  = types.I64
		)
		return types.NewStruct(dataType, lenType)
	case gotypes.UnsafePointer:
		return types.NewInt(cpuWordSize)
	// types for untyped values
	case gotypes.UntypedBool:
		return types.I1
	case gotypes.UntypedInt:
		untypedInt := types.NewInt(64)
		untypedInt.SetName("untyped_int")
		gen.new.typeDefs["untyped_int"] = untypedInt
		return untypedInt
	case gotypes.UntypedRune:
		untypedRune := types.NewInt(32)
		untypedRune.SetName("untyped_rune")
		gen.new.typeDefs["untyped_rune"] = untypedRune
		return untypedRune
	case gotypes.UntypedFloat:
		untypedFloat := &types.FloatType{Kind: types.FloatKindDouble}
		untypedFloat.SetName("untyped_float")
		gen.new.typeDefs["untyped_float"] = untypedFloat
		return untypedFloat
	case gotypes.UntypedComplex:
		untypedFloat := &types.FloatType{Kind: types.FloatKindDouble}
		untypedFloat.SetName("untyped_float")
		var (
			realType    = untypedFloat
			complexType = untypedFloat
		)
		untypedComplex := types.NewStruct(realType, complexType)
		untypedComplex.SetName("untyped_complex")
		gen.new.typeDefs["untyped_complex"] = untypedComplex
		return untypedComplex
	case gotypes.UntypedString:
		var (
			dataType = types.NewPointer(types.I8)
			lenType  = types.I64
		)
		untypedString := types.NewStruct(dataType, lenType)
		untypedString.SetName("untyped_string")
		gen.new.typeDefs["untyped_string"] = untypedString
		return untypedString
	case gotypes.UntypedNil:
		untypedNil := types.NewPointer(types.I8)
		untypedNil.SetName("untyped_nil")
		gen.new.typeDefs["untyped_nil"] = untypedNil
		return untypedNil
	default:
		panic(fmt.Errorf("support for basic type of kind %v not yet implemented", goType.Kind()))
	}
}
