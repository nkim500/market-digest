package insiders

import (
	"context"
	"encoding/xml"
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

// TestSelectOwnershipFilename verifies the two-pass filename selection logic:
// known SEC prefixes are preferred over generic .xml files, and comparisons
// are case-insensitive.
func TestSelectOwnershipFilename(t *testing.T) {
	cases := []struct {
		name  string
		items []indexEntry
		want  string
	}{
		{
			name:  "wk-form4 preferred over generic xml",
			items: []indexEntry{{Name: "other.xml"}, {Name: "wk-form4_123.xml"}},
			want:  "wk-form4_123.xml",
		},
		{
			name:  "wf-form4 preferred over generic xml",
			items: []indexEntry{{Name: "other.xml"}, {Name: "wf-form4_456.xml"}},
			want:  "wf-form4_456.xml",
		},
		{
			name:  "case-insensitive prefix match",
			items: []indexEntry{{Name: "generic.xml"}, {Name: "WK-FORM4_789.XML"}},
			want:  "WK-FORM4_789.XML",
		},
		{
			name:  "falls back to first non-index xml",
			items: []indexEntry{{Name: "0001-index.xml"}, {Name: "filing.xml"}},
			want:  "filing.xml",
		},
		{
			name:  "index files are skipped",
			items: []indexEntry{{Name: "0001-index.xml"}, {Name: "0001-index.htm"}},
			want:  "",
		},
		{
			name:  "empty list returns empty string",
			items: nil,
			want:  "",
		},
		{
			name:  "case-insensitive suffix match (.XML extension)",
			items: []indexEntry{{Name: "DOCUMENT.XML"}},
			want:  "DOCUMENT.XML",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := selectOwnershipFilename(c.items)
			if got != c.want {
				t.Errorf("selectOwnershipFilename() = %q, want %q", got, c.want)
			}
		})
	}
}

// TestStripXMLEncodingNoOpWhenAbsent verifies that stripXMLEncoding produces
// output that Go's xml.Unmarshal can parse regardless of whether the input has
// an encoding attribute, no prolog, or just a version-only prolog.
func TestStripXMLEncodingNoOpWhenAbsent(t *testing.T) {
	cases := []struct{ name, body string }{
		{"no prolog", "<root><a/></root>"},
		{"prolog without encoding", `<?xml version="1.0"?><root><a/></root>`},
		{"prolog with encoding", `<?xml version="1.0" encoding="UTF-8"?><root><a/></root>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(stripXMLEncoding([]byte(c.body)))
			var x struct {
				XMLName xml.Name
			}
			if err := xml.Unmarshal([]byte(got), &x); err != nil {
				t.Errorf("xml.Unmarshal failed after stripXMLEncoding: %v\noutput: %s", err, got)
			}
		})
	}
}

// TestFetchForm4RespectsCutoff verifies that FetchForm4 stops processing once
// all filings in the atom feed are older than the cutoff timestamp.
// The fixture's newest entry has <updated>2026-03-24T17:13:39-04:00</updated>;
// a cutoff in 2027 is after all fixture filings, so zero trades are expected.
func TestFetchForm4RespectsCutoff(t *testing.T) {
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

	// Cutoff set to 2027-01-01, which is after all fixture filings.
	// FetchForm4 should break immediately and return zero trades.
	futureCutoff := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	trades, err := client.FetchForm4(context.Background(), srv.URL, "0001045810", futureCutoff)
	if err != nil {
		t.Fatalf("FetchForm4: %v", err)
	}
	if len(trades) != 0 {
		t.Errorf("expected 0 trades with future cutoff, got %d", len(trades))
	}

	// Sanity check: the unfiltered fetch (cutoff=0) should still return trades,
	// confirming the fixture is valid and the cutoff is the reason for zero above.
	allTrades, err := client.FetchForm4(context.Background(), srv.URL, "0001045810", 0)
	if err != nil {
		t.Fatalf("FetchForm4 (no cutoff): %v", err)
	}
	if len(allTrades) == 0 {
		t.Error("sanity check failed: expected trades with cutoff=0")
	}
}
