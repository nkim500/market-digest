package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/config"
)

func TestLoad_readsExampleFiles(t *testing.T) {
	root, err := filepath.Abs("../..")
	require.NoError(t, err)

	cfg, err := config.Load(root)
	require.NoError(t, err)

	require.Equal(t, "America/New_York", cfg.Profile.User.Timezone)
	require.Equal(t, 500000, cfg.Profile.Reporting.DollarThresholds.Watch)
	require.Equal(t, 1000000, cfg.Profile.Reporting.DollarThresholds.Act)

	require.NotEmpty(t, cfg.Watchlist.Tickers)
	require.Equal(t, "NVDA", cfg.Watchlist.Tickers[0].Ticker)

	require.True(t, cfg.Sources.Insiders["sec_form4"].Enabled)
	require.Equal(t, "SEC_USER_AGENT", cfg.Sources.Insiders["sec_form4"].UserAgentEnv)
	require.False(t, cfg.Sources.Insiders["quiver"].Enabled) // dormant until QUIVER_API_KEY is set
	require.Equal(t, 7, cfg.Sources.MaxLookbackDays)

	require.Equal(t, "watch", cfg.Sources.AlertRules.WatchlistHit.Severity)
}

func TestLoad_prefersNonExampleYml(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "config"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "profile.example.yml"), []byte(`
user: {display_name: "Example", timezone: "UTC"}
reporting: {dollar_thresholds: {watch: 1, act: 2}, cluster_window_days: 1}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "profile.yml"), []byte(`
user: {display_name: "Real", timezone: "America/New_York"}
reporting: {dollar_thresholds: {watch: 500000, act: 1000000}, cluster_window_days: 7}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "watchlist.example.yml"), []byte(`tickers: []`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "sources.yml"), []byte(`
insiders: {}
alert_rules: {watchlist_hit: {severity: watch}, amount_over_500k: {severity: watch}, amount_over_1m: {severity: act}, cluster_3_in_7d: {severity: watch}}
http: {timeout_seconds: 30, max_retries: 3, backoff_ms: 1000}
`), 0o644))

	cfg, err := config.Load(dir)
	require.NoError(t, err)
	require.Equal(t, "Real", cfg.Profile.User.DisplayName)
}

func TestSourcesParsesMaxLookbackAndEnvRefs(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "sources.yml"), []byte(`
max_lookback_days: 7

insiders:
  sec_form4:
    enabled: true
    user_agent_env: SEC_USER_AGENT
  finnhub:
    enabled: true
    api_key_env: FINNHUB_API_KEY
  quiver:
    enabled: false
    api_key_env: QUIVER_API_KEY

alert_rules:
  watchlist_hit:    { severity: watch }
  amount_over_500k: { severity: watch }
  amount_over_1m:   { severity: act }
  cluster_3_in_7d:  { severity: watch }

http:
  timeout_seconds: 30
  max_retries: 3
  backoff_ms: 1000
`), 0o644); err != nil {
		t.Fatalf("write sources.yml: %v", err)
	}
	// profile + watchlist are required by Load; write minimal stubs.
	if err := os.WriteFile(filepath.Join(configDir, "profile.yml"), []byte("user:\n  display_name: test\n"), 0o644); err != nil {
		t.Fatalf("write profile.yml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "watchlist.yml"), []byte("tickers: []\n"), 0o644); err != nil {
		t.Fatalf("write watchlist.yml: %v", err)
	}

	cfg, err := config.Load(tmp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Sources.MaxLookbackDays != 7 {
		t.Errorf("MaxLookbackDays = %d; want 7", cfg.Sources.MaxLookbackDays)
	}
	sec := cfg.Sources.Insiders["sec_form4"]
	if !sec.Enabled {
		t.Error("sec_form4 should be enabled")
	}
	if sec.UserAgentEnv != "SEC_USER_AGENT" {
		t.Errorf("sec_form4.UserAgentEnv = %q; want SEC_USER_AGENT", sec.UserAgentEnv)
	}
	fh := cfg.Sources.Insiders["finnhub"]
	if fh.APIKeyEnv != "FINNHUB_API_KEY" {
		t.Errorf("finnhub.APIKeyEnv = %q; want FINNHUB_API_KEY", fh.APIKeyEnv)
	}
	qv := cfg.Sources.Insiders["quiver"]
	if qv.Enabled {
		t.Error("quiver should be disabled by default")
	}
	if qv.APIKeyEnv != "QUIVER_API_KEY" {
		t.Errorf("quiver.APIKeyEnv = %q; want QUIVER_API_KEY", qv.APIKeyEnv)
	}
}
