package models

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestSqliteWriteDSN(t *testing.T) {
	got := sqliteWriteDSN("/tmp/x/main.db")
	for _, want := range []string{
		"/tmp/x/main.db?",
		"_busy_timeout=30000",
		"_journal_mode=WAL",
		"_synchronous=NORMAL",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sqliteWriteDSN = %q, want substring %q", got, want)
		}
	}

	// existing query string must be extended, not truncated
	got = sqliteWriteDSN("/tmp/x/main.db?cache=shared")
	if !strings.Contains(got, "cache=shared&_busy_timeout=") {
		t.Errorf("sqliteWriteDSN with query = %q, want params appended with &", got)
	}
}

func openTestDB(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open("sqlite3", sqliteWriteDSN(path))
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func queryPragma(t *testing.T, db *gorm.DB, pragma string, dest interface{}) {
	t.Helper()
	if err := db.DB().QueryRow("PRAGMA " + pragma).Scan(dest); err != nil {
		t.Fatalf("PRAGMA %s: %v", pragma, err)
	}
}

// TestSqlitePragmasApplied guards issue #2139: every sqlite handle must wait
// on locks (busy_timeout) and keep readers from blocking writers (WAL).
func TestSqlitePragmasApplied(t *testing.T) {
	db := openTestDB(t, filepath.Join(t.TempDir(), "pragma.db"))
	if err := db.Exec("CREATE TABLE t (id INTEGER)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	var journalMode string
	queryPragma(t, db, "journal_mode", &journalMode)
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}

	var busyTimeout int
	queryPragma(t, db, "busy_timeout", &busyTimeout)
	if busyTimeout != 30000 {
		t.Errorf("busy_timeout = %d, want 30000", busyTimeout)
	}
}

// TestSqliteLockWaitsInsteadOfFailing reproduces the #2139 mechanism: one
// connection holding a write lock while another writes. The driver's default
// 5s busy_timeout gives up on sustained contention (dozens of concurrent
// scrapers), failing with "database is locked"; the 30s timeout must ride
// out the hold, so the write blocks until release and succeeds.
func TestSqliteLockWaitsInsteadOfFailing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locked.db")
	setup := openTestDB(t, path)
	if err := setup.Exec("CREATE TABLE t (id INTEGER)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := setup.Exec("INSERT INTO t (id) VALUES (1)").Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	holder := openTestDB(t, path)
	writer := openTestDB(t, path)

	// Pin a single pooled connection so BEGIN/COMMIT bracket the same txn.
	ctx := context.Background()
	conn, err := holder.DB().Conn(ctx)
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("begin: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- writer.Exec("UPDATE t SET id = 2 WHERE id = 1").Error
	}()

	// Hold past the driver's 5s default: a pre-fix handle fails at ~5s while
	// the fixed handle keeps waiting for the release below.
	select {
	case err := <-done:
		t.Fatalf("write finished while lock held (no waiting?): %v", err)
	case <-time.After(8 * time.Second):
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("write after lock release failed: %v", err)
		}
	case <-time.After(35 * time.Second):
		t.Fatal("write did not proceed after lock release")
	}

	var id int
	if err := writer.DB().QueryRow("SELECT id FROM t").Scan(&id); err != nil || id != 2 {
		t.Fatalf("row = %d, err = %v; want 2, nil", id, err)
	}
}
