package detect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/pkg/errors"
)

type Result struct {
	Name     string
	filePath string
}

var (
	Interface Result
	Recv      Result
)

func RunForRecv() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fileNames, err := findFilesWithWalkDir(wd)
	if err != nil {
		log.Fatal(err)
	}

	i, err := fuzzyfinder.Find(
		fileNames,
		func(i int) string {
			return strings.ReplaceAll(fileNames[i], wd, "")
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	its, err := getStructTypes(fileNames[i])
	if err != nil {
		log.Fatal(err)
	}
	i, err = fuzzyfinder.Find(
		its,
		func(i int) string {
			return its[i].Name.String()
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}

			it, ok := its[i].Type.(*ast.StructType)
			if !ok {
				return ""
			}

			str := fmt.Sprintf("type %s struct {\n", its[i].Name.String())
			for _, field := range it.Fields.List {
				comment := field.Comment.Text()
				if comment != "" {
					str += comment + "\n"
				}

				str += fmt.Sprintf("\t%s\n", prettyMethodParam(field))
			}

			return str + "}"
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	Recv = Result{
		Name:     its[i].Name.String(),
		filePath: fileNames[i],
	}
}

func RunForInterface() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fileNames, err := findFilesWithWalkDir(wd)
	if err != nil {
		log.Fatal(err)
	}

	i, err := fuzzyfinder.Find(
		fileNames,
		func(i int) string {
			return strings.ReplaceAll(fileNames[i], wd, "")
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	its, err := getInterfaceTypes(fileNames[i])
	if err != nil {
		log.Fatal(err)
	}
	i, err = fuzzyfinder.Find(
		its,
		func(i int) string {
			return its[i].Name.String()
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}

			it, ok := its[i].Type.(*ast.InterfaceType)
			if !ok {
				return ""
			}

			var str string

			for _, field := range it.Methods.List {
				ft, ok := field.Type.(*ast.FuncType)
				if !ok {
				}

				comment := field.Comment.Text()
				if comment != "" {
					str += comment + "\n"
				}

				var params []string
				for _, p := range ft.Params.List {
					params = append(params, prettyMethodParam(p))
				}

				var results []string
				for _, r := range ft.Results.List {
					results = append(results, prettyMethodResult(r))
				}

				str += fmt.Sprintf("%s(%s) (%s)\n", field.Names[0], strings.Join(params, ", "), strings.Join(results, ", ")) + "\n"
			}

			return str
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	Interface = Result{
		Name:     its[i].Name.String(),
		filePath: fileNames[i],
	}
}

func prettyMethodParam(f *ast.Field) string {
	return prettyMethod(f, false)
}

func prettyMethodResult(f *ast.Field) string {
	return prettyMethod(f, true)
}

func prettyMethod(f *ast.Field, omitEmptyName bool) string {
	var name string
	if len(f.Names) != 0 {
		name = f.Names[0].Name
	}

	var typeName string
	switch t := f.Type.(type) {
	case *ast.SelectorExpr:
		x, _ := t.X.(*ast.Ident)
		typeName = fmt.Sprintf("%s.%s", x.Name, t.Sel.Name)
	case *ast.Ident:
		typeName = t.Name
	}

	if name == "" {
		if omitEmptyName {
			return typeName
		} else {
			return fmt.Sprintf("_ %s", typeName)
		}
	}

	return fmt.Sprintf("%s %s", name, typeName)
}

func getInterfaceTypes(filename string) ([]*ast.TypeSpec, error) {
	// ファイルごとのトークンの位置を記録する
	fset := token.NewFileSet()

	astf, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	genInterfaces := make([]*ast.TypeSpec, 0)

	// ASTを深さ優先でトラバースする
	ast.Inspect(astf, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		nType, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		for _, spec := range nType.Specs {
			t, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			_, ok = t.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			genInterfaces = append(genInterfaces, t)
		}

		return true
	})

	return genInterfaces, nil
}

func getStructTypes(filename string) ([]*ast.TypeSpec, error) {
	// ファイルごとのトークンの位置を記録する
	fset := token.NewFileSet()

	astf, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	genInterfaces := make([]*ast.TypeSpec, 0)

	// ASTを深さ優先でトラバースする
	ast.Inspect(astf, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		nType, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		for _, spec := range nType.Specs {
			t, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			_, ok = t.Type.(*ast.StructType)
			if !ok {
				continue
			}

			genInterfaces = append(genInterfaces, t)
		}

		return true
	})

	return genInterfaces, nil
}

func findFilesWithWalkDir(root string) ([]string, error) {
	findList := []string{}

	err := filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return errors.Wrap(err, "failed filepath.WalkDir")
		}

		if info.IsDir() {
			return nil
		}

		findList = append(findList, path)
		return nil
	})
	return findList, err
}
