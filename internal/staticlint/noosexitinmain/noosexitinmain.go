// Package noosexitinmainanalyzer Анализатор, проверяющий, что нет os.Exit в main.main()
package noosexitinmainanalyzer

import (
	"go/ast"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"strings"
)

var AnalyzerNoOsExit = &analysis.Analyzer{
	Name: "noosexitinmain",
	Doc:  "forbids calling os.Exit in main.main()",
	Run:  runNoOsExit,
}

func runNoOsExit(pass *analysis.Pass) (interface{}, error) {

	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, f := range pass.Files {

		pos := pass.Fset.Position(f.Pos())
		if strings.Contains(pos.Filename, "/.cache/go-build/") {
			return nil, nil
		}

		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
					if pkgIdent, ok := fun.X.(*ast.Ident); ok {
						if pkgName, ok := pass.TypesInfo.Uses[pkgIdent].(*types.PkgName); ok &&
							pkgName.Imported().Name() == "os" && fun.Sel.Name == "Exit" {
							//вместо call.Lparen можно также юзать fun.Pos()
							pass.Reportf(call.Lparen, "forbids call os.Exit in main.main()")
						}
					}
				}
				return true
			})
		}
	}

	return nil, nil
}
