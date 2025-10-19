// Package staticcheck staticcheck анализаторы кода
package staticcheck

import (
	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/staticcheck"
	"strings"
)

// GetStaticCheckAnalyzers Функция возвращает - всех анализаторов класса SA пакета staticcheck.io;
// не менее одного анализатора остальных классов пакета staticcheck.io;
func GetStaticCheckAnalyzers() (analyzers []*analysis.Analyzer) {

	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			analyzers = append(analyzers, v.Analyzer)
			continue
		}

		analyzers = append(analyzers, v.Analyzer)
	}

	return analyzers
}
