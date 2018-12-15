package main

import (
	"fmt"
	"go/ast"

	"github.com/llir/llvm/ir/types"
	"golang.org/x/tools/go/packages"
)

// indexPackage indexes the top-level declarations of the given Go package.
func (gen *generator) indexPackage(pkg *packages.Package) {
	// Index top-level declarations.
	for _, file := range pkg.Syntax {
		gen.indexFile(file)
	}
}

// indexFile indexes the top-level declarations of the given Go file.
func (gen *generator) indexFile(file *ast.File) {
	// Index top-level declarations.
	for _, decl := range file.Decls {
		gen.indexDecl(decl)
	}
}

// indexDecl indexes the given top-level declaration.
func (gen *generator) indexDecl(old ast.Decl) {
	switch old := old.(type) {
	case *ast.FuncDecl:
		gen.indexFuncDecl(old)
	case *ast.GenDecl:
		gen.indexGenDecl(old)
	default:
		panic(fmt.Errorf("support for declaration %T not yet implemented", old))
	}
}

// indexFuncDecl indexes the given function declaration.
func (gen *generator) indexFuncDecl(old *ast.FuncDecl) {
	funcName := old.Name.String()
	// Receiver.
	receivers := gen.irParams(old.Recv)
	// Function parameters.
	params := gen.irParams(old.Type.Params)
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
	results := gen.irParams(old.Type.Params)
	var retType types.Type
	switch len(results) {
	case 0:
		retType = types.Void
	case 1:
		retType = results[0].Typ
	default:
		var resultTypes []types.Type
		for _, result := range results {
			resultTypes = append(resultTypes, result.Typ)
		}
		retType = types.NewStruct(resultTypes...)
	}
	// Add function.
	f := gen.m.NewFunc(funcName, retType, params...)
	if prev, ok := gen.new.funcs[funcName]; ok {
		gen.c.Errorf("function %q already present; prev `%v`, new `%v`", funcName, prev, f)
		return
	}
	gen.new.funcs[funcName] = f
}

// indexGenDecl indexes the given top-level declaration.
func (gen *generator) indexGenDecl(old *ast.GenDecl) {
	for _, oldSpec := range old.Specs {
		gen.indexSpec(oldSpec)
	}
}

// indexSpec indexes the given specifier.
func (gen *generator) indexSpec(old ast.Spec) {
	switch old := old.(type) {
	case *ast.TypeSpec:
		gen.old.typeDefs[old.Name.String()] = old.Type
	default:
		panic(fmt.Errorf("support for specifier %T not yet implemented", old))
	}
}
