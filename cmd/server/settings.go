package main

import (
	"errors"
	"flag"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	yamlcomment "github.com/zijiren233/yaml-comment"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strconv"
	"strings"
)

type programSettings struct {
	Address         netAddress `yaml:"ADDRESS" lc:"net address host:port to run server"`
	StoreInterval   int        `yaml:"STORE_INTERVAL" lc:"интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)"`
	FileStoragePath string     `yaml:"FILE_STORAGE_PATH" lc:"путь до файла, куда сохраняются текущие значения"`
	Restore         bool       `yaml:"RESTORE" lc:"определяет загружать или нет ранее сохранённые значения из указанного файла при старте сервера"`
}

type netAddress struct {
	Host string
	Port int
}

var settings programSettings

func (s programSettings) MarshalLogObject(encoder zapcore.ObjectEncoder) error {

	err := encoder.AddObject("Address", &s.Address)
	encoder.AddInt("StoreInterval", s.StoreInterval)
	encoder.AddString("FileStoragePath", s.FileStoragePath)
	encoder.AddBool("Restore", s.Restore)
	return err
}

func (a *netAddress) MarshalLogObject(encoder zapcore.ObjectEncoder) error {

	encoder.AddString("Host", a.Host)
	encoder.AddInt("Port", a.Port)
	return nil
}

func (s programSettings) SaveYaml(filename string) error {

	settings := programSettings{Address: netAddress{
		Host: "localhost", Port: 8080},
		StoreInterval:   300,
		FileStoragePath: "store.txt",
		Restore:         true,
	}

	yamlFile, err := yamlcomment.Marshal(&settings)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.WriteString(f, string(yamlFile))
	if err != nil {
		return err
	}

	return nil
}

func (s programSettings) ReadYaml() error {

	filename := "settings.yaml"

	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := s.SaveYaml(filename)
			if err != nil {
				return err
			}
		}
	}

	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &settings)
	if err != nil {
		return err
	}

	return nil

}

func getSettings() error {

	settings = programSettings{}
	err := settings.ReadYaml()
	if err != nil {
		return err
	}

	flag.Var(&settings.Address, "a", "Net address host:port")
	//flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&settings.StoreInterval, "i", settings.StoreInterval, "Интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)")
	flag.StringVar(&settings.FileStoragePath, "f", settings.FileStoragePath, "Путь до файла, куда сохраняются текущие значения")
	flag.BoolVar(&settings.Restore, "r", settings.Restore, "Определяет загружать или нет ранее сохранённые значения из указанного файла при старте сервера")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		err := settings.Address.Set(envAddr)
		if err != nil {
			return err
		}
	}

	if value := os.Getenv("STORE_INTERVAL"); value != "" {
		storeInterval, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		settings.StoreInterval = storeInterval
	}

	if value := os.Getenv("FILE_STORAGE_PATH"); value != "" {
		settings.FileStoragePath = value
	}

	if value := os.Getenv("RESTORE"); value != "" {
		restore, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		settings.Restore = restore
	}

	logger.LogNoSugar.Info("Settings", zap.Inline(settings)) //если Sugar, то выводит без имен
	return nil
}

func (a *netAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *netAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	if len(hp) != 2 {
		return errors.New("need address in a form host:port")
	}
	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return err
	}
	a.Host = hp[0]
	a.Port = port
	return nil
}
