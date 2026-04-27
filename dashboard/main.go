package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nkim500/market-digest/dashboard/internal/screens"
	"github.com/nkim500/market-digest/dashboard/internal/theme"
	"github.com/nkim500/market-digest/internal/data"
	"github.com/nkim500/market-digest/internal/db"
)

type screen int

const (
	screenAlerts screen = iota + 1
	screenWatchlist
	screenTicker
	screenJobs
)

type root struct {
	conn    *sql.DB
	current screen
	width   int
	height  int
	alerts  screens.AlertsModel
	watch   screens.WatchlistModel
	ticker  screens.TickerModel
	jobs    screens.JobsModel
	dbPath  string
	lastRun *data.JobRunRow
}

func (r root) Init() tea.Cmd {
	return tea.Batch(r.alerts.Init(), r.watch.Init(), r.jobs.Init(), refreshLastRunCmd(r.conn))
}

type lastRunMsg struct{ row *data.JobRunRow }

func refreshLastRunCmd(conn *sql.DB) tea.Cmd {
	return func() tea.Msg {
		row, _ := data.LastJobRun(context.Background(), conn, "fetch-insiders")
		return lastRunMsg{row}
	}
}

func (r root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width, r.height = msg.Width, msg.Height
		// Broadcast to all screens so their viewports size correctly before first visit.
		r.alerts, _ = r.alerts.Update(msg)
		r.watch, _ = r.watch.Update(msg)
		r.ticker, _ = r.ticker.Update(msg)
		r.jobs, _ = r.jobs.Update(msg)
		return r, nil
	case lastRunMsg:
		r.lastRun = msg.row
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return r, tea.Quit
		case "1":
			r.current = screenAlerts
		case "2":
			r.current = screenWatchlist
		case "3":
			r.current = screenTicker
			return r, r.ticker.Init()
		case "4":
			r.current = screenJobs
		case "enter":
			if r.current == screenWatchlist {
				if t := r.watch.SelectedTicker(); t != "" {
					r.ticker.SetTicker(t)
					r.current = screenTicker
					return r, r.ticker.Init()
				}
			}
		}
	}
	switch r.current {
	case screenAlerts:
		m, cmd := r.alerts.Update(msg)
		r.alerts = m
		cmds = append(cmds, cmd)
	case screenWatchlist:
		m, cmd := r.watch.Update(msg)
		r.watch = m
		cmds = append(cmds, cmd)
	case screenTicker:
		m, cmd := r.ticker.Update(msg)
		r.ticker = m
		cmds = append(cmds, cmd)
	case screenJobs:
		m, cmd := r.jobs.Update(msg)
		r.jobs = m
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

func (r root) View() string {
	var body string
	switch r.current {
	case screenAlerts:
		body = r.alerts.View()
	case screenWatchlist:
		body = r.watch.View()
	case screenTicker:
		body = r.ticker.View()
	case screenJobs:
		body = r.jobs.View()
	}
	footerTxt := fmt.Sprintf("digest.db @ %s  ·  last fetch-insiders: %s", r.dbPath, formatLastRun(r.lastRun))
	return body + "\n" + lipgloss.PlaceHorizontal(r.width, lipgloss.Left, theme.Footer.Render(footerTxt))
}

func formatLastRun(r *data.JobRunRow) string {
	if r == nil {
		return "never"
	}
	ago := time.Since(time.Unix(r.StartedTS, 0)).Truncate(time.Minute)
	return fmt.Sprintf("%s ago (%s)", ago, r.Status)
}

func digestHome() string {
	if h := os.Getenv("DIGEST_HOME"); h != "" {
		return h
	}
	cwd, _ := os.Getwd()
	return cwd
}

func main() {
	home := digestHome()
	dbPath := filepath.Join(home, "data", "digest.db")
	ctx := context.Background()
	conn, err := db.Open(ctx, dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dashboard:", err)
		os.Exit(1)
	}
	defer conn.Close()

	r := root{
		conn:    conn,
		current: screenWatchlist,
		dbPath:  dbPath,
		alerts:  screens.NewAlertsModel(conn),
		watch:   screens.NewWatchlistModel(conn),
		ticker:  screens.NewTickerModel(conn),
		jobs:    screens.NewJobsModel(conn),
	}
	if _, err := tea.NewProgram(r, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dashboard:", err)
		os.Exit(1)
	}
}
