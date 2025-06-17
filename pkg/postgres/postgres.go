package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jmoiron/sqlx"

	_ "github.com/golang-migrate/migrate/v4/source/file" // Необходим для миграции
	_ "github.com/jackc/pgx/v5/stdlib"                   // драйвер для sqlx
)

type Postgres struct {
	DB *sqlx.DB
}

type Config struct {
	DataSource        string
	ApplicationSchema string
}

const (
	driverName = "pgx"
)

func NewPostgres(cfg Config) (*Postgres, error) {
	conn, err := sqlx.Connect(driverName, cfg.DataSource)
	if err != nil {
		return nil, fmt.Errorf("sqlx.Connect: %w", err)
	}

	if _, err := conn.Exec("create schema if not exists " + cfg.ApplicationSchema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	var dbName []string
	err = conn.Select(&dbName, "SELECT current_database()")
	if err != nil {
		return nil, fmt.Errorf("get current database: %w", err)
	}

	if _, err := conn.Exec(fmt.Sprintf("ALTER DATABASE %s SET search_path TO %s,public;", dbName[0], cfg.ApplicationSchema)); err != nil {
		return nil, fmt.Errorf("use schema: %w", err)
	}

	return &Postgres{
		DB: conn,
	}, nil
}

func (p *Postgres) Close(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("<-ctx.Done(): context canceled")
	default:
		return p.DB.Close() //nolint:wrapcheck
	}
}

func (p *Postgres) RunMigrations(pathToMigrations string, applicationSchema string) (uint, error) {
	driver, err := postgres.WithInstance(p.DB.DB, &postgres.Config{
		MigrationsTable:       "",
		MigrationsTableQuoted: false,
		MultiStatementEnabled: false,
		DatabaseName:          "",
		SchemaName:            applicationSchema,
		StatementTimeout:      0,
		MultiStatementMaxSize: 0,
	})
	if err != nil {
		return 0, fmt.Errorf("create driver with instance: %w", err)
	}

	migrateInst, err := migrate.NewWithDatabaseInstance(pathToMigrations, driverName, driver)
	if err != nil {
		return 0, fmt.Errorf("create migrate instance: %w", err)
	}

	if err = migrateInst.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return 0, fmt.Errorf("up migrations: %w", err)
	}

	version, _, _ := migrateInst.Version()
	return version, nil
}
