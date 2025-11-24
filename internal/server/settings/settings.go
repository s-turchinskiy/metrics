// Package settings Загрузка настроек при запуске программы
package settings

import (
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	configutils "github.com/s-turchinskiy/metrics/internal/utils/configutil"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/utils/fileutil"
	rsautil "github.com/s-turchinskiy/metrics/internal/utils/rsautil"
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
	StoreInterval                 int        `env:"STORE_INTERVAL" yaml:"STORE_INTERVAL" lc:"интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)"`
	FileStoragePath               string     `env:"FILE_STORAGE_PATH" yaml:"FILE_STORAGE_PATH" lc:"путь до файла, куда сохраняются текущие значения"`
	Restore                       bool       `env:"RESTORE" yaml:"RESTORE" lc:"определяет загружать или нет ранее сохранённые значения из указанного файла при старте сервера"`
	Database                      database   `env:"DATABASE_DSN" yaml:"DATABASE_DSN" lc:"данные для подключения к базе данных"`
	HashKey                       string     `env:"KEY" yaml:"HASH_KEY" lc:"HashSHA256 ключ для обмена между агентом и сервером"`
	RSAPrivateKeyPath             string     `env:"CRYPTO_KEY" yaml:"CRYPTO_KEY" lc:"Путь к приватному ключу RSA"`
	EnableHTTPS                   bool       `env:"ENABLE_HTTPS" yaml:"ENABLE_HTTPS" lc:"Включить HTTPS"`
	TrustedSubnet                 string     `env:"TRUSTED_SUBNET" yaml:"TRUSTED_SUBNET" lc:"Строковое представление бесклассовой адресации (CIDR)"`
	RSAPrivateKey                 *rsa.PrivateKey
	AsynchronousWritingDataToFile bool
	Store                         Store
	TrustedSubnetTyped            *net.IPNet
	PortGRPC                      string
}

type SecretSettings struct {
	DBPassword string `yaml:"DBPassword" lc:"пароль для подключения к базе данных"`
}

type netAddress struct {
	Host string
	Port int
}

type database struct {
	Host            string
	DBName          string
	Login           string
	Password        string
	FlagDatabaseDSN string
}

var Settings ProgramSettings

func (s ProgramSettings) MarshalLogObject(encoder zapcore.ObjectEncoder) error {

	err := encoder.AddObject("Address", &s.Address)
	if err != nil {
		return err
	}
	encoder.AddInt("StoreInterval", s.StoreInterval)
	encoder.AddString("FileStoragePath", s.FileStoragePath)
	encoder.AddBool("Restore", s.Restore)
	err = encoder.AddObject("Database", &s.Database)
	if err != nil {
		return err
	}

	encoder.AddBool("AsynchronousWritingDataToFile", s.AsynchronousWritingDataToFile)

	switch s.Store {
	case Database:
		{
			encoder.AddString("Store", "Database")
		}
	case File:
		{
			encoder.AddString("Store", "File")
		}
	case Memory:
		{
			encoder.AddString("Store", "Memory")
		}
	default:
		logger.Log.Fatal("unhandled default case")
	}

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
		PortGRPC:        "3032",
	}

	configFilePath := configutils.GetConfigFilePath()
	if configFilePath != "" {
		if err := loadConfigFromJSON(&Settings, configFilePath); err != nil {
			return fmt.Errorf("failed to load configutil from JSON: %w", err)
		}
	}

	err := fileutil.ReadSaveYaml(&Settings, filenameSettings)
	if err != nil {
		return err
	}

	secretSettings := SecretSettings{}
	err = fileutil.ReadSaveYaml(&secretSettings, filenameSecretSettings)

	if err != nil {
		return err
	}
	Settings.Database.Password = secretSettings.DBPassword

	parseFlags()

	err = env.ParseWithOptions(&Settings, env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(database{}): func(incomingData string) (interface{}, error) {
				db := database{}
				err = db.Set(incomingData)
				if err != nil {
					return nil, err
				}

				db.FlagDatabaseDSN = incomingData

				return db, nil
			},
		},
	})
	if err != nil {
		return err
	}

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		err = Settings.Address.Set(envAddr)
		if err != nil {
			return err
		}
	}

	logger.Log.Debug("Received DatabaseDsn from env: ", os.Getenv("DATABASE_DSN"))

	Settings.AsynchronousWritingDataToFile = Settings.StoreInterval != 0

	if os.Getenv("FILE_STORAGE_PATH") != "" || isFlagPassed("f") {
		Settings.Store = File
	}

	if os.Getenv("DATABASE_DSN") != "" || isFlagPassed("d") {
		Settings.Store = Database
	}

	if Settings.RSAPrivateKeyPath != "" {
		Settings.RSAPrivateKey, err = rsautil.ReadPrivateKey(Settings.RSAPrivateKeyPath)
		if err != nil {
			err = fmt.Errorf("path: %s, error: %w", Settings.RSAPrivateKeyPath, err)
			return err
		}
	}

	if Settings.TrustedSubnet != "" {
		_, Settings.TrustedSubnetTyped, err = net.ParseCIDR(Settings.TrustedSubnet)
		if err != nil {
			logger.Log.Infow("Invalid subnet configuration",
				zap.String("subnet", Settings.TrustedSubnet),
				zap.Error(err))
			return err
		}
	}

	logger.LogNoSugar.Info("Settings", zap.Inline(Settings)) //если Sugar, то выводит без имен
	return nil
}

func parseFlags() {

	flag.Var(&Settings.Address, "a", "Net address host:port")
	//flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&Settings.StoreInterval, "i", Settings.StoreInterval, "Интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)")
	flag.StringVar(&Settings.FileStoragePath, "f", Settings.FileStoragePath, "Путь до файла, куда сохраняются текущие значения")
	flag.BoolVar(&Settings.Restore, "r", Settings.Restore, "Определяет загружать или нет ранее сохранённые значения из указанного файла при старте сервера")
	flag.Var(&Settings.Database, "d", "path to database")
	flag.StringVar(&Settings.HashKey, "k", "", "HashSHA256 key")
	flag.StringVar(&Settings.RSAPrivateKeyPath, "crypto-key", "", "Путь до файла с приватным ключом")
	flag.BoolVar(&Settings.EnableHTTPS, "s", Settings.EnableHTTPS, "Определяет включен ли HTTPS")
	flag.StringVar(&Settings.TrustedSubnet, "t", Settings.TrustedSubnet, "Строковое представление бесклассовой адресации (CIDR)")
	flag.Parse()

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

func (d *database) Set(s string) error {

	s = strings.Replace(s, "://", " ", 1)
	s = strings.Replace(s, ":", " ", 1)
	s = strings.Replace(s, "@", " ", 1)
	s = strings.Replace(s, "/", " ", 1)
	s = strings.Replace(s, "?", " ", 1)

	hp := strings.Split(s, " ")
	if len(hp) != 6 {
		//return errors.New("need address in a form host=%s user=%s password=%s dbname=%s sslmode=disable")
		return errors.New("incorrect format database-dsn")
	}

	d.Login = hp[1]
	d.Password = hp[2]
	d.Host = strings.Split(hp[3], ":")[0]
	d.DBName = hp[4]

	return nil
}
