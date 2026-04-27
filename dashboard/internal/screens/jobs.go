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

type JobsModel struct {
	Conn     *sql.DB
	Rows     []data.JobRunRow
	Viewport viewport.Model
	Width    int
	Height   int
	Error    string
}

type jobsLoadedMsg struct {
	Rows []data.JobRunRow
	Err  error
}

func NewJobsModel(conn *sql.DB) JobsModel {
	return JobsModel{Conn: conn, Viewport: viewport.New()}
}

func (m JobsModel) Init() tea.Cmd {
	return func() tea.Msg {
		rows, err := data.RecentJobRuns(context.Background(), m.Conn, 100)
		return jobsLoadedMsg{rows, err}
	}
}

func (m JobsModel) Update(msg tea.Msg) (JobsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case jobsLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
			return m, nil
		}
		m.Rows = msg.Rows
		m.Viewport = m.Viewport.SetRows(renderJobsRows(m.Rows))
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

func (m JobsModel) View() string {
	var b strings.Builder
	b.WriteString(theme.Header.Render("Jobs") + "\n\n")
	if m.Error != "" {
		return b.String() + "ERROR: " + m.Error + "\n"
	}
	if len(m.Rows) == 0 {
		b.WriteString("  (no job runs yet)\n")
		return b.String()
	}
	b.WriteString(m.Viewport.View())
	b.WriteString("\n" + theme.Footer.Render("↑/↓ j/k move · PgUp/PgDn page · r reload · 1-4 screens · q quit") + "\n")
	return b.String()
}

func renderJobsRows(rows []data.JobRunRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		started := time.Unix(r.StartedTS, 0).Format("2006-01-02 15:04")
		out[i] = fmt.Sprintf("  %s  %-20s  %-6s  in:%-5d new:%-5d  %s",
			started, r.Job, r.Status, r.RowsIn, r.RowsNew, r.Error)
	}
	return out
}
