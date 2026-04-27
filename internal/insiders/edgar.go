package insiders

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Form4Filing is a single entry from an issuer's Form 4 atom feed.
type Form4Filing struct {
	AccessionNumber string // dashed format, e.g. 0001199039-26-000003
	FilingDetailURL string // URL to the -index.htm page
	FilingTS        int64  // unix, from <updated>
}

// atomFeed is the top-level Atom feed envelope.
// The SEC feed root is in the Atom namespace; Go's xml decoder matches
// by local name when the struct tag omits the namespace, but we include
// it explicitly for correctness.
type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Entries []atomEntry `xml:"http://www.w3.org/2005/Atom entry"`
}

type atomEntry struct {
	Updated string      `xml:"http://www.w3.org/2005/Atom updated"`
	Content atomContent `xml:"http://www.w3.org/2005/Atom content"`
}

// atomContent holds SEC-specific sub-elements inside <content type="text/xml">.
// These child elements are NOT in the Atom namespace (SEC injects them directly),
// so we match them without a namespace prefix.
type atomContent struct {
	AccessionNum string `xml:"accession-number"`
	FilingHREF   string `xml:"filing-href"`
}

// parseAtomFeed turns the SEC Atom response into a list of Form4Filing.
// The SEC feed declares encoding="ISO-8859-1" in its XML prolog. Go's
// xml.Unmarshal rejects non-UTF-8 declarations unless a CharsetReader is
// provided. Since the actual content is ASCII-safe, we strip the declaration
// before decoding rather than pulling in a full charset translation library.
func parseAtomFeed(body []byte) ([]Form4Filing, error) {
	// Strip any <?xml ... encoding="..."?> prolog that declares a non-UTF-8
	// encoding; the body is ASCII-compatible so raw bytes are fine.
	body = stripXMLEncoding(body)

	var feed atomFeed
	dec := xml.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&feed); err != nil {
		return nil, fmt.Errorf("parse atom: %w", err)
	}
	out := make([]Form4Filing, 0, len(feed.Entries))
	for _, e := range feed.Entries {
		ts := parseAtomDate(e.Updated)
		acc := strings.TrimSpace(e.Content.AccessionNum)
		if acc == "" {
			acc = accessionFromURL(e.Content.FilingHREF)
		}
		if acc == "" {
			continue
		}
		out = append(out, Form4Filing{
			AccessionNumber: acc,
			FilingDetailURL: e.Content.FilingHREF,
			FilingTS:        ts,
		})
	}
	return out, nil
}

func parseAtomDate(s string) int64 {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC().Unix()
	}
	return 0
}

// xmlProcInstrRE matches an XML prolog that includes an encoding attribute,
// e.g. <?xml version="1.0" encoding="ISO-8859-1" ?>.
var xmlProcInstrRE = regexp.MustCompile(`(?i)<\?xml[^?]*encoding="[^"]*"[^?]*\?>`)

// stripXMLEncoding removes the encoding attribute from an XML prolog so that
// Go's xml decoder (which requires UTF-8 or no declaration) can parse it.
// SEC Form 4 atom feeds declare ISO-8859-1 but contain only ASCII text.
func stripXMLEncoding(body []byte) []byte {
	replaced := xmlProcInstrRE.ReplaceAll(body, []byte(`<?xml version="1.0"?>`))
	return replaced
}

// accessionFromURL extracts "0001199039-26-000003" from a URL path.
var accessionRE = regexp.MustCompile(`(\d{10}-\d{2}-\d{6})`)

func accessionFromURL(u string) string {
	if m := accessionRE.FindString(u); m != "" {
		return m
	}
	return ""
}

// indexEntry is one item in the SEC filing directory listing.
type indexEntry struct {
	Name string `json:"name"`
}

// indexResponse is the top-level structure of /<folder>/index.json.
type indexResponse struct {
	Directory struct {
		Item []indexEntry `json:"item"`
	} `json:"directory"`
}

// discoverOwnershipURL fetches the filing folder's index.json and returns
// the URL to the first .xml file that isn't an index file.
//
// detailURL looks like:
//
//	https://www.sec.gov/Archives/edgar/data/1045810/000119903926000003/0001199039-26-000003-index.htm
//
// We strip the filename, append index.json, fetch it, and find the ownership XML.
func discoverOwnershipURL(ctx context.Context, c *Client, detailURL string) (string, error) {
	u, err := url.Parse(detailURL)
	if err != nil || detailURL == "" {
		return "", fmt.Errorf("invalid detail URL %q: %w", detailURL, err)
	}
	// Strip filename to get folder URL.
	path := u.Path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[:idx]
	}
	u.Path = path + "/index.json"
	u.RawQuery = ""
	indexURL := u.String()

	body, err := c.get(ctx, indexURL)
	if err != nil {
		return "", fmt.Errorf("fetch index.json %s: %w", indexURL, err)
	}

	var resp indexResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse index.json: %w", err)
	}

	for _, item := range resp.Directory.Item {
		name := item.Name
		if strings.HasSuffix(name, ".xml") && !strings.Contains(name, "-index") {
			// Build the full URL: same scheme+host+folder/filename.
			u.Path = path + "/" + name
			return u.String(), nil
		}
	}
	return "", fmt.Errorf("no ownership XML found in %s", indexURL)
}

