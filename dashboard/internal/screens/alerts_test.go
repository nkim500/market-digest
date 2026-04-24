package screens_test

import (
	"context"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/dashboard/internal/screens"
	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/db"
)

func TestAlertsModel_markSeenDecrementsUnseen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../../migrations")
	require.NoError(t, err)

	_, err = alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "watch", Title: "one"})
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
