// Package errutil Общие процедуры обработки ошибок
package errutil

import (
	"fmt"
	"runtime"
)

func WrapError(err error) error {

	_, filename, line, _ := runtime.Caller(1)
	return fmt.Errorf("[error] %s %d: %w", filename, line, err)
}
