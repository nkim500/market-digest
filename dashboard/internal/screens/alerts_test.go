package screens_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/dashboard/internal/screens"
	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/db"
)

// testDB opens a fresh temp-file SQLite database with migrations applied.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	_, err = db.Migrate(ctx, conn, "../../../migrations")
	require.NoError(t, err)
	return conn
}

func updateSync(t *testing.T, m screens.AlertsModel, cmd tea.Cmd) screens.AlertsModel {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	m, _ = m.Update(msg)
	return m
}

func TestAlertsModel_markSeenDecrementsUnseen(t *testing.T) {
	ctx := context.Background()
	conn := testDB(t) // uses shared testDB helper above

	_, err := alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "watch", Title: "one"})
	require.NoError(t, err)
	_, err = alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "info", Title: "two"})
	require.NoError(t, err)

	m := screens.NewAlertsModel(conn)
	cmd := m.Init()
	msg := cmd()
	m, _ = m.Update(msg)
	require.Len(t, m.Rows, 2)

	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd)
	msg = cmd()
	m, _ = m.Update(msg)

	seenCount := 0
	for _, r := range m.Rows {
		if r.Seen() {
			seenCount++
		}
	}
	require.Equal(t, 1, seenCount)
}

func TestAlertsViewShowsDualDates(t *testing.T) {
	ctx := context.Background()
	conn := testDB(t)

	txnTS := int64(1712707200)  // 2024-04-10 00:00 UTC
	alertTS := int64(1713943860) // 2024-04-24 07:31 UTC
	_, err := conn.ExecContext(ctx, `
		INSERT INTO alerts (source, severity, ticker, title, body, payload, created_ts)
		VALUES ('insiders', 'watch', 'NVDA', 'Pelosi buy NVDA', '', ?, ?)
	`, fmt.Sprintf(`{"transaction_ts":%d}`, txnTS), alertTS)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	m := screens.NewAlertsModel(conn)
	m = updateSync(t, m, m.Init())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := m.View()

	if !strings.Contains(view, "txn 2024-04-10") {
		t.Errorf("view should contain 'txn 2024-04-10'; got:\n%s", view)
	}
	if !strings.Contains(view, "alert 04-24 07:31") {
		t.Errorf("view should contain 'alert 04-24 07:31'; got:\n%s", view)
	}
}

func TestAlertsViewClusterShowsDashForTxn(t *testing.T) {
	ctx := context.Background()
	conn := testDB(t)

	_, err := conn.ExecContext(ctx, `
		INSERT INTO alerts (source, severity, ticker, title, body, payload, created_ts)
		VALUES ('insiders', 'watch', 'NVDA', 'Cluster: 4 filers on NVDA in last 7d', '', '{}', ?)
	`, time.Now().Unix())
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	m := screens.NewAlertsModel(conn)
	m = updateSync(t, m, m.Init())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := m.View()

	if !strings.Contains(view, "txn —") {
		t.Errorf("cluster row should render 'txn —'; got:\n%s", view)
	}
}
