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

type Database struct {
	*gorm.DB
}

// const authToken = "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIyMDIzLTEyLTE5VDAyOjM0OjUzLjk4MDU0MzQzNloiLCJpZCI6ImI4NjNmYTY2LTllMTUtMTFlZS1iNTk2LTEyYWIwZGY3MGIxZiJ9.hLTP9Iv-ZUfzZAAV077Bzylfsug1A_cakR7VcwP7DW4bbeU4DEl2yaM0k-ggrd75TQKPYfcx8J3VUZmxAwDfCA"
// const dbURL = "libsql://golang-edgarsilva.turso.io?authToken="

func NewDatabase() *Database {

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Millisecond * 250, // Slow SQL threshold
			LogLevel:                  logger.Info,            // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,                   // Don't include params in the SQL log
			Colorful:                  true,                   // Disable color
		},
	)

	// db, err := gorm.Open(libsql.Open("turso-embed.db", dbURL, authToken), &gorm.Config{})
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&Todo{})

	return &Database{
		DB: db,
	}
}
