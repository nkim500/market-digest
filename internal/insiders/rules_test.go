package insiders_test

import (
	"context"
	"database/sql"
	"fmt"
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

func TestEvaluateRulesSuppressesNonOpenMarketForm4Codes(t *testing.T) {
	ctx := context.Background()
	conn := testDB(t)
	seedWatchlist(t, conn, []string{"NVDA"})

	// code 'A' is an award/grant — should not produce an alert even though
	// ticker is on the watchlist and amount exceeds both thresholds.
	shares := 10000
	price := 900.00
	t4 := insiders.Trade{
		Source:          "sec-form4",
		Filer:           "Jensen Huang",
		Role:            "CEO",
		Ticker:          "NVDA",
		Side:            "buy",
		AmountLow:       9000000,
		AmountHigh:      9000000,
		TransactionTS:   time.Now().Unix(),
		FilingTS:        time.Now().Unix(),
		Shares:          &shares,
		PricePerShare:   &price,
		TransactionCode: "A",
		SecurityType:    "common",
		Hash:            insiders.HashForm4("0001045810-26-000001", 0),
	}
	_, err := insiders.StoreInserts(ctx, conn, []insiders.Trade{t4})
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	inserted, err := insiders.EvaluateRules(ctx, conn, []insiders.Trade{t4}, testSources(), testProfile())
	if err != nil {
		t.Fatalf("EvaluateRules: %v", err)
	}
	if inserted != 0 {
		t.Errorf("code 'A' should not produce alerts; got %d inserted", inserted)
	}

	var count int
	_ = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM alerts").Scan(&count)
	if count != 0 {
		t.Errorf("alerts table should be empty; got %d", count)
	}
}

func TestEvaluateRulesFiresForOpenMarketForm4Purchase(t *testing.T) {
	ctx := context.Background()
	conn := testDB(t)
	seedWatchlist(t, conn, []string{"NVDA"})

	shares := 1000
	price := 900.00
	t4 := insiders.Trade{
		Source:          "sec-form4",
		Filer:           "Jensen Huang",
		Role:            "CEO",
		Ticker:          "NVDA",
		Side:            "buy",
		AmountLow:       900000,
		AmountHigh:      900000,
		TransactionTS:   time.Now().Unix(),
		FilingTS:        time.Now().Unix(),
		Shares:          &shares,
		PricePerShare:   &price,
		TransactionCode: "P",
		SecurityType:    "common",
		Hash:            insiders.HashForm4("0001045810-26-000002", 0),
	}
	_, err := insiders.StoreInserts(ctx, conn, []insiders.Trade{t4})
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	inserted, err := insiders.EvaluateRules(ctx, conn, []insiders.Trade{t4}, testSources(), testProfile())
	if err != nil {
		t.Fatalf("EvaluateRules: %v", err)
	}
	if inserted < 1 {
		t.Errorf("code 'P' watchlist hit should produce >=1 alert; got %d", inserted)
	}
}

func TestEvaluateRulesClusterExcludesNonOpenMarketForm4(t *testing.T) {
	ctx := context.Background()
	conn := testDB(t)

	now := time.Now().Unix()
	// Four Form 4 rows on NVDA — all code 'A' (grants). No open-market activity.
	// Cluster should NOT fire because the rule excludes non-P/S Form 4.
	for i := 0; i < 4; i++ {
		shares := 100
		price := 900.00
		t4 := insiders.Trade{
			Source:          "sec-form4",
			Filer:           fmt.Sprintf("Insider %d", i),
			Ticker:          "NVDA",
			Side:            "buy",
			AmountLow:       90000, AmountHigh: 90000,
			TransactionTS:   now,
			FilingTS:        now,
			Shares:          &shares,
			PricePerShare:   &price,
			TransactionCode: "A",
			SecurityType:    "common",
			Hash:            insiders.HashForm4(fmt.Sprintf("0001045810-26-00%04d", i), 0),
		}
		if _, err := insiders.StoreInserts(ctx, conn, []insiders.Trade{t4}); err != nil {
			t.Fatalf("store: %v", err)
		}
	}

	_, err := insiders.EvaluateRules(ctx, conn, nil, testSources(), testProfile())
	if err != nil {
		t.Fatalf("EvaluateRules: %v", err)
	}
	var clusters int
	_ = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM alerts WHERE title LIKE 'Cluster:%'").Scan(&clusters)
	if clusters != 0 {
		t.Errorf("cluster should not fire when all Form 4 codes are non-P/S; got %d cluster alerts", clusters)
	}
}

// --- helpers used by the new tests ---

func seedWatchlist(t *testing.T, conn *sql.DB, tickers []string) {
	t.Helper()
	for _, tk := range tickers {
		_, err := conn.ExecContext(context.Background(),
			"INSERT INTO watchlist (ticker, added_ts) VALUES (?, ?)", tk, time.Now().Unix())
		if err != nil {
			t.Fatalf("seed watchlist: %v", err)
		}
	}
}

func testSources() config.Sources {
	return config.Sources{
		AlertRules: config.AlertRules{
			WatchlistHit:   config.AlertRule{Severity: "watch"},
			AmountOver500k: config.AlertRule{Severity: "watch"},
			AmountOver1m:   config.AlertRule{Severity: "act"},
			Cluster3In7d:   config.AlertRule{Severity: "watch"},
		},
	}
}

func testProfile() config.Profile {
	var p config.Profile
	p.Reporting.DollarThresholds.Watch = 500000
	p.Reporting.DollarThresholds.Act = 1000000
	p.Reporting.ClusterWindowDays = 7
	return p
}
