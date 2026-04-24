package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/db"
)

func TestOpen_createsFileAndEnablesWAL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	conn, err := db.Open(context.Background(), path)
	require.NoError(t, err)
	defer conn.Close()

	var mode string
	require.NoError(t, conn.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&mode))
	require.Equal(t, "wal", mode)

	var busy int
	require.NoError(t, conn.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&busy))
	require.Equal(t, 5000, busy)
}

func TestOpen_parentDirCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "path", "test.db")

	conn, err := db.Open(context.Background(), path)
	require.NoError(t, err)
	defer conn.Close()
}
