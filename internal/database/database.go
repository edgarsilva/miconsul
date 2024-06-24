package database

import (
	// libsql "github.com/edgarsilva/gorm-libsql"

	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const DBOpts = "?mode=rwc&_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"

type Database struct {
	*gorm.DB
}

func New(DBPath string) *Database {
	loglevel := logger.Info
	if os.Getenv("APP_ENV") == "production" {
		loglevel = logger.Warn
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Millisecond * 100, // Slow SQL threshold
			LogLevel:                  loglevel,               // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,                  // Don't include params in the SQL log
			Colorful:                  true,                   // Enable/Disable color
		},
	)

	DB, err := gorm.Open(sqlite.Open(DBPath+DBOpts), &gorm.Config{
		Logger:                 newLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		panic("failed to connect database")
	}

	return &Database{
		DB: DB,
	}
}
