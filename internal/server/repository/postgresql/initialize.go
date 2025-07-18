package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/s-turchinskiy/metrics/internal/server/logger"
	"github.com/s-turchinskiy/metrics/internal/server/service"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
	"log"
	"strings"
	"time"
)

func Initialize(ctx context.Context) (service.Repository, error) {

	db := sqlx.MustOpen("pgx", settings.Settings.Database.String())
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	p := &PostgreSQL{db: db}
	p.tableSchema = "postgres"

	p.runCommand(ctx, "DROP TABLE postgres.gauges IF EXIST")
	//p.runCommand("DROP TABLE postgres.counters IF EXIST")

	err := p.loggingData(ctx,
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
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, tableSchema))
	return err
}

func (p *PostgreSQL) createTableGauges(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s.gauges (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value DOUBLE PRECISION,
    date TIMESTAMPTZ)`,
		p.tableSchema))
	return err
}

func (p *PostgreSQL) createTableCounters(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s.counters (
    id SERIAL PRIMARY KEY,
    metrics_name TEXT NOT NULL,
    value INT,
    date TIMESTAMPTZ)`,
		p.tableSchema))
	return err
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

	var rows *sql.Rows
	var err error

	if parameter == "" {
		rows, err = p.db.QueryContext(ctx, query)

	} else {
		rows, err = p.db.QueryContext(ctx, query, parameter)
	}

	defer rows.Close()

	if err != nil {
		return err
	}

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
	_, err := p.db.ExecContext(ctx, command)
	return err
}
