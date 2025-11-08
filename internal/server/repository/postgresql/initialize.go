// Package postgresql Хранение данных в postgresql
package postgresql

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/common/errutil"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"github.com/s-turchinskiy/metrics/internal/server/repository"
)

type PostgreSQL struct {
	db   *sqlx.DB
	pool *pgxpool.Pool
}

func Initialize(ctx context.Context, dbAddr, dbName string) (repository.Repository, error) {

	logger.Log.Debug("addr for Sql.Open: ", dbAddr)

	db, err := sqlx.Open("pgx", dbAddr)
	if err != nil {
		return nil, errutil.WrapError(err)
	}
	if err = db.PingContext(ctx); err != nil {
		return nil, errutil.WrapError(err)
	}

	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	pool, err := pgxpool.New(ctx, dbAddr)
	if err != nil {
		return nil, errutil.WrapError(err)
	}

	p := &PostgreSQL{db: db, pool: pool}

	_, err = p.db.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, "postgres"))
	if err != nil {
		return nil, errutil.WrapError(err)
	}

	if err = runMigrations(db.DB, dbName); err != nil {
		return nil, err
	}
	err = p.LoggingStateDatabase(ctx)
	if err != nil {
		return nil, errutil.WrapError(err)
	}

	return p, nil

}

//go:embed migrations/*.sql
var _ embed.FS

func runMigrations(db *sql.DB, dbname string) error {

	driver, err := postgres.WithInstance(db, &postgres.Config{SchemaName: "postgres"})
	if err != nil {
		return errutil.WrapError(err)
	}

	var pathToMigrations string
	_, err = os.Stat("./internal/server/repository/postgresql/migrations")
	if err == nil {
		pathToMigrations = "file://internal/server/repository/postgresql/migrations"
	} else {
		pathToMigrations = "file://migrations"
	}

	m, err := migrate.NewWithDatabaseInstance(pathToMigrations, dbname, driver)
	if err != nil {
		return errutil.WrapError(err)
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return errutil.WrapError(err)
	}

	return nil
}

func (p *PostgreSQL) LoggingStateDatabase(ctx context.Context) error {

	/*err = p.loggingData(ctx,
		"tables",
		"SELECT table_name FROM information_schema.tables WHERE table_schema = $1",
		p.tableSchema)
	if err != nil {
		return internal.WrapError(err)
	}*/

	err := p.loggingData(ctx,
		"view tables",
		"SELECT table_schema || '.' || table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')")
	if err != nil {
		return errutil.WrapError(err)
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

func (p *PostgreSQL) Close(ctx context.Context) error {

	p.pool.Close()

	err := p.db.Close()

	if err != nil {
		logger.Log.Infow("PostgreSQL stopped with error", zap.String("error", err.Error()))
	} else {
		logger.Log.Infow("PostgreSQL stopped")
	}
	return err

}
