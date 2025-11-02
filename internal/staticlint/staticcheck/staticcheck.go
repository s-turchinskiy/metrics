// Package staticcheck staticcheck анализаторы кода
package staticcheck

import (
	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
	"honnef.co/go/tools/unused"
)

// GetStaticCheckAnalyzers Функция возвращает - всех анализаторов класса SA пакета staticcheck.io;
// не менее одного анализатора остальных классов пакета staticcheck.io;
func GetStaticCheckAnalyzers() (analyzers []*analysis.Analyzer) {

	//SA
	for _, v := range staticcheck.Analyzers {
		analyzers = append(analyzers, v.Analyzer)
	}

	//S
	for _, v := range simple.Analyzers {
		analyzers = append(analyzers, v.Analyzer)
	}

	//ST
	for _, v := range stylecheck.Analyzers {
		analyzers = append(analyzers, v.Analyzer)
	}

	//QF
	for _, v := range quickfix.Analyzers {
		analyzers = append(analyzers, v.Analyzer)
	}

	//U1000
	analyzers = append(analyzers, unused.Analyzer.Analyzer)

	return analyzers
}
