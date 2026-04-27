package data_test

import (
	"context"
	"path/filepath"
	"strings"
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

func TestRecentAlertsExtractsTransactionTSFromPayload(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	// Seed an alert with transaction_ts in the payload.
	txnTS := int64(1713897600)
	_, err = alert.Insert(ctx, conn, alert.Alert{
		Source:   "insiders",
		Severity: "watch",
		Title:    "test",
		Body:     "body",
		Payload:  map[string]any{"transaction_ts": txnTS},
	})
	require.NoError(t, err)

	// And a cluster-style alert with no transaction_ts in the payload.
	_, err = conn.ExecContext(ctx, `
		INSERT INTO alerts (source, severity, title, body, payload, created_ts)
		VALUES ('insiders', 'watch', 'Cluster: 3 filers on NVDA in last 7d', '', '{}', ?)
	`, time.Now().Unix())
	require.NoError(t, err)

	rows, err := data.RecentAlerts(ctx, conn, 100)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	var hasTxn, hasCluster bool
	for _, r := range rows {
		if r.Title == "test" {
			hasTxn = true
			if r.TransactionTS == nil || *r.TransactionTS != txnTS {
				t.Errorf("expected TransactionTS=%d, got %v", txnTS, r.TransactionTS)
			}
		}
		if strings.HasPrefix(r.Title, "Cluster:") {
			hasCluster = true
			if r.TransactionTS != nil {
				t.Errorf("cluster alert should have nil TransactionTS, got %v", *r.TransactionTS)
			}
		}
	}
	if !hasTxn || !hasCluster {
		t.Error("missing expected rows")
	}
}
