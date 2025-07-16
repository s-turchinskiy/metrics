package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"log"
	"strings"
	"time"
)

const MaxConns = 10
const MinConns = 2
const MaxConnLifetime = time.Hour
const MaxConnIdleTime = time.Minute * 30

type PostgreSQL struct {
	db          *pgxpool.Pool
	tableSchema string
}

func (p *PostgreSQL) Ping(ctx context.Context) ([]byte, error) {

	err := p.db.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(p.db.Stat(), "", "   ")

}

func (p *PostgreSQL) UpdateGauge(ctx context.Context, metricsName string, newValue float64) error {

	var sqlStatement string
	_, exist, err := p.GetGauge(ctx, metricsName)
	if err != nil {
		return err
	}

	if exist {
		sqlStatement = `UPDATE postgres.gauges SET value = $1, date = $2 WHERE metrics_name = $3`
	} else {
		sqlStatement = `INSERT INTO postgres.gauges (value, date, metrics_name) VALUES ($1, $2, $3)`
	}

	_, err = p.db.Exec(ctx, sqlStatement, newValue, time.Now(), metricsName)
	if err != nil {
		err = fmt.Errorf("PostgreSQL.UpdateGauge error in p.DB.Exec, %w", err)
	}
	return err

}

func (p *PostgreSQL) UpdateCounter(ctx context.Context, metricsName string, newValue int64) error {

	err := p.loggingData(
		ctx,
		"view new tables",
		"SELECT table_schema || '.' || table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')",
		"")
	if err != nil {
		return err
	}

	err = p.withLoggingCreatingTable(ctx, "counters", p.createTableCounters)
	if err != nil {
		return err
	}

	var sqlStatement string
	_, exist, err := p.GetCounter(ctx, metricsName)
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

	_, err = p.db.Exec(ctx, sqlStatement, newValue, time.Now(), metricsName)

	return err

}

func (p *PostgreSQL) CountGauges(ctx context.Context) int {

	row := p.db.QueryRow(ctx, "SELECT COUNT(*) FROM postgres.gauges")
	var count int
	_ = row.Scan(&count)

	return count

}

func (p *PostgreSQL) CountCounters(ctx context.Context) int {

	row := p.db.QueryRow(ctx, "SELECT COUNT(*) FROM postgres.counters")
	var count int
	_ = row.Scan(&count)

	return count

}

func (p *PostgreSQL) GetGauge(ctx context.Context, metricsName string) (value float64, isExist bool, err error) {

	row := p.db.QueryRow(ctx, fmt.Sprintf("SELECT value FROM %s.gauges WHERE metrics_name = $1", p.tableSchema), metricsName)
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

func (p *PostgreSQL) GetCounter(ctx context.Context, metricsName string) (value int64, isExist bool, err error) {

	row := p.db.QueryRow(ctx, "SELECT value FROM postgres.counters WHERE metrics_name = $1", metricsName)
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

func (p *PostgreSQL) GetAllGauges(ctx context.Context) (map[string]float64, error) {

	result := make(map[string]float64)

	rows, err := p.db.Query(ctx, "SELECT metrics_name, value from postgres.gauges")
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

func (p *PostgreSQL) GetAllCounters(ctx context.Context) (map[string]int64, error) {

	result := make(map[string]int64)

	rows, err := p.db.Query(ctx, "SELECT metrics_name, value from postgres.counters")
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

func (p *PostgreSQL) ReloadAllGauges(ctx context.Context, data map[string]float64) error {

	for metricsName, newValue := range data {
		err := p.UpdateGauge(ctx, metricsName, newValue)
		if err != nil {
			return err
		}
	}

	return nil

}

func (p *PostgreSQL) ReloadAllCounters(ctx context.Context, data map[string]int64) error {

	for metricsName, newValue := range data {
		err := p.UpdateCounter(ctx, metricsName, newValue)
		if err != nil {
			return err
		}
	}

	return nil
}

func InitializePostgreSQL(ctx context.Context) (service.Repository, error) {

	/*db, err := sql.Open("pgx", settings.Settings.Database.String())
	if err != nil {
		return nil, err
	}

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}*/

	config, err := pgxpool.ParseConfig(settings.Settings.Database.String())
	if err != nil {
		return nil, err
	}

	config.MaxConns = MaxConns
	config.MinConns = MinConns
	config.MaxConnLifetime = MaxConnLifetime
	config.MaxConnIdleTime = MaxConnIdleTime

	pool, _ := pgxpool.NewWithConfig(ctx, config)

	p := &PostgreSQL{db: pool}
	p.tableSchema = "postgres"

	p.runCommand(ctx, "DROP TABLE postgres.gauges IF EXIST")
	//p.runCommand("DROP TABLE postgres.counters IF EXIST")

	err = p.loggingData(ctx,
		"schemas",
		"SELECT schema_name FROM information_schema.schemata WHERE catalog_name = $1;",
		settings.Settings.Database.DBName)
	if err != nil {
		return nil, err
	}

	err = p.loggingData(
		ctx,
		"tables",
		"SELECT table_name FROM information_schema.tables WHERE table_schema = $1",
		p.tableSchema)
	if err != nil {
		return nil, err
	}

	err = p.loggingData(
		ctx,
		"view new tables",
		"SELECT table_schema || '.' || table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')",
		"")
	if err != nil {
		return nil, err
	}

	err = p.createSchema(ctx, p.tableSchema)
	if err != nil {
		return nil, err
	}

	err = p.withLoggingCreatingTable(ctx, "gauges", p.createTableGauges)
	if err != nil {
		return nil, err
	}

	err = p.withLoggingCreatingTable(ctx, "counters", p.createTableCounters)
	if err != nil {
		return nil, err
	}

	return p, nil

}

func (p *PostgreSQL) createSchema(ctx context.Context, tableSchema string) error {
	_, err := p.db.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, tableSchema))
	return err
}

func (p *PostgreSQL) createTableGauges(ctx context.Context) error {
	_, err := p.db.Exec(ctx, fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s.gauges (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value DOUBLE PRECISION,
    date TIMESTAMPTZ)`,
		p.tableSchema))
	return err
}

func (p *PostgreSQL) createTableCounters(ctx context.Context) error {
	_, err := p.db.Exec(ctx, fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s.counters (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value INT,
    date TIMESTAMPTZ)`,
		p.tableSchema))
	return err
}

func (p *PostgreSQL) tableExist(ctx context.Context, tableName string) bool {
	row := p.db.QueryRow(ctx, fmt.Sprintf(`select exists (select *
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

func (p *PostgreSQL) withLoggingCreatingTable(ctx context.Context, tableName string, createTable func(context.Context) error) error {

	existBefore := p.tableExist(ctx, tableName)
	err := createTable(ctx)
	if err != nil {
		return err
	}
	existAfter := p.tableExist(ctx, tableName)
	if !existBefore && existAfter {
		logger.Log.Info(strings.ToUpper("created table "), p.tableSchema+"."+tableName)
	}

	return nil
}

func (p *PostgreSQL) loggingData(ctx context.Context, title, query, parameter string) error {

	var data []string

	var rows pgx.Rows
	var err error

	if parameter == "" {
		rows, err = p.db.Query(ctx, query)

	} else {
		rows, err = p.db.Query(ctx, query, parameter)
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

func (p *PostgreSQL) runCommand(ctx context.Context, command string) error {
	_, err := p.db.Exec(ctx, command)
	return err
}
