package insiders_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/insiders"
)

func TestHash_stableForSameInputs(t *testing.T) {
	a := insiders.Trade{Source: "senate", Filer: "Smith, John", Ticker: "NVDA",
		TransactionTS: 1700000000, AmountLow: 15000, AmountHigh: 50000, Side: "buy"}
	b := a
	require.Equal(t, insiders.Hash(a), insiders.Hash(b))
}

func TestHash_differsOnSide(t *testing.T) {
	a := insiders.Trade{Source: "senate", Filer: "Smith, John", Ticker: "NVDA",
		TransactionTS: 1700000000, AmountLow: 15000, AmountHigh: 50000, Side: "buy"}
	b := a
	b.Side = "sell"
	require.NotEqual(t, insiders.Hash(a), insiders.Hash(b))
}

func TestHash_caseInsensitiveTicker(t *testing.T) {
	a := insiders.Trade{Source: "senate", Filer: "Smith, John", Ticker: "NVDA",
		TransactionTS: 1700000000, AmountLow: 15000, AmountHigh: 50000, Side: "buy"}
	b := a
	b.Ticker = "nvda"
	require.Equal(t, insiders.Hash(a), insiders.Hash(b))
}

func TestHash_filerWhitespaceIgnored(t *testing.T) {
	a := insiders.Trade{Source: "senate", Filer: "Smith, John", Ticker: "NVDA",
		TransactionTS: 1700000000, AmountLow: 15000, AmountHigh: 50000, Side: "buy"}
	b := a
	b.Filer = "  Smith,  John  "
	require.Equal(t, insiders.Hash(a), insiders.Hash(b))
}
