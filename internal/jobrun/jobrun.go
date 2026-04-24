// Package jobrun wraps a job invocation in a job_runs row, capturing
// start/finish timestamps, result counts, errors, and panics.
package jobrun

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime/debug"
	"time"
)

// ErrNoop lets a job signal "nothing new to do, success." Reported as
// status='noop' in job_runs.
var ErrNoop = errors.New("jobrun: no-op")

// Fn is the job body. Returns (rows_in, rows_new, err).
type Fn func(ctx context.Context) (rowsIn, rowsNew int, err error)

// Track inserts a row with status='running', invokes fn, and updates the row
// with the final status + counts on exit. Panics are captured as status='error'
// and re-raised after the DB is updated.
func Track(ctx context.Context, conn *sql.DB, job string, fn Fn) (retErr error) {
	res, err := conn.ExecContext(ctx,
		`INSERT INTO job_runs (job, started_ts, status) VALUES (?, ?, 'running')`,
		job, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert job_runs: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last id: %w", err)
	}

	var (
		rowsIn  int
		rowsNew int
		runErr  error
		panicV  any
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicV = r
			}
		}()
		rowsIn, rowsNew, runErr = fn(ctx)
	}()

	status := "ok"
	var errText sql.NullString
	switch {
	case panicV != nil:
		status = "error"
		errText = sql.NullString{String: fmt.Sprintf("panic: %v\n%s", panicV, debug.Stack()), Valid: true}
	case errors.Is(runErr, ErrNoop):
		status = "noop"
	case runErr != nil:
		status = "error"
		errText = sql.NullString{String: runErr.Error(), Valid: true}
	}

	if _, err := conn.ExecContext(ctx,
		`UPDATE job_runs
		 SET finished_ts=?, status=?, rows_in=?, rows_new=?, error=?
		 WHERE id=?`,
		time.Now().Unix(), status, rowsIn, rowsNew, errText, id,
	); err != nil {
		fmt.Printf("jobrun: finalize failed: %v\n", err)
	}

	if panicV != nil {
		panic(panicV) // preserve original panic for process exit
	}
	if runErr != nil && !errors.Is(runErr, ErrNoop) {
		return runErr
	}
	return nil
}
