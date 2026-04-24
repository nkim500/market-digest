package insiders

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/config"
)

// EvaluateRules runs all alert rules against newTrades. Returns the number of
// alerts inserted. Rules:
//
//  1. Per-trade: watchlist hit, amount thresholds. One alert per trade at the
//     highest matching severity.
//  2. Per-run: cluster detection (>= 3 distinct filers on same ticker within
//     profile.Reporting.ClusterWindowDays days). One alert per ticker per UTC
//     day (dedup via title+DATE(created_ts)).
func EvaluateRules(
	ctx context.Context, conn *sql.DB, newTrades []Trade,
	cfg config.Sources, profile config.Profile,
) (int, error) {
	inserted := 0

	watchlist, err := loadWatchlist(ctx, conn)
	if err != nil {
		return 0, err
	}

	for _, t := range newTrades {
		sev := ""
		reasons := []string{}
		if t.Ticker != "" {
			if _, ok := watchlist[t.Ticker]; ok {
				sev = maxSev(sev, cfg.AlertRules.WatchlistHit.Severity)
				reasons = append(reasons, "on watchlist")
			}
		}
		if t.AmountHigh >= profile.Reporting.DollarThresholds.Act {
			sev = maxSev(sev, cfg.AlertRules.AmountOver1m.Severity)
			reasons = append(reasons, fmt.Sprintf("amount >= $%d", profile.Reporting.DollarThresholds.Act))
		} else if t.AmountHigh >= profile.Reporting.DollarThresholds.Watch {
			sev = maxSev(sev, cfg.AlertRules.AmountOver500k.Severity)
			reasons = append(reasons, fmt.Sprintf("amount >= $%d", profile.Reporting.DollarThresholds.Watch))
		}
		if sev == "" {
			continue
		}

		title := fmt.Sprintf("%s %s %s ($%d-$%d)",
			t.Filer, t.Side, nonEmpty(t.Ticker, t.AssetDesc), t.AmountLow, t.AmountHigh)
		body := fmt.Sprintf(
			"**Filer:** %s  \n**Ticker:** %s  \n**Side:** %s  \n**Amount:** $%d - $%d  \n**Reasons:** %s  \n**Filing:** %s",
			t.Filer, nonEmpty(t.Ticker, "(no ticker)"), t.Side, t.AmountLow, t.AmountHigh,
			strings.Join(reasons, ", "), t.RawURL,
		)
		ticker := t.Ticker
		a := alert.Alert{
			Source: "insiders", Severity: sev, Title: title, Body: body,
			Payload: map[string]any{
				"filer": t.Filer, "ticker": t.Ticker, "side": t.Side,
				"amount_low": t.AmountLow, "amount_high": t.AmountHigh,
				"transaction_ts": t.TransactionTS, "raw_url": t.RawURL,
			},
		}
		if ticker != "" {
			a.Ticker = &ticker
		}
		if _, err := alert.Insert(ctx, conn, a); err != nil {
			return inserted, err
		}
		inserted++
	}

	// Cluster rule — one query across the whole insider_trades table.
	windowDays := profile.Reporting.ClusterWindowDays
	if windowDays <= 0 {
		windowDays = 7
	}
	since := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour).Unix()

	// Collect all cluster hits first so we can close the rows before issuing
	// additional queries (single-connection SQLite would deadlock otherwise).
	type clusterHit struct {
		ticker string
		filers int
	}
	var clusters []clusterHit

	clusterRows, err := conn.QueryContext(ctx, `
		SELECT ticker, COUNT(DISTINCT filer) AS filers
		FROM insider_trades
		WHERE ticker IS NOT NULL AND ticker != '' AND transaction_ts >= ?
		GROUP BY ticker
		HAVING filers >= 3
	`, since)
	if err != nil {
		return inserted, err
	}
	for clusterRows.Next() {
		var h clusterHit
		if err := clusterRows.Scan(&h.ticker, &h.filers); err != nil {
			clusterRows.Close()
			return inserted, err
		}
		clusters = append(clusters, h)
	}
	if err := clusterRows.Close(); err != nil {
		return inserted, err
	}
	if err := clusterRows.Err(); err != nil {
		return inserted, err
	}

	for _, h := range clusters {
		title := fmt.Sprintf("Cluster: %d filers on %s in last %dd", h.filers, h.ticker, windowDays)

		// Dedup per UTC day.
		var exists int
		if err := conn.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM alerts
			WHERE source='insiders' AND title=?
			  AND DATE(created_ts,'unixepoch') = DATE('now')
		`, title).Scan(&exists); err != nil {
			return inserted, err
		}
		if exists > 0 {
			continue
		}
		tk := h.ticker
		if _, err := alert.Insert(ctx, conn, alert.Alert{
			Source: "insiders", Severity: cfg.AlertRules.Cluster3In7d.Severity,
			Ticker: &tk, Title: title,
			Body:    fmt.Sprintf("%d distinct filers transacted in %s within the last %d days. Worth a look.", h.filers, h.ticker, windowDays),
			Payload: map[string]any{"ticker": h.ticker, "filers": h.filers, "window_days": windowDays},
		}); err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, nil
}

// maxSev returns whichever of (current, candidate) is more severe. Order: info < watch < act.
func maxSev(current, candidate string) string {
	rank := map[string]int{"": 0, "info": 1, "watch": 2, "act": 3}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}

func nonEmpty(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func loadWatchlist(ctx context.Context, conn *sql.DB) (map[string]struct{}, error) {
	rows, err := conn.QueryContext(ctx, "SELECT ticker FROM watchlist")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out[strings.ToUpper(t)] = struct{}{}
	}
	return out, rows.Err()
}
