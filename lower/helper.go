package lower

import (
	"go/ast"

	"github.com/llir/llvm/ir"
)

// irParams returns the LLVM IR parameters based on the given Go field list.
func (gen *Generator) irParams(old *ast.FieldList) []*ir.Param {
	if old == nil {
		return nil
	}
	var params []*ir.Param
	for _, oldParam := range old.List {
		typ, err := gen.irTypeOf(oldParam.Type)
		if err != nil {
			gen.eh(err)
			continue
		}
		if len(oldParam.Names) > 0 {
			for _, name := range oldParam.Names {
				param := ir.NewParam(name.String(), typ)
				params = append(params, param)
			}
		} else {
			// Unnamed parameter.
			param := ir.NewParam("", typ)
			params = append(params, param)
		}
	}
	return params
}
