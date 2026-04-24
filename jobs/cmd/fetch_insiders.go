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

// RunFetchInsiders is exported so end-to-end tests can call it with a temp
// home without going through the cobra command.
func RunFetchInsiders(ctx context.Context, home string) (rowsIn, rowsNew int, err error) {
	cfg, err := config.Load(home)
	if err != nil {
		return 0, 0, err
	}
	conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
	if err != nil {
		return 0, 0, err
	}
	defer conn.Close()
	if _, err := db.Migrate(ctx, conn, filepath.Join(home, "migrations")); err != nil {
		return 0, 0, err
	}

	// Sync watchlist YAML -> DB (authored source wins; TUI edits write both).
	if err := syncWatchlist(ctx, conn, cfg.Watchlist); err != nil {
		return 0, 0, fmt.Errorf("sync watchlist: %w", err)
	}

	client := insiders.NewClient(insiders.ClientOptions{
		Timeout:    time.Duration(cfg.Sources.HTTP.TimeoutSeconds) * time.Second,
		MaxRetries: cfg.Sources.HTTP.MaxRetries,
		BackoffMS:  cfg.Sources.HTTP.BackoffMS,
	})

	var all []insiders.Trade
	if src, ok := cfg.Sources.Insiders["senate"]; ok && src.Enabled {
		trades, err := client.FetchSenate(ctx, src.URL)
		if err != nil {
			return 0, 0, fmt.Errorf("senate: %w", err)
		}
		all = append(all, trades...)
	}
	if src, ok := cfg.Sources.Insiders["house"]; ok && src.Enabled {
		trades, err := client.FetchHouse(ctx, src.URL)
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
	for _, entry := range w.Tickers {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO watchlist (ticker, note, added_ts) VALUES (?, ?, ?)",
			entry.Ticker, entry.Note, time.Now().Unix(),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
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
			return RunFetchInsiders(ctx, home)
		})
	},
}

func init() {
	rootCmd.AddCommand(fetchInsidersCmd)
}
