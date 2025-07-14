package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/file"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"strconv"
	"strings"
)

const (
	filenameSettings       = "settings.yaml"
	filenameSecretSettings = "secretSettings.yaml"
)

type ProgramSettings struct {
	Address                       netAddress `yaml:"ADDRESS" lc:"net address host:port to run server"`
	StoreInterval                 int        `yaml:"STORE_INTERVAL" lc:"интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)"`
	FileStoragePath               string     `yaml:"FILE_STORAGE_PATH" lc:"путь до файла, куда сохраняются текущие значения"`
	Restore                       bool       `yaml:"RESTORE" lc:"определяет загружать или нет ранее сохранённые значения из указанного файла при старте сервера"`
	Database                      database   `yaml:"DATABASE_DSN" lc:"данные для подключения к базе данных"`
	asynchronousWritingDataToFile bool
}

type SecretSettings struct {
	DBPassword string `yaml:"DBPassword" lc:"пароль для подключения к базе данных"`
}

type netAddress struct {
	Host string
	Port int
}

type database struct {
	Host     string
	DbName   string
	Login    string
	Password string
}

var settings ProgramSettings

func (s ProgramSettings) MarshalLogObject(encoder zapcore.ObjectEncoder) error {

	err := encoder.AddObject("Address", &s.Address)
	if err != nil {
		return nil
	}
	encoder.AddInt("StoreInterval", s.StoreInterval)
	encoder.AddString("FileStoragePath", s.FileStoragePath)
	encoder.AddBool("Restore", s.Restore)
	err = encoder.AddObject("Database", &s.Database)
	return err
}

func (a *netAddress) MarshalLogObject(encoder zapcore.ObjectEncoder) error {

	encoder.AddString("Host", a.Host)
	encoder.AddInt("Port", a.Port)
	return nil
}

func (d *database) MarshalLogObject(encoder zapcore.ObjectEncoder) error {

	encoder.AddString("Host", d.Host)
	encoder.AddString("DbName", d.DbName)
	encoder.AddString("Login", d.Login)
	encoder.AddString("Password", "********")
	return nil

}

func getSettings() error {

	settings = ProgramSettings{
		Address: netAddress{
			Host: "localhost", Port: 8080},
		StoreInterval:   300,
		FileStoragePath: "store.txt",
		Restore:         true,
		Database:        database{Host: "localhost", DbName: "metrics", Login: "metrics"},
	}

	err := file.ReadSaveYaml(&settings, filenameSettings)
	if err != nil {
		return err
	}

	secretSettings := SecretSettings{}
	err = file.ReadSaveYaml(&secretSettings, filenameSecretSettings)
	if err != nil {
		return err
	}
	settings.Database.Password = secretSettings.DBPassword

	flag.Var(&settings.Address, "a", "Net address host:port")
	//flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&settings.StoreInterval, "i", settings.StoreInterval, "Интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)")
	flag.StringVar(&settings.FileStoragePath, "f", settings.FileStoragePath, "Путь до файла, куда сохраняются текущие значения")
	flag.BoolVar(&settings.Restore, "r", settings.Restore, "Определяет загружать или нет ранее сохранённые значения из указанного файла при старте сервера")
	flag.Var(&settings.Database, "d", "path to database")
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

	if value := os.Getenv("DATABASE_DSN"); value != "" {
		err := settings.Database.Set(value)
		if err != nil {
			return err
		}
	}

	settings.asynchronousWritingDataToFile = settings.StoreInterval != 0

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

func (d *database) String() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		d.Host, d.Login, d.Password, d.DbName)
}

// 'postgres://postgres:postgres@postgres:5432/praktikum?sslmode=disable'
func (d *database) Set(s string) error {

	s = strings.Replace(s, "://", " ", 1)
	s = strings.Replace(s, ":", " ", 1)
	s = strings.Replace(s, "@", " ", 1)
	s = strings.Replace(s, ":", " ", 1)

	hp := strings.Split(s, " ")
	if len(hp) < 4 {
		//return errors.New("need address in a form host=%s user=%s password=%s dbname=%s sslmode=disable")
		return errors.New("Incorrect format database-dsn")
	}
	
	d.Host = hp[0]
	d.Login = hp[1]
	d.Password = hp[2]
	d.DbName = hp[3]

	return nil
}
