package cmd

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/config"
	"github.com/nkim500/market-digest/internal/db"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check DB, config, and source reachability",
	// RunE always returns nil so exit code is 0 even when individual components
	// fail — this is a diagnostic command; callers want the full report.
	RunE: func(cmd *cobra.Command, args []string) error {
		home := digestHome()
		ctx := context.Background()

		fmt.Printf("DIGEST_HOME: %s\n", home)

		// Config
		cfg, err := config.Load(home)
		configOK := err == nil
		if err != nil {
			fmt.Printf("config: FAIL — %v\n", err)
		} else {
			fmt.Printf("config: ok (user=%q, sources=%d)\n", cfg.Profile.User.DisplayName, len(cfg.Sources.Insiders))
		}

		// DB
		dbPath := filepath.Join(home, "data", "digest.db")
		conn, err := db.Open(ctx, dbPath)
		if err != nil {
			fmt.Printf("db: FAIL — %v\n", err)
		} else {
			defer conn.Close()
			var n int
			if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master").Scan(&n); err != nil {
				fmt.Printf("db: FAIL (schema count) — %v\n", err)
			} else {
				fmt.Printf("db: ok (%s, %d schema objects)\n", dbPath, n)
			}
		}

		if !configOK {
			fmt.Println("sources: skipped (config unavailable)")
			return nil
		}

		// Sources (HEAD with short timeout)
		client := &http.Client{Timeout: 10 * time.Second}
		for name, src := range cfg.Sources.Insiders {
			if !src.Enabled {
				fmt.Printf("source %s: disabled\n", name)
				continue
			}
			req, _ := http.NewRequestWithContext(ctx, http.MethodHead, src.URL, nil)
			if src.UserAgent != "" {
				req.Header.Set("User-Agent", src.UserAgent)
			}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("source %s: FAIL — %v\n", name, err)
				continue
			}
			resp.Body.Close()
			fmt.Printf("source %s: %s (HTTP %d)\n", name, src.URL, resp.StatusCode)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
