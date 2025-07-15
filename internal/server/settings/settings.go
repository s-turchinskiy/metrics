package settings

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

type Store int

const (
	Memory Store = iota
	File
	Database
)

type ProgramSettings struct {
	Address                       netAddress `yaml:"ADDRESS" lc:"net address host:port to run server"`
	StoreInterval                 int        `yaml:"STORE_INTERVAL" lc:"интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)"`
	FileStoragePath               string     `yaml:"FILE_STORAGE_PATH" lc:"путь до файла, куда сохраняются текущие значения"`
	Restore                       bool       `yaml:"RESTORE" lc:"определяет загружать или нет ранее сохранённые значения из указанного файла при старте сервера"`
	Database                      database   `yaml:"DATABASE_DSN" lc:"данные для подключения к базе данных"`
	AsynchronousWritingDataToFile bool
	store                         Store
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
	DBName   string
	Login    string
	Password string
}

var Settings ProgramSettings

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
	encoder.AddString("DbName", d.DBName)
	encoder.AddString("Login", d.Login)
	encoder.AddString("Password", "********")
	return nil

}

func GetSettings() error {

	Settings = ProgramSettings{
		Address: netAddress{
			Host: "localhost", Port: 8080},
		StoreInterval:   300,
		FileStoragePath: "store.txt",
		Restore:         true,
		Database:        database{Host: "localhost", DBName: "metrics", Login: "metrics"},
	}

	err := file.ReadSaveYaml(&Settings, filenameSettings)
	if err != nil {
		return err
	}

	secretSettings := SecretSettings{}
	err = file.ReadSaveYaml(&secretSettings, filenameSecretSettings)

	if err != nil {
		return err
	}
	Settings.Database.Password = secretSettings.DBPassword

	flag.Var(&Settings.Address, "a", "Net address host:port")
	//flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&Settings.StoreInterval, "i", Settings.StoreInterval, "Интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)")
	flag.StringVar(&Settings.FileStoragePath, "f", Settings.FileStoragePath, "Путь до файла, куда сохраняются текущие значения")
	flag.BoolVar(&Settings.Restore, "r", Settings.Restore, "Определяет загружать или нет ранее сохранённые значения из указанного файла при старте сервера")
	flag.Var(&Settings.Database, "d", "path to database")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		err := Settings.Address.Set(envAddr)
		if err != nil {
			return err
		}
	}

	if value := os.Getenv("STORE_INTERVAL"); value != "" {
		storeInterval, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		Settings.StoreInterval = storeInterval
	}

	FileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	if FileStoragePath != "" {
		Settings.FileStoragePath = FileStoragePath
	}

	if value := os.Getenv("RESTORE"); value != "" {
		restore, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		Settings.Restore = restore
	}

	DatabaseDsn := os.Getenv("DATABASE_DSN")
	if DatabaseDsn != "" {
		err := Settings.Database.Set(DatabaseDsn)
		if err != nil {
			return err
		}
	}

	Settings.AsynchronousWritingDataToFile = Settings.StoreInterval != 0

	if FileStoragePath != "" || isFlagPassed("f") {
		Settings.store = File
	}

	if DatabaseDsn != "" || isFlagPassed("d") {
		Settings.store = Database
	}

	logger.LogNoSugar.Info("Settings", zap.Inline(Settings)) //если Sugar, то выводит без имен
	return nil
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
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
		d.Host, d.Login, d.Password, d.DBName)
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
		return errors.New("incorrect format database-dsn")
	}

	d.Host = hp[0]
	d.Login = hp[1]
	d.Password = hp[2]
	d.DBName = hp[3]

	return nil
}