// rebaseURL takes a URL from the atom feed (pointing to www.sec.gov) and
// replaces the scheme+host with the baseURL that was passed to FetchForm4.
// This is necessary so integration tests (and any baseURL override) work correctly.
func rebaseURL(original, baseURL string) string {
	orig, err := url.Parse(original)
	if err != nil || original == "" {
		return original
	}
	base, err := url.Parse(baseURL)
	if err != nil || baseURL == "" {
		return original
	}
	orig.Scheme = base.Scheme
	orig.Host = base.Host
	return orig.String()
}

// --- Ownership XML (the actual Form 4 document) ---

type ownershipDoc struct {
	XMLName     xml.Name      `xml:"ownershipDocument"`
	Issuer      ownerIssuer   `xml:"issuer"`
	ReportOwner []reportOwner `xml:"reportingOwner"`
	NonDeriv    struct {
		Txns []ownershipTxn `xml:"nonDerivativeTransaction"`
	} `xml:"nonDerivativeTable"`
	Derivative struct {
		Txns []ownershipTxn `xml:"derivativeTransaction"`
	} `xml:"derivativeTable"`
}

type ownerIssuer struct {
	CIK           string `xml:"issuerCik"`
	Name          string `xml:"issuerName"`
	TradingSymbol string `xml:"issuerTradingSymbol"`
}

type reportOwner struct {
	ID struct {
		CIK  string `xml:"rptOwnerCik"`
		Name string `xml:"rptOwnerName"`
	} `xml:"reportingOwnerId"`
	Rel struct {
		IsDirector        string `xml:"isDirector"`
		IsOfficer         string `xml:"isOfficer"`
		IsTenPercentOwner string `xml:"isTenPercentOwner"`
		IsOther           string `xml:"isOther"`
		OfficerTitle      string `xml:"officerTitle"`
		OtherText         string `xml:"otherText"`
	} `xml:"reportingOwnerRelationship"`
}

// ownershipTxn represents a single non-derivative or derivative transaction.
// NOTE (Correction A): <transactionCode> is DIRECT text — no nested <value>.
// All other monetary/count fields use nested <value> elements.
type ownershipTxn struct {
	SecurityTitle struct {
		Value string `xml:"value"`
	} `xml:"securityTitle"`
	TransactionDate struct {
		Value string `xml:"value"`
	} `xml:"transactionDate"`
	TransactionCoding struct {
		Code string `xml:"transactionCode"` // direct text, NOT nested <value>
	} `xml:"transactionCoding"`
	TransactionAmounts struct {
		Shares struct {
			Value string `xml:"value"`
		} `xml:"transactionShares"`
		PricePerShare struct {
			Value string `xml:"value"`
		} `xml:"transactionPricePerShare"`
		AcquiredDisposed struct {
			Value string `xml:"value"`
		} `xml:"transactionAcquiredDisposedCode"`
	} `xml:"transactionAmounts"`
}

// parseOwnershipXML turns a single Form 4 XML document into []Trade.
func parseOwnershipXML(body []byte, accession, rawURL string) ([]Trade, error) {
	var doc ownershipDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse ownership xml: %w", err)
	}
	ticker := strings.ToUpper(strings.TrimSpace(doc.Issuer.TradingSymbol))

	filer := ""
	role := ""
	if len(doc.ReportOwner) > 0 {
		ro := doc.ReportOwner[0]
		filer = strings.TrimSpace(ro.ID.Name)
		role = deriveRole(ro)
	}

	var trades []Trade
	idx := 0
	for _, txn := range doc.NonDeriv.Txns {
		trades = append(trades, buildForm4Trade(txn, filer, role, ticker, "common", accession, idx, rawURL))
		idx++
	}
	for _, txn := range doc.Derivative.Txns {
		trades = append(trades, buildForm4Trade(txn, filer, role, ticker, "derivative", accession, idx, rawURL))
		idx++
	}
	return trades, nil
}

