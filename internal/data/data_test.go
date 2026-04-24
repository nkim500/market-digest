package data_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/data"
	"github.com/nkim500/market-digest/internal/db"
)

func TestRecentAlerts_ordersUnseenFirst(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	oldSeenID, err := alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "info", Title: "old seen"})
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `UPDATE alerts SET seen_ts=? WHERE id=?`, time.Now().Unix(), oldSeenID)
	require.NoError(t, err)

	_, err = alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "watch", Title: "new unseen"})
	require.NoError(t, err)

	rows, err := data.RecentAlerts(ctx, conn, 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "new unseen", rows[0].Title, "unseen must come first")
	require.Equal(t, "old seen", rows[1].Title)
}

func TestLastJobRun_returnsNilWhenEmpty(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	run, err := data.LastJobRun(ctx, conn, "fetch-insiders")
	require.NoError(t, err)
	require.Nil(t, run)
}
