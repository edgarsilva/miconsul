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

const DBOpts = "?cache=shared&mode=rwc&_journal_mode=WAL&_foreign_keys=true"

type Database struct {
	*gorm.DB
}

func New(DBPath string) *Database {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Millisecond * 100, // Slow SQL threshold
			LogLevel:                  logger.Info,            // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,                   // Don't include params in the SQL log
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

	// Migrate the schema
	// DB.AutoMigrate(
	// &model.User{},
	// &model.Todo{},
	// &model.Clinic{},
	// &model.Patient{},
	// &model.FeedEvent{},
	// &model.Appointment{},
	// &model.Alert{},
	// &Journal|Logbook  TODO: Logbook to log extraneous events (No 20k USD Datatog bill)
	// )

	return &Database{
		DB: DB,
	}
}
