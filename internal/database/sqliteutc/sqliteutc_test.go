package sqliteutc

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestOpenBindsTimesInUTC(t *testing.T) {
	mx, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	local := time.Local
	time.Local = mx
	t.Cleanup(func() { time.Local = local })

	db, err := gorm.Open(Open(filepath.Join(t.TempDir(), "utc.sqlite")), &gorm.Config{PrepareStmt: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := db.Exec("CREATE TABLE stamps (name TEXT, at DATETIME)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	// 20:30 in Mexico City is 02:30 the next day in UTC.
	at := time.Date(2026, 10, 28, 20, 30, 0, 0, mx)
	args := map[string]any{
		"time":       at,
		"pointer":    &at,
		"null time":  sql.NullTime{Time: at, Valid: true},
		"deleted at": gorm.DeletedAt{Time: at, Valid: true},
	}
	for name, arg := range args {
		if err := db.Exec("INSERT INTO stamps (name, at) VALUES (?, ?)", name, arg).Error; err != nil {
			t.Fatalf("insert %s: %v", name, err)
		}
	}
	if _, err := sqlDB.Exec("INSERT INTO stamps (name, at) VALUES (?, ?)", "database/sql", at); err != nil {
		t.Fatalf("insert through database/sql: %v", err)
	}

	rows, err := sqlDB.Query("SELECT name, CAST(at AS TEXT) FROM stamps")
	if err != nil {
		t.Fatalf("read stamps: %v", err)
	}
	defer func() { _ = rows.Close() }()
	count := 0
	for rows.Next() {
		var name, stored string
		if err := rows.Scan(&name, &stored); err != nil {
			t.Fatalf("scan: %v", err)
		}
		count++
		if stored != "2026-10-29 02:30:00+00:00" {
			t.Errorf("%s stored as %q, want 2026-10-29 02:30:00+00:00", name, stored)
		}
	}
	if count != len(args)+1 {
		t.Fatalf("read %d rows, want %d", count, len(args)+1)
	}

	var before int64
	bound := time.Date(2026, 10, 28, 21, 0, 0, 0, mx)
	if err := db.Raw("SELECT COUNT(*) FROM stamps WHERE at < ?", bound).Scan(&before).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if before != int64(count) {
		t.Fatalf("rows before 21:00 Mexico City = %d, want %d", before, count)
	}
}

func TestCheckNamedValueLeavesOtherValues(t *testing.T) {
	conn := &utcConn{}
	for _, v := range []any{nil, int64(7), "text", []byte("b"), true, 1.5} {
		nv := driver.NamedValue{Value: v}
		if err := conn.CheckNamedValue(&nv); err != nil {
			t.Fatalf("CheckNamedValue(%v): %v", v, err)
		}
		if fmt.Sprintf("%T", nv.Value) != fmt.Sprintf("%T", v) {
			t.Errorf("CheckNamedValue(%v) changed type to %T", v, nv.Value)
		}
	}
}

func BenchmarkCheckNamedValue(b *testing.B) {
	conn := &utcConn{}
	now := time.Now()
	b.Run("time", func(b *testing.B) {
		for b.Loop() {
			nv := driver.NamedValue{Value: now}
			_ = conn.CheckNamedValue(&nv)
		}
	})
	b.Run("time default converter", func(b *testing.B) {
		for b.Loop() {
			_, _ = driver.DefaultParameterConverter.ConvertValue(now)
		}
	})
	b.Run("string", func(b *testing.B) {
		for b.Loop() {
			nv := driver.NamedValue{Value: "x"}
			_ = conn.CheckNamedValue(&nv)
		}
	})
}
