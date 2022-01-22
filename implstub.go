package implstub

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/types"
	"html/template"
	"implstub/detect"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

type alreadyDecl struct {
	recvName string
	methods  map[string]struct{}
}

// methodSig represents a methodSig signature.
type methodSig struct {
	Recv string
	funcSig
}

// funcSig represents a function signature.
type funcSig struct {
	Name     string
	Params   string
	Res      string
	Comments string
}

// paramSig represents a parameter in a function or method signature.
type paramSig struct {
	Name string
	Type string
}

const doc = "implstub is ..."

// ImplStubAnalyzer is ...
var ImplStubAnalyzer = &analysis.Analyzer{
	Name: "implstub",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

var (
	output            string
	overWrite         bool
	pointerReciever   bool
	detectedInterface *detect.Result
	detectedRecv      *detect.Result
)

const stub = "{{if .Comments}}{{.Comments}}{{end}}" +
	"func ({{.Recv}}) {{.Name}}" +
	"{{.Params}}" +
	"{{.Res}}" +
	"{\n" + "panic(\"not implemented\") // TODO: Implement" + "\n}\n\n"

var tmpl = template.Must(template.New("test").Parse(stub))

func init() {
	ImplStubAnalyzer.Flags.StringVar(&output, "o", output, "specify the output file path")
	ImplStubAnalyzer.Flags.BoolVar(&overWrite, "w", overWrite, "overwrite the specified receiver file")
	ImplStubAnalyzer.Flags.BoolVar(&pointerReciever, "p", pointerReciever, "create a stub with the pointer receiver")

	srcPath := strings.TrimSuffix(os.Args[len(os.Args)-1], "...")

	var err error
	detectedInterface, err = detect.DetectInterface(srcPath)
	if err != nil {
		log.Fatal(err)
	}
	detectedRecv, err = detect.DetectReciever(srcPath)
	if err != nil {
		log.Fatal(err)
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	var (
		targetInterface *types.Interface
		targetRecv      *types.TypeName
	)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// 指定したInterfaceとRecieverと同名の宣言を探す
	inspect.Preorder([]ast.Node{(*ast.GenDecl)(nil)}, func(n ast.Node) {
		nGenDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return
		}

		for _, spec := range nGenDecl.Specs {
			t, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			defType, ok := pass.TypesInfo.Defs[t.Name].(*types.TypeName)
			if !ok {
				continue
			}

			switch defT := defType.Type().Underlying().(type) {
			case *types.Interface:
				if defType.Name() == detectedInterface.Name {
					targetInterface = defT
				}
			case *types.Struct:
				if defType.Name() == detectedRecv.Name {
					targetRecv = defType
				}
			}
		}
	})

	fmt.Printf("targetInterface: %v\n", targetInterface)
	fmt.Printf("targetRecv: %v\n", targetRecv)
	// ターゲットが見つかるまでパッケージを走査する
	if targetInterface == nil || targetRecv == nil {
		return nil, nil
	}

	// ターゲットが見つかったらスタブを書き出す
	return write(inspect, targetInterface, targetRecv)
}

func write(inspect *inspector.Inspector, targetInterface *types.Interface, targetRecv *types.TypeName) (interface{}, error) {
	var (
		f   io.WriteCloser = os.Stdout
		err error
	)
	if overWrite {
		f, err = os.OpenFile(detectedRecv.FilePath, os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		defer f.Close()
	} else if output != "" {
		f, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		defer f.Close()
	}

	decl := getAlreadyDecl(inspect, targetRecv)
	recvPkgPath := targetRecv.Pkg().Path()

	// スタブメソッドを書き出す
	for i := 0; i < targetInterface.NumMethods(); i++ {
		m := targetInterface.Method(i)

		mSig := m.Type().Underlying().(*types.Signature)

		funcName := m.Name()
		funcParams := trimPackage(strings.ReplaceAll(mSig.Params().String(), recvPkgPath+".", ""))
		funcResults := trimPackage(strings.ReplaceAll(mSig.Results().String(), recvPkgPath+".", ""))

		// 実装済みのメソッドはスキップ
		if _, ok := decl.methods[decl.methodKey(funcName, funcParams, funcResults)]; ok {
			fmt.Println("skip already defined: " + decl.methodKey(funcName, funcParams, funcResults))
			continue
		}

		stub, err := genStubs(decl.recvName+" *"+detectedRecv.Name, []funcSig{
			{
				Name:     funcName,
				Params:   funcParams,
				Res:      funcResults,
				Comments: fmt.Sprintf("// %s comments...\n", funcName),
			},
		})
		if err != nil {
			return nil, err
		}

		fmt.Fprintf(f, string(stub))
	}

	return nil, nil
}

// genStubs prints nicely formatted method stubs
func genStubs(recv string, fns []funcSig) ([]byte, error) {
	var buf bytes.Buffer
	for _, fn := range fns {
		meth := methodSig{Recv: recv, funcSig: fn}
		tmpl.Execute(&buf, meth)
	}

	pretty, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return pretty, nil
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

// getAlreadyDecl 対象のレシーバに既に実装されている情報を取得する
func getAlreadyDecl(inspect *inspector.Inspector, targetRecv *types.TypeName) *alreadyDecl {
	result := &alreadyDecl{
		recvName: strings.ToLower(detectedRecv.Name),
	}

	inspect.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		nType, ok := n.(*ast.FuncDecl)
		if !ok || nType.Recv == nil {
			return
		}

		for _, r := range nType.Recv.List {
			// 値レシーバーとポインターレシーバーそれぞれチェックする必要がある
			switch t := r.Type.(type) {
			case *ast.Ident:
				if detectedRecv.Name != t.Name {
					continue
				}
			case *ast.StarExpr:
				if detectedRecv.Name != fmt.Sprint(t.X) {
					continue
				}
			}

			// 対象のオブジェクトに既にレシーバ名が宣言されている場合は合わせる
			// 違う名前がついていることは考慮しない
			if len(r.Names) > 0 {
				result.recvName = r.Names[0].String()
			}

			// 指定されたレシーバにメソッドが存在する場合はメソッド名・引数・返り値を保存する
			// 指定されたインターフェースを満たすためのメソッドが既に実装されていれば出力をスキップすることになる
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
			result.appendMethod(nType.Name.String(), fmt.Sprintf("(%s)", strings.Join(params, ", ")), fmt.Sprintf("(%s)", strings.Join(results, ", ")))
		}
	})

	return result
}

func (a *alreadyDecl) appendMethod(name, params, results string) {
	if a == nil {
		return
	}

	a.methods[a.methodKey(name, params, results)] = struct{}{}
}

func (a *alreadyDecl) methodKey(name, params, results string) string {
	return fmt.Sprintf("%s(%s) (%s)", name, params, results)
}
