package alert_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/db"
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

func TestInsert_setsFields(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	ticker := "NVDA"
	id, err := alert.Insert(ctx, conn, alert.Alert{
		Source:   "insiders",
		Severity: "act",
		Ticker:   &ticker,
		Title:    "Senator X bought $1M NVDA",
		Body:     "...",
		Payload:  map[string]any{"filer": "Senator X"},
	})
	require.NoError(t, err)
	require.Positive(t, id)

	var got struct {
		source, severity, title string
		ticker                  sql.NullString
		payload                 string
		seenTS                  sql.NullInt64
	}
	row := conn.QueryRowContext(ctx,
		`SELECT source, severity, ticker, title, payload, seen_ts FROM alerts WHERE id=?`, id,
	)
	require.NoError(t, row.Scan(&got.source, &got.severity, &got.ticker, &got.title, &got.payload, &got.seenTS))
	require.Equal(t, "insiders", got.source)
	require.Equal(t, "act", got.severity)
	require.Equal(t, "NVDA", got.ticker.String)
	require.False(t, got.seenTS.Valid)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(got.payload), &payload))
	require.Equal(t, "Senator X", payload["filer"])
}

func TestInsert_rejectsUnknownSeverity(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)
	_, err := alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "critical", Title: "t"})
	require.Error(t, err)
}
