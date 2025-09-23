// Package database provides a gorm database connection
package database

import (
	// libsql "github.com/edgarsilva/gorm-libsql"

	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

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

func New(DBPath string) *Database {
	loglevel := logger.Warn
	hideParamValues := true
	if os.Getenv("APP_ENV") == "development" {
		loglevel = logger.Info
		hideParamValues = false
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             150 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  loglevel,               // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      hideParamValues,        // If true, don't include params values in the SQL log
			Colorful:                  true,                   // Enable/Disable color
		},
	)

	dsn := DBPath + PragmaOpts
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:      newLogger,
		PrepareStmt: true,
	})
	if err != nil {
		log.Panic("Failed to connect database")
	}
	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		panic(err)
	}

	return &Database{
		DB: db,
	}
}

func ApplyMigrations(dbPath string) error {
	fmt.Println(" Applying migrations...")
	dsn := dbPath
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// Usage
	if os.Getenv("APP_ENV") == "development" || os.Getenv("APP_ENV") == "production" {
		logger, _ := zap.NewProduction()
		sugar := logger.Sugar()
		goose.SetLogger(ZapLogger{l: sugar})
	}

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

func NewLogger() logger.Interface {
	loglevel := logger.Warn
	hideParamValues := true
	if os.Getenv("APP_ENV") == "development" {
		loglevel = logger.Info
		hideParamValues = false
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             150 * time.Millisecond,
			LogLevel:                  loglevel,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      hideParamValues,
			Colorful:                  true,
		},
	)

	return newLogger
}
