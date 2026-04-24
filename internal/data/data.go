// Package data contains typed read helpers for the dashboard TUI.
// Writes still go through jobs (except watchlist add/remove and mark-seen).
package data

import (
	"context"
	"database/sql"
	"time"
)

type AlertRow struct {
	ID        int64
	CreatedTS int64
	Source    string
	Severity  string
	Ticker    string
	Title     string
	Body      string
	SeenTS    *int64
}

func (a AlertRow) Time() time.Time { return time.Unix(a.CreatedTS, 0) }
func (a AlertRow) Seen() bool      { return a.SeenTS != nil }

func RecentAlerts(ctx context.Context, conn *sql.DB, limit int) ([]AlertRow, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, created_ts, source, severity, COALESCE(ticker,''), title, COALESCE(body,''), seen_ts
		FROM alerts
		ORDER BY (seen_ts IS NULL) DESC, created_ts DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AlertRow
	for rows.Next() {
		var a AlertRow
		var seen sql.NullInt64
		if err := rows.Scan(&a.ID, &a.CreatedTS, &a.Source, &a.Severity, &a.Ticker, &a.Title, &a.Body, &seen); err != nil {
			return nil, err
		}
		if seen.Valid {
			s := seen.Int64
			a.SeenTS = &s
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type WatchlistRow struct {
	Ticker       string
	Note         string
	AddedTS      int64
	AlertCount   int
	TradesCount  int
}

func Watchlist(ctx context.Context, conn *sql.DB) ([]WatchlistRow, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT
		  w.ticker, COALESCE(w.note,''), w.added_ts,
		  (SELECT COUNT(*) FROM alerts a WHERE a.ticker=w.ticker AND a.created_ts >= strftime('%s','now','-30 days')),
		  (SELECT COUNT(*) FROM insider_trades t WHERE t.ticker=w.ticker AND t.transaction_ts >= strftime('%s','now','-30 days'))
		FROM watchlist w
		ORDER BY w.ticker
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WatchlistRow
	for rows.Next() {
		var r WatchlistRow
		if err := rows.Scan(&r.Ticker, &r.Note, &r.AddedTS, &r.AlertCount, &r.TradesCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type InsiderTradeRow struct {
	ID             int64
	Source, Filer  string
	Role           string
	Ticker         string
	Side           string
	AmountLow      int
	AmountHigh     int
	TransactionTS  int64
	FilingTS       int64
	RawURL         string
}

func RecentInsiderTrades(ctx context.Context, conn *sql.DB, ticker string, limit int) ([]InsiderTradeRow, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, source, filer, COALESCE(role,''), COALESCE(ticker,''), COALESCE(side,''),
		       COALESCE(amount_low,0), COALESCE(amount_high,0),
		       COALESCE(transaction_ts,0), COALESCE(filing_ts,0), COALESCE(raw_url,'')
		FROM insider_trades
		WHERE ticker = ?
		ORDER BY transaction_ts DESC
		LIMIT ?
	`, ticker, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InsiderTradeRow
	for rows.Next() {
		var r InsiderTradeRow
		if err := rows.Scan(&r.ID, &r.Source, &r.Filer, &r.Role, &r.Ticker, &r.Side,
			&r.AmountLow, &r.AmountHigh, &r.TransactionTS, &r.FilingTS, &r.RawURL); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type JobRunRow struct {
	ID                          int64
	Job                         string
	StartedTS, FinishedTS       int64
	Status                      string
	RowsIn, RowsNew             int
	Error                       string
}

func RecentJobRuns(ctx context.Context, conn *sql.DB, limit int) ([]JobRunRow, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, job, started_ts, COALESCE(finished_ts,0), status,
		       COALESCE(rows_in,0), COALESCE(rows_new,0), COALESCE(error,'')
		FROM job_runs
		ORDER BY id DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobRunRow
	for rows.Next() {
		var r JobRunRow
		if err := rows.Scan(&r.ID, &r.Job, &r.StartedTS, &r.FinishedTS, &r.Status,
			&r.RowsIn, &r.RowsNew, &r.Error); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LastJobRun returns the most recent run for a given job, or nil if none.
func LastJobRun(ctx context.Context, conn *sql.DB, job string) (*JobRunRow, error) {
	row := conn.QueryRowContext(ctx, `
		SELECT id, job, started_ts, COALESCE(finished_ts,0), status,
		       COALESCE(rows_in,0), COALESCE(rows_new,0), COALESCE(error,'')
		FROM job_runs WHERE job=? ORDER BY id DESC LIMIT 1
	`, job)
	var r JobRunRow
	if err := row.Scan(&r.ID, &r.Job, &r.StartedTS, &r.FinishedTS, &r.Status,
		&r.RowsIn, &r.RowsNew, &r.Error); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// MarkSeen flips seen_ts.
func MarkSeen(ctx context.Context, conn *sql.DB, alertID int64) error {
	_, err := conn.ExecContext(ctx,
		`UPDATE alerts SET seen_ts=? WHERE id=? AND seen_ts IS NULL`,
		time.Now().Unix(), alertID)
	return err
}
