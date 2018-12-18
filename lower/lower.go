// Package lower lowers Go source code in AST-form to LLVM IR assembly.
package lower

import (
	"fmt"
	"go/ast"
	gotypes "go/types"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
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
	// Package scope.
	scope *gotypes.Scope
	// Current child scope.
	curChildScope int
	// LLVM IR module being generated.
	m *ir.Module
	// index of LLVM IR top-level entities.
	new newIndex
}

// NewGenerator returns a new generator for lowering the source code of the
// given Go package to LLVM IR assembly. The error handler eh is invoked when an
// error is encountered during compilation.
func NewGenerator(eh func(error), pkg *packages.Package) *Generator {
	gen := &Generator{
		eh:    eh,
		pkg:   pkg,
		scope: pkg.Types.Scope(),
		m:     ir.NewModule(),
		new: newIndex{
			typeDefs: make(map[string]types.Type),
			globals:  make(map[string]*ir.Global),
			funcs:    make(map[string]*ir.Function),
		},
	}
	return gen
}

// Lower lowers the source code of the given Go package to LLVM IR.
func (gen *Generator) Lower() *ir.Module {
	gen.lowerPackage()
	// Append type definitions to module.
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
	// LLVM IR function generator.
	fgen := gen.newFuncGen()
	// Function name.
	funcName := old.Name.String()
	// Function scope.
	funcScope := gen.scope.Innermost(old.Name.Pos())
	fgen.scope = funcScope
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
	results := gen.irParams(old.Type.Results)
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
		gen.Errorf("function %q already present; prev `%v`, new `%v`", funcName, prev, f)
		return
	}
	fgen.f = f
	gen.new.funcs[funcName] = f
	// Lower function body.
	if old.Body != nil {
		fgen.lowerFuncBody(old.Body)
	}
}

// lowerGenDecl lowers the given Go generic declaration to LLVM IR.
func (gen *Generator) lowerGenDecl(old *ast.GenDecl) {
	for _, oldSpec := range old.Specs {
		gen.lowerSpec(oldSpec)
	}
}

// lowerSpec lowers the given Go specifier to LLVM IR, emitting to m.
func (gen *Generator) lowerSpec(old ast.Spec) {
	switch old := old.(type) {
	case *ast.TypeSpec:
		// handled by resolveTypeDefs.
	case *ast.ValueSpec:
		gen.lowerValueSpec(old)
	default:
		panic(fmt.Errorf("support for specifier %T not yet implemented", old))
	}
}

// lowerValueSpec lowers the given Go variable declaration to LLVM IR, emitting
// to m.
func (gen *Generator) lowerValueSpec(old *ast.ValueSpec) {
	for i, oldName := range old.Names {
		name := oldName.String()
		if len(old.Values) > 0 {
			oldValue := old.Values[i]
			init, err := gen.lowerGlobalInitExpr(oldValue)
			if err != nil {
				gen.eh(err)
				continue
			}
			v := gen.m.NewGlobalDef(name, init)
			gen.new.globals[name] = v
		} else {
			typ, err := gen.irTypeOf(old.Type)
			if err != nil {
				gen.eh(err)
				return
			}
			v := gen.m.NewGlobalDecl(name, typ)
			gen.new.globals[name] = v
		}
	}
}

// lowerGlobalInitExpr lowers the given Go global initialization expression to
// LLVM IR.
func (gen *Generator) lowerGlobalInitExpr(old ast.Expr) (constant.Constant, error) {
	switch old := old.(type) {
	// Constant.
	case *ast.BasicLit:
		return gen.lowerBasicLit(old), nil
	// Non-constant, generate init functions.
	default:
		panic(fmt.Errorf("support for global initialization expression %T not yet implemented", old))
	}
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
