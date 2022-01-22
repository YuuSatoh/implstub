package implstub

import (
	"go/ast"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ImplStubAnalyzer is ...
var FindInterfaceAnalyzer = &analysis.Analyzer{
	Name: "findInterface",
	Doc:  "findInterface is ...",
	Run:  findInterface,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	RunDespiteErrors: true,
	ResultType:       reflect.TypeOf([]*types.TypeName{}),
}

func findInterface(pass *analysis.Pass) (interface{}, error) {
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

			switch defType.Type().Underlying().(type) {
			case *types.Interface:
				candidates = append(candidates, defType)
				// if defType.Name() == interf {
				// 	fmt.Println("@@" + defType.Name())

				// 	for i := 0; i < typ.NumMethods(); i++ {
				// 		m := typ.Method(i)

				// 		mSig := m.Type().Underlying().(*types.Signature)
				// 		fmt.Printf("##### %v %v %v\n", m.Name(), mSig.Params(), mSig.Results())
				// 	}
				// }
			}
		}
	})

	return candidates, nil
}
