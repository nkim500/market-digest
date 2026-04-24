package insiders_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/config"
	"github.com/nkim500/market-digest/internal/insiders"
)

func TestEvaluateRules_watchlistHit(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	_, err := conn.ExecContext(ctx,
		`INSERT INTO watchlist (ticker, note, added_ts) VALUES (?, ?, ?)`,
		"NVDA", "", time.Now().Unix())
	require.NoError(t, err)

	trade := insiders.Trade{Source: "senate", Filer: "Sen X", Ticker: "NVDA",
		Side: "buy", AmountLow: 15001, AmountHigh: 50000,
		TransactionTS: time.Now().Unix(), FilingTS: time.Now().Unix()}
	trade.Hash = insiders.Hash(trade)
	_, err = insiders.StoreInserts(ctx, conn, []insiders.Trade{trade})
	require.NoError(t, err)

	cfg := config.Sources{
		AlertRules: config.AlertRules{
			WatchlistHit:   config.AlertRule{Severity: "watch"},
			AmountOver500k: config.AlertRule{Severity: "watch"},
			AmountOver1m:   config.AlertRule{Severity: "act"},
			Cluster3In7d:   config.AlertRule{Severity: "watch"},
		},
	}
	profile := config.Profile{}
	profile.Reporting.DollarThresholds.Watch = 500000
	profile.Reporting.DollarThresholds.Act = 1000000
	profile.Reporting.ClusterWindowDays = 7

	n, err := insiders.EvaluateRules(ctx, conn, []insiders.Trade{trade}, cfg, profile)
	require.NoError(t, err)
	require.Equal(t, 1, n, "expected 1 alert row for watchlist hit")

	var sev, title string
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT severity, title FROM alerts WHERE source='insiders' ORDER BY id DESC LIMIT 1`,
	).Scan(&sev, &title))
	require.Equal(t, "watch", sev)
	require.Contains(t, title, "NVDA")
}

func TestEvaluateRules_amountOver1mIsAct(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	trade := insiders.Trade{Source: "senate", Filer: "Sen X", Ticker: "ZZZZ",
		Side: "buy", AmountLow: 1000001, AmountHigh: 5000000,
		TransactionTS: time.Now().Unix()}
	trade.Hash = insiders.Hash(trade)
	_, err := insiders.StoreInserts(ctx, conn, []insiders.Trade{trade})
	require.NoError(t, err)

	cfg := config.Sources{
		AlertRules: config.AlertRules{
			WatchlistHit:   config.AlertRule{Severity: "watch"},
			AmountOver500k: config.AlertRule{Severity: "watch"},
			AmountOver1m:   config.AlertRule{Severity: "act"},
			Cluster3In7d:   config.AlertRule{Severity: "watch"},
		},
	}
	profile := config.Profile{}
	profile.Reporting.DollarThresholds.Watch = 500000
	profile.Reporting.DollarThresholds.Act = 1000000
	profile.Reporting.ClusterWindowDays = 7

	_, err = insiders.EvaluateRules(ctx, conn, []insiders.Trade{trade}, cfg, profile)
	require.NoError(t, err)

	var sev string
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT severity FROM alerts WHERE source='insiders' ORDER BY id DESC LIMIT 1`,
	).Scan(&sev))
	require.Equal(t, "act", sev)
}

func TestEvaluateRules_cluster3FilersOneTickerWithin7d(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	now := time.Now().Unix()
	var trades []insiders.Trade
	for i, f := range []string{"A", "B", "C"} {
		tr := insiders.Trade{Source: "senate", Filer: f, Ticker: "XYZ",
			Side: "buy", AmountLow: 1000, AmountHigh: 15000,
			TransactionTS: now - int64(i*86400)}
		tr.Hash = insiders.Hash(tr)
		trades = append(trades, tr)
	}
	_, err := insiders.StoreInserts(ctx, conn, trades)
	require.NoError(t, err)

	cfg := config.Sources{
		AlertRules: config.AlertRules{
			WatchlistHit:   config.AlertRule{Severity: "watch"},
			AmountOver500k: config.AlertRule{Severity: "watch"},
			AmountOver1m:   config.AlertRule{Severity: "act"},
			Cluster3In7d:   config.AlertRule{Severity: "watch"},
		},
	}
	profile := config.Profile{}
	profile.Reporting.DollarThresholds.Watch = 500000
	profile.Reporting.DollarThresholds.Act = 1000000
	profile.Reporting.ClusterWindowDays = 7

	_, err = insiders.EvaluateRules(ctx, conn, trades, cfg, profile)
	require.NoError(t, err)

	var count int
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM alerts WHERE title LIKE 'Cluster:%' AND ticker='XYZ'`,
	).Scan(&count))
	require.Equal(t, 1, count)

	_, err = insiders.EvaluateRules(ctx, conn, trades, cfg, profile)
	require.NoError(t, err)
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM alerts WHERE title LIKE 'Cluster:%' AND ticker='XYZ'`,
	).Scan(&count))
	require.Equal(t, 1, count, "cluster alert must dedup per UTC day")
}
