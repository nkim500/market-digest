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

	require.True(t, cfg.Sources.Insiders["senate"].Enabled)
	require.False(t, cfg.Sources.Insiders["sec_form4"].Enabled)

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
