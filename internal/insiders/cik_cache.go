package insiders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ErrCIKNotFound signals that a ticker has no SEC mapping (ADR, OTC, foreign issuer).
var ErrCIKNotFound = errors.New("CIK not found for ticker")

const defaultCIKSourceURL = "https://www.sec.gov/files/company_tickers.json"

type CIKCacheOptions struct {
	SourceURL string        // SEC tickers JSON URL; defaults to official endpoint if empty
	CachePath string        // absolute path to the on-disk cache file
	MaxAge    time.Duration // refresh if file is older than this
	UserAgent string        // SEC-policy-compliant UA for fetches
}

type CIKCache struct {
	opts CIKCacheOptions
	mu   sync.Mutex
	mem  map[string]string // ticker (upper) -> zero-padded CIK
}

func NewCIKCache(opts CIKCacheOptions) *CIKCache {
	if opts.SourceURL == "" {
		opts.SourceURL = defaultCIKSourceURL
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 30 * 24 * time.Hour
	}
	return &CIKCache{opts: opts}
}

// Resolve returns the 10-digit zero-padded CIK for a ticker. Loads the cache
// on first call (disk or fresh fetch), returns ErrCIKNotFound for missing tickers.
func (c *CIKCache) Resolve(ctx context.Context, ticker string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mem == nil {
		if err := c.load(ctx); err != nil {
			return "", err
		}
	}
	cik, ok := c.mem[strings.ToUpper(strings.TrimSpace(ticker))]
	if !ok {
		return "", ErrCIKNotFound
	}
	return cik, nil
}

// load populates c.mem from disk if the file is fresh, otherwise refetches.
func (c *CIKCache) load(ctx context.Context) error {
	body, err := c.readFreshFile()
	if err == nil {
		return c.parse(body)
	}
	if !os.IsNotExist(err) && !errors.Is(err, errCacheStale) {
		return err
	}
	body, err = c.fetch(ctx)
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.opts.CachePath, body, 0o644); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	return c.parse(body)
}

var errCacheStale = errors.New("cache stale")

func (c *CIKCache) readFreshFile() ([]byte, error) {
	fi, err := os.Stat(c.opts.CachePath)
	if err != nil {
		return nil, err
	}
	if time.Since(fi.ModTime()) > c.opts.MaxAge {
		return nil, errCacheStale
	}
	return os.ReadFile(c.opts.CachePath)
}

func (c *CIKCache) fetch(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.opts.SourceURL, nil)
	if err != nil {
		return nil, err
	}
	if c.opts.UserAgent != "" {
		req.Header.Set("User-Agent", c.opts.UserAgent)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch CIK map: http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// parse handles SEC's native JSON shape: {"0":{"cik_str":N,"ticker":"X","title":"..."}}
func (c *CIKCache) parse(body []byte) error {
	var raw map[string]struct {
		CIKStr int    `json:"cik_str"`
		Ticker string `json:"ticker"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("parse CIK JSON: %w", err)
	}
	c.mem = make(map[string]string, len(raw))
	for _, entry := range raw {
		c.mem[strings.ToUpper(entry.Ticker)] = fmt.Sprintf("%010d", entry.CIKStr)
	}
	return nil
}
