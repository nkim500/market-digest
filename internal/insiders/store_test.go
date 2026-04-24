package insiders_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/internal/insiders"
)

// testDB is an alias for newDB so both names work in this file.
func testDB(t *testing.T) *sql.DB { return newDB(t) }

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	return conn
}

func TestStoreInserts_dedupsOnHash(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	trades := []insiders.Trade{
		{Source: "senate", Filer: "X", Ticker: "NVDA", Side: "buy",
			AmountLow: 1000, AmountHigh: 15000, TransactionTS: 1700000000},
		{Source: "senate", Filer: "X", Ticker: "NVDA", Side: "buy",
			AmountLow: 1000, AmountHigh: 15000, TransactionTS: 1700000000},
	}
	for i := range trades {
		trades[i].Hash = insiders.Hash(trades[i])
	}
	result, err := insiders.StoreInserts(ctx, conn, trades)
	require.NoError(t, err)
	require.Len(t, result.IDs, 1, "identical trades must dedup")
	require.Len(t, result.Trades, 1, "identical trades must dedup")

	result, err = insiders.StoreInserts(ctx, conn, trades)
	require.NoError(t, err)
	require.Empty(t, result.IDs)
	require.Empty(t, result.Trades)
}

func TestStoreInsertsWritesForm4Fields(t *testing.T) {
	ctx := context.Background()
	conn := testDB(t)

	shares := -5000
	price := 842.00
	t4 := insiders.Trade{
		Source:          "sec-form4",
		Filer:           "Jensen Huang",
		Role:            "CEO",
		Ticker:          "NVDA",
		Side:            "sell",
		AmountLow:       4210000,
		AmountHigh:      4210000,
		TransactionTS:   1713897600,
		FilingTS:        1713984000,
		RawURL:          "https://www.sec.gov/Archives/edgar/data/1045810/...",
		Hash:            insiders.HashForm4("0001127602-26-000123", 0),
		Shares:          &shares,
		PricePerShare:   &price,
		TransactionCode: "S",
		SecurityType:    "common",
	}
	result, err := insiders.StoreInserts(ctx, conn, []insiders.Trade{t4})
	if err != nil {
		t.Fatalf("StoreInserts: %v", err)
	}
	if len(result.IDs) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(result.IDs))
	}

	var (
		gotShares  sql.NullInt64
		gotPrice   sql.NullFloat64
		gotTxnCode sql.NullString
		gotSecType sql.NullString
	)
	err = conn.QueryRowContext(ctx, `
		SELECT shares, price_per_share, transaction_code, security_type
		FROM insider_trades WHERE id = ?
	`, result.IDs[0]).Scan(&gotShares, &gotPrice, &gotTxnCode, &gotSecType)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !gotShares.Valid || gotShares.Int64 != int64(shares) {
		t.Errorf("shares: got %v, want %d", gotShares, shares)
	}
	if !gotPrice.Valid || gotPrice.Float64 != price {
		t.Errorf("price_per_share: got %v, want %f", gotPrice, price)
	}
	if !gotTxnCode.Valid || gotTxnCode.String != "S" {
		t.Errorf("transaction_code: got %v, want S", gotTxnCode)
	}
	if !gotSecType.Valid || gotSecType.String != "common" {
		t.Errorf("security_type: got %v, want common", gotSecType)
	}
}

func TestStoreInsertsPoliticianFieldsAreNull(t *testing.T) {
	ctx := context.Background()
	conn := testDB(t)

	p := insiders.Trade{
		Source:        "finnhub",
		Filer:         "Nancy Pelosi",
		Role:          "Representative",
		Ticker:        "NVDA",
		Side:          "buy",
		AmountLow:     50001,
		AmountHigh:    100000,
		TransactionTS: 1713897600,
		FilingTS:      1713984000,
		Hash:          "politicianhash0001",
		// No Form 4 fields set.
	}
	result, err := insiders.StoreInserts(ctx, conn, []insiders.Trade{p})
	if err != nil {
		t.Fatalf("StoreInserts: %v", err)
	}

	var (
		gotShares  sql.NullInt64
		gotPrice   sql.NullFloat64
		gotTxnCode sql.NullString
		gotSecType sql.NullString
	)
	_ = conn.QueryRowContext(ctx, `
		SELECT shares, price_per_share, transaction_code, security_type
		FROM insider_trades WHERE id = ?
	`, result.IDs[0]).Scan(&gotShares, &gotPrice, &gotTxnCode, &gotSecType)
	if gotShares.Valid {
		t.Error("shares should be NULL for politician trade")
	}
	if gotPrice.Valid {
		t.Error("price_per_share should be NULL for politician trade")
	}
	if gotTxnCode.Valid {
		t.Error("transaction_code should be NULL for politician trade")
	}
	if gotSecType.Valid {
		t.Error("security_type should be NULL for politician trade")
	}
}
