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

func TestHashForm4IsDeterministic(t *testing.T) {
	got1 := insiders.HashForm4("0001127602-26-000123", 0)
	got2 := insiders.HashForm4("0001127602-26-000123", 0)
	if got1 != got2 {
		t.Errorf("non-deterministic: %s vs %s", got1, got2)
	}
}

func TestHashForm4DifferentIndicesDifferentHashes(t *testing.T) {
	a := insiders.HashForm4("0001127602-26-000123", 0)
	b := insiders.HashForm4("0001127602-26-000123", 1)
	if a == b {
		t.Error("same filing, different indices must produce different hashes")
	}
}

func TestHashForm4DifferentAccessionsDifferentHashes(t *testing.T) {
	a := insiders.HashForm4("0001127602-26-000123", 0)
	b := insiders.HashForm4("0001127602-26-000124", 0)
	if a == b {
		t.Error("different accessions must produce different hashes")
	}
}

func TestHashForm4DoesNotCollideWithHash(t *testing.T) {
	// A hypothetically structured Trade that Hash() would produce, vs a
	// Form 4 hash with the same "identity" inputs: they must differ.
	t4 := insiders.Trade{
		Source: "sec-form4", Filer: "Jensen Huang", Ticker: "NVDA",
		TransactionTS: 1713897600, AmountLow: 4210000, AmountHigh: 4210000, Side: "sell",
	}
	hashTrade := insiders.Hash(t4)
	hashF4 := insiders.HashForm4("0001127602-26-000123", 0)
	if hashTrade == hashF4 {
		t.Error("Hash() and HashForm4() must occupy different output spaces")
	}
}
