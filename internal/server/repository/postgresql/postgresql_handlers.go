package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/utils/errutil"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/models"
)

const (
	QueryInsertUpdateCounter = `
	INSERT INTO postgres.counters (metrics_name, value, updated) 
	VALUES ($1, $2, $3)
	ON CONFLICT (metrics_name) DO UPDATE SET
			value = EXCLUDED.value + counters.value,
			updated = EXCLUDED.updated`

	QueryInsertUpdateGauge = `
		INSERT INTO postgres.gauges (metrics_name, value, updated) 
		VALUES ($1, $2, $3)
		ON CONFLICT (metrics_name) DO UPDATE SET
			value = EXCLUDED.value,
			updated = EXCLUDED.updated`
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

	_, err := p.db.ExecContext(ctx, QueryInsertUpdateGauge, metricsName, newValue, time.Now())
	if err != nil {
		err = fmt.Errorf("PostgreSQL.UpdateGauge error in p.DB.Exec, %w", err)
	}
	return err

}

func (p *PostgreSQL) UpdateCounter(ctx context.Context, metricsName string, newValue int64) error {

	logger.Log.Debugw("PostgreSQL.UpdateCounter try",
		"metricsName", metricsName,
		"value", newValue,
	)

	_, err := p.db.ExecContext(ctx, QueryInsertUpdateCounter, metricsName, newValue, time.Now())
	if err != nil {
		var pgErr *pgconn.PgError
		errors.As(err, &pgErr)
		if pgerrcode.IsSyntaxErrororAccessRuleViolation(pgErr.Code) {
			return errutil.WrapError(fmt.Errorf("%w syntax error in request QueryInsertUpdateCounter", err))
		}
		return errutil.WrapError(err)
	}

	return nil

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

	row := p.db.QueryRowContext(ctx, "SELECT value FROM postgres.gauges WHERE metrics_name = $1", metricsName)
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
			return 0, false, errutil.WrapError(err)
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
		return nil, errutil.WrapError(err)
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
		return errutil.WrapError(err)
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "TRUNCATE postgres.gauges")
	if err != nil {
		return errutil.WrapError(err)
	}

	portionData := make(map[string]float64, 10)

	for metricsName, newValue := range data {
		portionData[metricsName] = newValue

		if len(portionData) == 10 {

			err = insertUpdatePortionGauges(ctx, portionData, tx)
			if err != nil {
				return errutil.WrapError(err)
			}

			for k := range portionData {
				delete(portionData, k)
			}
		}
	}

	err = insertUpdatePortionGauges(ctx, portionData, tx)
	if err != nil {
		return errutil.WrapError(err)
	}

	err = tx.Commit()
	if err != nil {
		return errutil.WrapError(err)
	}

	return nil

}

func insertUpdatePortionGauges(ctx context.Context, portionData map[string]float64, tx *sql.Tx) error {

	for metricsName, newValue := range portionData {
		_, err := tx.ExecContext(ctx, QueryInsertUpdateGauge, metricsName, newValue, time.Now())
		if err != nil {
			return err

		}
	}
	return nil
}

func (p *PostgreSQL) ReloadAllCounters(ctx context.Context, data map[string]int64) error {

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "TRUNCATE postgres.counters")
	if err != nil {
		return errutil.WrapError(err)
	}

	portionData := make(map[string]int64, 10)

	for metricsName, newValue := range data {
		portionData[metricsName] = newValue

		if len(portionData) == 10 {

			err = insertUpdatePortionCounters(ctx, portionData, tx)
			if err != nil {
				return errutil.WrapError(err)
			}

			for k := range portionData {
				delete(portionData, k)
			}
		}
	}

	err = insertUpdatePortionCounters(ctx, portionData, tx)
	if err != nil {
		return errutil.WrapError(err)
	}

	err = tx.Commit()
	if err != nil {
		return errutil.WrapError(err)
	}

	return nil
}

func insertUpdatePortionCounters(ctx context.Context, portionData map[string]int64, tx *sql.Tx) error {

	for metricsName, newValue := range portionData {
		_, err := tx.ExecContext(ctx, QueryInsertUpdateCounter, metricsName, newValue, time.Now())
		if err != nil {
			return err

		}
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

	batch.Queue("TRUNCATE postgres.counters")
	batch.Queue("TRUNCATE postgres.gauges")

	for _, metric := range metrics {

		switch metric.MType {
		case "gauge":
			batch.Queue(QueryInsertUpdateGauge, metric.Name, &metric.Value, time.Now())
		case "counter":
			batch.Queue(QueryInsertUpdateCounter, metric.Name, &metric.Delta, time.Now())
		default:
			return 0, fmt.Errorf("unclown MType %s", metric.MType)
		}
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, errutil.WrapError(err)
	}

	result := tx.SendBatch(ctx, batch)

	tag, err := result.Exec()
	if err != nil {
		return 0, errutil.WrapError(err)
	}

	err = result.Close()
	if err != nil {
		_ = tx.Rollback(ctx)
		return 0, errutil.WrapError(err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return 0, errutil.WrapError(err)
	}

	return tag.RowsAffected(), nil

}
