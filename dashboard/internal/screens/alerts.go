// Package screens contains one bubbletea model per dashboard screen.
package screens

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nkim500/market-digest/dashboard/internal/theme"
	"github.com/nkim500/market-digest/dashboard/internal/viewport"
	"github.com/nkim500/market-digest/internal/data"
)

type AlertsLoadedMsg struct {
	Rows []data.AlertRow
	Err  error
}

type AlertsModel struct {
	Conn     *sql.DB
	Rows     []data.AlertRow
	Viewport viewport.Model
	Width    int
	Height   int
	Error    string
}

func NewAlertsModel(conn *sql.DB) AlertsModel {
	return AlertsModel{Conn: conn, Viewport: viewport.New()}
}

func (m AlertsModel) Init() tea.Cmd { return m.loadCmd() }

func (m AlertsModel) loadCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := data.RecentAlerts(context.Background(), m.Conn, 200)
		return AlertsLoadedMsg{Rows: rows, Err: err}
	}
}

func (m AlertsModel) Update(msg tea.Msg) (AlertsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case AlertsLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
			return m, nil
		}
		m.Rows = msg.Rows
		m.Viewport = m.Viewport.SetRows(renderAlertRows(m.Rows))
		return m, nil
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		m.Viewport = m.Viewport.SetHeight(m.Height - 4) // header + footer reserve
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "x":
			if m.Viewport.Cursor() < len(m.Rows) {
				id := m.Rows[m.Viewport.Cursor()].ID
				return m, func() tea.Msg {
					if err := data.MarkSeen(context.Background(), m.Conn, id); err != nil {
						return AlertsLoadedMsg{Err: err}
					}
					rows, err := data.RecentAlerts(context.Background(), m.Conn, 200)
					return AlertsLoadedMsg{Rows: rows, Err: err}
				}
			}
		case "r":
			return m, m.loadCmd()
		}
	}
	// Delegate nav keys to viewport.
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}

func (m AlertsModel) View() string {
	var b strings.Builder
	b.WriteString(theme.Header.Render("Alerts") + "\n\n")
	if m.Error != "" {
		b.WriteString("ERROR: " + m.Error + "\n")
		return b.String()
	}
	if len(m.Rows) == 0 {
		b.WriteString("  (no alerts)\n")
		return b.String()
	}
	b.WriteString(m.Viewport.View())
	b.WriteString("\n" + theme.Footer.Render("↑/↓ j/k move · PgUp/PgDn page · x mark seen · r reload · 1-4 screens · q quit") + "\n")
	return b.String()
}

func renderAlertRows(rows []data.AlertRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		txnStr := "—"
		if r.TransactionTS != nil {
			txnStr = time.Unix(*r.TransactionTS, 0).UTC().Format("2006-01-02")
		}
		alertStr := time.Unix(r.CreatedTS, 0).UTC().Format("01-02 15:04")
		line := fmt.Sprintf("  txn %-10s  alert %-11s  %-8s  %-6s  %-6s  %s",
			txnStr, alertStr, r.Source, r.Severity, r.Ticker, r.Title)
		line = theme.SeverityStyle(r.Severity).Render(line)
		if r.Seen() {
			line = theme.Seen.Render(line)
		}
		out[i] = line
	}
	return out
}

// SelectedBody returns the markdown body of the cursor's row, if any.
func (m AlertsModel) SelectedBody() string {
	if m.Viewport.Cursor() >= len(m.Rows) {
		return ""
	}
	return m.Rows[m.Viewport.Cursor()].Body
}
