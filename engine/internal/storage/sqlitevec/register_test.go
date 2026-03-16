package sqlitevec_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mutecomm/go-sqlcipher/v4"
	"github.com/welife-os/welife-os/engine/internal/storage/sqlitevec"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if err := sqlitevec.Init(); err != nil {
		t.Fatalf("sqlitevec.Init: %v", err)
	}
	dsn := filepath.Join(t.TempDir(), "test.db") + "?_pragma_key=testkey&_pragma_foreign_keys=ON"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestVecVersionReturnsValidVersion(t *testing.T) {
	db := openTestDB(t)

	var version string
	err := db.QueryRowContext(context.Background(), "SELECT vec_version()").Scan(&version)
	if err != nil {
		t.Fatalf("vec_version() failed: %v", err)
	}
	if version == "" {
		t.Fatal("vec_version() returned empty string")
	}
	t.Logf("sqlite-vec version: %s", version)
}

func TestVec0VirtualTableCanBeCreated(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `CREATE VIRTUAL TABLE test_vec USING vec0(embedding float[4])`)
	if err != nil {
		t.Fatalf("CREATE vec0 table: %v", err)
	}

	// Verify we can query the empty table.
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_vec").Scan(&count)
	if err != nil {
		t.Fatalf("COUNT(*) from vec0: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}
}
