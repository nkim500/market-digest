package insiders

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFinnhubParseResponse(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "finnhub_congress.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	trades, err := parseFinnhubResponse(body)
	if err != nil {
		t.Fatalf("parseFinnhubResponse: %v", err)
	}
	if len(trades) < 2 {
		t.Fatalf("expected >=2 trades from fixture, got %d", len(trades))
	}
	first := trades[0]
	if first.Source != "finnhub" {
		t.Errorf("Source = %q, want finnhub", first.Source)
	}
	if first.Filer == "" {
		t.Error("Filer should be populated")
	}
	if first.Role == "" {
		t.Error("Role should be populated from position")
	}
	if first.Ticker == "" {
		t.Error("Ticker should be populated from symbol")
	}
	if first.AmountLow == 0 || first.AmountHigh == 0 {
		t.Errorf("amounts not parsed: low=%d high=%d", first.AmountLow, first.AmountHigh)
	}
	if first.Side == "" {
		t.Error("Side should be normalized from transactionType")
	}
	if first.Hash == "" {
		t.Error("Hash should be populated via Hash()")
	}
	// Form 4 fields should be nil / empty.
	if first.Shares != nil || first.PricePerShare != nil {
		t.Error("Finnhub trades should leave Form 4 numeric fields nil")
	}
	if first.TransactionCode != "" || first.SecurityType != "" {
		t.Error("Finnhub trades should leave Form 4 string fields empty")
	}
}

func TestFinnhubFetchIntegration(t *testing.T) {
	body, _ := os.ReadFile(filepath.Join("testdata", "finnhub_congress.json"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("symbol") != "AAPL" {
			t.Errorf("missing symbol param: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("token") != "testkey" {
			t.Errorf("missing token param")
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	client := NewClient(ClientOptions{Timeout: 5 * time.Second, MaxRetries: 1, BackoffMS: 10})
	trades, err := client.FetchFinnhubCongressional(context.Background(), srv.URL, "testkey", "AAPL", 0)
	if err != nil {
		t.Fatalf("FetchFinnhubCongressional: %v", err)
	}
	if len(trades) < 2 {
		t.Errorf("expected >=2 trades, got %d", len(trades))
	}
}
