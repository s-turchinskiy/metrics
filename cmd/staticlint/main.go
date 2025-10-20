// Package staticlint Статический анализ кода
package main

//cd /home/stanislav/go/metrics && go build ./cmd/staticlint

import (
	noosexitinmainanalyzer "github.com/s-turchinskiy/metrics/internal/staticlint/noosexitinmain"
	"github.com/s-turchinskiy/metrics/internal/staticlint/standard"
	"github.com/s-turchinskiy/metrics/internal/staticlint/staticcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.yandex/linters/passes/copyproto"
	"golang.yandex/linters/passes/ctxcheck"
	"golang.yandex/linters/passes/deepequalproto"
	"golang.yandex/linters/passes/goodpackagenames"
	"golang.yandex/linters/passes/nonakedreturn"
	"golang.yandex/linters/passes/remindercheck"
	"golang.yandex/linters/passes/returnstruct"
	"golang.yandex/linters/passes/structtagcase"
	"honnef.co/go/tools/stylecheck"
)

func main() {

	var analyzers []*analysis.Analyzer

	analyzers = append(analyzers, standard.GetStandardAnalyzers()...)
	analyzers = append(analyzers, staticcheck.GetStaticCheckAnalyzers()...)

	for _, a := range stylecheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	analyzers = append(analyzers,
		noosexitinmainanalyzer.AnalyzerNoOsExit,
		copyproto.Analyzer,
		ctxcheck.CtxArgAnalyzer,
		ctxcheck.CtxSaveAnalyzer,
		deepequalproto.Analyzer,
		goodpackagenames.Analyzer,
		nonakedreturn.Analyzer,
		remindercheck.Analyzer(),
		returnstruct.Analyzer,
		structtagcase.Analyzer)

	multichecker.Main(
		analyzers...,
	)

	//https://github.com/yandex/go-linters тут используют unitchecker, а не multichecker
	/*unitchecker.Main(
		analyzers...,
	)*/

}
