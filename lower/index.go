package lower

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir/types"
)

// indexPackage indexes global identifiers and creates scaffolding IR type
// definitions, global variable and function declarations and definitions
// (without bodies but with types) of the Go package.
func (gen *Generator) indexPackage() {
	for _, file := range gen.pkg.Syntax {
		gen.indexFile(file)
	}
}

// indexFile indexes global identifiers and creates scaffolding IR type
// definitions, global variable and function declarations and definitions
// (without bodies but with types) of the Go source file.
func (gen *Generator) indexFile(file *ast.File) {
	// Index top-level declarations.
	for _, goDecl := range file.Decls {
		gen.indexDecl(goDecl)
	}
}

// === [ Declarations ] ========================================================

// indexDecl indexes the global identifier and creates a scaffolding IR type
// definition, global variable or function declaration or definition (without
// bodies but with types) of the Go top-level declaration.
func (gen *Generator) indexDecl(goDecl ast.Decl) {
	switch goDecl := goDecl.(type) {
	case *ast.FuncDecl:
		gen.indexFuncDecl(goDecl)
	case *ast.GenDecl:
		gen.indexGenDecl(goDecl)
	default:
		panic(fmt.Errorf("support for declaration %T not yet implemented", goDecl))
	}
}

// --- [ Function declarations ] -----------------------------------------------

// indexFuncDecl indexes the global identifier and creates a scaffolding IR
// function declaration or definition (without bodies but with types) of the Go
// function declaration.
func (gen *Generator) indexFuncDecl(goFuncDecl *ast.FuncDecl) {
	// Receiver.
	receivers := gen.irParams(goFuncDecl.Recv)
	// Function parameters.
	params := gen.irParams(goFuncDecl.Type.Params)
	// Add reciver to function parameters if present.
	funcName := goFuncDecl.Name.String()
	switch len(receivers) {
	case 0:
		// nothing to do.
	case 1:
		// To avoid function name collisions, rename "M" to "T.M".
		recvType := receivers[0].Typ
		funcName = fmt.Sprintf("%s.%s", recvType.Name(), funcName)
		// Prepend receiver as first parameter of function.
		params = append(receivers, params...)
	default:
		panic(fmt.Errorf("support for multiple receivers not yet implemented; %q has %d receivers", funcName, len(receivers)))
	}
	// Return type.
	results := gen.irParams(goFuncDecl.Type.Results)
	var retType types.Type
	switch len(results) {
	case 0:
		// void return.
		retType = types.Void
	case 1:
		// single value return.
		retType = results[0].Typ
	default:
		// multiple value return.
		var resultTypes []types.Type
		for _, result := range results {
			resultTypes = append(resultTypes, result.Typ)
		}
		retType = types.NewStruct(resultTypes...)
	}
	// Add function.
	f := gen.m.NewFunc(funcName, retType, params...)
	if prev, ok := gen.funcs[funcName]; ok {
		gen.Errorf("function %q already present; prev `%v`, new `%v`", funcName, prev, f)
		return
	}
	gen.funcs[funcName] = f
}

// --- [ Generic declarations ] ------------------------------------------------

// indexGenDecl indexes the global identifier and creates a scaffolding IR type
// definition, or global variable declaration or definition (without bodies but
// with types) of the Go generic declaration.
func (gen *Generator) indexGenDecl(goGenDecl *ast.GenDecl) {
	for _, goSpec := range goGenDecl.Specs {
		gen.indexSpec(goSpec)
	}
}

// indexSpec indexes the global identifier and creates a scaffolding IR type
// definition, or global variable declaration or definition (without bodies but
// with types) of the Go specifier.
func (gen *Generator) indexSpec(goSpec ast.Spec) {
	switch goSpec := goSpec.(type) {
	case *ast.ImportSpec:
		// handled by import graph traversal.
	case *ast.TypeSpec:
		// handled by lowerTypeSpec.
	case *ast.ValueSpec:
		gen.indexValueSpec(goSpec)
	default:
		panic(fmt.Errorf("support for specifier %T not yet implemented", goSpec))
	}
}

// indexTypeSpec indexes the global identifier and creates a scaffolding IR
// global variable declaration or definition of the Go value specifier.
func (gen *Generator) indexValueSpec(goSpec *ast.ValueSpec) {
	for _, goName := range goSpec.Names {
		name := goName.String()
		// Global variable declaration or definition.
		typ, err := gen.irTypeOf(goSpec.Type)
		if err != nil {
			gen.eh(err)
			continue
		}
		v := gen.m.NewGlobalDecl(name, typ)
		gen.globals[name] = v
	}
}
