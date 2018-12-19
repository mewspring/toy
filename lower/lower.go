// Package lower lowers Go source code in AST-form to LLVM IR assembly.
package lower

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir"
	"github.com/rickypai/natsort"
)

// Lower lowers the source code of the Go package to LLVM IR.
func (gen *Generator) Lower() *ir.Module {
	// Index top-level declarations.
	gen.indexPackage()
	// Lower Go package to LLVM IR.
	gen.lowerPackage()
	// Append type definitions to module.
	var typeNames []string
	for typeName := range gen.typeDefs {
		typeNames = append(typeNames, typeName)
	}
	natsort.Strings(typeNames)
	for _, typeName := range typeNames {
		t := gen.typeDefs[typeName]
		gen.m.NewTypeDef(typeName, t)
	}
	return gen.m
}

// lowerPackage lowers the Go package to LLVM IR, emitting to m.
func (gen *Generator) lowerPackage() {
	for _, file := range gen.pkg.Syntax {
		gen.lowerFile(file)
	}
}

// lowerFile lowers the Go source file to LLVM IR, emitting to m.
func (gen *Generator) lowerFile(file *ast.File) {
	// Lower top-level declarations.
	for _, goDecl := range file.Decls {
		gen.lowerDecl(goDecl)
	}
}

// === [ Declarations ] ========================================================

// lowerDecl lowers the Go top-level declaration to LLVM IR, emitting to m.
func (gen *Generator) lowerDecl(goDecl ast.Decl) {
	switch goDecl := goDecl.(type) {
	case *ast.FuncDecl:
		gen.lowerFuncDecl(goDecl)
	case *ast.GenDecl:
		gen.lowerGenDecl(goDecl)
	default:
		panic(fmt.Errorf("support for declaration %T not yet implemented", goDecl))
	}
}

// --- [ Function declarations ] -----------------------------------------------

// lowerFuncDecl lowers the Go function declaration to LLVM IR, emitting to m.
func (gen *Generator) lowerFuncDecl(goFuncDecl *ast.FuncDecl) {
	if goFuncDecl.Body == nil {
		// Function declaration.
		return
	}
	// Locate function definition.
	funcName := goFuncDecl.Name.String()
	f, ok := gen.funcs[funcName]
	if !ok {
		gen.Errorf("unable to locate function definition %q", funcName)
		return
	}
	// Create LLVM IR function generator.
	fgen := gen.newFuncGen()
	fgen.f = f
	// Function scope.
	fgen.scope = gen.scope.Innermost(goFuncDecl.Name.Pos())
	// Lower function body.
	fgen.cur = fgen.f.NewBlock("entry")
	fgen.lowerStmt(goFuncDecl.Body)
}

// --- [ Generic declarations ] ------------------------------------------------

// lowerGenDecl lowers the Go generic declaration to LLVM IR.
func (gen *Generator) lowerGenDecl(goGenDecl *ast.GenDecl) {
	for _, goSpec := range goGenDecl.Specs {
		gen.lowerSpec(goSpec)
	}
}

// lowerSpec lowers the Go specifier to LLVM IR, emitting to m.
func (gen *Generator) lowerSpec(goSpec ast.Spec) {
	switch goSpec := goSpec.(type) {
	case *ast.ImportSpec:
		// handled by import graph traversal.
	case *ast.TypeSpec:
		gen.lowerTypeSpec(goSpec)
	case *ast.ValueSpec:
		gen.lowerValueSpec(goSpec)
	default:
		panic(fmt.Errorf("support for specifier %T not yet implemented", goSpec))
	}
}

// lowerTypeSpec lowers the Go type specifier to LLVM IR, emitting to m.
func (gen *Generator) lowerTypeSpec(goSpec *ast.TypeSpec) {
	typ, err := gen.irTypeOf(goSpec.Type)
	if err != nil {
		gen.eh(err)
		return
	}
	name := goSpec.Name.String()
	typ.SetName(name)
	gen.typeDefs[name] = typ
}

// lowerValueSpec lowers the Go value specifier to LLVM IR, emitting to m.
func (gen *Generator) lowerValueSpec(goSpec *ast.ValueSpec) {
	for i, goName := range goSpec.Names {
		if len(goSpec.Values) == 0 {
			// Global variable declaration.
			continue
		}
		// Global variable definition.
		name := goName.String()
		v, ok := gen.globals[name]
		if !ok {
			gen.Errorf("unable to locate global variable definition %q", name)
			return
		}
		goExpr := goSpec.Values[i]
		init, err := gen.lowerGlobalInitExpr(goExpr)
		if err != nil {
			gen.eh(err)
			continue
		}
		v.Init = init
	}
}
