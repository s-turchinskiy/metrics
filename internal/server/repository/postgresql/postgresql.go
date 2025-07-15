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

	row := p.DB.QueryRow("SELECT COUNT(*) FROM gauges")
	var count int
	_ = row.Scan(&count)

	return count

}

func (p PostgreSQL) CountCounters() int {

	row := p.DB.QueryRow("SELECT COUNT(*) FROM counters")
	var count int
	_ = row.Scan(&count)

	return count

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

func ConnectToDatabase() (*sql.DB, error) {

	dbSettings := settings.Settings.Database
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbSettings.Host, dbSettings.Login, dbSettings.Password, dbSettings.DBName)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		return nil, err
	}

	err = CreateTableGauges(db)
	if err != nil {
		return nil, err
	}

	err = CreateTableCounters(db)
	if err != nil {
		return nil, err
	}

	return db, nil

}

func CreateTableGauges(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS gauges (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value DOUBLE PRECISION)`)
	return err
}

func CreateTableCounters(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS counters (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value INT)`)
	return err
}
