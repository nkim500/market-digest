package insiders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type quiverRow struct {
	Representative  string `json:"Representative"`
	ReportDate      string `json:"ReportDate"`
	TransactionDate string `json:"TransactionDate"`
	Ticker          string `json:"Ticker"`
	Transaction     string `json:"Transaction"`
	Range           string `json:"Range"`
	House           string `json:"House"`
	Amount          int    `json:"Amount"`
	Party           string `json:"Party"`
}

func parseQuiverResponse(body []byte) ([]Trade, error) {
	var rows []quiverRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("parse quiver json: %w", err)
	}
	out := make([]Trade, 0, len(rows))
	for _, r := range rows {
		low, high := parseAmountRange(r.Range)
		if high == 0 && r.Amount > 0 {
			high = r.Amount
			if low == 0 {
				low = r.Amount
			}
		}
		t := Trade{
			Source:        "quiver",
			Filer:         strings.TrimSpace(r.Representative),
			Role:          strings.TrimSpace(r.House),
			Ticker:        normalizeTicker(r.Ticker),
			Side:          normalizeSide(r.Transaction),
			AmountLow:     low,
			AmountHigh:    high,
			TransactionTS: parseDate(r.TransactionDate),
			FilingTS:      parseDate(r.ReportDate),
		}
		t.Hash = Hash(t)
		out = append(out, t)
	}
	return out, nil
}

// FetchQuiverCongressional calls /historical/congresstrading/{ticker} with a Bearer token.
func (c *Client) FetchQuiverCongressional(ctx context.Context, baseURL, apiKey, ticker string, cutoff int64) ([]Trade, error) {
	u := fmt.Sprintf("%s/historical/congresstrading/%s", strings.TrimRight(baseURL, "/"), ticker)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if c.opts.UserAgent != "" {
		req.Header.Set("User-Agent", c.opts.UserAgent)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("quiver http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	trades, err := parseQuiverResponse(body)
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
