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

func TestEdgarParseOwnershipXML(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "form4_ownership.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	rawURL := "https://www.sec.gov/Archives/edgar/data/1045810/000119903926000003/wk-form4_1774386816.xml"
	trades, err := parseOwnershipXML(body, "0001199039-26-000003", rawURL)
	if err != nil {
		t.Fatalf("parseOwnershipXML: %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(trades))
	}
	first := trades[0]
	if first.Source != "sec-form4" {
		t.Errorf("Source = %q, want sec-form4", first.Source)
	}
	if first.Filer != "STEVENS MARK A" {
		t.Errorf("Filer = %q, want STEVENS MARK A", first.Filer)
	}
	if first.Ticker != "NVDA" {
		t.Errorf("Ticker = %q, want NVDA", first.Ticker)
	}
	if first.TransactionCode != "S" {
		t.Errorf("TransactionCode = %q, want S", first.TransactionCode)
	}
	if first.SecurityType != "common" {
		t.Errorf("SecurityType = %q, want common", first.SecurityType)
	}
	if first.Shares == nil || *first.Shares != -100000 {
		t.Errorf("Shares = %v, want pointer to -100000 (negative because A/D=D)", first.Shares)
	}
	if first.PricePerShare == nil || *first.PricePerShare != 172.6068 {
		t.Errorf("PricePerShare = %v, want pointer to 172.6068", first.PricePerShare)
	}
	if first.AmountLow != first.AmountHigh {
		t.Errorf("Form 4 should set AmountLow == AmountHigh, got low=%d high=%d", first.AmountLow, first.AmountHigh)
	}
	expectedAmt := int(100000 * 172.6068) // 17,260,680
	if first.AmountHigh != expectedAmt {
		t.Errorf("AmountHigh = %d, want %d (abs(shares)*price)", first.AmountHigh, expectedAmt)
	}
	if first.Side != "sell" {
		t.Errorf("Side = %q, want sell", first.Side)
	}
	if first.Hash == "" {
		t.Error("Hash should be populated via HashForm4")
	}
	if first.Hash == trades[1].Hash {
		t.Error("two transactions in the same filing must have different hashes (different indices)")
	}
}

func TestEdgarParseAtomFeed(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "form4_atom.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	filings, err := parseAtomFeed(body)
	if err != nil {
		t.Fatalf("parseAtomFeed: %v", err)
	}
	if len(filings) == 0 {
		t.Fatal("expected at least one filing in atom feed")
	}
	first := filings[0]
	if first.AccessionNumber != "0001199039-26-000003" {
		t.Errorf("AccessionNumber = %q, want 0001199039-26-000003", first.AccessionNumber)
	}
	if !strings.HasPrefix(first.FilingDetailURL, "https://www.sec.gov/Archives/edgar/data/1045810/") {
		t.Errorf("FilingDetailURL = %q, want NVDA archives prefix", first.FilingDetailURL)
	}
	if first.FilingTS == 0 {
		t.Error("FilingTS should be populated from <updated>")
	}
}

func TestEdgarFetchIntegration(t *testing.T) {
	atomBody, _ := os.ReadFile(filepath.Join("testdata", "form4_atom.xml"))
	indexBody, _ := os.ReadFile(filepath.Join("testdata", "form4_filing_index.json"))
	ownershipBody, _ := os.ReadFile(filepath.Join("testdata", "form4_ownership.xml"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.RawQuery, "output=atom"):
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = w.Write(atomBody)
		case strings.HasSuffix(r.URL.Path, "/index.json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(indexBody)
		case strings.HasSuffix(r.URL.Path, ".xml"):
			w.Header().Set("Content-Type", "text/xml")
			_, _ = w.Write(ownershipBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(ClientOptions{
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		BackoffMS:  10,
		UserAgent:  "market-digest test",
	})
	// cutoff = 0 means include everything from the fixture
	trades, err := client.FetchForm4(context.Background(), srv.URL, "0001045810", 0)
	if err != nil {
		t.Fatalf("FetchForm4: %v", err)
	}
	if len(trades) == 0 {
		t.Fatal("expected at least one trade from the integration fetch")
	}
	// Spot-check: first trade should match what parseOwnershipXML produces from the fixture.
	if trades[0].Source != "sec-form4" {
		t.Errorf("trades[0].Source = %q, want sec-form4", trades[0].Source)
	}
	if trades[0].TransactionCode != "S" {
		t.Errorf("trades[0].TransactionCode = %q, want S", trades[0].TransactionCode)
	}
}
