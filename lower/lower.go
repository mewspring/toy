// Package lower lowers Go source code in AST-form to LLVM IR assembly.
package lower

import (
	"fmt"
	"go/ast"
	"log"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/rickypai/natsort"
	"golang.org/x/tools/go/packages"
)

// Generator keeps track of top-level entities when translating from Go AST to
// LLVM IR representation.
type Generator struct {
	// Error handler used to report errors encountered during compilation.
	eh func(error)
	// Go package being compiled.
	pkg *packages.Package
	// LLVM IR module being generated.
	m *ir.Module
	// index of Go AST top-level entities.
	old oldIndex
	// index of LLVM IR top-level entities.
	new newIndex
}

// NewGenerator returns a new generator for lowering the source code of the
// given Go package to LLVM IR assembly. The error handler eh is invoked when an
// error is encountered during compilation.
func NewGenerator(eh func(error), pkg *packages.Package) *Generator {
	gen := &Generator{
		eh:  eh,
		pkg: pkg,
		m:   ir.NewModule(),
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
	//
	// * int
	intType := types.NewInt(64)
	intType.SetName("int")
	gen.new.typeDefs["int"] = intType
	// * string
	stringType := types.NewStruct(
		types.NewPointer(types.I8), // data
		types.I64,                  // len
	)
	stringType.SetName("string")
	gen.new.typeDefs["string"] = stringType
	// TODO: add remaining built-in types of Go.
	return gen
}

// Lower lowers the source code of the given Go package to LLVM IR.
func (gen *Generator) Lower() *ir.Module {
	gen.resolveTypeDefs()
	gen.indexPackage()
	gen.lowerPackage()
	var typeNames []string
	for typeName := range gen.new.typeDefs {
		typeNames = append(typeNames, typeName)
	}
	natsort.Strings(typeNames)
	for _, typeName := range typeNames {
		t := gen.new.typeDefs[typeName]
		gen.m.NewTypeDef(typeName, t)
	}
	return gen.m
}

// lowerPackage lowers the given Go package to LLVM IR.
func (gen *Generator) lowerPackage() {
	// Compile top-level declarations.
	for _, file := range gen.pkg.Syntax {
		gen.lowerFile(file)
	}
}

// lowerFile lowers the given Go file to LLVM IR.
func (gen *Generator) lowerFile(file *ast.File) {
	// Compile top-level declarations.
	for _, old := range file.Decls {
		gen.lowerDecl(old)
	}
}

// lowerDecl lowers the given Go declaration to LLVM IR.
func (gen *Generator) lowerDecl(old ast.Decl) {
	switch old := old.(type) {
	case *ast.FuncDecl:
		gen.lowerFuncDecl(old)
	case *ast.GenDecl:
		gen.lowerGenDecl(old)
	default:
		panic(fmt.Errorf("support for declaration %T not yet implemented", old))
	}
}

// lowerFuncDecl lowers the given Go function declaration to LLVM IR.
func (gen *Generator) lowerFuncDecl(old *ast.FuncDecl) {
	funcName := old.Name.String()
	f, ok := gen.new.funcs[funcName]
	if !ok {
		gen.Errorf("unable to locate function %q", funcName)
		return
	}
	fgen := gen.newFuncGen(f)
	if old.Body != nil {
		fgen.lowerFuncBody(old.Body)
	}
}

// lowerGenDecl lowers the given Go generic declaration to LLVM IR.
func (gen *Generator) lowerGenDecl(old *ast.GenDecl) {
	log.Printf("support for top-level declaration %T not yet implemented", old)
}

// oldIndex is an index of AST top-level entities.
type oldIndex struct {
	// typeDefs maps from type identifier to the underlying type definition.
	typeDefs map[string]ast.Expr // Go type
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
