package postgresql

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/s-turchinskiy/metrics/internal"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"log"
	"strings"
	"time"
)

const (
	queryCreateTableGauges = `CREATE TABLE IF NOT EXISTS %s.gauges (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value DOUBLE PRECISION,
    date TIMESTAMPTZ)`

	queryCreateTableCounters = `CREATE TABLE IF NOT EXISTS %s.counters (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value INT,
    date TIMESTAMPTZ)`
)

func Initialize(ctx context.Context) (service.Repository, error) {

	addr := settings.Settings.Database.String()
	logger.Log.Debug("addr for Sql.Open: ", addr)
	logger.Log.Debug("FlagDatabaseDSN for Sql.Open: ", settings.Settings.Database.FlagDatabaseDSN)

	db, err := sqlx.Open("pgx", addr)
	if err != nil {
		return nil, internal.WrapError(err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, internal.WrapError(err)
	}

	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	p := &PostgreSQL{db: db}
	p.tableSchema = "postgres"

	_, err = p.db.ExecContext(ctx, "DROP TABLE IF EXISTS postgres.testtable")
	if err != nil {
		return nil, internal.WrapError(err)
	}

	err = p.LoggingStateDatabase(ctx)
	if err != nil {
		return nil, internal.WrapError(err)
	}

	_, err = p.db.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, p.tableSchema))
	if err != nil {
		return nil, internal.WrapError(err)
	}

	err = p.withLoggingCreatingTable(ctx, "testtable", "CREATE TABLE IF NOT EXISTS %s.testtable (id SERIAL PRIMARY KEY)")
	if err != nil {
		return nil, internal.WrapError(err)
	}

	err = p.withLoggingCreatingTable(ctx, "gauges", queryCreateTableGauges)
	if err != nil {
		return nil, internal.WrapError(err)
	}

	err = p.withLoggingCreatingTable(ctx, "counters", queryCreateTableCounters)
	if err != nil {
		return nil, internal.WrapError(err)
	}

	err = p.LoggingStateDatabase(ctx)
	if err != nil {
		return nil, internal.WrapError(err)
	}

	return p, nil

}

func (p *PostgreSQL) LoggingStateDatabase(ctx context.Context) error {

	err := p.loggingData(ctx, "schemas",
		"SELECT schema_name FROM information_schema.schemata WHERE catalog_name = $1;",
		settings.Settings.Database.DBName)
	if err != nil {
		return internal.WrapError(err)
	}

	/*err = p.loggingData(ctx,
		"tables",
		"SELECT table_name FROM information_schema.tables WHERE table_schema = $1",
		p.tableSchema)
	if err != nil {
		return internal.WrapError(err)
	}*/

	err = p.loggingData(ctx,
		"view tables",
		"SELECT table_schema || '.' || table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')")
	if err != nil {
		return internal.WrapError(err)
	}
	return nil
}

func (p *PostgreSQL) tableExist(ctx context.Context, tableName string) bool {
	row := p.db.QueryRowContext(ctx, fmt.Sprintf(`select exists (select *
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

func (p *PostgreSQL) withLoggingCreatingTable(ctx context.Context, tableName, query string) error {

	existBefore := p.tableExist(ctx, tableName)
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(query, p.tableSchema))
	if err != nil {
		return err
	}

	if existBefore {
		logger.Log.Debug(fmt.Sprintf("table %s.%s already exist", p.tableSchema, tableName))
		return nil
	}
	existAfter := p.tableExist(ctx, tableName)
	if !existBefore && existAfter {
		logger.Log.Info(strings.ToUpper("created table "), p.tableSchema+"."+tableName)
	}

	return nil
}

func (p *PostgreSQL) loggingData(ctx context.Context, title, query string, args ...interface{}) error {

	var data []string

	err := p.db.SelectContext(
		ctx,
		&data,
		query,
		args...)

	if err != nil {
		return err
	}

	logger.Log.Debugw(title, "values", strings.Join(data, ","))

	return nil
}
