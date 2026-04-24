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
	"github.com/nkim500/market-digest/internal/data"
)

type AlertsLoadedMsg struct {
	Rows []data.AlertRow
	Err  error
}

type AlertsModel struct {
	Conn   *sql.DB
	Rows   []data.AlertRow
	Cursor int
	Width  int
	Height int
	Error  string
}

func NewAlertsModel(conn *sql.DB) AlertsModel {
	return AlertsModel{Conn: conn}
}

func (m AlertsModel) Init() tea.Cmd { return m.loadCmd() }

func (m AlertsModel) loadCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := data.RecentAlerts(context.Background(), m.Conn, 100)
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
		if m.Cursor >= len(m.Rows) {
			m.Cursor = 0
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Rows)-1 {
				m.Cursor++
			}
		case "x":
			if m.Cursor < len(m.Rows) {
				id := m.Rows[m.Cursor].ID
				return m, func() tea.Msg {
					if err := data.MarkSeen(context.Background(), m.Conn, id); err != nil {
						return AlertsLoadedMsg{Err: err}
					}
					rows, err := data.RecentAlerts(context.Background(), m.Conn, 100)
					return AlertsLoadedMsg{Rows: rows, Err: err}
				}
			}
		case "r":
			return m, m.loadCmd()
		}
	}
	return m, nil
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
	for i, r := range m.Rows {
		line := fmt.Sprintf("  %-16s  %-8s  %-8s  %-6s  %s",
			time.Unix(r.CreatedTS, 0).Format("2006-01-02 15:04"),
			r.Source, r.Severity, r.Ticker, r.Title)
		line = theme.SeverityStyle(r.Severity).Render(line)
		if r.Seen() {
			line = theme.Seen.Render(line)
		}
		if i == m.Cursor {
			line = theme.Cursor.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + theme.Footer.Render("↑/↓ move · x mark seen · r reload · 1-4 screens · q quit") + "\n")
	return b.String()
}

// SelectedBody returns the markdown body of the currently-selected row, if any.
func (m AlertsModel) SelectedBody() string {
	if m.Cursor >= len(m.Rows) {
		return ""
	}
	return m.Rows[m.Cursor].Body
}
