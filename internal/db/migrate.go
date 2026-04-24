package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migrate applies every *.sql file in migrationsDir whose leading integer is
// greater than the max version recorded in schema_version. Returns the list of
// filenames applied, in order.
//
// Filename convention: NNNN_short_name.sql, where NNNN is a zero-padded
// monotonic integer. Filenames without a leading integer are skipped with an
// error.
func Migrate(ctx context.Context, conn *sql.DB, migrationsDir string) ([]string, error) {
	if _, err := conn.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_version (
			version    INTEGER PRIMARY KEY,
			applied_ts INTEGER NOT NULL
		)`,
	); err != nil {
		return nil, fmt.Errorf("bootstrap schema_version: %w", err)
	}

	applied, err := loadAppliedVersions(ctx, conn)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var pending []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		ver, err := parseVersion(e.Name())
		if err != nil {
			return nil, err
		}
		if _, ok := applied[ver]; ok {
			continue
		}
		pending = append(pending, migration{version: ver, filename: e.Name()})
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].version < pending[j].version })

	var appliedNames []string
	for _, m := range pending {
		body, err := os.ReadFile(filepath.Join(migrationsDir, m.filename))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", m.filename, err)
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("begin tx for %s: %w", m.filename, err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("apply %s: %w", m.filename, err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_version (version, applied_ts) VALUES (?, ?)",
			m.version, time.Now().Unix(),
		); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("record %s: %w", m.filename, err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit %s: %w", m.filename, err)
		}
		appliedNames = append(appliedNames, m.filename)
	}
	return appliedNames, nil
}

type migration struct {
	version  int
	filename string
}

func loadAppliedVersions(ctx context.Context, conn *sql.DB) (map[int]struct{}, error) {
	rows, err := conn.QueryContext(ctx, "SELECT version FROM schema_version")
	if err != nil {
		return nil, fmt.Errorf("load applied: %w", err)
	}
	defer rows.Close()
	out := map[int]struct{}{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = struct{}{}
	}
	return out, rows.Err()
}

func parseVersion(filename string) (int, error) {
	underscore := strings.IndexByte(filename, '_')
	if underscore <= 0 {
		return 0, fmt.Errorf("migration %q: no leading version", filename)
	}
	v, err := strconv.Atoi(filename[:underscore])
	if err != nil {
		return 0, fmt.Errorf("migration %q: version: %w", filename, err)
	}
	return v, nil
}
