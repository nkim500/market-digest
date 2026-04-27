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

type WatchlistModel struct {
	Conn     *sql.DB
	Rows     []data.WatchlistRow
	Viewport viewport.Model
	Width    int
	Height   int
	Error    string
}

type watchlistLoadedMsg struct {
	Rows []data.WatchlistRow
	Err  error
}

func NewWatchlistModel(conn *sql.DB) WatchlistModel {
	return WatchlistModel{Conn: conn, Viewport: viewport.New()}
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
			return m, nil
		}
		m.Rows = msg.Rows
		m.Viewport = m.Viewport.SetRows(renderWatchlistRows(m.Rows))
		return m, nil
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		m.Viewport = m.Viewport.SetHeight(m.Height - 4)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "r" {
			return m, m.Init()
		}
	}
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}

func (m WatchlistModel) View() string {
	var b strings.Builder
	b.WriteString(theme.Header.Render("Watchlist") + "\n\n")
	if m.Error != "" {
		return b.String() + "ERROR: " + m.Error + "\n"
	}
	if len(m.Rows) == 0 {
		b.WriteString("  (empty — edit config/watchlist.yml and run `jobs fetch-insiders`)\n")
		return b.String()
	}
	b.WriteString(m.Viewport.View())
	b.WriteString("\n" + theme.Footer.Render("↑/↓ j/k move · PgUp/PgDn page · r reload · enter → ticker detail · 1-4 screens · q quit") + "\n")
	return b.String()
}

func renderWatchlistRows(rows []data.WatchlistRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = fmt.Sprintf("  %-6s  alerts(30d):%-3d  trades(30d):%-3d  added:%s  %s",
			r.Ticker, r.AlertCount, r.TradesCount,
			time.Unix(r.AddedTS, 0).Format("2006-01-02"), r.Note)
	}
	return out
}

func (m WatchlistModel) SelectedTicker() string {
	if m.Viewport.Cursor() >= len(m.Rows) {
		return ""
	}
	return m.Rows[m.Viewport.Cursor()].Ticker
}
