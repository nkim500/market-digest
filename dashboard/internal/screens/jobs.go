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

type JobsModel struct {
	Conn  *sql.DB
	Rows  []data.JobRunRow
	Error string
}

type jobsLoadedMsg struct {
	Rows []data.JobRunRow
	Err  error
}

func NewJobsModel(conn *sql.DB) JobsModel {
	return JobsModel{Conn: conn}
}

func (m JobsModel) Init() tea.Cmd {
	return func() tea.Msg {
		rows, err := data.RecentJobRuns(context.Background(), m.Conn, 20)
		return jobsLoadedMsg{rows, err}
	}
}

func (m JobsModel) Update(msg tea.Msg) (JobsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case jobsLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
		} else {
			m.Rows = msg.Rows
		}
	case tea.KeyMsg:
		if msg.String() == "r" {
			return m, m.Init()
		}
	}
	return m, nil
}

func (m JobsModel) View() string {
	var b strings.Builder
	b.WriteString(theme.Header.Render("Jobs") + "\n\n")
	if m.Error != "" {
		return b.String() + "ERROR: " + m.Error + "\n"
	}
	for _, r := range m.Rows {
		dur := "(running)"
		if r.FinishedTS > 0 {
			dur = time.Unix(r.FinishedTS, 0).Sub(time.Unix(r.StartedTS, 0)).Truncate(time.Second).String()
		}
		errTxt := ""
		if r.Error != "" {
			errTxt = " ERR=" + truncate(r.Error, 60)
		}
		line := fmt.Sprintf("  %s  %-18s  %-6s  %5s  in=%-5d new=%-5d%s",
			time.Unix(r.StartedTS, 0).Format("2006-01-02 15:04"),
			r.Job, r.Status, dur, r.RowsIn, r.RowsNew, errTxt)
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + theme.Footer.Render("r reload · 1-4 screens · q quit") + "\n")
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
