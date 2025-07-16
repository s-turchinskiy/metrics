package postgresql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"log"
	"strings"
	"time"
)

type PostgreSQL struct {
	DB          *sql.DB
	tableSchema string
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
		sqlStatement = `UPDATE postgres.gauges SET value = $1, date = $2 WHERE metrics_name = $3`
	} else {
		sqlStatement = `INSERT INTO postgres.gauges (value, date, metrics_name) VALUES ($1, $2, $3)`
	}

	_, err = p.DB.Exec(sqlStatement, newValue, time.Now(), metricsName)
	if err != nil {
		err = fmt.Errorf("PostgreSQL.UpdateGauge error in p.DB.Exec, %w", err)
	}
	return err

}

func (p PostgreSQL) UpdateCounter(metricsName string, newValue int64) error {

	err := p.loggingData(
		"view new tables",
		"SELECT table_schema || '.' || table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')",
		"")
	if err != nil {
		return err
	}

	err = p.withLoggingCreatingTable("counters", p.createTableCounters)
	if err != nil {
		return err
	}

	var sqlStatement string
	_, exist, err := p.GetCounter(metricsName)
	if err != nil {
		return err
	}

	if exist {
		sqlStatement = `UPDATE postgres.counters SET value = $1, date = $2 WHERE metrics_name = $3`
	} else {
		sqlStatement = `INSERT INTO postgres.counters (value, date, metrics_name) VALUES ($1, $2, $3)`
	}

	logger.Log.Debugw("PostgreSQL.UpdateCounter try",
		"metricsName", metricsName,
		"value", newValue,
	)

	_, err = p.DB.Exec(sqlStatement, newValue, time.Now(), metricsName)

	return err

}

func (p PostgreSQL) CountGauges() int {

	row := p.DB.QueryRow("SELECT COUNT(*) FROM postgres.gauges")
	var count int
	_ = row.Scan(&count)

	return count

}

func (p PostgreSQL) CountCounters() int {

	row := p.DB.QueryRow("SELECT COUNT(*) FROM postgres.counters")
	var count int
	_ = row.Scan(&count)

	return count

}

func (p PostgreSQL) GetGauge(metricsName string) (value float64, isExist bool, err error) {

	row := p.DB.QueryRow(fmt.Sprintf("SELECT value FROM %s.gauges WHERE metrics_name = $1", p.tableSchema), metricsName)
	err = row.Scan(&value)

	isExist = true

	if err != nil {
		isExist = false
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		} else {
			err = fmt.Errorf("PostgreSQL.GetGauge error in p.DB.QueryRow, %w", err)
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

	row := p.DB.QueryRow("SELECT value FROM postgres.counters WHERE metrics_name = $1", metricsName)
	err = row.Scan(&value)

	isExist = true

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			isExist = false
			err = nil
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

	rows, err := p.DB.Query("SELECT metrics_name, value from postgres.gauges")
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

	rows, err := p.DB.Query("SELECT metrics_name, value from postgres.counters")
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

func InitializePostgreSQL() (*PostgreSQL, error) {

	driverConfig := stdlib.DriverConfig{
		ConnConfig: pgx.ConnConfig{
			PreferSimpleProtocol: true,
		},
	}
	stdlib.RegisterDriverConfig(&driverConfig)

	conn, err := sql.Open("pgx", driverConfig.ConnectionString(settings.Settings.Database.FlagDatabaseDSN))

	/*dbSettings := settings.Settings.Database
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbSettings.Host, dbSettings.Login, dbSettings.Password, dbSettings.DBName)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		return nil, err
	}*/

	p := &PostgreSQL{DB: conn}
	p.tableSchema = "postgres"

	p.runCommand("DROP TABLE postgres.gauges IF EXIST")
	p.runCommand("DROP TABLE postgres.counters IF EXIST")

	err = p.loggingData(
		"schemas",
		"SELECT schema_name FROM information_schema.schemata WHERE catalog_name = $1;",
		settings.Settings.Database.DBName)
	if err != nil {
		return nil, err
	}

	err = p.loggingData(
		"tables",
		"SELECT table_name FROM information_schema.tables WHERE table_schema = $1",
		p.tableSchema)
	if err != nil {
		return nil, err
	}

	err = p.loggingData(
		"view new tables",
		"SELECT table_schema || '.' || table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')",
		"")
	if err != nil {
		return nil, err
	}

	err = p.createSchema(p.tableSchema)
	if err != nil {
		return nil, err
	}

	err = p.withLoggingCreatingTable("gauges", p.createTableGauges)
	if err != nil {
		return nil, err
	}

	err = p.withLoggingCreatingTable("counters", p.createTableCounters)
	if err != nil {
		return nil, err
	}

	return p, nil

}

func (p PostgreSQL) createSchema(tableSchema string) error {
	_, err := p.DB.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, tableSchema))
	return err
}

func (p PostgreSQL) createTableGauges() error {
	_, err := p.DB.Exec(fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s.gauges (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value DOUBLE PRECISION,
    date TIMESTAMPTZ)`,
		p.tableSchema))
	return err
}

func (p PostgreSQL) createTableCounters() error {
	_, err := p.DB.Exec(fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s.counters (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value INT,
    date TIMESTAMPTZ)`,
		p.tableSchema))
	return err
}

func (p PostgreSQL) tableExist(tableName string) bool {
	row := p.DB.QueryRow(fmt.Sprintf(`select exists (select *
               from information_schema.tables
               where table_name = '%s' 
                 and table_schema = '%s') as table_exists;`, tableName, p.tableSchema))

	var isExist bool
	err := row.Scan(&isExist)
	if err != nil {
		log.Fatal(err)
	}

	return isExist
}

func (p PostgreSQL) withLoggingCreatingTable(tableName string, createTable func() error) error {

	existBefore := p.tableExist(tableName)
	err := createTable()
	if err != nil {
		return err
	}
	existAfter := p.tableExist(tableName)
	if !existBefore && existAfter {
		logger.Log.Info(strings.ToUpper("created table "), p.tableSchema+"."+tableName)
	}

	return nil
}

func (p PostgreSQL) loggingData(title, query, parameter string) error {

	var data []string

	var rows *sql.Rows
	var err error

	if parameter == "" {
		rows, err = p.DB.Query(query)

	} else {
		rows, err = p.DB.Query(query, parameter)
	}

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		err = rows.Scan(&s)
		if err != nil {
			return err
		}

		data = append(data, s)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	logger.Log.Debugw(title, "values", strings.Join(data, ","))
	return nil

}

func (p PostgreSQL) runCommand(command string) error {
	_, err := p.DB.Exec(command)
	return err
}
