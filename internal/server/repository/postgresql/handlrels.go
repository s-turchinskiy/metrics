package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"time"
)

type PostgreSQL struct {
	db          *sqlx.DB
	tableSchema string
}

func (p *PostgreSQL) Ping(ctx context.Context) ([]byte, error) {

	err := p.db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(p.db.Stats(), "", "   ")

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

	_, err = p.db.ExecContext(ctx, sqlStatement, newValue, time.Now(), metricsName)
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

	_, err = p.db.ExecContext(ctx, sqlStatement, newValue, time.Now(), metricsName)

	return err

}

func (p *PostgreSQL) CountGauges(ctx context.Context) int {

	row := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM postgres.gauges")
	var count int
	_ = row.Scan(&count)

	return count

}

func (p *PostgreSQL) CountCounters(ctx context.Context) int {

	row := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM postgres.counters")
	var count int
	_ = row.Scan(&count)

	return count

}

func (p *PostgreSQL) GetGauge(ctx context.Context, metricsName string) (value float64, isExist bool, err error) {

	row := p.db.QueryRowContext(ctx, fmt.Sprintf("SELECT value FROM %s.gauges WHERE metrics_name = $1", p.tableSchema), metricsName)
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

	row := p.db.QueryRowContext(ctx, "SELECT value FROM postgres.counters WHERE metrics_name = $1", metricsName)
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

	rows, err := p.db.QueryContext(ctx, "SELECT metrics_name, value from postgres.gauges")
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

	rows, err := p.db.QueryContext(ctx, "SELECT metrics_name, value from postgres.counters")
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
