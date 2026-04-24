package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/db"
)

type ListAlertsOptions struct {
	Unseen bool
	Since  time.Duration
	Source string
}

func ListAlerts(ctx context.Context, conn *sql.DB, opts ListAlertsOptions, w io.Writer) error {
	q := `SELECT id, created_ts, source, severity, COALESCE(ticker, ''), title FROM alerts WHERE 1=1`
	args := []any{}
	if opts.Unseen {
		q += " AND seen_ts IS NULL"
	}
	if opts.Since > 0 {
		q += " AND created_ts >= ?"
		args = append(args, time.Now().Add(-opts.Since).Unix())
	}
	if opts.Source != "" {
		q += " AND source = ?"
		args = append(args, opts.Source)
	}
	q += " ORDER BY created_ts DESC LIMIT 100"

	rows, err := conn.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id         int64
			createdTS  int64
			source     string
			severity   string
			ticker     string
			title      string
		)
		if err := rows.Scan(&id, &createdTS, &source, &severity, &ticker, &title); err != nil {
			return err
		}
		fmt.Fprintf(w, "#%d  %s  %-7s  %-8s  %-6s  %s\n",
			id, time.Unix(createdTS, 0).Format("2006-01-02 15:04"),
			source, severity, ticker, title)
	}
	return rows.Err()
}

var (
	listAlertsUnseen bool
	listAlertsSince  string
	listAlertsSource string
)

var listAlertsCmd = &cobra.Command{
	Use:   "list-alerts",
	Short: "Print recent alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		home := digestHome()
		conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
		if err != nil {
			return err
		}
		defer conn.Close()
		opts := ListAlertsOptions{Unseen: listAlertsUnseen, Source: listAlertsSource}
		if listAlertsSince != "" {
			d, err := time.ParseDuration(listAlertsSince)
			if err != nil {
				return fmt.Errorf("--since: %w", err)
			}
			opts.Since = d
		}
		return ListAlerts(ctx, conn, opts, os.Stdout)
	},
}

func init() {
	listAlertsCmd.Flags().BoolVar(&listAlertsUnseen, "unseen", false, "only unseen alerts")
	listAlertsCmd.Flags().StringVar(&listAlertsSince, "since", "", "time window, e.g. 168h, 24h (Go duration — 'd' not supported)")
	listAlertsCmd.Flags().StringVar(&listAlertsSource, "source", "", "filter by source (insiders|momentum|...)")
	rootCmd.AddCommand(listAlertsCmd)
}
