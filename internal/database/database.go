// Package database provides a gorm database connection
package database

import (
	// libsql "github.com/edgarsilva/gorm-libsql"

	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"miconsul/goose/migrations"
	"miconsul/internal/lib/appenv"
	obslogging "miconsul/internal/observability/logging"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const PragmaOpts = "?mode=rwc" +
	"&_journal_mode=WAL" +
	"&_synchronous=NORMAL" +
	"&_busy_timeout=10000" +
	"&_cache_size=-16384" +
	"&_temp_store=MEMORY" +
	"&_wal_autocheckpoint=1000" +
	"&_journal_size_limit=67108864" +
	"&_mmap_size=268435456"

type Database struct {
	*gorm.DB
}

func New(env *appenv.Env, obsLogger obslogging.Logger) (*Database, error) {
	if env == nil {
		return nil, fmt.Errorf("database: environment config is nil")
	}

	dbLogger := NewLogger(env, obsLogger)
	dsn := env.DBPath + PragmaOpts
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:      dbLogger,
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("database: open sqlite: %w", err)
	}
	pluginOpts := []otelgorm.Option{}
	dbName := filepath.Base(env.DBPath)
	if dbName != "" && dbName != "." {
		pluginOpts = append(pluginOpts, otelgorm.WithDBName(dbName))
	}

	if err := db.Use(otelgorm.NewPlugin(pluginOpts...)); err != nil {
		return nil, fmt.Errorf("database: register otel gorm plugin: %w", err)
	}

	return &Database{
		DB: db,
	}, nil
}

func (d *Database) GormDB() *gorm.DB {
	if d == nil {
		return nil
	}

	return d.DB
}

func (d *Database) SQLDB() (*sql.DB, error) {
	if d == nil || d.DB == nil {
		return nil, nil
	}

	sqlDB, err := d.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db handle: %w", err)
	}

	return sqlDB, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.SQLDB()
	if err != nil {
		return err
	}

	if sqlDB == nil {
		return nil
	}

	return sqlDB.Close()
}

func ApplyMigrations(database *Database, env *appenv.Env) error {
	return applyMigrations(database, env, false)
}

func ApplyMigrationsSilent(database *Database, env *appenv.Env) error {
	return applyMigrations(database, env, true)
}

func applyMigrations(database *Database, env *appenv.Env, silent bool) error {
	if !silent {
		fmt.Println(" Applying migrations...")
	}

	if database == nil {
		return fmt.Errorf("database is not initialized")
	}

	sqlDB, err := database.SQLDB()
	if err != nil {
		return fmt.Errorf("resolve sql db handle: %w", err)
	}
	if sqlDB == nil {
		return fmt.Errorf("database is not initialized")
	}

	if silent {
		goose.SetLogger(NoopLogger{})
	} else if env != nil && (appenv.IsDevelopment(env.Environment) || appenv.IsProduction(env.Environment)) {
		logger, _ := zap.NewProduction()
		sugar := logger.Sugar()
		goose.SetLogger(ZapLogger{l: sugar})
	}

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("run goose migrations: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	return nil
}

func NewLogger(env *appenv.Env, obsLogger obslogging.Logger) logger.Interface {
	loglevel := logger.Warn
	hideParamValues := true
	if env != nil && appenv.IsDevelopment(env.Environment) {
		loglevel = logger.Info
		hideParamValues = false
	}

	baseLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             150 * time.Millisecond,
			LogLevel:                  loglevel,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      hideParamValues,
			Colorful:                  true,
		},
	)

	return NewGormObsLogger(baseLogger, obsLogger)
}
