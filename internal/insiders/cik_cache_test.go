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

const fakeCIKJSON = `{
	"0": {"cik_str": 1045810, "ticker": "NVDA", "title": "NVIDIA Corp"},
	"1": {"cik_str": 320193,  "ticker": "AAPL", "title": "Apple Inc"},
	"2": {"cik_str": 1757898, "ticker": "ICHR", "title": "Ichor Holdings"}
}`

func TestCIKCacheColdLoadFetchesAndWritesFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fakeCIKJSON))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	cache := NewCIKCache(CIKCacheOptions{
		SourceURL: srv.URL,
		CachePath: filepath.Join(tmp, "cik_cache.json"),
		MaxAge:    30 * 24 * time.Hour,
		UserAgent: "market-digest test",
	})

	got, err := cache.Resolve(context.Background(), "NVDA")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "0001045810" {
		t.Errorf("NVDA CIK = %q, want 0001045810 (10-digit padded)", got)
	}

	if _, err := os.Stat(filepath.Join(tmp, "cik_cache.json")); err != nil {
		t.Errorf("cache file not written: %v", err)
	}
}

func TestCIKCacheWarmLoadSkipsFetch(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(fakeCIKJSON))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	cache := NewCIKCache(CIKCacheOptions{
		SourceURL: srv.URL,
		CachePath: filepath.Join(tmp, "cik_cache.json"),
		MaxAge:    30 * 24 * time.Hour,
		UserAgent: "market-digest test",
	})

	if _, err := cache.Resolve(context.Background(), "NVDA"); err != nil {
		t.Fatalf("cold resolve: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected 1 fetch, got %d", hits)
	}

	// New cache instance pointed at the same file — should read, not fetch.
	cache2 := NewCIKCache(CIKCacheOptions{
		SourceURL: srv.URL,
		CachePath: filepath.Join(tmp, "cik_cache.json"),
		MaxAge:    30 * 24 * time.Hour,
		UserAgent: "market-digest test",
	})
	if _, err := cache2.Resolve(context.Background(), "AAPL"); err != nil {
		t.Fatalf("warm resolve: %v", err)
	}
	if hits != 1 {
		t.Errorf("warm read triggered a fetch; hits=%d", hits)
	}
}

func TestCIKCacheStaleFileTriggersRefresh(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(fakeCIKJSON))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "cik_cache.json")
	if err := os.WriteFile(cachePath, []byte(fakeCIKJSON), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Backdate the file by 40 days.
	old := time.Now().Add(-40 * 24 * time.Hour)
	if err := os.Chtimes(cachePath, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	cache := NewCIKCache(CIKCacheOptions{
		SourceURL: srv.URL,
		CachePath: cachePath,
		MaxAge:    30 * 24 * time.Hour,
		UserAgent: "market-digest test",
	})
	if _, err := cache.Resolve(context.Background(), "NVDA"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if hits != 1 {
		t.Errorf("expected refresh to fetch once; hits=%d", hits)
	}
}

func TestCIKCacheUnknownTickerReturnsErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fakeCIKJSON))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	cache := NewCIKCache(CIKCacheOptions{
		SourceURL: srv.URL,
		CachePath: filepath.Join(tmp, "cik_cache.json"),
		MaxAge:    30 * 24 * time.Hour,
		UserAgent: "market-digest test",
	})

	_, err := cache.Resolve(context.Background(), "ZZZZ")
	if err == nil {
		t.Fatal("expected error for unknown ticker")
	}
	if err != ErrCIKNotFound {
		t.Errorf("expected ErrCIKNotFound, got %v", err)
	}
}
