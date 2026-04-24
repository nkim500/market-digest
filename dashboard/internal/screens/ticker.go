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

type TickerModel struct {
	Conn   *sql.DB
	Ticker string
	Rows   []data.InsiderTradeRow
	Error  string
}

type tickerLoadedMsg struct {
	Rows []data.InsiderTradeRow
	Err  error
}

func NewTickerModel(conn *sql.DB) TickerModel {
	return TickerModel{Conn: conn}
}

// SetTicker lets the root model pass a ticker in when switching screens.
func (m *TickerModel) SetTicker(t string) { m.Ticker = t }

func (m TickerModel) Init() tea.Cmd {
	if m.Ticker == "" {
		return nil
	}
	ticker := m.Ticker
	conn := m.Conn
	return func() tea.Msg {
		rows, err := data.RecentInsiderTrades(context.Background(), conn, ticker, 50)
		return tickerLoadedMsg{rows, err}
	}
}

func (m TickerModel) Update(msg tea.Msg) (TickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickerLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
		} else {
			m.Rows = msg.Rows
		}
	}
	return m, nil
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
	b.WriteString("  Price pane: momentum mode not yet implemented.\n\n")
	b.WriteString("  Recent insider trades:\n")
	if len(m.Rows) == 0 {
		b.WriteString("    (none)\n")
	}
	for _, r := range m.Rows {
		b.WriteString(fmt.Sprintf("    %s  %-20s  %-7s  $%d-$%d\n",
			time.Unix(r.TransactionTS, 0).Format("2006-01-02"),
			r.Filer, r.Side, r.AmountLow, r.AmountHigh,
		))
	}
	return b.String()
}
