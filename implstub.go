package implstub

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/types"
	"html/template"
	"io"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
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

var (
	detectedInterface *Result
	detectedRecv      *Result
)

const stub = "{{if .Comments}}{{.Comments}}{{end}}" +
	"func ({{.Recv}}) {{.Name}}" +
	"{{.Params}}" +
	"{{.Res}}" +
	"{\n" + "panic(\"not implemented\") // TODO: Implement" + "\n}\n\n"

var tmpl = template.Must(template.New("test").Parse(stub))

const pkgPath = "command-line-arguments"

func Exec(output *string, overwrite, pointerReciever bool) error {
	srcPath := strings.TrimSuffix(os.Args[len(os.Args)-1], "...")
	var err error
	detectedInterface, err = DetectInterface(srcPath)
	if err != nil {
		return err
	}

	detectedRecv, err = DetectReciever(srcPath)
	if err != nil {
		return err
	}

	config := &packages.Config{
		Mode: packages.LoadAllSyntax,
	}

	interfacePkgs, err := packages.Load(config, detectedInterface.FilePath)
	if err != nil {
		return err
	}
	interfacePkg := interfacePkgs[0]

	recvPkgs, err := packages.Load(config, detectedRecv.FilePath)
	if err != nil {
		return err
	}
	recvPkg := recvPkgs[0]

	var (
		targetInterface *types.Interface
		targetRecv      *types.TypeName
	)

	for _, syntax := range interfacePkg.Syntax {
		ast.Inspect(syntax, func(node ast.Node) bool {
			nGenDecl, ok := node.(*ast.GenDecl)
			if !ok {
				return true
			}

			for _, spec := range nGenDecl.Specs {
				t, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				defType, ok := interfacePkg.TypesInfo.Defs[t.Name].(*types.TypeName)
				if !ok {
					continue
				}

				if defT, ok := defType.Type().Underlying().(*types.Interface); ok && defType.Name() == detectedInterface.Name {
					targetInterface = defT
					return false
				}
			}

			return true
		})
	}

	var decl *alreadyDecl
	for _, syntax := range recvPkg.Syntax {
		ast.Inspect(syntax, func(node ast.Node) bool {
			nGenDecl, ok := node.(*ast.GenDecl)
			if !ok {
				return true
			}

			for _, spec := range nGenDecl.Specs {
				t, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				defType, ok := recvPkg.TypesInfo.Defs[t.Name].(*types.TypeName)
				if !ok {
					continue
				}

				if _, ok := defType.Type().Underlying().(*types.Struct); ok && defType.Name() == detectedRecv.Name {
					targetRecv = defType

					decl = getAlreadyDecl(inspector.New(recvPkg.Syntax), targetRecv)
					return false
				}
			}

			return true
		})
	}

	// ターゲットが見つからないケースはないはずだが一応エラーにしておく
	if targetInterface == nil || targetRecv == nil || decl == nil {
		return errors.New("not found target")
	}

	// ターゲットが見つかったらスタブを書き出す
	return write(targetInterface, targetRecv, decl, output, overwrite, pointerReciever)
}

func write(targetInterface *types.Interface, targetRecv *types.TypeName, decl *alreadyDecl, output *string, overwrite, pointerReciever bool) error {
	var (
		f   io.WriteCloser = os.Stdout
		err error
	)
	if overwrite {
		f, err = os.OpenFile(detectedRecv.FilePath, os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		defer f.Close()
	} else if output != nil {
		f, err = os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		defer f.Close()
	}

	// スタブメソッドを書き出す
	for i := 0; i < targetInterface.NumMethods(); i++ {
		m := targetInterface.Method(i)

		mSig := m.Type().Underlying().(*types.Signature)

		funcName := m.Name()
		funcParams := ArrangePackagePath(detectedRecv.FilePath, detectedInterface.FilePath, mSig.Params().String())
		funcResults := ArrangePackagePath(detectedRecv.FilePath, detectedInterface.FilePath, mSig.Results().String())

		// 実装済みのメソッドはスキップ
		if _, ok := decl.methods[decl.methodKey(funcName, funcParams, funcResults)]; ok {
			fmt.Println("skip already defined: " + decl.methodKey(funcName, funcParams, funcResults))
			continue
		}

		pointer := ""
		if pointerReciever {
			pointer = "*"
		}

		stub, err := genStubs(fmt.Sprintf("%s %s%s", decl.recvName, pointer, detectedRecv.Name), []funcSig{
			{
				Name:     funcName,
				Params:   funcParams,
				Res:      funcResults,
				Comments: fmt.Sprintf("// %s comments...\n", funcName),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to generate stub, funcName: %s, funcParams: %s, funcResults: %s, err: %w", funcName, funcParams, funcResults, err)
		}

		fmt.Fprintf(f, string(stub))
	}

	return nil
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

// ArrangePackagePath 配置先のファイルパッケージに合わせてパラメーターのパッケージ指定を調整する
func ArrangePackagePath(dstFilePath, srcFilePath, srcParams string) string {
	// ファイル名まで指定されているので取り除く
	dstFilePath = trimFileName(dstFilePath)
	srcFilePath = trimFileName(srcFilePath)

	// packages.Loadが内部でgo listを使用しておりgoのファイルを指定するとpackage.pathがcommand-line-argumentsとなる
	// 扱いづらいのでgo listで指定したファイルのパッケージパスに置き換える
	params := strings.ReplaceAll(srcParams, pkgPath, srcFilePath)
	// 配置先のパッケージが含まれている場合は指定不要なので取り除く
	params = strings.ReplaceAll(params, dstFilePath+".", "")
	// implstub/src/a.ADB のようにパッケージパスを含んだ形式になっているため一律取り除く
	return trimPackage(params)
}

func trimPackage(str string) string {
	str = strings.ReplaceAll(str, "(", "")
	str = strings.ReplaceAll(str, ")", "")

	// adb implstub/testdata/src/a.DB, db *implstub/src/b.DB
	whiteSpaceSplitStr := strings.Split(str, " ")

	result := make([]string, 0, len(whiteSpaceSplitStr))
	for _, splitStr := range whiteSpaceSplitStr {
		// [ implstub, testdata, src, a.DB]
		slashSplitStr := strings.Split(splitStr, "/")
		// a.DB,
		pkgRemovedStr := slashSplitStr[len(slashSplitStr)-1]
		if strings.HasPrefix(splitStr, "*") {
			pkgRemovedStr = "*" + pkgRemovedStr
		}
		result = append(result, pkgRemovedStr)
	}

	// (adb a.DB, db b.DB)
	return fmt.Sprintf("(%s)", strings.Join(result, " "))
}

func trimFileName(str string) string {
	// implstub/testdata/src/a.DB
	// [ implstub, testdata, src, a.DB ]
	slashSplitStr := strings.Split(str, "/")
	if len(slashSplitStr) < 2 {
		return str
	}

	// [ implstub, testdata, src]
	removedFileName := slashSplitStr[:len(slashSplitStr)-1]

	// implstub/testdata/src
	return strings.Join(removedFileName, "/")
}

// getAlreadyDecl 対象のレシーバに既に実装されている情報を取得する
func getAlreadyDecl(inspect *inspector.Inspector, targetRecv *types.TypeName) *alreadyDecl {
	result := &alreadyDecl{
		recvName: strings.ToLower(detectedRecv.Name),
		methods:  make(map[string]struct{}),
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
			if nType.Type.Params != nil {
				for _, v := range nType.Type.Params.List {
					// ここらへん怪しい
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
			}

			var results []string
			if nType.Type.Results != nil {
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
