package insiders

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQuiverParseResponse(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "quiver_congress.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	trades, err := parseQuiverResponse(body)
	if err != nil {
		t.Fatalf("parseQuiverResponse: %v", err)
	}
	if len(trades) < 2 {
		t.Fatalf("expected >=2 trades, got %d", len(trades))
	}
	first := trades[0]
	if first.Source != "quiver" {
		t.Errorf("Source = %q, want quiver", first.Source)
	}
	if first.Filer == "" {
		t.Error("Filer should be populated from Representative")
	}
	if first.Role != "House" && first.Role != "Senate" {
		t.Errorf("Role = %q, want House or Senate", first.Role)
	}
	if first.Ticker != "AAPL" {
		t.Errorf("Ticker = %q, want AAPL", first.Ticker)
	}
	if first.AmountLow == 0 {
		t.Error("AmountLow should be parsed from Range string")
	}
	if first.AmountHigh == 0 {
		t.Error("AmountHigh should be parsed from Range string or Amount field")
	}
	if first.Hash == "" {
		t.Error("Hash should be populated")
	}
}

func TestQuiverFetchIntegration(t *testing.T) {
	body, _ := os.ReadFile(filepath.Join("testdata", "quiver_congress.json"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/AAPL") {
			t.Errorf("path should contain ticker: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer testkey" {
			t.Errorf("missing or wrong Authorization header: %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	client := NewClient(ClientOptions{Timeout: 5 * time.Second, MaxRetries: 1, BackoffMS: 10})
	trades, err := client.FetchQuiverCongressional(context.Background(), srv.URL, "testkey", "AAPL", 0)
	if err != nil {
		t.Fatalf("FetchQuiverCongressional: %v", err)
	}
	if len(trades) < 2 {
		t.Errorf("expected >=2 trades, got %d", len(trades))
	}
}
