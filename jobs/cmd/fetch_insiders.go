package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/config"
	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/internal/insiders"
	"github.com/nkim500/market-digest/internal/jobrun"
)

// RunFetchInsiders opens its own DB connection and runs the full fetch pipeline.
// Exported so end-to-end tests can call it with a temp home without going
// through the cobra command. The cobra command path reuses its already-open
// conn via runFetchInsidersWithConn to avoid stacking two *sql.DB handles
// against the same sqlite file (each handle keeps its own MaxOpenConns(1)
// pool).
func RunFetchInsiders(ctx context.Context, home string) (rowsIn, rowsNew int, err error) {
	conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
	if err != nil {
		return 0, 0, err
	}
	defer conn.Close()
	if _, err := db.Migrate(ctx, conn, filepath.Join(home, "migrations")); err != nil {
		return 0, 0, err
	}
	return runFetchInsidersWithConn(ctx, conn, home)
}

// runFetchInsidersWithConn is the internal entry point that does the work on
// a caller-provided connection. Callers are responsible for having already
// run migrations.
func runFetchInsidersWithConn(ctx context.Context, conn *sql.DB, home string) (rowsIn, rowsNew int, err error) {
	cfg, err := config.Load(home)
	if err != nil {
		return 0, 0, err
	}

	// Sync watchlist YAML -> DB (YAML is the authored source of truth).
	if err := syncWatchlist(ctx, conn, cfg.Watchlist); err != nil {
		return 0, 0, fmt.Errorf("sync watchlist: %w", err)
	}

	// Per-source clients so each can carry its own User-Agent header. Senate
	// and House don't need one; SEC EDGAR requires it.
	newClient := func(ua string) *insiders.Client {
		return insiders.NewClient(insiders.ClientOptions{
			Timeout:    time.Duration(cfg.Sources.HTTP.TimeoutSeconds) * time.Second,
			MaxRetries: cfg.Sources.HTTP.MaxRetries,
			BackoffMS:  cfg.Sources.HTTP.BackoffMS,
			UserAgent:  ua,
		})
	}

	var all []insiders.Trade
	if src, ok := cfg.Sources.Insiders["senate"]; ok && src.Enabled {
		trades, err := newClient(src.UserAgent).FetchSenate(ctx, src.URL)
		if err != nil {
			return 0, 0, fmt.Errorf("senate: %w", err)
		}
		all = append(all, trades...)
	}
	if src, ok := cfg.Sources.Insiders["house"]; ok && src.Enabled {
		trades, err := newClient(src.UserAgent).FetchHouse(ctx, src.URL)
		if err != nil {
			return 0, 0, fmt.Errorf("house: %w", err)
		}
		all = append(all, trades...)
	}
	rowsIn = len(all)

	result, err := insiders.StoreInserts(ctx, conn, all)
	if err != nil {
		return rowsIn, 0, err
	}
	rowsNew = len(result.IDs)

	// Evaluate rules only against newly inserted trades.
	if _, err := insiders.EvaluateRules(ctx, conn, result.Trades, cfg.Sources, cfg.Profile); err != nil {
		return rowsIn, rowsNew, err
	}
	return rowsIn, rowsNew, nil
}

func syncWatchlist(ctx context.Context, conn *sql.DB, w config.Watchlist) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM watchlist"); err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, entry := range w.Tickers {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO watchlist (ticker, note, added_ts) VALUES (?, ?, ?)",
			entry.Ticker, entry.Note, parseAddedTS(entry.Added, now),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// parseAddedTS turns the user-authored `added: YYYY-MM-DD` string into a unix
// timestamp. Falls back to `now` if the string is empty or unparseable so a
// forgotten/malformed field doesn't block the sync.
func parseAddedTS(raw string, now int64) int64 {
	if raw == "" {
		return now
	}
	for _, layout := range []string{"2006-01-02", "01/02/2006", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Unix()
		}
	}
	return now
}

var fetchInsidersCmd = &cobra.Command{
	Use:   "fetch-insiders",
	Short: "Fetch Senate + House political trades, dedup, write alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		home := digestHome()
		conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
		if err != nil {
			return err
		}
		defer conn.Close()
		if _, err := db.Migrate(ctx, conn, filepath.Join(home, "migrations")); err != nil {
			return err
		}
		return jobrun.Track(ctx, conn, "fetch-insiders", func(ctx context.Context) (int, int, error) {
			return runFetchInsidersWithConn(ctx, conn, home)
		})
	},
}

func init() {
	rootCmd.AddCommand(fetchInsidersCmd)
}
