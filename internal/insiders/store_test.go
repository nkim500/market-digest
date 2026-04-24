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
	newIDs, err := insiders.StoreInserts(ctx, conn, trades)
	require.NoError(t, err)
	require.Len(t, newIDs, 1, "identical trades must dedup")

	newIDs, err = insiders.StoreInserts(ctx, conn, trades)
	require.NoError(t, err)
	require.Empty(t, newIDs)
}
