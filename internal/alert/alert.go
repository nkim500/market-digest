// Package alert writes rows to the alerts table. Consumers pass a typed Alert;
// payload is serialized to JSON.
package alert

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Severity = string

const (
	Info  Severity = "info"
	Watch Severity = "watch"
	Act   Severity = "act"
)

type Alert struct {
	Source   string
	Severity Severity
	Ticker   *string // nullable
	Title    string
	Body     string
	Payload  map[string]any
}

func Insert(ctx context.Context, conn *sql.DB, a Alert) (int64, error) {
	switch a.Severity {
	case Info, Watch, Act:
	default:
		return 0, fmt.Errorf("alert: invalid severity %q", a.Severity)
	}
	if a.Source == "" || a.Title == "" {
		return 0, fmt.Errorf("alert: source and title required")
	}

	var payload sql.NullString
	if a.Payload != nil {
		b, err := json.Marshal(a.Payload)
		if err != nil {
			return 0, fmt.Errorf("marshal payload: %w", err)
		}
		payload = sql.NullString{String: string(b), Valid: true}
	}
	var ticker sql.NullString
	if a.Ticker != nil {
		ticker = sql.NullString{String: *a.Ticker, Valid: true}
	}

	res, err := conn.ExecContext(ctx,
		`INSERT INTO alerts (created_ts, source, severity, ticker, title, body, payload)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), a.Source, a.Severity, ticker, a.Title, a.Body, payload,
	)
	if err != nil {
		return 0, fmt.Errorf("insert alert: %w", err)
	}
	return res.LastInsertId()
}
