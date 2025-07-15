package postgresql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
)

type PostgreSQL struct {
	DB *sql.DB
}

func (p PostgreSQL) Ping() ([]byte, error) {

	err := p.DB.Ping()
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(p.DB.Stats(), "", "   ")

}

func (p PostgreSQL) UpdateGauge(metricsName string, newValue float64) error {
	//TODO implement me
	panic("implement me")
}

func (p PostgreSQL) UpdateCounter(metricsName string, newValue int64) error {
	//TODO implement me
	panic("implement me")
}

func (p PostgreSQL) CountGauges() int {
	//TODO implement me
	panic("implement me")
}

func (p PostgreSQL) CountCounters() int {
	//TODO implement me
	panic("implement me")
}

func (p PostgreSQL) GetGauge(metricsName string) (float64, bool, error) {
	//TODO implement me
	panic("implement me")
}

func (p PostgreSQL) GetCounter(metricsName string) (int64, bool, error) {
	//TODO implement me
	panic("implement me")
}

func (p PostgreSQL) GetAllGauges() (map[string]float64, error) {
	//TODO implement me
	panic("implement me")
}

func (p PostgreSQL) GetAllCounters() (map[string]int64, error) {
	//TODO implement me
	panic("implement me")
}

func (p PostgreSQL) ReloadAllGauges(m map[string]float64) error {

	return nil

}

func (p PostgreSQL) ReloadAllCounters(m map[string]int64) error {

	return nil
}

func ConnectToStore() (*sql.DB, error) {

	dbSettings := settings.Settings.Database
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbSettings.Host, dbSettings.Login, dbSettings.Password, dbSettings.DBName)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		return nil, err
	}

	return db, nil

}
