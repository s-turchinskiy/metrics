// Package codegeneration Кодогенерация
package codegeneration

import (
	"bytes"
	"go/format"
	"log"
	"os"
	"text/template"
)

func Codogeneration(data any, templateStr, filename string) {

	var buf bytes.Buffer
	tmpl := template.Must(template.New("data").Parse(templateStr))
	err := tmpl.Execute(&buf, data)
	if err != nil {
		log.Fatal(err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(filename, formatted, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
