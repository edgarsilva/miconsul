package database

import (
	// libsql "github.com/edgarsilva/gorm-libsql"

	"log"
	"os"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const PragmaOpts = "?mode=rwc&" +
	"_foreign_keys=on&" +
	"_journal_mode=WAL&" +
	"_busy_timeout=5000&" +
	"_sync=NORMAL&" +
	"_cache_size=2000"

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
			SlowThreshold:             time.Millisecond * 150, // Slow SQL threshold
			LogLevel:                  loglevel,               // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      hideParamValues,        // If true, don't include params values in the SQL log
			Colorful:                  true,                   // Enable/Disable color
		},
	)

	db, err := gorm.Open(sqlite.Open(DBPath+PragmaOpts), &gorm.Config{
		Logger: newLogger,
		// SkipDefaultTransaction: false,
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
