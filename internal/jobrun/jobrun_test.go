package jobrun_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/internal/jobrun"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	return conn
}

func TestTrack_successRecordsOK(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	err := jobrun.Track(ctx, conn, "demo", func(ctx context.Context) (int, int, error) {
		return 10, 3, nil
	})
	require.NoError(t, err)

	var status string
	var rowsIn, rowsNew sql.NullInt64
	var errText sql.NullString
	err = conn.QueryRowContext(ctx,
		"SELECT status, rows_in, rows_new, error FROM job_runs WHERE job='demo'",
	).Scan(&status, &rowsIn, &rowsNew, &errText)
	require.NoError(t, err)
	require.Equal(t, "ok", status)
	require.Equal(t, int64(10), rowsIn.Int64)
	require.Equal(t, int64(3), rowsNew.Int64)
	require.False(t, errText.Valid)
}

func TestTrack_errorRecordsError(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	err := jobrun.Track(ctx, conn, "demo", func(ctx context.Context) (int, int, error) {
		return 0, 0, errors.New("boom")
	})
	require.Error(t, err)

	var status string
	var errText sql.NullString
	require.NoError(t, conn.QueryRowContext(ctx,
		"SELECT status, error FROM job_runs WHERE job='demo'",
	).Scan(&status, &errText))
	require.Equal(t, "error", status)
	require.Contains(t, errText.String, "boom")
}

func TestTrack_panicRecordsError(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	require.Panics(t, func() {
		_ = jobrun.Track(ctx, conn, "demo", func(ctx context.Context) (int, int, error) {
			panic("kaboom")
		})
	})

	var status string
	var errText sql.NullString
	require.NoError(t, conn.QueryRowContext(ctx,
		"SELECT status, error FROM job_runs WHERE job='demo'",
	).Scan(&status, &errText))
	require.Equal(t, "error", status)
	require.Contains(t, errText.String, "kaboom")
}

func TestTrack_noopStatus(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	err := jobrun.Track(ctx, conn, "demo", func(ctx context.Context) (int, int, error) {
		return 0, 0, jobrun.ErrNoop
	})
	require.NoError(t, err)

	var status string
	require.NoError(t, conn.QueryRowContext(ctx,
		"SELECT status FROM job_runs WHERE job='demo'",
	).Scan(&status))
	require.Equal(t, "noop", status)
}
