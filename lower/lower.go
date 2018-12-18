// Package lower lowers Go source code in AST-form to LLVM IR assembly.
package lower

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/rickypai/natsort"
)

// Lower lowers the source code of the Go package to LLVM IR.
func (gen *Generator) Lower() *ir.Module {
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

// lowerFile lowers the Go file to LLVM IR, emitting to m.
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
	// Create LLVM IR function generator.
	fgen := gen.newFuncGen()
	funcName := goFuncDecl.Name.String()
	// Function scope.
	fgen.scope = gen.scope.Innermost(goFuncDecl.Name.Pos())
	// Receiver.
	receivers := gen.irParams(goFuncDecl.Recv)
	// Function parameters.
	params := gen.irParams(goFuncDecl.Type.Params)
	// Add reciver to function parameters if present.
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
	fgen.f = f
	if prev, ok := gen.funcs[funcName]; ok {
		gen.Errorf("function %q already present; prev `%v`, new `%v`", funcName, prev, f)
		return
	}
	gen.funcs[funcName] = f
	// Lower function body.
	if goFuncDecl.Body != nil {
		fgen.lowerFuncBody(goFuncDecl.Body)
	}
}

// lowerFuncBody lowers the Go function body block statement to LLVM IR,
// emitting to f.
func (fgen *funcGen) lowerFuncBody(goBlockStmt *ast.BlockStmt) {
	fgen.cur = fgen.f.NewBlock("entry")
	for _, goStmt := range goBlockStmt.List {
		fgen.lowerStmt(goStmt)
	}
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
		name := goName.String()
		if len(goSpec.Values) > 0 {
			// Global variable definition.
			goExpr := goSpec.Values[i]
			init, err := gen.lowerGlobalInitExpr(goExpr)
			if err != nil {
				gen.eh(err)
				continue
			}
			v := gen.m.NewGlobalDef(name, init)
			gen.globals[name] = v
		} else {
			// Global variable declaration.
			typ, err := gen.irTypeOf(goSpec.Type)
			if err != nil {
				gen.eh(err)
				continue
			}
			v := gen.m.NewGlobalDecl(name, typ)
			gen.globals[name] = v
		}
	}
}
