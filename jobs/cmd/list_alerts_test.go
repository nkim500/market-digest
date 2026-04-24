package cmd_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/jobs/cmd"
)

func TestListAlerts_filtersUnseenAndSince(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	ticker := "NVDA"
	_, err = alert.Insert(ctx, conn, alert.Alert{Source: "insiders", Severity: "watch", Ticker: &ticker, Title: "fresh unseen"})
	require.NoError(t, err)
	oldID, err := alert.Insert(ctx, conn, alert.Alert{Source: "insiders", Severity: "info", Title: "old"})
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, `UPDATE alerts SET created_ts=? WHERE id=?`,
		time.Now().Add(-30*24*time.Hour).Unix(), oldID)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = cmd.ListAlerts(ctx, conn, cmd.ListAlertsOptions{Unseen: true, Since: 7 * 24 * time.Hour}, &buf)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "fresh unseen")
	require.NotContains(t, buf.String(), "old")
}
