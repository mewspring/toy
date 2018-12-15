package main

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/kr/pretty"
	"github.com/llir/llvm/ir/types"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

// resolveTypeDefs resolves the type definitions of the given Go package.
func (gen *generator) resolveTypeDefs(pkg *packages.Package) {
	// Index type identifiers and create scaffolding IR type definitions (without
	// bodies).
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			if decl, ok := decl.(*ast.GenDecl); ok {
				if decl.Tok != token.TYPE {
					continue
				}
				for _, spec := range decl.Specs {
					ts := spec.(*ast.TypeSpec)
					typeName := ts.Name.String()
					gen.old.typeDefs[typeName] = ts.Type
				}
			}
		}
	}
	for typeName, oldType := range gen.old.typeDefs {
		t := gen.newType(typeName, oldType)
		t.SetName(typeName)
		gen.new.typeDefs[typeName] = t
	}
	pretty.Println("gen.old.typeDefs", gen.old.typeDefs)
	// Translate AST type definitions to IR.
	for typeName, oldType := range gen.old.typeDefs {
		new := gen.new.typeDefs[typeName]
		gen.irTypeDef(new, oldType)
	}
}

// newType creates a new LLVM IR type (without body) based on the given Go type.
func (gen *generator) newType(typeName string, old ast.Expr) types.Type {
	switch old := old.(type) {
	case *ast.Ident:
		newName := old.String()
		newType := gen.old.typeDefs[newName]
		return gen.newType(newName, newType)
	case *ast.StarExpr:
		return &types.PointerType{TypeName: typeName}
	case *ast.StructType:
		return &types.StructType{TypeName: typeName}
	default:
		panic(fmt.Errorf("support for type %T not yet implemented", old))
	}
}

// irTypeDef translates the AST type into an equivalent IR type. A new IR type
// correspoding to the AST type is created if t is nil, otherwise the body of t
// is populated. Named types are resolved through gen.new.typeDefs.
func (gen *generator) irTypeDef(t types.Type, old ast.Expr) (types.Type, error) {
	switch old := old.(type) {
	case *ast.Ident:
		return gen.irNamedType(t, old)
	case *ast.StarExpr:
		return gen.irPointerType(t, old)
	case *ast.StructType:
		return gen.irStructType(t, old)
	default:
		panic(fmt.Errorf("support for type %T not yet implemented", old))
	}
}

// --- [ Pointer type ] --------------------------------------------------------

// irPointerType translates the AST pointer type into an equivalent IR type. A
// new IR type correspoding to the AST type is created if t is nil, otherwise
// the body of t is populated.
func (gen *generator) irPointerType(t types.Type, old *ast.StarExpr) (types.Type, error) {
	typ, ok := t.(*types.PointerType)
	if t == nil {
		typ = &types.PointerType{}
	} else if !ok {
		panic(fmt.Errorf("invalid IR type for AST pointer type; expected *types.PointerType, got %T", t))
	}
	// Element type.
	elemType, err := gen.irType(old.X)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	typ.ElemType = elemType
	return typ, nil
}

// --- [ Named type ] ----------------------------------------------------------

// irNamedType translates the AST named type into an equivalent IR type.
func (gen *generator) irNamedType(t types.Type, old *ast.Ident) (types.Type, error) {
	// TODO: make use of t?
	// Resolve named type.
	typeName := old.String()
	typ, ok := gen.new.typeDefs[typeName]
	if !ok {
		return nil, errors.Errorf("unable to locate type definition of named type %q", typeName)
	}
	return typ, nil
}

// --- [ Struct type ] ---------------------------------------------------------

// irStructType translates the AST struct type into an equivalent IR type. A new
// IR type correspoding to the AST type is created if t is nil, otherwise the
// body of t is populated.
func (gen *generator) irStructType(t types.Type, old *ast.StructType) (types.Type, error) {
	typ, ok := t.(*types.StructType)
	if t == nil {
		typ = &types.StructType{}
	} else if !ok {
		panic(fmt.Errorf("invalid IR type for AST struct type; expected *types.StructType, got %T", t))
	}
	// Fields.
	fields := gen.irParams(old.Fields)
	for _, field := range fields {
		typ.Fields = append(typ.Fields, field.Typ)
	}
	return typ, nil
}

// ### [ Helpers ] #############################################################

// irType returns the IR type corresponding to the given AST type.
func (gen *generator) irType(old ast.Expr) (types.Type, error) {
	return gen.irTypeDef(nil, old)
}
