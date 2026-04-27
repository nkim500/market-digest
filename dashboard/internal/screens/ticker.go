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

type TickerModel struct {
	Conn     *sql.DB
	Ticker   string
	Rows     []data.InsiderTradeRow
	Viewport viewport.Model
	Width    int
	Height   int
	Error    string
}

type tickerLoadedMsg struct {
	Rows []data.InsiderTradeRow
	Err  error
}

func NewTickerModel(conn *sql.DB) TickerModel {
	return TickerModel{Conn: conn, Viewport: viewport.New()}
}

func (m *TickerModel) SetTicker(t string) { m.Ticker = t }

func (m TickerModel) Init() tea.Cmd {
	if m.Ticker == "" {
		return nil
	}
	ticker := m.Ticker
	conn := m.Conn
	return func() tea.Msg {
		rows, err := data.RecentInsiderTrades(context.Background(), conn, ticker, 200)
		return tickerLoadedMsg{rows, err}
	}
}

func (m TickerModel) Update(msg tea.Msg) (TickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickerLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
			return m, nil
		}
		m.Rows = msg.Rows
		m.Viewport = m.Viewport.SetRows(renderTickerRows(m.Rows))
		return m, nil
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		m.Viewport = m.Viewport.SetHeight(m.Height - 5)
		return m, nil
	}
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}

func (m TickerModel) View() string {
	var b strings.Builder
	title := "Ticker detail"
	if m.Ticker != "" {
		title = "Ticker detail — " + m.Ticker
	}
	b.WriteString(theme.Header.Render(title) + "\n\n")
	if m.Ticker == "" {
		b.WriteString("  (no ticker selected — pick one from Watchlist screen)\n")
		return b.String()
	}
	if m.Error != "" {
		return b.String() + "ERROR: " + m.Error + "\n"
	}
	b.WriteString("  Recent insider trades:\n")
	if len(m.Rows) == 0 {
		b.WriteString("    (none)\n")
		return b.String()
	}
	b.WriteString(m.Viewport.View())
	b.WriteString("\n" + theme.Footer.Render("↑/↓ j/k move · PgUp/PgDn page · 1-4 screens · q quit") + "\n")
	return b.String()
}

func renderTickerRows(rows []data.InsiderTradeRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = fmt.Sprintf("    %s  %-20s  %-7s  $%d-$%d",
			time.Unix(r.TransactionTS, 0).Format("2006-01-02"),
			r.Filer, r.Side, r.AmountLow, r.AmountHigh,
		)
	}
	return out
}
