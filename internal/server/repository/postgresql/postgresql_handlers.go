package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/s-turchinskiy/metrics/internal"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
	"time"
)

const (
	QueryInsertCounters = `INSERT INTO postgres.counters (value, date, metrics_name) VALUES ($1, $2, $3)`
	QueryInsertGauges   = `INSERT INTO postgres.gauges (value, date, metrics_name) VALUES ($1, $2, $3)`
)

type keyTx string

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
		sqlStatement = QueryInsertGauges
	}

	_, err = p.db.ExecContext(ctx, sqlStatement, newValue, time.Now(), metricsName)
	if err != nil {
		err = fmt.Errorf("PostgreSQL.UpdateGauge error in p.DB.Exec, %w", err)
	}
	return err

}

func (p *PostgreSQL) UpdateCounter(ctx context.Context, metricsName string, newValue int64) error {

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	ctx2 := context.WithValue(ctx, keyTx("tx"), tx)

	_, exist, err := p.GetCounter(ctx2, metricsName)
	if err != nil {
		return err
	}

	var sqlStatement string
	if exist {
		sqlStatement = `UPDATE postgres.counters SET value = $1, date = $2 WHERE metrics_name = $3`
	} else {
		sqlStatement = QueryInsertCounters
	}

	logger.Log.Debugw("PostgreSQL.UpdateCounter try",
		"metricsName", metricsName,
		"value", newValue,
	)

	_, err = tx.ExecContext(ctx, sqlStatement, newValue, time.Now(), metricsName)
	if err != nil {
		return internal.WrapError(err)
	}

	err = tx.Commit()
	if err != nil {
		return internal.WrapError(err)
	}

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

	query := "SELECT value FROM postgres.counters WHERE metrics_name = $1"

	var row *sql.Row

	tx := ctx.Value(keyTx("tx"))

	if tx != nil {
		row = tx.(*sql.Tx).QueryRowContext(ctx, query, metricsName)
	} else {
		row = p.db.QueryRowContext(ctx, query, metricsName)
	}
	err = row.Scan(&value)

	isExist = true

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			isExist = false
			err = nil
		} else {
			return 0, false, internal.WrapError(err)
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

	var metrics []models.DatabaseTableGauges
	err := p.db.SelectContext(ctx, &metrics, "SELECT metrics_name, value from postgres.gauges")

	if err != nil {
		return nil, internal.WrapError(err)
	}
	for _, data := range metrics {
		result[data.MetricsName] = data.Value
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

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "TRUNCATE metrics.gauges")
	if err != nil {
		return internal.WrapError(err)
	}

	portionData := make(map[string]float64, 10)

	for metricsName, newValue := range data {
		portionData[metricsName] = newValue

		if len(portionData) == 10 {

			for pMetricsname, pNewvalue := range portionData {
				_, err = tx.ExecContext(ctx, QueryInsertGauges, pNewvalue, time.Now(), pMetricsname)
				if err != nil {
					return internal.WrapError(err)

				}
			}

			for k := range portionData {
				delete(portionData, k)
			}
		}
	}

	for pMetricsname, pNewvalue := range portionData {
		_, err = tx.ExecContext(ctx, QueryInsertGauges, pNewvalue, time.Now(), pMetricsname)
		if err != nil {
			return internal.WrapError(err)

		}
	}

	err = tx.Commit()
	if err != nil {
		return internal.WrapError(err)
	}

	return nil

}

func (p *PostgreSQL) ReloadAllCounters(ctx context.Context, data map[string]int64) error {

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "TRUNCATE metrics.counters")
	if err != nil {
		return internal.WrapError(err)
	}

	portionData := make(map[string]int64, 10)

	for metricsName, newValue := range data {
		portionData[metricsName] = newValue

		if len(portionData) == 10 {

			for pMetricsname, pNewvalue := range portionData {
				_, err = tx.ExecContext(ctx, QueryInsertCounters, pNewvalue, time.Now(), pMetricsname)
				if err != nil {
					return internal.WrapError(err)

				}
			}

			for k := range portionData {
				delete(portionData, k)
			}
		}
	}

	for pMetricsname, pNewvalue := range portionData {
		_, err = tx.ExecContext(ctx, QueryInsertCounters, pNewvalue, time.Now(), pMetricsname)
		if err != nil {
			return internal.WrapError(err)

		}
	}

	err = tx.Commit()
	if err != nil {
		return internal.WrapError(err)
	}

	return nil
}

func (p *PostgreSQL) ReloadAllMetrics(ctx context.Context, metrics []models.StorageMetrics) (int64, error) {

	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	batch := new(pgx.Batch)

	batch.Queue("TRUNCATE postgres.gauges")
	batch.Queue("TRUNCATE postgres.counters")
	for _, metric := range metrics {

		switch metric.MType {
		case "gauge":

			batch.Queue(QueryInsertGauges, metric.Value, time.Now(), metric.Name)
		case "counter":
			batch.Queue(QueryInsertCounters, metric.Delta, time.Now(), metric.Name)
		default:
			return 0, fmt.Errorf("unclown MType %s", metric.MType)
		}
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, internal.WrapError(err)
	}
	result := tx.SendBatch(ctx, batch)

	tag, err := result.Exec()
	if err != nil {
		return 0, internal.WrapError(err)
	}

	err = result.Close()
	if err != nil {
		_ = tx.Rollback(ctx)
		return 0, internal.WrapError(err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return 0, internal.WrapError(err)
	}

	return tag.RowsAffected(), nil

}
