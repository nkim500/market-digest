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

type WatchlistModel struct {
	Conn   *sql.DB
	Rows   []data.WatchlistRow
	Cursor int
	Error  string
}

type watchlistLoadedMsg struct {
	Rows []data.WatchlistRow
	Err  error
}

func NewWatchlistModel(conn *sql.DB) WatchlistModel {
	return WatchlistModel{Conn: conn}
}

func (m WatchlistModel) Init() tea.Cmd {
	return func() tea.Msg {
		rows, err := data.Watchlist(context.Background(), m.Conn)
		return watchlistLoadedMsg{rows, err}
	}
}

func (m WatchlistModel) Update(msg tea.Msg) (WatchlistModel, tea.Cmd) {
	switch msg := msg.(type) {
	case watchlistLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
		} else {
			m.Rows = msg.Rows
		}
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
		case "r":
			return m, m.Init()
		}
	}
	return m, nil
}

func (m WatchlistModel) View() string {
	var b strings.Builder
	b.WriteString(theme.Header.Render("Watchlist") + "\n\n")
	if m.Error != "" {
		return b.String() + "ERROR: " + m.Error + "\n"
	}
	if len(m.Rows) == 0 {
		b.WriteString("  (empty — edit config/watchlist.yml and run `jobs migrate`)\n")
		return b.String()
	}
	for i, r := range m.Rows {
		line := fmt.Sprintf("  %-6s  alerts(30d):%-3d  trades(30d):%-3d  added:%s  %s",
			r.Ticker, r.AlertCount, r.TradesCount,
			time.Unix(r.AddedTS, 0).Format("2006-01-02"), r.Note)
		if i == m.Cursor {
			line = theme.Cursor.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + theme.Footer.Render("r reload · enter → ticker detail · 1-4 screens · q quit") + "\n")
	return b.String()
}

func (m WatchlistModel) SelectedTicker() string {
	if m.Cursor >= len(m.Rows) {
		return ""
	}
	return m.Rows[m.Cursor].Ticker
}
