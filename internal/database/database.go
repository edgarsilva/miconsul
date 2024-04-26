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

func New(dbPath string) *Database {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Millisecond * 100, // Slow SQL threshold
			LogLevel:                  logger.Info,            // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,                   // Don't include params in the SQL log
			Colorful:                  true,                   // Disable color
		},
	)

	// db, err := gorm.Open(libsql.Open("turso-embed.db", dbURL, authToken), &gorm.Config{})
	// refer https://github.com/go-sql-driver/mysql#dsn-data-source-name for details
	// dsn := "root:mysql@tcp(127.0.0.1:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"
	// db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{

	DB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger:                 newLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	DB.AutoMigrate(
		&User{},
		&Todo{},
		&Post{},
		&Comment{},
		// &PurchaseOrder{},
		// &LineItem{},
	)

	return &Database{
		DB: DB,
	}
}
