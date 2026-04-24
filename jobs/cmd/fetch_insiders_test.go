package cmd_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/config"
	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/jobs/cmd"
)

func TestFetchInsidersRun_endToEnd(t *testing.T) {
	senateBody, err := os.ReadFile("../../internal/insiders/testdata/senate_sample.json")
	require.NoError(t, err)
	houseBody, err := os.ReadFile("../../internal/insiders/testdata/house_sample.json")
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/senate", func(w http.ResponseWriter, _ *http.Request) { w.Write(senateBody) })
	mux.HandleFunc("/house", func(w http.ResponseWriter, _ *http.Request) { w.Write(houseBody) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "migrations"), 0o755))
	migSrc := "../../migrations"
	entries, err := os.ReadDir(migSrc)
	require.NoError(t, err)
	for _, e := range entries {
		src, err := os.ReadFile(filepath.Join(migSrc, e.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(home, "migrations", e.Name()), src, 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "profile.yml"), []byte(`
user: {display_name: "Test", timezone: "UTC"}
reporting: {dollar_thresholds: {watch: 500000, act: 1000000}, cluster_window_days: 7}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "watchlist.yml"), []byte(`
tickers:
  - {ticker: NVDA, note: "", added: 2026-01-01}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "sources.yml"), []byte(`
insiders:
  senate: {url: "`+srv.URL+`/senate", enabled: true}
  house:  {url: "`+srv.URL+`/house",  enabled: true}
  sec_form4: {url: "", enabled: false}
alert_rules:
  watchlist_hit:    {severity: watch}
  amount_over_500k: {severity: watch}
  amount_over_1m:   {severity: act}
  cluster_3_in_7d:  {severity: watch}
http: {timeout_seconds: 5, max_retries: 2, backoff_ms: 1}
`), 0o644))

	ctx := context.Background()
	rowsIn, rowsNew, err := cmd.RunFetchInsiders(ctx, home)
	require.NoError(t, err)
	require.Equal(t, 3, rowsIn, "2 senate + 1 house rows from fixtures")
	require.Equal(t, 3, rowsNew)

	_, rowsNew2, err := cmd.RunFetchInsiders(ctx, home)
	require.NoError(t, err)
	require.Equal(t, 0, rowsNew2)

	conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
	require.NoError(t, err)
	defer conn.Close()
	var watchCount int
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM alerts WHERE source='insiders' AND severity='watch' AND ticker='NVDA'`,
	).Scan(&watchCount))
	require.GreaterOrEqual(t, watchCount, 1)

	cfg, err := config.Load(home)
	require.NoError(t, err)
	require.True(t, cfg.Sources.Insiders["senate"].Enabled)
}