func buildForm4Trade(txn ownershipTxn, filer, role, ticker, secType, accession string, idx int, rawURL string) Trade {
	shares := parseFloatStr(txn.TransactionAmounts.Shares.Value)
	price := parseFloatStr(txn.TransactionAmounts.PricePerShare.Value)
	// Correction A: Code is already a string, not a nested struct.
	code := strings.TrimSpace(txn.TransactionCoding.Code)
	acqDisp := strings.ToUpper(strings.TrimSpace(txn.TransactionAmounts.AcquiredDisposed.Value))

	sharesInt := int(shares)
	if acqDisp == "D" {
		sharesInt = -sharesInt
	}
	amount := int(float64(absInt(sharesInt)) * price)
	side := form4Side(code, acqDisp)

	t := Trade{
		Source:          "sec-form4",
		Filer:           filer,
		Role:            role,
		Ticker:          ticker,
		Side:            side,
		AmountLow:       amount,
		AmountHigh:      amount,
		TransactionTS:   parseDate(txn.TransactionDate.Value),
		FilingTS:        parseDate(txn.TransactionDate.Value), // overridden in FetchForm4 with atom <updated>
		RawURL:          rawURL,
		TransactionCode: code,
		SecurityType:    secType,
	}
	if sharesInt != 0 {
		t.Shares = &sharesInt
	}
	if price > 0 {
		t.PricePerShare = &price
	}
	t.Hash = HashForm4(accession, idx)
	return t
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func parseFloatStr(s string) float64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func form4Side(code, acqDisp string) string {
	switch code {
	case "P":
		return "buy"
	case "S":
		return "sell"
	case "M", "X":
		return "exchange"
	}
	// Fallback to acquired/disposed bucket.
	switch acqDisp {
	case "A":
		return "buy"
	case "D":
		return "sell"
	}
	return ""
}

func deriveRole(ro reportOwner) string {
	var parts []string
	if ro.Rel.IsOfficer == "1" || strings.EqualFold(ro.Rel.IsOfficer, "true") {
		title := strings.TrimSpace(ro.Rel.OfficerTitle)
		if title == "" {
			title = "Officer"
		}
		parts = append(parts, title)
	}
	if ro.Rel.IsDirector == "1" || strings.EqualFold(ro.Rel.IsDirector, "true") {
		parts = append(parts, "Director")
	}
	if ro.Rel.IsTenPercentOwner == "1" || strings.EqualFold(ro.Rel.IsTenPercentOwner, "true") {
		parts = append(parts, "10%-Owner")
	}
	if len(parts) == 0 && (ro.Rel.IsOther == "1" || strings.EqualFold(ro.Rel.IsOther, "true")) {
		parts = append(parts, strings.TrimSpace(ro.Rel.OtherText))
	}
	return strings.Join(parts, ", ")
}

// FetchForm4 pulls the Atom feed for a specific CIK, discovers each filing's
// ownership XML via the filing folder's index.json (Correction B: filenames vary),
// parses non-derivative + derivative transactions into []Trade with HashForm4 for
// dedup. Stops pagination once filing_ts < cutoff. Per-filing errors are logged
// and skipped so a single bad filing doesn't abort the whole run.
func (c *Client) FetchForm4(ctx context.Context, baseURL, cik string, cutoff int64) ([]Trade, error) {
	atomURL := fmt.Sprintf(
		"%s/cgi-bin/browse-edgar?action=getcompany&CIK=%s&type=4&dateb=&owner=include&count=40&output=atom",
		strings.TrimRight(baseURL, "/"), cik,
	)
	atomBody, err := c.get(ctx, atomURL)
	if err != nil {
		return nil, fmt.Errorf("atom: %w", err)
	}
	filings, err := parseAtomFeed(atomBody)
	if err != nil {
		return nil, err
	}

	var all []Trade
	for _, f := range filings {
		if cutoff > 0 && f.FilingTS < cutoff {
			break
		}

		// Rebase the filing detail URL to our baseURL so tests and URL overrides work.
		detailURL := rebaseURL(f.FilingDetailURL, baseURL)

		ownershipURL, err := discoverOwnershipURL(ctx, c, detailURL)
		if err != nil {
			log.Printf("form4: skipping %s: discover ownership URL: %v", f.AccessionNumber, err)
			continue
		}

		ownershipBody, err := c.get(ctx, ownershipURL)
		if err != nil {
			log.Printf("form4: skipping %s: fetch ownership XML: %v", f.AccessionNumber, err)
			continue
		}

		trades, err := parseOwnershipXML(ownershipBody, f.AccessionNumber, ownershipURL)
		if err != nil {
			log.Printf("form4: skipping %s: parse ownership XML: %v", f.AccessionNumber, err)
			continue
		}

		// Override FilingTS with the atom feed's <updated> timestamp.
		for i := range trades {
			trades[i].FilingTS = f.FilingTS
		}
		all = append(all, trades...)
	}
	return all, nil
}
