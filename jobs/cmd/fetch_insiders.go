package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	httpOpts := insiders.ClientOptions{
		Timeout:    time.Duration(cfg.Sources.HTTP.TimeoutSeconds) * time.Second,
		MaxRetries: cfg.Sources.HTTP.MaxRetries,
		BackoffMS:  cfg.Sources.HTTP.BackoffMS,
	}

	lookbackDays := cfg.Sources.MaxLookbackDays
	if lookbackDays <= 0 {
		lookbackDays = 7
	}
	cutoff := time.Now().Add(-time.Duration(lookbackDays) * 24 * time.Hour).Unix()

	var all []insiders.Trade

	// --- SEC Form 4 ---
	if src, ok := cfg.Sources.Insiders["sec_form4"]; ok && src.Enabled {
		ua := os.Getenv(src.UserAgentEnv)
		if ua == "" || strings.Contains(ua, "your-email@example.com") {
			return 0, 0, fmt.Errorf("SEC user agent required: set %s to a string containing your contact email", src.UserAgentEnv)
		}
		edgarBase := os.Getenv("DIGEST_EDGAR_BASE_URL")
		if edgarBase == "" {
			edgarBase = "https://www.sec.gov"
		}
		cikSrc := os.Getenv("DIGEST_CIK_SOURCE_URL")
		if cikSrc == "" {
			cikSrc = "https://www.sec.gov/files/company_tickers.json"
		}
		cik := insiders.NewCIKCache(insiders.CIKCacheOptions{
			SourceURL: cikSrc,
			CachePath: filepath.Join(home, "data", "cik_cache.json"),
			UserAgent: ua,
		})
		opts := httpOpts
		opts.UserAgent = ua
		client := insiders.NewClient(opts)
		for _, wl := range cfg.Watchlist.Tickers {
			cikStr, err := cik.Resolve(ctx, wl.Ticker)
			if err != nil {
				if err == insiders.ErrCIKNotFound {
					continue
				}
				return 0, 0, fmt.Errorf("cik resolve %s: %w", wl.Ticker, err)
			}
			watermark := resolveWatermark(ctx, conn, "sec-form4", wl.Ticker)
			effectiveCutoff := cutoff
			if watermark > cutoff {
				effectiveCutoff = watermark
			}
			trades, err := client.FetchForm4(ctx, edgarBase, cikStr, effectiveCutoff)
			if err != nil {
				return 0, 0, fmt.Errorf("form4 %s: %w", wl.Ticker, err)
			}
			all = append(all, trades...)
		}
	}

	// --- Finnhub Congressional ---
	if src, ok := cfg.Sources.Insiders["finnhub"]; ok && src.Enabled {
		key := os.Getenv(src.APIKeyEnv)
		if key == "" {
			fmt.Fprintf(os.Stderr, "[finnhub] %s unset; skipping source\n", src.APIKeyEnv)
		} else {
			base := os.Getenv("DIGEST_FINNHUB_BASE_URL")
			if base == "" {
				base = "https://finnhub.io/api/v1"
			}
			client := insiders.NewClient(httpOpts)
			for _, wl := range cfg.Watchlist.Tickers {
				watermark := resolveWatermark(ctx, conn, "finnhub", wl.Ticker)
				effectiveCutoff := cutoff
				if watermark > cutoff {
					effectiveCutoff = watermark
				}
				trades, err := client.FetchFinnhubCongressional(ctx, base, key, wl.Ticker, effectiveCutoff)
				if err != nil {
					return 0, 0, fmt.Errorf("finnhub %s: %w", wl.Ticker, err)
				}
				all = append(all, trades...)
			}
		}
	}

	// --- QuiverQuant Congressional ---
	if src, ok := cfg.Sources.Insiders["quiver"]; ok && src.Enabled {
		key := os.Getenv(src.APIKeyEnv)
		if key == "" {
			fmt.Fprintf(os.Stderr, "[quiver] %s unset; skipping source\n", src.APIKeyEnv)
		} else {
			base := os.Getenv("DIGEST_QUIVER_BASE_URL")
			if base == "" {
				base = "https://api.quiverquant.com/beta"
			}
			client := insiders.NewClient(httpOpts)
			for _, wl := range cfg.Watchlist.Tickers {
				watermark := resolveWatermark(ctx, conn, "quiver", wl.Ticker)
				effectiveCutoff := cutoff
				if watermark > cutoff {
					effectiveCutoff = watermark
				}
				trades, err := client.FetchQuiverCongressional(ctx, base, key, wl.Ticker, effectiveCutoff)
				if err != nil {
					return 0, 0, fmt.Errorf("quiver %s: %w", wl.Ticker, err)
				}
				all = append(all, trades...)
			}
		}
	}

	rowsIn = len(all)

	result, err := insiders.StoreInserts(ctx, conn, all)
	if err != nil {
		return rowsIn, 0, err
	}
	rowsNew = len(result.IDs)

	if _, err := insiders.EvaluateRules(ctx, conn, result.Trades, cfg.Sources, cfg.Profile); err != nil {
		return rowsIn, rowsNew, err
	}
	return rowsIn, rowsNew, nil
}

func resolveWatermark(ctx context.Context, conn *sql.DB, source, ticker string) int64 {
	var ts sql.NullInt64
	_ = conn.QueryRowContext(ctx,
		`SELECT MAX(filing_ts) FROM insider_trades WHERE source = ? AND ticker = ?`,
		source, ticker,
	).Scan(&ts)
	if ts.Valid {
		return ts.Int64
	}
	return 0
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
	Short: "Fetch insider trades (SEC Form 4, Finnhub, QuiverQuant), dedup, write alerts",
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
