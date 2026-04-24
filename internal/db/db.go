// Package db owns sqlite open + migrations for market-digest.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open opens (and creates if needed) a sqlite database at path, returning a
// *sql.DB configured with WAL journaling and a 5s busy_timeout so the TUI can
// read concurrently with writes from the jobs binary.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	q := url.Values{}
	q.Set("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(ON)")
	dsn := fmt.Sprintf("file:%s?%s", path, q.Encode())

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	// modernc.org/sqlite accepts concurrent reads + a single writer.
	// One connection is the simplest correct choice at this scale.
	conn.SetMaxOpenConns(1)
	return conn, nil
}
