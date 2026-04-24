package insiders_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/insiders"
)

func TestFetchSenate_parsesFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/senate_sample.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	client := insiders.NewClient(insiders.ClientOptions{
		Timeout: 5 * time.Second, MaxRetries: 1,
	})
	trades, err := client.FetchSenate(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Len(t, trades, 2)

	nvda := trades[0]
	require.Equal(t, "senate", nvda.Source)
	require.Equal(t, "Tommy Tuberville", nvda.Filer)
	require.Equal(t, "NVDA", nvda.Ticker)
	require.Equal(t, "buy", nvda.Side)
	require.Equal(t, 15001, nvda.AmountLow)
	require.Equal(t, 50000, nvda.AmountHigh)
	require.NotZero(t, nvda.TransactionTS)
	require.NotZero(t, nvda.FilingTS)
	require.NotEmpty(t, nvda.Hash)

	bond := trades[1]
	require.Equal(t, "", bond.Ticker)
	require.Equal(t, "sell", bond.Side)
	require.Equal(t, 250001, bond.AmountLow)
	require.Equal(t, 500000, bond.AmountHigh)
}

func TestFetchHouse_parsesFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/house_sample.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	client := insiders.NewClient(insiders.ClientOptions{Timeout: 5 * time.Second, MaxRetries: 1})
	trades, err := client.FetchHouse(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Len(t, trades, 1)
	require.Equal(t, "house", trades[0].Source)
	require.Equal(t, "AAPL", trades[0].Ticker)
	require.Equal(t, "buy", trades[0].Side)
}

func TestFetch_retriesOn5xx(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()
	client := insiders.NewClient(insiders.ClientOptions{
		Timeout: 5 * time.Second, MaxRetries: 3, BackoffMS: 1,
	})
	_, err := client.FetchSenate(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Equal(t, 3, attempts)
}
