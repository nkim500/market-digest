package insiders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type finnhubResponse struct {
	Data   []finnhubRow `json:"data"`
	Symbol string       `json:"symbol"`
}

type finnhubRow struct {
	AmountFrom      int    `json:"amountFrom"`
	AmountTo        int    `json:"amountTo"`
	AssetName       string `json:"assetName"`
	FilingDate      string `json:"filingDate"`
	Name            string `json:"name"`
	OwnerType       string `json:"ownerType"`
	Position        string `json:"position"`
	Symbol          string `json:"symbol"`
	TransactionDate string `json:"transactionDate"`
	TransactionType string `json:"transactionType"`
}

func parseFinnhubResponse(body []byte) ([]Trade, error) {
	var resp finnhubResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse finnhub json: %w", err)
	}
	out := make([]Trade, 0, len(resp.Data))
	for _, r := range resp.Data {
		t := Trade{
			Source:        "finnhub",
			Filer:         strings.TrimSpace(r.Name),
			Role:          strings.TrimSpace(r.Position),
			Ticker:        normalizeTicker(r.Symbol),
			AssetDesc:     r.AssetName,
			Side:          normalizeSide(r.TransactionType),
			AmountLow:     r.AmountFrom,
			AmountHigh:    r.AmountTo,
			TransactionTS: parseDate(r.TransactionDate),
			FilingTS:      parseDate(r.FilingDate),
		}
		t.Hash = Hash(t)
		out = append(out, t)
	}
	return out, nil
}

// FetchFinnhubCongressional calls /stock/congressional-trading for a ticker,
// parses, and filters by cutoff filing_ts.
func (c *Client) FetchFinnhubCongressional(ctx context.Context, baseURL, apiKey, ticker string, cutoff int64) ([]Trade, error) {
	u := fmt.Sprintf("%s/stock/congressional-trading?symbol=%s&token=%s",
		strings.TrimRight(baseURL, "/"), url.QueryEscape(ticker), url.QueryEscape(apiKey))
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	trades, err := parseFinnhubResponse(body)
	if err != nil {
		return nil, err
	}
	if cutoff <= 0 {
		return trades, nil
	}
	filtered := trades[:0]
	for _, t := range trades {
		if t.FilingTS >= cutoff {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}
