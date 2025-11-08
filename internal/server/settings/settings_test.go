package settings

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_database_Set(t *testing.T) {

	type args struct {
		s string
	}
	tests := []struct {
		name          string
		args          args
		expectedValue *database
		wantErr       bool
	}{
		{
			name:          "Разбор строки подключения к базе данных в структуру с портом. Успешно",
			args:          args{s: "postgres://login:password@host:5432/praktikum?sslmode=disable"},
			expectedValue: &database{Host: "host", DBName: "praktikum", Login: "login", Password: "password", FlagDatabaseDSN: ""},
			wantErr:       false,
		},
		{
			name:          "Разбор строки подключения к базе данных в структуру без порта. Успешно",
			args:          args{s: "postgres://login:password@host/praktikum?sslmode=disable"},
			expectedValue: &database{Host: "host", DBName: "praktikum", Login: "login", Password: "password", FlagDatabaseDSN: ""},
			wantErr:       false,
		},
		{
			name:          "Разбор строки подключения к базе данных в структуру. Ошибка",
			args:          args{s: "postgres://postgres:postgres@postgres"},
			expectedValue: &database{},
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &database{}

			err := d.Set(tt.args.s)
			assert.EqualValues(t, tt.expectedValue, d)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_netAddress_Set(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name          string
		args          args
		expectedValue *netAddress
		wantErr       bool
	}{
		{
			name:          "Разбор строки запуска вебсервиса в структуру. Успешно",
			args:          args{s: "localhost:8081"},
			expectedValue: &netAddress{Host: "localhost", Port: 8081},
			wantErr:       false,
		},
		{
			name:          "Разбор строки запуска вебсервиса в структуру. Ошибочная строка 1",
			args:          args{s: "localhost8081"},
			expectedValue: &netAddress{},
			wantErr:       true,
		},
		{
			name:          "Разбор строки запуска вебсервиса в структуру. Ошибочная строка 2",
			args:          args{s: "localhost:8081ж"},
			expectedValue: &netAddress{},
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &netAddress{}

			err := d.Set(tt.args.s)
			assert.EqualValues(t, tt.expectedValue, d)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
