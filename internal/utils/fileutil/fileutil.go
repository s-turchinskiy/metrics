// Package fileutil Общие процедуры взаимодействия с файлами
package fileutil

import (
	"errors"
	"io"
	"os"

	yamlcomment "github.com/zijiren233/yaml-comment"
	"gopkg.in/yaml.v3"
)

func SaveYaml(data *any, filename string) error {
	yamlFile, err := yamlcomment.Marshal(*data)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Writer.Write(f, yamlFile)
	if err != nil {
		return err
	}

	return nil
}

func ReadSaveYaml(data any, filename string) error {

	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := SaveYaml(&data, filename)
			if err != nil {
				return err
			}
		}
	}

	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, data)
	if err != nil {
		return err
	}

	return nil

}
