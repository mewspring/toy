package main

import (
	"fmt"
	"go/ast"
	"log"

	"github.com/llir/llvm/ir"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

// compiler tracks the state of the compiler, including any errors encountered
// during compilation.
type compiler struct {
	// Compiled LLVM IR modules.
	modules []*ir.Module
	// List of errors encountered during compilation.
	errs []error
}

// newCompiler returns a new compiler.
func newCompiler() *compiler {
	return &compiler{}
}

// Errorf appends a new error based on the given format specific and arguments
// to the list of encountered compiler errors.
func (c *compiler) Errorf(format string, args ...interface{}) {
	err := errors.Errorf(format, args...)
	c.errs = append(c.errs, err)
}

// pre is invoked in pre-order traversal of the import graph. The returned
// boolean value determines whether imports of pkg are visited.
func (c *compiler) pre(pkg *packages.Package) bool {
	dbg.Println("pre:", pkg.Name)
	return true
}

// post is invoked in post-order traversal of the import graph.
func (c *compiler) post(pkg *packages.Package) {
	dbg.Println("post:", pkg.Name)
	gen := c.newGenerator(pkg)
	gen.resolveTypeDefs(pkg)
	gen.indexPackage(pkg)
	gen.compilePackage(pkg)
	c.modules = append(c.modules, gen.m)
}

// compilePackage compiles the given Go package.
func (gen *generator) compilePackage(pkg *packages.Package) {
	// Compile top-level declarations.
	for _, file := range pkg.Syntax {
		gen.compileFile(file)
	}
}

// compileFile compiles the given Go file.
func (gen *generator) compileFile(file *ast.File) {
	// Compile top-level declarations.
	for _, old := range file.Decls {
		gen.compileDecl(old)
	}
}

// compileDecl compiles the given declaration.
func (gen *generator) compileDecl(old ast.Decl) {
	switch old := old.(type) {
	case *ast.FuncDecl:
		gen.compileFuncDecl(old)
	case *ast.GenDecl:
		gen.compileGenDecl(old)
	default:
		panic(fmt.Errorf("support for declaration %T not yet implemented", old))
	}
}

// compileFuncDecl compiles the given function declaration.
func (gen *generator) compileFuncDecl(old *ast.FuncDecl) {
	funcName := old.Name.String()
	f, ok := gen.new.funcs[funcName]
	if !ok {
		gen.c.Errorf("unable to locate function %q", funcName)
		return
	}
	fgen := gen.newFuncGen(f)
	if old.Body != nil {
		fgen.lowerFuncBody(old.Body)
	}
}

// compileGenDecl compiles the given generic declaration.
func (gen *generator) compileGenDecl(old *ast.GenDecl) {
	log.Printf("support for top-level declaration %T not yet implemented", old)
}
