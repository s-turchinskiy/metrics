package postgresql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"time"
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

	var sqlStatement string
	_, exist, err := p.GetGauge(metricsName)
	if err != nil {
		return err
	}

	if exist {
		sqlStatement = `UPDATE gauges SET value = $1, date = $2 WHERE metrics_name = $3`
	} else {
		sqlStatement = `INSERT INTO gauges (value, date, metrics_name) VALUES ($1, $2, $3)`
	}

	_, err = p.DB.Exec(sqlStatement, newValue, time.Now(), metricsName)
	return err

}

func (p PostgreSQL) UpdateCounter(metricsName string, newValue int64) error {

	var sqlStatement string
	_, exist, err := p.GetCounter(metricsName)
	if err != nil {
		return err
	}

	if exist {
		sqlStatement = `UPDATE counters SET value = $1, date = $2 WHERE metrics_name = $3`
	} else {
		sqlStatement = `INSERT INTO counters (value, date, metrics_name) VALUES ($1, $2, $3)`
	}

	_, err = p.DB.Exec(sqlStatement, newValue, time.Now(), metricsName)
	return err

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

func (p PostgreSQL) GetGauge(metricsName string) (value float64, isExist bool, err error) {

	row := p.DB.QueryRow("SELECT value FROM gauges WHERE metrics_name = $1", metricsName)
	err = row.Scan(&value)

	isExist = true

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			isExist = false
		}
	}

	logger.Log.Debugw("PostgreSQL.GetGauge",
		"metricsName", metricsName,
		"isExist", isExist,
		"value", value,
	)

	return value, isExist, err

}

func (p PostgreSQL) GetCounter(metricsName string) (value int64, isExist bool, err error) {

	row := p.DB.QueryRow("SELECT value FROM counters WHERE metrics_name = $1", metricsName)
	err = row.Scan(&value)

	isExist = true

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			isExist = false
		}
	}

	logger.Log.Debugw("PostgreSQL.GetCounter",
		"metricsName", metricsName,
		"isExist", isExist,
		"value", value,
	)

	return value, isExist, err

}

func (p PostgreSQL) GetAllGauges() (map[string]float64, error) {

	result := make(map[string]float64)

	rows, err := p.DB.Query("SELECT metrics_name, value from gauges")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var metricsName string
		var value float64
		err = rows.Scan(&metricsName, &value)
		if err != nil {
			return nil, err
		}

		result[metricsName] = value
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil

}

func (p PostgreSQL) GetAllCounters() (map[string]int64, error) {

	result := make(map[string]int64)

	rows, err := p.DB.Query("SELECT metrics_name, value from counters")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var metricsName string
		var value int64
		err = rows.Scan(&metricsName, &value)
		if err != nil {
			return nil, err
		}

		result[metricsName] = value
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil

}

func (p PostgreSQL) ReloadAllGauges(data map[string]float64) error {

	for metricsName, newValue := range data {
		err := p.UpdateGauge(metricsName, newValue)
		if err != nil {
			return err
		}
	}

	return nil

}

func (p PostgreSQL) ReloadAllCounters(data map[string]int64) error {

	for metricsName, newValue := range data {
		err := p.UpdateCounter(metricsName, newValue)
		if err != nil {
			return err
		}
	}

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
    value DOUBLE PRECISION,
    date TIMESTAMPTZ)`)
	return err
}

func CreateTableCounters(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS counters (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value INT,
    date TIMESTAMPTZ)`)
	return err
}
