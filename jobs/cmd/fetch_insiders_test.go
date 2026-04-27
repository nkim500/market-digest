package cmd_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nkim500/market-digest/jobs/cmd"
)

func TestFetchInsidersForm4HappyPath(t *testing.T) {
	home := setupTestHome(t) // existing helper — creates tmp dir with config/, data/, migrations/
	writeSourcesYAML(t, home, `
max_lookback_days: 90

insiders:
  sec_form4:
    enabled: true
    user_agent_env: TEST_SEC_UA
  finnhub:
    enabled: false
  quiver:
    enabled: false

alert_rules:
  watchlist_hit:    { severity: watch }
  amount_over_500k: { severity: watch }
  amount_over_1m:   { severity: act }
  cluster_3_in_7d:  { severity: watch }

http:
  timeout_seconds: 5
  max_retries: 1
  backoff_ms: 100
`)
	writeWatchlistYAML(t, home, `tickers:
  - ticker: NVDA
    note: test
    added: 2026-01-01
`)

	ownershipBody, _ := os.ReadFile(filepath.Join("..", "..", "internal", "insiders", "testdata", "form4_ownership.xml"))
	atomBody, _ := os.ReadFile(filepath.Join("..", "..", "internal", "insiders", "testdata", "form4_atom.xml"))
	indexBody, _ := os.ReadFile(filepath.Join("..", "..", "internal", "insiders", "testdata", "form4_filing_index.json"))
	cikBody := []byte(`{"0":{"cik_str":1045810,"ticker":"NVDA","title":"NVIDIA Corp"}}`)

	var edgarCalls int
	edgarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		edgarCalls++
		switch {
		case strings.Contains(r.URL.Path, "company_tickers.json"):
			_, _ = w.Write(cikBody)
		case strings.Contains(r.URL.RawQuery, "output=atom"):
			_, _ = w.Write(atomBody)
		case strings.HasSuffix(r.URL.Path, "/index.json"):
			_, _ = w.Write(indexBody)
		case strings.HasSuffix(r.URL.Path, ".xml"):
			_, _ = w.Write(ownershipBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer edgarSrv.Close()

	t.Setenv("TEST_SEC_UA", "market-digest <test@example.com>")
	t.Setenv("DIGEST_EDGAR_BASE_URL", edgarSrv.URL)
	t.Setenv("DIGEST_CIK_SOURCE_URL", edgarSrv.URL+"/files/company_tickers.json")

	rowsIn, rowsNew, err := cmd.RunFetchInsiders(context.Background(), home)
	if err != nil {
		t.Fatalf("RunFetchInsiders: %v", err)
	}
	if rowsIn == 0 {
		t.Errorf("expected rowsIn > 0, got %d", rowsIn)
	}
	if rowsNew == 0 {
		t.Errorf("expected rowsNew > 0 on first run, got %d", rowsNew)
	}
	if edgarCalls < 2 {
		t.Errorf("expected at least atom+ownership calls; got %d", edgarCalls)
	}
}

func TestFetchInsidersFailsLoudOnMissingSECUserAgent(t *testing.T) {
	home := setupTestHome(t)
	writeSourcesYAML(t, home, `
max_lookback_days: 7
insiders:
  sec_form4: { enabled: true, user_agent_env: DEFINITELY_UNSET_XYZ }
alert_rules:
  watchlist_hit: { severity: watch }
  amount_over_500k: { severity: watch }
  amount_over_1m: { severity: act }
  cluster_3_in_7d: { severity: watch }
http: { timeout_seconds: 5, max_retries: 1, backoff_ms: 100 }
`)
	writeWatchlistYAML(t, home, "tickers: []\n")

	_, _, err := cmd.RunFetchInsiders(context.Background(), home)
	if err == nil {
		t.Fatal("expected error when SEC user agent env var is unset")
	}
	if !strings.Contains(err.Error(), "SEC") {
		t.Errorf("error should mention SEC user agent, got %v", err)
	}
}

func setupTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	for _, dir := range []string{"config", "data", "migrations"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	// Copy migrations from repo.
	migSrc, _ := filepath.Abs(filepath.Join("..", "..", "migrations"))
	entries, _ := os.ReadDir(migSrc)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			body, _ := os.ReadFile(filepath.Join(migSrc, e.Name()))
			_ = os.WriteFile(filepath.Join(home, "migrations", e.Name()), body, 0o644)
		}
	}
	// Minimal profile.yml so config.Load succeeds.
	_ = os.WriteFile(filepath.Join(home, "config", "profile.yml"),
		[]byte("reporting:\n  dollar_thresholds:\n    watch: 500000\n    act: 1000000\n  cluster_window_days: 7\n"), 0o644)
	return home
}

func writeSourcesYAML(t *testing.T, home, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, "config", "sources.yml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write sources.yml: %v", err)
	}
}

func writeWatchlistYAML(t *testing.T, home, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, "config", "watchlist.yml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write watchlist.yml: %v", err)
	}
}
