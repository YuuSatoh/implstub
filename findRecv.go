package implstub

import (
	"go/ast"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// FindRecvAnalyzer is ...
var FindRecvAnalyzer = &analysis.Analyzer{
	Name: "findRecv",
	Doc:  doc,
	Run:  findRecv,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	RunDespiteErrors: true,
	ResultType:       reflect.TypeOf([]*types.TypeName{}),
}

func findRecv(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.GenDecl)(nil),
	}

	var candidates []*types.TypeName
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		nType, ok := n.(*ast.GenDecl)
		if !ok {
			return
		}

		for _, spec := range nType.Specs {
			t, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			defType, ok := pass.TypesInfo.Defs[t.Name].(*types.TypeName)
			if !ok {
				continue
			}

			candidates = append(candidates, defType)
			// switch defType.Type().Underlying().(type) {
			// case *types.Struct:
			// if defType.Name() == recv {
			// }
			// }
		}
	})

	return candidates, nil
}
