package implstub

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/types"
	"html/template"
	"implstub/detect"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "implstub is ..."

// ImplStubAnalyzer is ...
var ImplStubAnalyzer = &analysis.Analyzer{
	Name: "findInterface",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

var (
	interf string
	recv   string
)

func init() {
	ImplStubAnalyzer.Flags.StringVar(&interf, "interf", interf, "name of the interface")
	ImplStubAnalyzer.Flags.StringVar(&recv, "recv", recv, "name of the reciever")
	detect.RunForRecv()
	detect.RunForInterface()
}

// var (
// 	recvOnce        sync.Once
// 	interfaceOnce   sync.Once
// 	recvObject      *types.TypeName
// 	interfaceObject *types.TypeName
// )

func run(pass *analysis.Pass) (interface{}, error) {
	// fmt.Println(pass.Pkg)

	// recvObjects := pass.ResultOf[FindRecvAnalyzer].([]*types.TypeName)
	// if len(recvObjects) == 0 {
	// 	return nil, nil
	// }

	// var (
	// 	err error
	// 	i   int
	// )
	// recvOnce.Do(func() {
	// 	i, err = fuzzyfinder.Find(
	// 		recvObjects,
	// 		func(i int) string {
	// 			return fmt.Sprintf("%s.%s", recvObjects[i].Pkg().Path(), recvObjects[i].Name())
	// 		},
	// 	)
	// 	recvObject = recvObjects[i]
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// interfaceObjects := pass.ResultOf[FindInterfaceAnalyzer].([]*types.TypeName)
	// if len(interfaceObjects) == 0 {
	// 	return nil, nil
	// }

	// interfaceOnce.Do(func() {
	// 	i, err = fuzzyfinder.Find(
	// 		interfaceObjects,
	// 		func(i int) string {
	// 			return fmt.Sprintf("%s.%s", interfaceObjects[i].Pkg().Path(), interfaceObjects[i].Name())
	// 		},
	// 	)
	// 	interfaceObject = interfaceObjects[i]
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// objectType := interfaceObject.Type()
	// if objectType == nil {
	// 	return nil, nil
	// }

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	recvName := strings.ToLower(string(recvObject.Name()[0]))

	var implementedMethods []implementedMethod
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		nType, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		if nType.Recv != nil {
			for _, r := range nType.Recv.List {
				// FIXME: type特定してちゃんと名前取得したほうがきれいかも
				if recvObject.Name() == fmt.Sprint(r.Type) {
					// レシーバが存在する場合はレシーバ名を合わせる
					if len(r.Names) > 0 {
						recvName = r.Names[0].String()
					}

					// 指定されたレシーバオブジェクトにメソッドが存在する場合は引数と返り値を一旦保存する
					// 指定されたinterfaceを満たすためのメソッドが既に実装されていれば出力をスキップすることになる
					var params []string
					for _, v := range nType.Type.Params.List {
						name := "_"
						if len(v.Names) != 0 {
							name = v.Names[0].String()
						}

						if se, ok := v.Type.(*ast.SelectorExpr); ok {
							params = append(params, fmt.Sprintf("%s %s.%s", name, se.X.(*ast.Ident).Name, se.Sel.Name))
						} else {
							params = append(params, fmt.Sprintf("%s %s", name, v.Type))
						}
					}

					var results []string
					for _, v := range nType.Type.Results.List {
						name := ""
						if len(v.Names) != 0 {
							name = v.Names[0].String() + " "
						}

						if se, ok := v.Type.(*ast.SelectorExpr); ok {
							results = append(results, fmt.Sprintf("%s%s.%s", name, se.X.(*ast.Ident).Name, se.Sel.Name))
						} else {
							results = append(results, fmt.Sprintf("%s%s", name, v.Type))
						}
					}

					implementedMethods = append(implementedMethods, implementedMethod{
						name:    nType.Name.String(),
						params:  fmt.Sprintf("(%s)", strings.Join(params, ", ")),
						results: fmt.Sprintf("(%s)", strings.Join(results, ", ")),
					})
				}
			}
		}
	})

	recvPkgPath := recvObject.Pkg().Path()
	if typeI, ok := objectType.Underlying().(*types.Interface); ok {
	implementLoop:
		for i := 0; i < typeI.NumMethods(); i++ {
			m := typeI.Method(i)
			mSig := m.Type().Underlying().(*types.Signature)

			funcName := m.Name()
			funcParams := trimPackage(strings.ReplaceAll(mSig.Params().String(), recvPkgPath+".", ""))
			funcResults := trimPackage(strings.ReplaceAll(mSig.Results().String(), recvPkgPath+".", ""))

			for _, m := range implementedMethods {
				// 実装済みのメソッドはスキップ
				if m.name == funcName && m.params == funcParams && m.results == funcResults {
					fmt.Printf("m.name: %v\n", m.name)
					fmt.Printf("m.params: %v\n", m.params)
					fmt.Printf("m.results: %v\n", m.results)
					continue implementLoop
				}
			}

			fmt.Println(string(genStubs(recvName+" *"+recvObject.Name(), []Func{
				{
					Name:     funcName,
					Params:   funcParams,
					Res:      funcResults,
					Comments: "// hoge method\n",
				},
			}, nil)))
		}
	}

	f, err := os.Create("./output.txt")
	if err != nil {
		return nil, err
	}

	// fmt.Fprint(f, string(src))

	if err := f.Close(); err != nil {
		return nil, err
	}

	return nil, nil
}

// Method represents a method signature.
type Method struct {
	Recv string
	Func
}

// Func represents a function signature.
type Func struct {
	Name     string
	Params   string
	Res      string
	Comments string
}

// Param represents a parameter in a function or method signature.
type Param struct {
	Name string
	Type string
}

const stub = "{{if .Comments}}{{.Comments}}{{end}}" +
	"func ({{.Recv}}) {{.Name}}" +
	"{{.Params}}" +
	"{{.Res}}" +
	"{\n" + "panic(\"not implemented\") // TODO: Implement" + "\n}\n\n"

var tmpl = template.Must(template.New("test").Parse(stub))

// genStubs prints nicely formatted method stubs
// for fns using receiver expression recv.
// If recv is not a valid receiver expression,
// genStubs will panic.
// genStubs won't generate stubs for
// already implemented methods of receiver.
func genStubs(recv string, fns []Func, implemented map[string]bool) []byte {
	var buf bytes.Buffer
	for _, fn := range fns {
		if implemented[fn.Name] {
			continue
		}
		meth := Method{Recv: recv, Func: fn}
		tmpl.Execute(&buf, meth)
	}

	pretty, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	return pretty
}

func trimPackage(str string) string {
	// (adb implstub/testdata/src/a.DB, db implstub/src/b.DB)
	whiteSpaceSplitStr := strings.Split(str, " ")

	result := make([]string, 0, len(whiteSpaceSplitStr))
	for _, splitStr := range whiteSpaceSplitStr {
		// implstub/testdata/src/a.DB,
		slashSplitStr := strings.Split(splitStr, "/")

		// a.DB,
		result = append(result, slashSplitStr[len(slashSplitStr)-1])
	}

	// (adb a.DB, db b.DB)
	return strings.Join(result, " ")
}

type implementedMethod struct {
	name    string
	params  string
	results string
}
