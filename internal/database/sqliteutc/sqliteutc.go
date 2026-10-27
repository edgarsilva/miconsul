// Package sqliteutc provides a SQLite driver that binds every time.Time in UTC.
package sqliteutc

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DriverName is the database/sql name of the driver.
const DriverName = "sqlite3_utc"

func init() {
	sql.Register(DriverName, &utcDriver{})
}

// Open returns a GORM dialector for dsn that uses the UTC driver.
func Open(dsn string) gorm.Dialector {
	return sqlite.New(sqlite.Config{DriverName: DriverName, DSN: dsn})
}

// utcDriver opens SQLite connections that bind time values in UTC.
type utcDriver struct {
	sqlite sqlite3.SQLiteDriver
}

// Open opens a SQLite connection wrapped in utcConn.
func (d *utcDriver) Open(dsn string) (driver.Conn, error) {
	conn, err := d.sqlite.Open(dsn)
	if err != nil {
		return nil, err
	}

	sqliteConn, ok := conn.(*sqlite3.SQLiteConn)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("open sqlite: unexpected connection type %T", conn)
	}

	return &utcConn{SQLiteConn: sqliteConn}, nil
}

// utcConn is a SQLite connection that binds time values in UTC.
type utcConn struct {
	*sqlite3.SQLiteConn
}

// CheckNamedValue converts an argument the way database/sql would by default,
// then turns a time.Time into UTC.
func (c *utcConn) CheckNamedValue(nv *driver.NamedValue) error {
	v, err := driver.DefaultParameterConverter.ConvertValue(nv.Value)
	if err != nil {
		return err
	}
	if t, ok := v.(time.Time); ok {
		v = t.UTC()
	}
	nv.Value = v

	return nil
}
