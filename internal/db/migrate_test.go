package db_test

import (
	"context"
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
