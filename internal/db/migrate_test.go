package db_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/db"
)

func TestMigrate_appliesAllPendingThenIdempotent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()

	applied1, err := db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(applied1), 1, "expected at least 0001_init to apply")

	// Second run applies nothing.
	applied2, err := db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	require.Empty(t, applied2)

	// schema_version has rows.
	var count int
	require.NoError(t, conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_version").Scan(&count))
	require.Equal(t, len(applied1), count)
}

func TestMigrate_createsCoreTables(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()

	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	for _, table := range []string{"watchlist", "alerts", "job_runs", "schema_version"} {
		var got string
		err := conn.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&got)
		require.NoError(t, err, "missing table %s", table)
	}
}

func TestMigrate0003AddsForm4Columns(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	conn, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.Close()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := db.Migrate(ctx, conn, filepath.Join(repoRoot, "migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rows, err := conn.QueryContext(ctx, `PRAGMA table_info(insider_trades)`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer rows.Close()
	have := map[string]string{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		have[name] = typ
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}

	want := map[string]string{
		"shares":           "INTEGER",
		"price_per_share":  "REAL",
		"transaction_code": "TEXT",
		"security_type":    "TEXT",
	}
	for col, typ := range want {
		got, ok := have[col]
		if !ok {
			t.Errorf("column %s missing", col)
			continue
		}
		if got != typ {
			t.Errorf("column %s: type %s want %s", col, got, typ)
		}
	}

	var idxName string
	if err := conn.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='index' AND name='insider_trades_txcode'`,
	).Scan(&idxName); err != nil {
		t.Errorf("partial index insider_trades_txcode missing: %v", err)
	}
}
