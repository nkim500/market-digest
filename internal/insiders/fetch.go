package insiders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ClientOptions struct {
	Timeout    time.Duration
	MaxRetries int
	BackoffMS  int
	UserAgent  string
}

type Client struct {
	http *http.Client
	opts ClientOptions
}

func NewClient(opts ClientOptions) *Client {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.BackoffMS == 0 {
		opts.BackoffMS = 1000
	}
	return &Client{
		http: &http.Client{Timeout: opts.Timeout},
		opts: opts,
	}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= c.opts.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		if c.opts.UserAgent != "" {
			req.Header.Set("User-Agent", c.opts.UserAgent)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
		} else {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return nil, err
				}
				return body, nil
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return nil, lastErr // don't retry 4xx
			}
		}
		if attempt < c.opts.MaxRetries {
			time.Sleep(time.Duration(c.opts.BackoffMS) * time.Millisecond * time.Duration(attempt))
		}
	}
	return nil, fmt.Errorf("fetch %s: %w", url, lastErr)
}

type senateRaw struct {
	TransactionDate  string `json:"transaction_date"`
	Ticker           string `json:"ticker"`
	AssetDescription string `json:"asset_description"`
	Type             string `json:"type"`
	Amount           string `json:"amount"`
	Senator          string `json:"senator"`
	PTRLink          string `json:"ptr_link"`
	DisclosureDate   string `json:"disclosure_date"`
}

func (c *Client) FetchSenate(ctx context.Context, url string) ([]Trade, error) {
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	var raws []senateRaw
	if err := json.Unmarshal(body, &raws); err != nil {
		return nil, fmt.Errorf("parse senate json: %w", err)
	}
	out := make([]Trade, 0, len(raws))
	for _, r := range raws {
		t := Trade{
			Source:        "senate",
			Filer:         strings.TrimSpace(r.Senator),
			Role:          "Senator",
			Ticker:        normalizeTicker(r.Ticker),
			AssetDesc:     r.AssetDescription,
			Side:          normalizeSide(r.Type),
			TransactionTS: parseDate(r.TransactionDate),
			FilingTS:      parseDate(r.DisclosureDate),
			RawURL:        r.PTRLink,
		}
		t.AmountLow, t.AmountHigh = parseAmountRange(r.Amount)
		t.Hash = Hash(t)
		out = append(out, t)
	}
	return out, nil
}

type houseRaw struct {
	TransactionDate  string `json:"transaction_date"`
	Ticker           string `json:"ticker"`
	AssetDescription string `json:"asset_description"`
	Type             string `json:"type"`
	Amount           string `json:"amount"`
	Representative   string `json:"representative"`
	PTRLink          string `json:"ptr_link"`
	DisclosureDate   string `json:"disclosure_date"`
}

func (c *Client) FetchHouse(ctx context.Context, url string) ([]Trade, error) {
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	var raws []houseRaw
	if err := json.Unmarshal(body, &raws); err != nil {
		return nil, fmt.Errorf("parse house json: %w", err)
	}
	out := make([]Trade, 0, len(raws))
	for _, r := range raws {
		t := Trade{
			Source:        "house",
			Filer:         strings.TrimSpace(r.Representative),
			Role:          "Representative",
			Ticker:        normalizeTicker(r.Ticker),
			AssetDesc:     r.AssetDescription,
			Side:          normalizeSide(r.Type),
			TransactionTS: parseDate(r.TransactionDate),
			FilingTS:      parseDate(r.DisclosureDate),
			RawURL:        r.PTRLink,
		}
		t.AmountLow, t.AmountHigh = parseAmountRange(r.Amount)
		t.Hash = Hash(t)
		out = append(out, t)
	}
	return out, nil
}

func normalizeTicker(raw string) string {
	t := strings.ToUpper(strings.TrimSpace(raw))
	if t == "--" || t == "N/A" || t == "" {
		return ""
	}
	return t
}

func normalizeSide(typ string) string {
	low := strings.ToLower(typ)
	switch {
	case strings.Contains(low, "purchase"), strings.Contains(low, "buy"):
		return "buy"
	case strings.Contains(low, "sale"), strings.Contains(low, "sell"):
		return "sell"
	case strings.Contains(low, "exchange"):
		return "exchange"
	default:
		return ""
	}
}

func parseDate(s string) int64 {
	if s == "" {
		return 0
	}
	for _, layout := range []string{"2006-01-02", "01/02/2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Unix()
		}
	}
	return 0
}

var amountNumRE = regexp.MustCompile(`\$?([\d,]+)`)

func parseAmountRange(s string) (low, high int) {
	matches := amountNumRE.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, 0
	}
	nums := make([]int, 0, len(matches))
	for _, m := range matches {
		n, err := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
		if err != nil {
			continue
		}
		nums = append(nums, n)
	}
	switch len(nums) {
	case 0:
		return 0, 0
	case 1:
		return nums[0], nums[0]
	default:
		return nums[0], nums[len(nums)-1]
	}
}
