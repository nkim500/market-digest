# Market-Digest Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold a fork-friendly skeleton (Go TUI dashboard + Go jobs binary + SQLite + Claude modes layer) with one fully-wired end-to-end example (insider/politician trades from Senate/House Stock Watcher), matching the design in [`../specs/2026-04-23-market-digest-skeleton-design.md`](../specs/2026-04-23-market-digest-skeleton-design.md).

**Architecture:** Single Go module at repo root producing two binaries (`dashboard`, `jobs`) that share packages in `internal/`. SQLite via pure-Go `modernc.org/sqlite` (no cgo) is the contract between the Go runtime and the Claude modes layer (markdown files invoked via slash command). OS cron/launchd drives scheduled fetches; alerts are rows in the DB.

**Tech Stack:** Go 1.24, `modernc.org/sqlite`, `github.com/spf13/cobra`, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `github.com/stretchr/testify/require`, `gopkg.in/yaml.v3`.

**Plan-board protocol:** Per spec §10, this plan should be copied to `implementation_plan.md` (gitignored) at the start of execution. Agents mark phases `[IN PROGRESS — <id> — <ts>]` before starting and `[DONE — <PR or "local">]` after. See CLAUDE.md (created in Task 17) for the full protocol.

---

## Phase roadmap

| Phase | Tasks | Milestone |
|-------|-------|-----------|
| 1. Foundation | 1–3 | `jobs migrate` creates `data/digest.db` |
| 2. Shared packages | 4–6 | Config loads; `jobrun.Track` records runs; `alert.Insert` writes alerts |
| 3. Jobs binary | 7–12 | `jobs fetch-insiders` works end-to-end; `jobs list-alerts` returns rows |
| 4. Dashboard TUI | 13–16 | `dashboard` opens, shows alerts/watchlist, marks seen |
| 5. Modes + skill | 17–19 | `/digest insiders` and `/digest alerts` work in Claude Code |
| 6. Scheduling + docs | 20–22 | launchd plist installs; docs cover getting-started + extension |

---

## Phase 1 — Foundation

### Task 1: Scaffold Go module and repo files

**Files:**
- Create: `go.mod`, `.gitignore`, `README.md`, `Makefile`, `data/.gitkeep`

- [ ] **Step 1: Initialize the Go module**

```bash
cd /path/to/market-digest   # repo root (currently `day-trader-digest`, rename later)
go mod init github.com/nkim500/market-digest
```

Expected: creates `go.mod` with `module github.com/nkim500/market-digest` and `go 1.24` (or current stable).

- [ ] **Step 2: Write `.gitignore`**

Create `.gitignore` with exactly (matches spec §4):

```gitignore
# Binaries
dashboard/dashboard
jobs/jobs
/bin/

# Local DB and derived data
data/*.db
data/*.db-*
data/*.csv
!data/.gitkeep

# User config (only *.example.yml is tracked)
config/*.yml
!config/*.example.yml
!config/sources.yml

# User profile
modes/_profile.md

# Local Claude context
CLAUDE.local.md
.claude/settings.local.json
.claude/memory/

# Agent plan board
implementation_plan.md

# OS noise
.DS_Store

# Go build artifacts
*.out
*.test
*.prof

# Reports written by modes
data/reports/
```

- [ ] **Step 3: Create `data/.gitkeep`**

```bash
mkdir -p data && touch data/.gitkeep
```

- [ ] **Step 4: Write `README.md`**

Create with this content:

```markdown
# market-digest

A personal, fork-friendly tool for ideating around US equity positions using free public data and Claude as the synthesis engine. Inspired by [santifer/career-ops](https://github.com/santifer/career-ops).

**What it does (v1):** Fetches politician stock disclosures from Senate/House Stock Watcher, surfaces meaningful activity on your watchlist, and lets Claude narrate what matters via `/digest insiders`.

**What it is not:** A trading bot. A real-time tool. A substitute for licensed research.

See [`docs/getting-started.md`](docs/getting-started.md) and the [design spec](docs/superpowers/specs/2026-04-23-market-digest-skeleton-design.md).

## Quick start

```bash
make build            # produces bin/dashboard and bin/jobs
make migrate          # creates data/digest.db
cp config/profile.example.yml config/profile.yml     # edit to taste
cp config/watchlist.example.yml config/watchlist.yml # edit to taste
./bin/jobs fetch-insiders
./bin/dashboard
```

## Disclaimer

This tool surfaces public information for educational and ideational purposes. It is not investment advice. All data comes from public disclosures; no non-public information is used.
```

- [ ] **Step 5: Write `Makefile`**

```makefile
.PHONY: build migrate dashboard jobs test lint clean

BIN := bin
GO  := go

build: $(BIN)/jobs $(BIN)/dashboard

$(BIN)/jobs:
	$(GO) build -o $(BIN)/jobs ./jobs

$(BIN)/dashboard:
	$(GO) build -o $(BIN)/dashboard ./dashboard

migrate: $(BIN)/jobs
	./$(BIN)/jobs migrate

dashboard: $(BIN)/dashboard
	./$(BIN)/dashboard

jobs: $(BIN)/jobs

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...

clean:
	rm -rf $(BIN) data/*.db data/*.db-*
```

- [ ] **Step 6: Verify and commit**

```bash
go build ./...   # should succeed with nothing to build
git add go.mod .gitignore README.md Makefile data/.gitkeep
git commit -m "chore: scaffold go module and repo structure"
```

Expected: commit succeeds, `git status` is clean.

---

### Task 2: SQLite driver and `db.Open`

**Files:**
- Create: `internal/db/db.go`, `internal/db/db_test.go`
- Modify: `go.mod`, `go.sum` (via `go get`)

- [ ] **Step 1: Add dependencies**

```bash
go get modernc.org/sqlite@latest
go get github.com/stretchr/testify@latest
```

Expected: `go.mod` gains two require lines; `go.sum` populated.

- [ ] **Step 2: Write the failing test**

Create `internal/db/db_test.go`:

```go
package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/db"
)

func TestOpen_createsFileAndEnablesWAL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	conn, err := db.Open(context.Background(), path)
	require.NoError(t, err)
	defer conn.Close()

	var mode string
	require.NoError(t, conn.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&mode))
	require.Equal(t, "wal", mode)

	var busy int
	require.NoError(t, conn.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&busy))
	require.Equal(t, 5000, busy)
}

func TestOpen_parentDirCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "path", "test.db")

	conn, err := db.Open(context.Background(), path)
	require.NoError(t, err)
	defer conn.Close()
}
```

- [ ] **Step 3: Run test — confirm it fails**

```bash
go test ./internal/db/...
```

Expected: FAIL with `no Go files in .../internal/db` or `undefined: db.Open`.

- [ ] **Step 4: Implement `internal/db/db.go`**

```go
// Package db owns sqlite open + migrations for market-digest.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open opens (and creates if needed) a sqlite database at path, returning a
// *sql.DB configured with WAL journaling and a 5s busy_timeout so the TUI can
// read concurrently with writes from the jobs binary.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	q := url.Values{}
	q.Set("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(ON)")
	dsn := fmt.Sprintf("file:%s?%s", path, q.Encode())

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	// modernc.org/sqlite accepts concurrent reads + a single writer.
	// One connection is the simplest correct choice at this scale.
	conn.SetMaxOpenConns(1)
	return conn, nil
}
```

- [ ] **Step 5: Run test — confirm it passes**

```bash
go test ./internal/db/...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/db/db.go internal/db/db_test.go
git commit -m "feat(db): pure-Go sqlite Open with WAL"
```

---

### Task 3: Migrations runner and `0001_init.sql`

**Files:**
- Create: `internal/db/migrate.go`, `internal/db/migrate_test.go`, `migrations/0001_init.sql`

- [ ] **Step 1: Write `migrations/0001_init.sql`**

```sql
-- 0001_init.sql
CREATE TABLE IF NOT EXISTS watchlist (
  ticker      TEXT PRIMARY KEY,
  note        TEXT,
  added_ts    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS alerts (
  id          INTEGER PRIMARY KEY,
  created_ts  INTEGER NOT NULL,
  source      TEXT NOT NULL,
  severity    TEXT NOT NULL,
  ticker      TEXT,
  title       TEXT NOT NULL,
  body        TEXT,
  payload     TEXT,
  seen_ts     INTEGER
);
CREATE INDEX IF NOT EXISTS alerts_unseen ON alerts(seen_ts) WHERE seen_ts IS NULL;
CREATE INDEX IF NOT EXISTS alerts_created ON alerts(created_ts DESC);

CREATE TABLE IF NOT EXISTS job_runs (
  id          INTEGER PRIMARY KEY,
  job         TEXT NOT NULL,
  started_ts  INTEGER NOT NULL,
  finished_ts INTEGER,
  status      TEXT NOT NULL,
  rows_in     INTEGER,
  rows_new    INTEGER,
  error       TEXT
);
CREATE INDEX IF NOT EXISTS job_runs_job_started ON job_runs(job, started_ts DESC);

CREATE TABLE IF NOT EXISTS schema_version (
  version     INTEGER PRIMARY KEY,
  applied_ts  INTEGER NOT NULL
);
```

- [ ] **Step 2: Write the failing test**

Create `internal/db/migrate_test.go`:

```go
package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/db"
)

func TestMigrate_appliesAllPendingThenIdempotent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()

	applied1, err := db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(applied1), 1, "expected at least 0001_init to apply")

	// Second run applies nothing.
	applied2, err := db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	require.Empty(t, applied2)

	// schema_version has rows.
	var count int
	require.NoError(t, conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_version").Scan(&count))
	require.Equal(t, len(applied1), count)
}

func TestMigrate_createsCoreTables(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()

	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	for _, table := range []string{"watchlist", "alerts", "job_runs", "schema_version"} {
		var got string
		err := conn.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&got)
		require.NoError(t, err, "missing table %s", table)
	}
}
```

- [ ] **Step 3: Run the test — confirm it fails**

```bash
go test ./internal/db/... -run TestMigrate
```

Expected: FAIL with `undefined: db.Migrate`.

- [ ] **Step 4: Implement `internal/db/migrate.go`**

```go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migrate applies every *.sql file in migrationsDir whose leading integer is
// greater than the max version recorded in schema_version. Returns the list of
// filenames applied, in order.
//
// Filename convention: NNNN_short_name.sql, where NNNN is a zero-padded
// monotonic integer. Filenames without a leading integer are skipped with an
// error.
func Migrate(ctx context.Context, conn *sql.DB, migrationsDir string) ([]string, error) {
	if _, err := conn.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_version (
			version    INTEGER PRIMARY KEY,
			applied_ts INTEGER NOT NULL
		)`,
	); err != nil {
		return nil, fmt.Errorf("bootstrap schema_version: %w", err)
	}

	applied, err := loadAppliedVersions(ctx, conn)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var pending []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		ver, err := parseVersion(e.Name())
		if err != nil {
			return nil, err
		}
		if _, ok := applied[ver]; ok {
			continue
		}
		pending = append(pending, migration{version: ver, filename: e.Name()})
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].version < pending[j].version })

	var appliedNames []string
	for _, m := range pending {
		body, err := os.ReadFile(filepath.Join(migrationsDir, m.filename))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", m.filename, err)
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("begin tx for %s: %w", m.filename, err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("apply %s: %w", m.filename, err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_version (version, applied_ts) VALUES (?, ?)",
			m.version, time.Now().Unix(),
		); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("record %s: %w", m.filename, err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit %s: %w", m.filename, err)
		}
		appliedNames = append(appliedNames, m.filename)
	}
	return appliedNames, nil
}

type migration struct {
	version  int
	filename string
}

func loadAppliedVersions(ctx context.Context, conn *sql.DB) (map[int]struct{}, error) {
	rows, err := conn.QueryContext(ctx, "SELECT version FROM schema_version")
	if err != nil {
		return nil, fmt.Errorf("load applied: %w", err)
	}
	defer rows.Close()
	out := map[int]struct{}{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = struct{}{}
	}
	return out, rows.Err()
}

func parseVersion(filename string) (int, error) {
	// Expect NNNN_<anything>.sql
	underscore := strings.IndexByte(filename, '_')
	if underscore <= 0 {
		return 0, fmt.Errorf("migration %q: no leading version", filename)
	}
	v, err := strconv.Atoi(filename[:underscore])
	if err != nil {
		return 0, fmt.Errorf("migration %q: version: %w", filename, err)
	}
	return v, nil
}
```

- [ ] **Step 5: Run tests — confirm they pass**

```bash
go test ./internal/db/...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add migrations/0001_init.sql internal/db/migrate.go internal/db/migrate_test.go
git commit -m "feat(db): migrations runner with 0001_init"
```

---

## Phase 2 — Shared packages

### Task 4: Config loaders

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`, `config/profile.example.yml`, `config/watchlist.example.yml`, `config/sources.yml`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add yaml dependency**

```bash
go get gopkg.in/yaml.v3@latest
```

- [ ] **Step 2: Write `config/profile.example.yml`**

```yaml
user:
  display_name: "Your Name"
  timezone: "America/New_York"

risk:
  tolerance: "moderate"          # conservative | moderate | aggressive
  max_position_pct: 5
  notes: "No leverage. No options outside watchlist."

interests:
  sectors: ["semis", "energy", "biotech"]
  themes: ["AI infra", "onshoring", "grid buildout"]
  avoid: ["crypto-adjacent penny stocks"]

reporting:
  dollar_thresholds:
    watch: 500000
    act:   1000000
  cluster_window_days: 7

notifications:
  macos: false
  webhook_url: ""
```

- [ ] **Step 3: Write `config/watchlist.example.yml`**

```yaml
tickers:
  - ticker: NVDA
    note: "Core AI infra hold"
    added: 2026-01-15
  - ticker: CEG
    note: "Grid/nuclear thesis"
    added: 2026-02-01
```

- [ ] **Step 4: Write `config/sources.yml`** (tracked, not gitignored)

```yaml
insiders:
  senate:
    url: "https://raw.githubusercontent.com/jeremiak/senate-stock-watcher-data/main/aggregate/all_transactions.json"
    enabled: true
  house:
    url: "https://raw.githubusercontent.com/aviaryan/house-stock-watcher-data/main/transactions/all_transactions.json"
    enabled: true
  sec_form4:
    url: "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&type=4&output=atom"
    enabled: false
    user_agent: "market-digest <your-email@example.com>"

alert_rules:
  watchlist_hit:    { severity: watch }
  amount_over_500k: { severity: watch }
  amount_over_1m:   { severity: act }
  cluster_3_in_7d:  { severity: watch }

http:
  timeout_seconds: 30
  max_retries: 3
  backoff_ms: 1000
```

- [ ] **Step 5: Write the failing test**

Create `internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/config"
)

func TestLoad_readsExampleFiles(t *testing.T) {
	// The repo root is two levels up from this test file.
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
	// Watchlist + sources falls back to example.
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
```

- [ ] **Step 6: Run test — confirm it fails**

```bash
go test ./internal/config/...
```

Expected: FAIL (undefined package).

- [ ] **Step 7: Implement `internal/config/config.go`**

```go
// Package config loads market-digest YAML configuration from <root>/config/.
// It prefers *.yml over *.example.yml so users can override without editing
// tracked files.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Profile   Profile
	Watchlist Watchlist
	Sources   Sources
}

type Profile struct {
	User struct {
		DisplayName string `yaml:"display_name"`
		Timezone    string `yaml:"timezone"`
	} `yaml:"user"`
	Risk struct {
		Tolerance      string `yaml:"tolerance"`
		MaxPositionPct int    `yaml:"max_position_pct"`
		Notes          string `yaml:"notes"`
	} `yaml:"risk"`
	Interests struct {
		Sectors []string `yaml:"sectors"`
		Themes  []string `yaml:"themes"`
		Avoid   []string `yaml:"avoid"`
	} `yaml:"interests"`
	Reporting struct {
		DollarThresholds struct {
			Watch int `yaml:"watch"`
			Act   int `yaml:"act"`
		} `yaml:"dollar_thresholds"`
		ClusterWindowDays int `yaml:"cluster_window_days"`
	} `yaml:"reporting"`
	Notifications struct {
		MacOS      bool   `yaml:"macos"`
		WebhookURL string `yaml:"webhook_url"`
	} `yaml:"notifications"`
}

type Watchlist struct {
	Tickers []WatchlistEntry `yaml:"tickers"`
}

type WatchlistEntry struct {
	Ticker string `yaml:"ticker"`
	Note   string `yaml:"note"`
	Added  string `yaml:"added"`
}

type Sources struct {
	Insiders   map[string]InsiderSource `yaml:"insiders"`
	AlertRules AlertRules               `yaml:"alert_rules"`
	HTTP       HTTPConfig               `yaml:"http"`
}

type InsiderSource struct {
	URL       string `yaml:"url"`
	Enabled   bool   `yaml:"enabled"`
	UserAgent string `yaml:"user_agent,omitempty"`
}

type AlertRules struct {
	WatchlistHit    AlertRule `yaml:"watchlist_hit"`
	AmountOver500k  AlertRule `yaml:"amount_over_500k"`
	AmountOver1m    AlertRule `yaml:"amount_over_1m"`
	Cluster3In7d    AlertRule `yaml:"cluster_3_in_7d"`
}

type AlertRule struct {
	Severity string `yaml:"severity"`
}

type HTTPConfig struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
	MaxRetries     int `yaml:"max_retries"`
	BackoffMS      int `yaml:"backoff_ms"`
}

// Load reads config from <root>/config, preferring *.yml over *.example.yml.
// Missing optional files are tolerated only where a fallback makes sense.
func Load(root string) (Config, error) {
	var cfg Config
	if err := loadYAML(filepath.Join(root, "config"), "profile", &cfg.Profile); err != nil {
		return cfg, err
	}
	if err := loadYAML(filepath.Join(root, "config"), "watchlist", &cfg.Watchlist); err != nil {
		return cfg, err
	}
	if err := loadYAML(filepath.Join(root, "config"), "sources", &cfg.Sources); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func loadYAML(dir, name string, out any) error {
	// Prefer user file, fall back to example.
	for _, candidate := range []string{name + ".yml", name + ".example.yml"} {
		path := filepath.Join(dir, candidate)
		body, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", candidate, err)
		}
		if err := yaml.Unmarshal(body, out); err != nil {
			return fmt.Errorf("parse %s: %w", candidate, err)
		}
		return nil
	}
	return fmt.Errorf("config file %s.yml (or .example.yml) not found in %s", name, dir)
}
```

- [ ] **Step 8: Run tests — confirm they pass**

```bash
go test ./internal/config/...
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add config/ internal/config/ go.mod go.sum
git commit -m "feat(config): yaml loader with *.example.yml fallback"
```

---

### Task 5: `jobrun` helper

**Files:**
- Create: `internal/jobrun/jobrun.go`, `internal/jobrun/jobrun_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/jobrun/jobrun_test.go`:

```go
package jobrun_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/internal/jobrun"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	return conn
}

func TestTrack_successRecordsOK(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	err := jobrun.Track(ctx, conn, "demo", func(ctx context.Context) (int, int, error) {
		return 10, 3, nil
	})
	require.NoError(t, err)

	var status string
	var rowsIn, rowsNew sql.NullInt64
	var errText sql.NullString
	err = conn.QueryRowContext(ctx,
		"SELECT status, rows_in, rows_new, error FROM job_runs WHERE job='demo'",
	).Scan(&status, &rowsIn, &rowsNew, &errText)
	require.NoError(t, err)
	require.Equal(t, "ok", status)
	require.Equal(t, int64(10), rowsIn.Int64)
	require.Equal(t, int64(3), rowsNew.Int64)
	require.False(t, errText.Valid)
}

func TestTrack_errorRecordsError(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	err := jobrun.Track(ctx, conn, "demo", func(ctx context.Context) (int, int, error) {
		return 0, 0, errors.New("boom")
	})
	require.Error(t, err)

	var status string
	var errText sql.NullString
	require.NoError(t, conn.QueryRowContext(ctx,
		"SELECT status, error FROM job_runs WHERE job='demo'",
	).Scan(&status, &errText))
	require.Equal(t, "error", status)
	require.Contains(t, errText.String, "boom")
}

func TestTrack_panicRecordsError(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	require.Panics(t, func() {
		_ = jobrun.Track(ctx, conn, "demo", func(ctx context.Context) (int, int, error) {
			panic("kaboom")
		})
	})

	var status string
	var errText sql.NullString
	require.NoError(t, conn.QueryRowContext(ctx,
		"SELECT status, error FROM job_runs WHERE job='demo'",
	).Scan(&status, &errText))
	require.Equal(t, "error", status)
	require.Contains(t, errText.String, "kaboom")
}

func TestTrack_noopStatus(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	err := jobrun.Track(ctx, conn, "demo", func(ctx context.Context) (int, int, error) {
		return 0, 0, jobrun.ErrNoop
	})
	require.NoError(t, err)

	var status string
	require.NoError(t, conn.QueryRowContext(ctx,
		"SELECT status FROM job_runs WHERE job='demo'",
	).Scan(&status))
	require.Equal(t, "noop", status)
}
```

- [ ] **Step 2: Run test — confirm it fails**

```bash
go test ./internal/jobrun/...
```

Expected: FAIL — undefined package.

- [ ] **Step 3: Implement `internal/jobrun/jobrun.go`**

```go
// Package jobrun wraps a job invocation in a job_runs row, capturing
// start/finish timestamps, result counts, errors, and panics.
package jobrun

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime/debug"
	"time"
)

// ErrNoop lets a job signal "nothing new to do, success." Reported as
// status='noop' in job_runs.
var ErrNoop = errors.New("jobrun: no-op")

// Fn is the job body. Returns (rows_in, rows_new, err).
type Fn func(ctx context.Context) (rowsIn, rowsNew int, err error)

// Track inserts a row with status='running', invokes fn, and updates the row
// with the final status + counts on exit. Panics are captured as status='error'
// and re-raised after the DB is updated.
func Track(ctx context.Context, conn *sql.DB, job string, fn Fn) (retErr error) {
	res, err := conn.ExecContext(ctx,
		`INSERT INTO job_runs (job, started_ts, status) VALUES (?, ?, 'running')`,
		job, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert job_runs: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last id: %w", err)
	}

	var (
		rowsIn  int
		rowsNew int
		runErr  error
		panicV  any
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicV = r
			}
		}()
		rowsIn, rowsNew, runErr = fn(ctx)
	}()

	status := "ok"
	var errText sql.NullString
	switch {
	case panicV != nil:
		status = "error"
		errText = sql.NullString{String: fmt.Sprintf("panic: %v\n%s", panicV, debug.Stack()), Valid: true}
	case errors.Is(runErr, ErrNoop):
		status = "noop"
	case runErr != nil:
		status = "error"
		errText = sql.NullString{String: runErr.Error(), Valid: true}
	}

	if _, err := conn.ExecContext(ctx,
		`UPDATE job_runs
		 SET finished_ts=?, status=?, rows_in=?, rows_new=?, error=?
		 WHERE id=?`,
		time.Now().Unix(), status, rowsIn, rowsNew, errText, id,
	); err != nil {
		// Log but don't swallow the original error.
		fmt.Printf("jobrun: finalize failed: %v\n", err)
	}

	if panicV != nil {
		panic(panicV) // preserve original panic for process exit
	}
	if runErr != nil && !errors.Is(runErr, ErrNoop) {
		return runErr
	}
	return nil
}
```

- [ ] **Step 4: Run tests — confirm they pass**

```bash
go test ./internal/jobrun/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/jobrun/
git commit -m "feat(jobrun): track job runs with panic recovery"
```

---

### Task 6: `alert` helper

**Files:**
- Create: `internal/alert/alert.go`, `internal/alert/alert_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/alert/alert_test.go`:

```go
package alert_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/db"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	return conn
}

func TestInsert_setsFields(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	ticker := "NVDA"
	id, err := alert.Insert(ctx, conn, alert.Alert{
		Source:   "insiders",
		Severity: "act",
		Ticker:   &ticker,
		Title:    "Senator X bought $1M NVDA",
		Body:     "...",
		Payload:  map[string]any{"filer": "Senator X"},
	})
	require.NoError(t, err)
	require.Positive(t, id)

	var got struct {
		source, severity, title string
		ticker                  sql.NullString
		payload                 string
		seenTS                  sql.NullInt64
	}
	row := conn.QueryRowContext(ctx,
		`SELECT source, severity, ticker, title, payload, seen_ts FROM alerts WHERE id=?`, id,
	)
	require.NoError(t, row.Scan(&got.source, &got.severity, &got.ticker, &got.title, &got.payload, &got.seenTS))
	require.Equal(t, "insiders", got.source)
	require.Equal(t, "act", got.severity)
	require.Equal(t, "NVDA", got.ticker.String)
	require.False(t, got.seenTS.Valid)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(got.payload), &payload))
	require.Equal(t, "Senator X", payload["filer"])
}

func TestInsert_rejectsUnknownSeverity(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)
	_, err := alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "critical", Title: "t"})
	require.Error(t, err)
}
```

- [ ] **Step 2: Run — confirm failure**

```bash
go test ./internal/alert/...
```

Expected: FAIL (undefined).

- [ ] **Step 3: Implement `internal/alert/alert.go`**

```go
// Package alert writes rows to the alerts table. Consumers pass a typed Alert;
// payload is serialized to JSON.
package alert

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Severity = string

const (
	Info  Severity = "info"
	Watch Severity = "watch"
	Act   Severity = "act"
)

type Alert struct {
	Source   string
	Severity Severity
	Ticker   *string // nullable
	Title    string
	Body     string
	Payload  map[string]any
}

func Insert(ctx context.Context, conn *sql.DB, a Alert) (int64, error) {
	switch a.Severity {
	case Info, Watch, Act:
	default:
		return 0, fmt.Errorf("alert: invalid severity %q", a.Severity)
	}
	if a.Source == "" || a.Title == "" {
		return 0, fmt.Errorf("alert: source and title required")
	}

	var payload sql.NullString
	if a.Payload != nil {
		b, err := json.Marshal(a.Payload)
		if err != nil {
			return 0, fmt.Errorf("marshal payload: %w", err)
		}
		payload = sql.NullString{String: string(b), Valid: true}
	}
	var ticker sql.NullString
	if a.Ticker != nil {
		ticker = sql.NullString{String: *a.Ticker, Valid: true}
	}

	res, err := conn.ExecContext(ctx,
		`INSERT INTO alerts (created_ts, source, severity, ticker, title, body, payload)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), a.Source, a.Severity, ticker, a.Title, a.Body, payload,
	)
	if err != nil {
		return 0, fmt.Errorf("insert alert: %w", err)
	}
	return res.LastInsertId()
}
```

- [ ] **Step 4: Run tests — confirm pass**

```bash
go test ./internal/alert/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/alert/
git commit -m "feat(alert): typed Alert insert with JSON payload"
```

---

## Phase 3 — Jobs binary

### Task 7: Cobra scaffolding, `migrate` and `doctor`

**Files:**
- Create: `jobs/main.go`, `jobs/cmd/root.go`, `jobs/cmd/migrate.go`, `jobs/cmd/doctor.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add cobra**

```bash
go get github.com/spf13/cobra@latest
```

- [ ] **Step 2: Write `jobs/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/nkim500/market-digest/jobs/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Write `jobs/cmd/root.go`**

```go
// Package cmd assembles the cobra command tree for the jobs binary.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jobs",
	Short: "market-digest jobs binary — fetch, compute, alert",
	Long: `market-digest jobs binary.
All subcommands read/write data/digest.db (resolved relative to DIGEST_HOME).
Exit codes: 0 ok, 1 error, 2 noop.`,
}

// Execute runs the root command. Called from main.
func Execute() error {
	return rootCmd.Execute()
}

// digestHome returns DIGEST_HOME or CWD.
func digestHome() string {
	if h := os.Getenv("DIGEST_HOME"); h != "" {
		return h
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "jobs: cannot resolve cwd:", err)
		os.Exit(1)
	}
	return cwd
}
```

- [ ] **Step 4: Write `jobs/cmd/migrate.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/db"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply pending SQL migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		home := digestHome()
		ctx := context.Background()
		conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
		if err != nil {
			return err
		}
		defer conn.Close()
		applied, err := db.Migrate(ctx, conn, filepath.Join(home, "migrations"))
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			fmt.Println("migrate: up to date")
			return nil
		}
		for _, f := range applied {
			fmt.Printf("migrate: applied %s\n", f)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
```

- [ ] **Step 5: Write `jobs/cmd/doctor.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/config"
	"github.com/nkim500/market-digest/internal/db"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check DB, config, and source reachability",
	RunE: func(cmd *cobra.Command, args []string) error {
		home := digestHome()
		ctx := context.Background()

		fmt.Printf("DIGEST_HOME: %s\n", home)

		// Config
		cfg, err := config.Load(home)
		if err != nil {
			fmt.Printf("config: FAIL — %v\n", err)
		} else {
			fmt.Printf("config: ok (user=%q, sources=%d)\n", cfg.Profile.User.DisplayName, len(cfg.Sources.Insiders))
		}

		// DB
		dbPath := filepath.Join(home, "data", "digest.db")
		conn, err := db.Open(ctx, dbPath)
		if err != nil {
			fmt.Printf("db: FAIL — %v\n", err)
		} else {
			defer conn.Close()
			var n int
			_ = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master").Scan(&n)
			fmt.Printf("db: ok (%s, %d schema objects)\n", dbPath, n)
		}

		// Sources (HEAD with short timeout)
		client := &http.Client{Timeout: 10 * time.Second}
		for name, src := range cfg.Sources.Insiders {
			if !src.Enabled {
				fmt.Printf("source %s: disabled\n", name)
				continue
			}
			req, _ := http.NewRequestWithContext(ctx, http.MethodHead, src.URL, nil)
			if src.UserAgent != "" {
				req.Header.Set("User-Agent", src.UserAgent)
			}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("source %s: FAIL — %v\n", name, err)
				continue
			}
			resp.Body.Close()
			fmt.Printf("source %s: %s (HTTP %d)\n", name, src.URL, resp.StatusCode)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
```

- [ ] **Step 6: Verify `jobs migrate` works end-to-end**

```bash
go build -o bin/jobs ./jobs
./bin/jobs migrate
```

Expected: prints `migrate: applied 0001_init.sql`, creates `data/digest.db`.

```bash
./bin/jobs migrate
```

Expected: prints `migrate: up to date`.

```bash
./bin/jobs doctor
```

Expected: prints `DIGEST_HOME`, `config: ok ...`, `db: ok ...`, each source line. Exit 0.

- [ ] **Step 7: Commit**

```bash
git add jobs/ go.mod go.sum
git commit -m "feat(jobs): cobra scaffolding with migrate and doctor"
```

---

### Task 8: Insiders migration + types + dedup hash

**Files:**
- Create: `migrations/0002_insiders.sql`, `internal/insiders/types.go`, `internal/insiders/hash.go`, `internal/insiders/hash_test.go`

- [ ] **Step 1: Write `migrations/0002_insiders.sql`**

```sql
CREATE TABLE IF NOT EXISTS insider_trades (
  id             INTEGER PRIMARY KEY,
  source         TEXT NOT NULL,
  filer          TEXT NOT NULL,
  role           TEXT,
  ticker         TEXT,
  asset_desc     TEXT,
  side           TEXT,
  amount_low     INTEGER,
  amount_high    INTEGER,
  transaction_ts INTEGER,
  filing_ts      INTEGER,
  raw_url        TEXT,
  hash           TEXT UNIQUE NOT NULL
);
CREATE INDEX IF NOT EXISTS insider_trades_ticker_ts ON insider_trades(ticker, transaction_ts DESC);
CREATE INDEX IF NOT EXISTS insider_trades_filer ON insider_trades(filer);
CREATE INDEX IF NOT EXISTS insider_trades_filing_ts ON insider_trades(filing_ts DESC);
```

- [ ] **Step 2: Write `internal/insiders/types.go`**

```go
// Package insiders holds the types and fetchers for politician/corporate
// insider trade data.
package insiders

// Trade is the normalized row written to insider_trades.
type Trade struct {
	Source        string // 'senate' | 'house' | 'sec-form4'
	Filer         string
	Role          string
	Ticker        string
	AssetDesc     string
	Side          string // 'buy' | 'sell' | 'exchange'
	AmountLow     int    // USD
	AmountHigh    int
	TransactionTS int64 // unix
	FilingTS      int64
	RawURL        string
	Hash          string
}
```

- [ ] **Step 3: Write the failing hash test**

Create `internal/insiders/hash_test.go`:

```go
package insiders_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/insiders"
)

func TestHash_stableForSameInputs(t *testing.T) {
	a := insiders.Trade{Source: "senate", Filer: "Smith, John", Ticker: "NVDA",
		TransactionTS: 1700000000, AmountLow: 15000, AmountHigh: 50000, Side: "buy"}
	b := a
	require.Equal(t, insiders.Hash(a), insiders.Hash(b))
}

func TestHash_differsOnSide(t *testing.T) {
	a := insiders.Trade{Source: "senate", Filer: "Smith, John", Ticker: "NVDA",
		TransactionTS: 1700000000, AmountLow: 15000, AmountHigh: 50000, Side: "buy"}
	b := a
	b.Side = "sell"
	require.NotEqual(t, insiders.Hash(a), insiders.Hash(b))
}

func TestHash_caseInsensitiveTicker(t *testing.T) {
	a := insiders.Trade{Source: "senate", Filer: "Smith, John", Ticker: "NVDA",
		TransactionTS: 1700000000, AmountLow: 15000, AmountHigh: 50000, Side: "buy"}
	b := a
	b.Ticker = "nvda"
	require.Equal(t, insiders.Hash(a), insiders.Hash(b))
}

func TestHash_filerWhitespaceIgnored(t *testing.T) {
	a := insiders.Trade{Source: "senate", Filer: "Smith, John", Ticker: "NVDA",
		TransactionTS: 1700000000, AmountLow: 15000, AmountHigh: 50000, Side: "buy"}
	b := a
	b.Filer = "  Smith,  John  "
	require.Equal(t, insiders.Hash(a), insiders.Hash(b))
}
```

- [ ] **Step 4: Run — confirm failure**

```bash
go test ./internal/insiders/...
```

Expected: FAIL (undefined `insiders.Hash`).

- [ ] **Step 5: Implement `internal/insiders/hash.go`**

```go
package insiders

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
)

// Hash is the dedup key. Inputs are normalized (trim, uppercase ticker,
// collapse internal whitespace in filer) so cosmetic differences in upstream
// data don't produce duplicates.
func Hash(t Trade) string {
	filer := strings.Join(strings.Fields(t.Filer), " ")
	h := sha1.Sum([]byte(fmt.Sprintf(
		"%s|%s|%s|%d|%d|%d|%s",
		t.Source,
		filer,
		strings.ToUpper(t.Ticker),
		t.TransactionTS,
		t.AmountLow,
		t.AmountHigh,
		strings.ToLower(t.Side),
	)))
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 6: Run tests — confirm pass**

```bash
go test ./internal/insiders/...
```

Expected: PASS.

- [ ] **Step 7: Apply the new migration and verify**

```bash
./bin/jobs migrate
```

Expected: `migrate: applied 0002_insiders.sql`.

- [ ] **Step 8: Commit**

```bash
git add migrations/0002_insiders.sql internal/insiders/
git commit -m "feat(insiders): trade type, dedup hash, 0002 migration"
```

---

### Task 9: Insiders fetcher — HTTP + parse + normalize + insert

**Files:**
- Create: `internal/insiders/fetch.go`, `internal/insiders/fetch_test.go`, `internal/insiders/store.go`, `internal/insiders/store_test.go`
- Create: `internal/insiders/testdata/senate_sample.json`, `internal/insiders/testdata/house_sample.json`

**Background:** Senate Stock Watcher JSON schema (representative):

```json
[
  {
    "transaction_date": "2024-11-12",
    "owner": "Self",
    "ticker": "NVDA",
    "asset_description": "NVIDIA Corp",
    "type": "Purchase",
    "amount": "$15,001 - $50,000",
    "senator": "Tommy Tuberville",
    "ptr_link": "https://...",
    "disclosure_date": "2024-11-20"
  }
]
```

House Stock Watcher JSON uses similar keys: `transaction_date`, `ticker`, `asset_description`, `amount`, `representative`, `ptr_link`, `disclosure_date`, `type` (e.g., "purchase", "sale_full", "exchange").

- [ ] **Step 1: Create testdata fixtures**

Write `internal/insiders/testdata/senate_sample.json`:

```json
[
  {
    "transaction_date": "2024-11-12",
    "owner": "Self",
    "ticker": "NVDA",
    "asset_description": "NVIDIA Corp",
    "type": "Purchase",
    "amount": "$15,001 - $50,000",
    "senator": "Tommy Tuberville",
    "ptr_link": "https://efdsearch.senate.gov/search/view/ptr/abc/",
    "disclosure_date": "2024-11-20"
  },
  {
    "transaction_date": "2024-11-13",
    "owner": "Joint",
    "ticker": "--",
    "asset_description": "Treasury Bond",
    "type": "Sale (Full)",
    "amount": "$250,001 - $500,000",
    "senator": "Jane Doe",
    "ptr_link": "https://efdsearch.senate.gov/search/view/ptr/xyz/",
    "disclosure_date": "2024-11-20"
  }
]
```

Write `internal/insiders/testdata/house_sample.json`:

```json
[
  {
    "transaction_date": "2024-11-10",
    "ticker": "AAPL",
    "asset_description": "Apple Inc",
    "type": "purchase",
    "amount": "$1,001 - $15,000",
    "representative": "Hon. Example Rep",
    "ptr_link": "https://disclosures-clerk.house.gov/...",
    "disclosure_date": "2024-11-20"
  }
]
```

- [ ] **Step 2: Write the failing fetch/parse test**

Create `internal/insiders/fetch_test.go`:

```go
package insiders_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/insiders"
)

func TestFetchSenate_parsesFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/senate_sample.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	client := insiders.NewClient(insiders.ClientOptions{
		Timeout: 5 * time.Second, MaxRetries: 1,
	})
	trades, err := client.FetchSenate(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Len(t, trades, 2)

	nvda := trades[0]
	require.Equal(t, "senate", nvda.Source)
	require.Equal(t, "Tommy Tuberville", nvda.Filer)
	require.Equal(t, "NVDA", nvda.Ticker)
	require.Equal(t, "buy", nvda.Side)
	require.Equal(t, 15001, nvda.AmountLow)
	require.Equal(t, 50000, nvda.AmountHigh)
	require.NotZero(t, nvda.TransactionTS)
	require.NotZero(t, nvda.FilingTS)
	require.NotEmpty(t, nvda.Hash)

	// Row 2: '--' ticker should be normalized to empty; side='sell'
	bond := trades[1]
	require.Equal(t, "", bond.Ticker)
	require.Equal(t, "sell", bond.Side)
	require.Equal(t, 250001, bond.AmountLow)
	require.Equal(t, 500000, bond.AmountHigh)
}

func TestFetchHouse_parsesFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/house_sample.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	client := insiders.NewClient(insiders.ClientOptions{Timeout: 5 * time.Second, MaxRetries: 1})
	trades, err := client.FetchHouse(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Len(t, trades, 1)
	require.Equal(t, "house", trades[0].Source)
	require.Equal(t, "AAPL", trades[0].Ticker)
	require.Equal(t, "buy", trades[0].Side)
}

func TestFetch_retriesOn5xx(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("[]"))
	}))
	defer srv.Close()
	client := insiders.NewClient(insiders.ClientOptions{
		Timeout: 5 * time.Second, MaxRetries: 3, BackoffMS: 1,
	})
	_, err := client.FetchSenate(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Equal(t, 3, attempts)
}
```

- [ ] **Step 3: Run — confirm failure**

```bash
go test ./internal/insiders/... -run TestFetch
```

Expected: FAIL (undefined).

- [ ] **Step 4: Implement `internal/insiders/fetch.go`**

```go
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

// --- helpers ---

func normalizeTicker(raw string) string {
	t := strings.ToUpper(strings.TrimSpace(raw))
	if t == "--" || t == "N/A" || t == "" {
		return ""
	}
	return t
}

// normalizeSide maps source-specific type strings to 'buy' | 'sell' | 'exchange'.
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

// parseAmountRange turns "$15,001 - $50,000" into (15001, 50000). Returns
// (0, 0) if unparseable.
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
```

- [ ] **Step 5: Write the failing store test**

Create `internal/insiders/store_test.go`:

```go
package insiders_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/internal/insiders"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)
	return conn
}

func TestStoreInserts_dedupsOnHash(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	trades := []insiders.Trade{
		{Source: "senate", Filer: "X", Ticker: "NVDA", Side: "buy",
			AmountLow: 1000, AmountHigh: 15000, TransactionTS: 1700000000},
		{Source: "senate", Filer: "X", Ticker: "NVDA", Side: "buy",
			AmountLow: 1000, AmountHigh: 15000, TransactionTS: 1700000000},
	}
	for i := range trades {
		trades[i].Hash = insiders.Hash(trades[i])
	}
	newIDs, err := insiders.StoreInserts(ctx, conn, trades)
	require.NoError(t, err)
	require.Len(t, newIDs, 1, "identical trades must dedup")

	// Second call returns 0 new.
	newIDs, err = insiders.StoreInserts(ctx, conn, trades)
	require.NoError(t, err)
	require.Empty(t, newIDs)
}
```

- [ ] **Step 6: Run — confirm failure**

```bash
go test ./internal/insiders/... -run TestStore
```

Expected: FAIL.

- [ ] **Step 7: Implement `internal/insiders/store.go`**

```go
package insiders

import (
	"context"
	"database/sql"
	"fmt"
)

// StoreInserts upserts trades with INSERT OR IGNORE on hash. Returns the row
// ids of actually-inserted rows so the caller can evaluate alert rules only
// against new data.
func StoreInserts(ctx context.Context, conn *sql.DB, trades []Trade) ([]int64, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO insider_trades
		  (source, filer, role, ticker, asset_desc, side,
		   amount_low, amount_high, transaction_ts, filing_ts, raw_url, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var ids []int64
	for _, t := range trades {
		res, err := stmt.ExecContext(ctx,
			t.Source, t.Filer, t.Role, nullStr(t.Ticker), t.AssetDesc, t.Side,
			t.AmountLow, t.AmountHigh, t.TransactionTS, t.FilingTS, t.RawURL, t.Hash,
		)
		if err != nil {
			return nil, fmt.Errorf("insert trade hash=%s: %w", t.Hash, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected == 1 {
			id, err := res.LastInsertId()
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
```

- [ ] **Step 8: Run tests — confirm pass**

```bash
go test ./internal/insiders/...
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/insiders/
git commit -m "feat(insiders): fetch, parse, normalize, store with dedup"
```

---

### Task 10: Alert rules engine

**Files:**
- Create: `internal/insiders/rules.go`, `internal/insiders/rules_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/insiders/rules_test.go`:

```go
package insiders_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/config"
	"github.com/nkim500/market-digest/internal/insiders"
)

func TestEvaluateRules_watchlistHit(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	// Seed watchlist.
	_, err := conn.ExecContext(ctx,
		`INSERT INTO watchlist (ticker, note, added_ts) VALUES (?, ?, ?)`,
		"NVDA", "", time.Now().Unix())
	require.NoError(t, err)

	trade := insiders.Trade{Source: "senate", Filer: "Sen X", Ticker: "NVDA",
		Side: "buy", AmountLow: 15001, AmountHigh: 50000,
		TransactionTS: time.Now().Unix(), FilingTS: time.Now().Unix()}
	trade.Hash = insiders.Hash(trade)
	_, err = insiders.StoreInserts(ctx, conn, []insiders.Trade{trade})
	require.NoError(t, err)

	cfg := config.Sources{
		AlertRules: config.AlertRules{
			WatchlistHit:   config.AlertRule{Severity: "watch"},
			AmountOver500k: config.AlertRule{Severity: "watch"},
			AmountOver1m:   config.AlertRule{Severity: "act"},
			Cluster3In7d:   config.AlertRule{Severity: "watch"},
		},
	}
	profile := config.Profile{}
	profile.Reporting.DollarThresholds.Watch = 500000
	profile.Reporting.DollarThresholds.Act = 1000000
	profile.Reporting.ClusterWindowDays = 7

	n, err := insiders.EvaluateRules(ctx, conn, []insiders.Trade{trade}, cfg, profile)
	require.NoError(t, err)
	require.Equal(t, 1, n, "expected 1 alert row for watchlist hit")

	var sev, title string
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT severity, title FROM alerts WHERE source='insiders' ORDER BY id DESC LIMIT 1`,
	).Scan(&sev, &title))
	require.Equal(t, "watch", sev)
	require.Contains(t, title, "NVDA")
}

func TestEvaluateRules_amountOver1mIsAct(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	trade := insiders.Trade{Source: "senate", Filer: "Sen X", Ticker: "ZZZZ",
		Side: "buy", AmountLow: 1000001, AmountHigh: 5000000,
		TransactionTS: time.Now().Unix()}
	trade.Hash = insiders.Hash(trade)
	_, err := insiders.StoreInserts(ctx, conn, []insiders.Trade{trade})
	require.NoError(t, err)

	cfg := config.Sources{
		AlertRules: config.AlertRules{
			WatchlistHit:   config.AlertRule{Severity: "watch"},
			AmountOver500k: config.AlertRule{Severity: "watch"},
			AmountOver1m:   config.AlertRule{Severity: "act"},
			Cluster3In7d:   config.AlertRule{Severity: "watch"},
		},
	}
	profile := config.Profile{}
	profile.Reporting.DollarThresholds.Watch = 500000
	profile.Reporting.DollarThresholds.Act = 1000000
	profile.Reporting.ClusterWindowDays = 7

	_, err = insiders.EvaluateRules(ctx, conn, []insiders.Trade{trade}, cfg, profile)
	require.NoError(t, err)

	var sev string
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT severity FROM alerts WHERE source='insiders' ORDER BY id DESC LIMIT 1`,
	).Scan(&sev))
	require.Equal(t, "act", sev)
}

func TestEvaluateRules_cluster3FilersOneTickerWithin7d(t *testing.T) {
	ctx := context.Background()
	conn := newDB(t)

	now := time.Now().Unix()
	var trades []insiders.Trade
	for i, f := range []string{"A", "B", "C"} {
		tr := insiders.Trade{Source: "senate", Filer: f, Ticker: "XYZ",
			Side: "buy", AmountLow: 1000, AmountHigh: 15000,
			TransactionTS: now - int64(i*86400)}
		tr.Hash = insiders.Hash(tr)
		trades = append(trades, tr)
	}
	_, err := insiders.StoreInserts(ctx, conn, trades)
	require.NoError(t, err)

	cfg := config.Sources{
		AlertRules: config.AlertRules{
			WatchlistHit:   config.AlertRule{Severity: "watch"},
			AmountOver500k: config.AlertRule{Severity: "watch"},
			AmountOver1m:   config.AlertRule{Severity: "act"},
			Cluster3In7d:   config.AlertRule{Severity: "watch"},
		},
	}
	profile := config.Profile{}
	profile.Reporting.DollarThresholds.Watch = 500000
	profile.Reporting.DollarThresholds.Act = 1000000
	profile.Reporting.ClusterWindowDays = 7

	_, err = insiders.EvaluateRules(ctx, conn, trades, cfg, profile)
	require.NoError(t, err)

	var count int
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM alerts WHERE title LIKE 'Cluster:%' AND ticker='XYZ'`,
	).Scan(&count))
	require.Equal(t, 1, count)

	// Running again the same UTC-day must not duplicate.
	_, err = insiders.EvaluateRules(ctx, conn, trades, cfg, profile)
	require.NoError(t, err)
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM alerts WHERE title LIKE 'Cluster:%' AND ticker='XYZ'`,
	).Scan(&count))
	require.Equal(t, 1, count, "cluster alert must dedup per UTC day")
}
```

- [ ] **Step 2: Run — confirm failure**

```bash
go test ./internal/insiders/... -run TestEvaluateRules
```

Expected: FAIL.

- [ ] **Step 3: Implement `internal/insiders/rules.go`**

```go
package insiders

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/config"
)

// EvaluateRules runs all alert rules against newTrades. Returns the number of
// alerts inserted. Rules:
//
//  1. Per-trade: watchlist hit, amount thresholds. One alert per trade at the
//     highest matching severity.
//  2. Per-run: cluster detection (>= 3 distinct filers on same ticker within
//     profile.Reporting.ClusterWindowDays days). One alert per ticker per UTC
//     day (dedup via title+DATE(created_ts)).
func EvaluateRules(
	ctx context.Context, conn *sql.DB, newTrades []Trade,
	cfg config.Sources, profile config.Profile,
) (int, error) {
	inserted := 0

	watchlist, err := loadWatchlist(ctx, conn)
	if err != nil {
		return 0, err
	}

	for _, t := range newTrades {
		sev := ""
		reasons := []string{}
		if t.Ticker != "" {
			if _, ok := watchlist[t.Ticker]; ok {
				sev = maxSev(sev, cfg.AlertRules.WatchlistHit.Severity)
				reasons = append(reasons, "on watchlist")
			}
		}
		if t.AmountHigh >= profile.Reporting.DollarThresholds.Act {
			sev = maxSev(sev, cfg.AlertRules.AmountOver1m.Severity)
			reasons = append(reasons, fmt.Sprintf("amount ≥ $%d", profile.Reporting.DollarThresholds.Act))
		} else if t.AmountHigh >= profile.Reporting.DollarThresholds.Watch {
			sev = maxSev(sev, cfg.AlertRules.AmountOver500k.Severity)
			reasons = append(reasons, fmt.Sprintf("amount ≥ $%d", profile.Reporting.DollarThresholds.Watch))
		}
		if sev == "" {
			continue
		}

		title := fmt.Sprintf("%s %s %s ($%d-$%d)",
			t.Filer, t.Side, nonEmpty(t.Ticker, t.AssetDesc), t.AmountLow, t.AmountHigh)
		body := fmt.Sprintf(
			"**Filer:** %s  \n**Ticker:** %s  \n**Side:** %s  \n**Amount:** $%d - $%d  \n**Reasons:** %s  \n**Filing:** %s",
			t.Filer, nonEmpty(t.Ticker, "(no ticker)"), t.Side, t.AmountLow, t.AmountHigh,
			strings.Join(reasons, ", "), t.RawURL,
		)
		ticker := t.Ticker
		a := alert.Alert{
			Source: "insiders", Severity: sev, Title: title, Body: body,
			Payload: map[string]any{
				"filer": t.Filer, "ticker": t.Ticker, "side": t.Side,
				"amount_low": t.AmountLow, "amount_high": t.AmountHigh,
				"transaction_ts": t.TransactionTS, "raw_url": t.RawURL,
			},
		}
		if ticker != "" {
			a.Ticker = &ticker
		}
		if _, err := alert.Insert(ctx, conn, a); err != nil {
			return inserted, err
		}
		inserted++
	}

	// Cluster rule — one query across the whole insider_trades table.
	windowDays := profile.Reporting.ClusterWindowDays
	if windowDays <= 0 {
		windowDays = 7
	}
	since := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour).Unix()

	clusterRows, err := conn.QueryContext(ctx, `
		SELECT ticker, COUNT(DISTINCT filer) AS filers
		FROM insider_trades
		WHERE ticker IS NOT NULL AND ticker != '' AND transaction_ts >= ?
		GROUP BY ticker
		HAVING filers >= 3
	`, since)
	if err != nil {
		return inserted, err
	}
	defer clusterRows.Close()

	for clusterRows.Next() {
		var ticker string
		var filers int
		if err := clusterRows.Scan(&ticker, &filers); err != nil {
			return inserted, err
		}
		title := fmt.Sprintf("Cluster: %d filers on %s in last %dd", filers, ticker, windowDays)

		// Dedup per UTC day.
		var exists int
		if err := conn.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM alerts
			WHERE source='insiders' AND title=?
			  AND DATE(created_ts,'unixepoch') = DATE('now')
		`, title).Scan(&exists); err != nil {
			return inserted, err
		}
		if exists > 0 {
			continue
		}
		tk := ticker
		if _, err := alert.Insert(ctx, conn, alert.Alert{
			Source: "insiders", Severity: cfg.AlertRules.Cluster3In7d.Severity,
			Ticker: &tk, Title: title,
			Body:    fmt.Sprintf("%d distinct filers transacted in %s within the last %d days. Worth a look.", filers, ticker, windowDays),
			Payload: map[string]any{"ticker": ticker, "filers": filers, "window_days": windowDays},
		}); err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, clusterRows.Err()
}

// maxSev returns whichever of (current, candidate) is more severe. Order: info < watch < act.
func maxSev(current, candidate string) string {
	rank := map[string]int{"": 0, "info": 1, "watch": 2, "act": 3}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}

func nonEmpty(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func loadWatchlist(ctx context.Context, conn *sql.DB) (map[string]struct{}, error) {
	rows, err := conn.QueryContext(ctx, "SELECT ticker FROM watchlist")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out[strings.ToUpper(t)] = struct{}{}
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: Run tests — confirm pass**

```bash
go test ./internal/insiders/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/insiders/rules.go internal/insiders/rules_test.go
git commit -m "feat(insiders): alert rules engine (watchlist, amount, cluster)"
```

---

### Task 11: Wire `jobs fetch-insiders` end-to-end

**Files:**
- Create: `jobs/cmd/fetch_insiders.go`, `jobs/cmd/fetch_insiders_test.go`

- [ ] **Step 1: Write the failing end-to-end test**

Create `jobs/cmd/fetch_insiders_test.go`:

```go
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
	// Host both fixtures.
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
	// Copy migrations
	migSrc := "../../migrations"
	entries, err := os.ReadDir(migSrc)
	require.NoError(t, err)
	for _, e := range entries {
		src, err := os.ReadFile(filepath.Join(migSrc, e.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(home, "migrations", e.Name()), src, 0o644))
	}
	// Profile
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "profile.yml"), []byte(`
user: {display_name: "Test", timezone: "UTC"}
reporting: {dollar_thresholds: {watch: 500000, act: 1000000}, cluster_window_days: 7}
`), 0o644))
	// Watchlist: include NVDA so the senate sample row produces a watchlist-hit alert.
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "watchlist.yml"), []byte(`
tickers:
  - {ticker: NVDA, note: "", added: 2026-01-01}
`), 0o644))
	// Sources point at our httptest server.
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

	// Second run is idempotent.
	_, rowsNew2, err := cmd.RunFetchInsiders(ctx, home)
	require.NoError(t, err)
	require.Equal(t, 0, rowsNew2)

	// There should be at least one 'watch' alert (NVDA watchlist hit).
	conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
	require.NoError(t, err)
	defer conn.Close()
	var watchCount int
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM alerts WHERE source='insiders' AND severity='watch' AND ticker='NVDA'`,
	).Scan(&watchCount))
	require.GreaterOrEqual(t, watchCount, 1)

	// Ensure config.Load works too (basic sanity).
	cfg, err := config.Load(home)
	require.NoError(t, err)
	require.True(t, cfg.Sources.Insiders["senate"].Enabled)
}
```

- [ ] **Step 2: Run — confirm failure**

```bash
go test ./jobs/cmd/... -run TestFetchInsidersRun
```

Expected: FAIL (`undefined: cmd.RunFetchInsiders`).

- [ ] **Step 3: Implement `jobs/cmd/fetch_insiders.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/config"
	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/internal/insiders"
	"github.com/nkim500/market-digest/internal/jobrun"
)

// RunFetchInsiders is exported so end-to-end tests can call it with a temp
// home without going through the cobra command.
func RunFetchInsiders(ctx context.Context, home string) (rowsIn, rowsNew int, err error) {
	cfg, err := config.Load(home)
	if err != nil {
		return 0, 0, err
	}
	conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
	if err != nil {
		return 0, 0, err
	}
	defer conn.Close()
	if _, err := db.Migrate(ctx, conn, filepath.Join(home, "migrations")); err != nil {
		return 0, 0, err
	}

	// Sync watchlist YAML -> DB (authored source wins; TUI edits write both).
	if err := syncWatchlist(ctx, conn, cfg.Watchlist); err != nil {
		return 0, 0, fmt.Errorf("sync watchlist: %w", err)
	}

	client := insiders.NewClient(insiders.ClientOptions{
		Timeout:    time.Duration(cfg.Sources.HTTP.TimeoutSeconds) * time.Second,
		MaxRetries: cfg.Sources.HTTP.MaxRetries,
		BackoffMS:  cfg.Sources.HTTP.BackoffMS,
	})

	var all []insiders.Trade
	if src, ok := cfg.Sources.Insiders["senate"]; ok && src.Enabled {
		trades, err := client.FetchSenate(ctx, src.URL)
		if err != nil {
			return 0, 0, fmt.Errorf("senate: %w", err)
		}
		all = append(all, trades...)
	}
	if src, ok := cfg.Sources.Insiders["house"]; ok && src.Enabled {
		trades, err := client.FetchHouse(ctx, src.URL)
		if err != nil {
			return 0, 0, fmt.Errorf("house: %w", err)
		}
		all = append(all, trades...)
	}
	rowsIn = len(all)

	newIDs, err := insiders.StoreInserts(ctx, conn, all)
	if err != nil {
		return rowsIn, 0, err
	}
	rowsNew = len(newIDs)

	// Evaluate rules only against newly inserted trades.
	newSet := map[int64]struct{}{}
	for _, id := range newIDs {
		newSet[id] = struct{}{}
	}
	var newTrades []insiders.Trade
	for _, t := range all {
		// cheap filter: find by hash
		var id int64
		if err := conn.QueryRowContext(ctx,
			`SELECT id FROM insider_trades WHERE hash=?`, t.Hash,
		).Scan(&id); err == nil {
			if _, ok := newSet[id]; ok {
				newTrades = append(newTrades, t)
			}
		}
	}
	if _, err := insiders.EvaluateRules(ctx, conn, newTrades, cfg.Sources, cfg.Profile); err != nil {
		return rowsIn, rowsNew, err
	}
	return rowsIn, rowsNew, nil
}

func syncWatchlist(ctx context.Context, conn *sql.DB, w config.Watchlist) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM watchlist"); err != nil {
		return err
	}
	for _, entry := range w.Tickers {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO watchlist (ticker, note, added_ts) VALUES (?, ?, ?)",
			entry.Ticker, entry.Note, time.Now().Unix(),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

var fetchInsidersCmd = &cobra.Command{
	Use:   "fetch-insiders",
	Short: "Fetch Senate + House political trades, dedup, write alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		home := digestHome()
		conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
		if err != nil {
			return err
		}
		defer conn.Close()
		if _, err := db.Migrate(ctx, conn, filepath.Join(home, "migrations")); err != nil {
			return err
		}
		return jobrun.Track(ctx, conn, "fetch-insiders", func(ctx context.Context) (int, int, error) {
			return RunFetchInsiders(ctx, home)
		})
	},
}

func init() {
	rootCmd.AddCommand(fetchInsidersCmd)
}
```

**Note:** `RunFetchInsiders` uses `*sql.DB` via `syncWatchlist`; add the missing `"database/sql"` import. The full list of imports is:

```go
import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/config"
	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/internal/insiders"
	"github.com/nkim500/market-digest/internal/jobrun"
)
```

- [ ] **Step 4: Run tests — confirm pass**

```bash
go test ./jobs/cmd/...
```

Expected: PASS.

- [ ] **Step 5: Smoke test against real sources**

```bash
go build -o bin/jobs ./jobs
cp config/profile.example.yml config/profile.yml
cp config/watchlist.example.yml config/watchlist.yml
./bin/jobs fetch-insiders
```

Expected: no error. Exit code 0. `data/digest.db` contains `insider_trades` rows (may be large — several thousand — that's fine) and a `job_runs` row with `status='ok'`.

Verify:

```bash
sqlite3 data/digest.db "SELECT job, status, rows_in, rows_new FROM job_runs ORDER BY id DESC LIMIT 1"
sqlite3 data/digest.db "SELECT COUNT(*) FROM insider_trades"
sqlite3 data/digest.db "SELECT severity, COUNT(*) FROM alerts WHERE source='insiders' GROUP BY severity"
```

- [ ] **Step 6: Commit**

```bash
git add jobs/cmd/fetch_insiders.go jobs/cmd/fetch_insiders_test.go
git commit -m "feat(jobs): fetch-insiders command wired end-to-end"
```

---

### Task 12: `list-alerts`, `mark-seen`, and stub commands

**Files:**
- Create: `jobs/cmd/list_alerts.go`, `jobs/cmd/list_alerts_test.go`, `jobs/cmd/mark_seen.go`, `jobs/cmd/stubs.go`

- [ ] **Step 1: Write the failing test for list-alerts**

Create `jobs/cmd/list_alerts_test.go`:

```go
package cmd_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/db"
	"github.com/nkim500/market-digest/jobs/cmd"
)

func TestListAlerts_filtersUnseenAndSince(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	ticker := "NVDA"
	_, err = alert.Insert(ctx, conn, alert.Alert{Source: "insiders", Severity: "watch", Ticker: &ticker, Title: "fresh unseen"})
	require.NoError(t, err)
	oldID, err := alert.Insert(ctx, conn, alert.Alert{Source: "insiders", Severity: "info", Title: "old"})
	require.NoError(t, err)

	// Age the "old" alert.
	_, err = conn.ExecContext(ctx, `UPDATE alerts SET created_ts=? WHERE id=?`,
		time.Now().Add(-30*24*time.Hour).Unix(), oldID)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = cmd.ListAlerts(ctx, conn, cmd.ListAlertsOptions{Unseen: true, Since: 7 * 24 * time.Hour}, &buf)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "fresh unseen")
	require.NotContains(t, buf.String(), "old")
}
```

- [ ] **Step 2: Run — confirm failure**

```bash
go test ./jobs/cmd/... -run TestListAlerts
```

Expected: FAIL.

- [ ] **Step 3: Implement `jobs/cmd/list_alerts.go`**

```go
package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/db"
)

type ListAlertsOptions struct {
	Unseen bool
	Since  time.Duration
	Source string
}

func ListAlerts(ctx context.Context, conn *sql.DB, opts ListAlertsOptions, w io.Writer) error {
	q := `SELECT id, created_ts, source, severity, COALESCE(ticker, ''), title FROM alerts WHERE 1=1`
	args := []any{}
	if opts.Unseen {
		q += " AND seen_ts IS NULL"
	}
	if opts.Since > 0 {
		q += " AND created_ts >= ?"
		args = append(args, time.Now().Add(-opts.Since).Unix())
	}
	if opts.Source != "" {
		q += " AND source = ?"
		args = append(args, opts.Source)
	}
	q += " ORDER BY created_ts DESC LIMIT 100"

	rows, err := conn.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id         int64
			createdTS  int64
			source     string
			severity   string
			ticker     string
			title      string
		)
		if err := rows.Scan(&id, &createdTS, &source, &severity, &ticker, &title); err != nil {
			return err
		}
		fmt.Fprintf(w, "#%d  %s  %-7s  %-8s  %-6s  %s\n",
			id, time.Unix(createdTS, 0).Format("2006-01-02 15:04"),
			source, severity, ticker, title)
	}
	return rows.Err()
}

var (
	listAlertsUnseen bool
	listAlertsSince  string
	listAlertsSource string
)

var listAlertsCmd = &cobra.Command{
	Use:   "list-alerts",
	Short: "Print recent alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		home := digestHome()
		conn, err := db.Open(ctx, filepath.Join(home, "data", "digest.db"))
		if err != nil {
			return err
		}
		defer conn.Close()
		opts := ListAlertsOptions{Unseen: listAlertsUnseen, Source: listAlertsSource}
		if listAlertsSince != "" {
			d, err := time.ParseDuration(listAlertsSince)
			if err != nil {
				return fmt.Errorf("--since: %w", err)
			}
			opts.Since = d
		}
		return ListAlerts(ctx, conn, opts, os.Stdout)
	},
}

func init() {
	listAlertsCmd.Flags().BoolVar(&listAlertsUnseen, "unseen", false, "only unseen alerts")
	listAlertsCmd.Flags().StringVar(&listAlertsSince, "since", "", "time window, e.g. 7d, 24h (Go duration)")
	listAlertsCmd.Flags().StringVar(&listAlertsSource, "source", "", "filter by source (insiders|momentum|...)")
	rootCmd.AddCommand(listAlertsCmd)
}
```

**Note:** Go's `time.ParseDuration` doesn't understand `7d`. For v1 keep it to unit-of-hours (`168h`, `24h`); document this. If desired, add a small pre-parser — out of scope for v1.

- [ ] **Step 4: Implement `jobs/cmd/mark_seen.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/nkim500/market-digest/internal/db"
)

var markSeenCmd = &cobra.Command{
	Use:   "mark-seen <alert-id>",
	Short: "Mark an alert as seen",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("parse id: %w", err)
		}
		ctx := context.Background()
		conn, err := db.Open(ctx, filepath.Join(digestHome(), "data", "digest.db"))
		if err != nil {
			return err
		}
		defer conn.Close()
		res, err := conn.ExecContext(ctx,
			`UPDATE alerts SET seen_ts=? WHERE id=? AND seen_ts IS NULL`,
			time.Now().Unix(), id,
		)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			fmt.Println("no change (already seen or unknown id)")
			return nil
		}
		fmt.Printf("alert %d marked seen\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(markSeenCmd)
}
```

- [ ] **Step 5: Implement stubs in `jobs/cmd/stubs.go`**

```go
package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var ErrNotImplemented = errors.New("not implemented — see modes/momentum.md, modes/sector.md for context")

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "fetch-prices",
		Short: "(stub) Fetch EOD prices for watchlist — planned for momentum mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "fetch-prices is not implemented yet. See modes/momentum.md.")
			return ErrNotImplemented
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "compute-momentum",
		Short: "(stub) Compute momentum signals over prices table",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "compute-momentum is not implemented yet.")
			return ErrNotImplemented
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "report-sector <sector>",
		Short: "(stub) Generate a sector snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "report-sector is not implemented yet.")
			return ErrNotImplemented
		},
	})
}
```

- [ ] **Step 6: Run all tests**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Smoke test**

```bash
go build -o bin/jobs ./jobs
./bin/jobs list-alerts --unseen --since 168h
./bin/jobs mark-seen 1   # if at least one alert exists
./bin/jobs --help
```

Expected: the help output lists all commands (migrate, doctor, fetch-insiders, fetch-prices, compute-momentum, report-sector, list-alerts, mark-seen).

- [ ] **Step 8: Commit**

```bash
git add jobs/cmd/
git commit -m "feat(jobs): list-alerts, mark-seen, and stubbed commands"
```

---

## Phase 4 — Dashboard TUI

### Task 13: `internal/data` reader package

**Files:**
- Create: `internal/data/data.go`, `internal/data/data_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/data/data_test.go`:

```go
package data_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/data"
	"github.com/nkim500/market-digest/internal/db"
)

func TestRecentAlerts_ordersUnseenFirst(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	oldSeenID, err := alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "info", Title: "old seen"})
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `UPDATE alerts SET seen_ts=? WHERE id=?`, time.Now().Unix(), oldSeenID)
	require.NoError(t, err)

	_, err = alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "watch", Title: "new unseen"})
	require.NoError(t, err)

	rows, err := data.RecentAlerts(ctx, conn, 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "new unseen", rows[0].Title, "unseen must come first")
	require.Equal(t, "old seen", rows[1].Title)
}

func TestLastJobRun_returnsNilWhenEmpty(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../migrations")
	require.NoError(t, err)

	run, err := data.LastJobRun(ctx, conn, "fetch-insiders")
	require.NoError(t, err)
	require.Nil(t, run)
}
```

- [ ] **Step 2: Run — confirm failure**

```bash
go test ./internal/data/...
```

Expected: FAIL.

- [ ] **Step 3: Implement `internal/data/data.go`**

```go
// Package data contains typed read helpers for the dashboard TUI.
// Writes still go through jobs (except watchlist add/remove and mark-seen).
package data

import (
	"context"
	"database/sql"
	"time"
)

type AlertRow struct {
	ID        int64
	CreatedTS int64
	Source    string
	Severity  string
	Ticker    string
	Title     string
	Body      string
	SeenTS    *int64
}

func (a AlertRow) Time() time.Time { return time.Unix(a.CreatedTS, 0) }
func (a AlertRow) Seen() bool      { return a.SeenTS != nil }

func RecentAlerts(ctx context.Context, conn *sql.DB, limit int) ([]AlertRow, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, created_ts, source, severity, COALESCE(ticker,''), title, COALESCE(body,''), seen_ts
		FROM alerts
		ORDER BY (seen_ts IS NULL) DESC, created_ts DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AlertRow
	for rows.Next() {
		var a AlertRow
		var seen sql.NullInt64
		if err := rows.Scan(&a.ID, &a.CreatedTS, &a.Source, &a.Severity, &a.Ticker, &a.Title, &a.Body, &seen); err != nil {
			return nil, err
		}
		if seen.Valid {
			s := seen.Int64
			a.SeenTS = &s
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type WatchlistRow struct {
	Ticker       string
	Note         string
	AddedTS      int64
	AlertCount   int
	TradesCount  int
}

func Watchlist(ctx context.Context, conn *sql.DB) ([]WatchlistRow, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT
		  w.ticker, COALESCE(w.note,''), w.added_ts,
		  (SELECT COUNT(*) FROM alerts a WHERE a.ticker=w.ticker AND a.created_ts >= strftime('%s','now','-30 days')),
		  (SELECT COUNT(*) FROM insider_trades t WHERE t.ticker=w.ticker AND t.transaction_ts >= strftime('%s','now','-30 days'))
		FROM watchlist w
		ORDER BY w.ticker
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WatchlistRow
	for rows.Next() {
		var r WatchlistRow
		if err := rows.Scan(&r.Ticker, &r.Note, &r.AddedTS, &r.AlertCount, &r.TradesCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type InsiderTradeRow struct {
	ID             int64
	Source, Filer  string
	Role           string
	Ticker         string
	Side           string
	AmountLow      int
	AmountHigh     int
	TransactionTS  int64
	FilingTS       int64
	RawURL         string
}

func RecentInsiderTrades(ctx context.Context, conn *sql.DB, ticker string, limit int) ([]InsiderTradeRow, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, source, filer, COALESCE(role,''), COALESCE(ticker,''), COALESCE(side,''),
		       COALESCE(amount_low,0), COALESCE(amount_high,0),
		       COALESCE(transaction_ts,0), COALESCE(filing_ts,0), COALESCE(raw_url,'')
		FROM insider_trades
		WHERE ticker = ?
		ORDER BY transaction_ts DESC
		LIMIT ?
	`, ticker, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InsiderTradeRow
	for rows.Next() {
		var r InsiderTradeRow
		if err := rows.Scan(&r.ID, &r.Source, &r.Filer, &r.Role, &r.Ticker, &r.Side,
			&r.AmountLow, &r.AmountHigh, &r.TransactionTS, &r.FilingTS, &r.RawURL); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type JobRunRow struct {
	ID                          int64
	Job                         string
	StartedTS, FinishedTS       int64
	Status                      string
	RowsIn, RowsNew             int
	Error                       string
}

func RecentJobRuns(ctx context.Context, conn *sql.DB, limit int) ([]JobRunRow, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT id, job, started_ts, COALESCE(finished_ts,0), status,
		       COALESCE(rows_in,0), COALESCE(rows_new,0), COALESCE(error,'')
		FROM job_runs
		ORDER BY id DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobRunRow
	for rows.Next() {
		var r JobRunRow
		if err := rows.Scan(&r.ID, &r.Job, &r.StartedTS, &r.FinishedTS, &r.Status,
			&r.RowsIn, &r.RowsNew, &r.Error); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LastJobRun returns the most recent run for a given job, or nil if none.
func LastJobRun(ctx context.Context, conn *sql.DB, job string) (*JobRunRow, error) {
	row := conn.QueryRowContext(ctx, `
		SELECT id, job, started_ts, COALESCE(finished_ts,0), status,
		       COALESCE(rows_in,0), COALESCE(rows_new,0), COALESCE(error,'')
		FROM job_runs WHERE job=? ORDER BY id DESC LIMIT 1
	`, job)
	var r JobRunRow
	if err := row.Scan(&r.ID, &r.Job, &r.StartedTS, &r.FinishedTS, &r.Status,
		&r.RowsIn, &r.RowsNew, &r.Error); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// MarkSeen flips seen_ts.
func MarkSeen(ctx context.Context, conn *sql.DB, alertID int64) error {
	_, err := conn.ExecContext(ctx,
		`UPDATE alerts SET seen_ts=? WHERE id=? AND seen_ts IS NULL`,
		time.Now().Unix(), alertID)
	return err
}
```

- [ ] **Step 4: Run tests — confirm pass**

```bash
go test ./internal/data/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/data/
git commit -m "feat(data): typed reads + mark-seen for TUI"
```

---

### Task 14: Dashboard root + Alerts screen

**Files:**
- Create: `dashboard/main.go`, `dashboard/internal/theme/theme.go`, `dashboard/internal/screens/alerts.go`, `dashboard/internal/screens/alerts_test.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add bubbletea + lipgloss**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 2: Write `dashboard/internal/theme/theme.go`**

```go
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Header    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	Footer    = lipgloss.NewStyle().Faint(true)
	Cursor    = lipgloss.NewStyle().Reverse(true)
	Seen      = lipgloss.NewStyle().Faint(true)
	SevInfo   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	SevWatch  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	SevAct    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

func SeverityStyle(sev string) lipgloss.Style {
	switch sev {
	case "act":
		return SevAct
	case "watch":
		return SevWatch
	default:
		return SevInfo
	}
}
```

- [ ] **Step 3: Write `dashboard/internal/screens/alerts.go`**

```go
// Package screens contains one bubbletea model per dashboard screen.
package screens

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nkim500/market-digest/dashboard/internal/theme"
	"github.com/nkim500/market-digest/internal/data"
)

type AlertsLoadedMsg struct {
	Rows []data.AlertRow
	Err  error
}

type AlertsModel struct {
	Conn   *sql.DB
	Rows   []data.AlertRow
	Cursor int
	Width  int
	Height int
	Error  string
}

func NewAlertsModel(conn *sql.DB) AlertsModel {
	return AlertsModel{Conn: conn}
}

func (m AlertsModel) Init() tea.Cmd { return m.loadCmd() }

func (m AlertsModel) loadCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := data.RecentAlerts(context.Background(), m.Conn, 100)
		return AlertsLoadedMsg{Rows: rows, Err: err}
	}
}

func (m AlertsModel) Update(msg tea.Msg) (AlertsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case AlertsLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
			return m, nil
		}
		m.Rows = msg.Rows
		if m.Cursor >= len(m.Rows) {
			m.Cursor = 0
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Rows)-1 {
				m.Cursor++
			}
		case "x":
			if m.Cursor < len(m.Rows) {
				id := m.Rows[m.Cursor].ID
				return m, func() tea.Msg {
					if err := data.MarkSeen(context.Background(), m.Conn, id); err != nil {
						return AlertsLoadedMsg{Err: err}
					}
					rows, err := data.RecentAlerts(context.Background(), m.Conn, 100)
					return AlertsLoadedMsg{Rows: rows, Err: err}
				}
			}
		case "r":
			return m, m.loadCmd()
		}
	}
	return m, nil
}

func (m AlertsModel) View() string {
	var b strings.Builder
	b.WriteString(theme.Header.Render("Alerts") + "\n\n")
	if m.Error != "" {
		b.WriteString("ERROR: " + m.Error + "\n")
		return b.String()
	}
	if len(m.Rows) == 0 {
		b.WriteString("  (no alerts)\n")
		return b.String()
	}
	for i, r := range m.Rows {
		line := fmt.Sprintf("  %-16s  %-8s  %-8s  %-6s  %s",
			time.Unix(r.CreatedTS, 0).Format("2006-01-02 15:04"),
			r.Source, r.Severity, r.Ticker, r.Title)
		line = theme.SeverityStyle(r.Severity).Render(line)
		if r.Seen() {
			line = theme.Seen.Render(line)
		}
		if i == m.Cursor {
			line = theme.Cursor.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + theme.Footer.Render("↑/↓ move · x mark seen · r reload · 1-4 screens · q quit") + "\n")
	return b.String()
}

// SelectedBody returns the markdown body of the currently-selected row, if any.
func (m AlertsModel) SelectedBody() string {
	if m.Cursor >= len(m.Rows) {
		return ""
	}
	return m.Rows[m.Cursor].Body
}
```

- [ ] **Step 4: Write a simple model test**

Create `dashboard/internal/screens/alerts_test.go`:

```go
package screens_test

import (
	"context"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/nkim500/market-digest/dashboard/internal/screens"
	"github.com/nkim500/market-digest/internal/alert"
	"github.com/nkim500/market-digest/internal/db"
)

func TestAlertsModel_markSeenDecrementsUnseen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(ctx, path)
	require.NoError(t, err)
	defer conn.Close()
	_, err = db.Migrate(ctx, conn, "../../../migrations")
	require.NoError(t, err)

	_, err = alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "watch", Title: "one"})
	require.NoError(t, err)
	_, err = alert.Insert(ctx, conn, alert.Alert{Source: "x", Severity: "info", Title: "two"})
	require.NoError(t, err)

	m := screens.NewAlertsModel(conn)
	// Initial load (simulate)
	cmd := m.Init()
	msg := cmd()
	m, _ = m.Update(msg)
	require.Len(t, m.Rows, 2)

	// 'x' on cursor 0 marks it seen and reloads.
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd)
	msg = cmd()
	m, _ = m.Update(msg)

	seenCount := 0
	for _, r := range m.Rows {
		if r.Seen() {
			seenCount++
		}
	}
	require.Equal(t, 1, seenCount)
}
```

- [ ] **Step 5: Write `dashboard/main.go` (root model + screen switching — minimum for v1)**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nkim500/market-digest/dashboard/internal/screens"
	"github.com/nkim500/market-digest/dashboard/internal/theme"
	"github.com/nkim500/market-digest/internal/data"
	"github.com/nkim500/market-digest/internal/db"
)

type screen int

const (
	screenAlerts screen = iota + 1
	screenWatchlist
	screenTicker
	screenJobs
)

type root struct {
	conn     *sql.DB
	current  screen
	width    int
	height   int
	alerts   screens.AlertsModel
	watch    screens.WatchlistModel
	ticker   screens.TickerModel
	jobs     screens.JobsModel
	dbPath   string
	lastRun  *data.JobRunRow
}

func (r root) Init() tea.Cmd {
	return tea.Batch(r.alerts.Init(), r.watch.Init(), r.jobs.Init(), refreshLastRunCmd(r.conn))
}

type lastRunMsg struct{ row *data.JobRunRow }

func refreshLastRunCmd(conn *sql.DB) tea.Cmd {
	return func() tea.Msg {
		row, _ := data.LastJobRun(context.Background(), conn, "fetch-insiders")
		return lastRunMsg{row}
	}
}

func (r root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width, r.height = msg.Width, msg.Height
	case lastRunMsg:
		r.lastRun = msg.row
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return r, tea.Quit
		case "1":
			r.current = screenAlerts
		case "2":
			r.current = screenWatchlist
		case "3":
			r.current = screenTicker
		case "4":
			r.current = screenJobs
		}
	}
	// Delegate to the active screen.
	switch r.current {
	case screenAlerts:
		m, cmd := r.alerts.Update(msg)
		r.alerts = m
		cmds = append(cmds, cmd)
	case screenWatchlist:
		m, cmd := r.watch.Update(msg)
		r.watch = m
		cmds = append(cmds, cmd)
	case screenTicker:
		m, cmd := r.ticker.Update(msg)
		r.ticker = m
		cmds = append(cmds, cmd)
	case screenJobs:
		m, cmd := r.jobs.Update(msg)
		r.jobs = m
		cmds = append(cmds, cmd)
	}
	return r, tea.Batch(cmds...)
}

func (r root) View() string {
	var body string
	switch r.current {
	case screenAlerts:
		body = r.alerts.View()
	case screenWatchlist:
		body = r.watch.View()
	case screenTicker:
		body = r.ticker.View()
	case screenJobs:
		body = r.jobs.View()
	}
	footerTxt := fmt.Sprintf("digest.db @ %s  ·  last fetch-insiders: %s", r.dbPath, formatLastRun(r.lastRun))
	return body + "\n" + lipgloss.PlaceHorizontal(r.width, lipgloss.Left, theme.Footer.Render(footerTxt))
}

func formatLastRun(r *data.JobRunRow) string {
	if r == nil {
		return "never"
	}
	ago := time.Since(time.Unix(r.StartedTS, 0)).Truncate(time.Minute)
	return fmt.Sprintf("%s ago (%s)", ago, r.Status)
}

func digestHome() string {
	if h := os.Getenv("DIGEST_HOME"); h != "" {
		return h
	}
	cwd, _ := os.Getwd()
	return cwd
}

func main() {
	home := digestHome()
	dbPath := filepath.Join(home, "data", "digest.db")
	ctx := context.Background()
	conn, err := db.Open(ctx, dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dashboard:", err)
		os.Exit(1)
	}
	defer conn.Close()

	r := root{
		conn:    conn,
		current: screenAlerts,
		dbPath:  dbPath,
		alerts:  screens.NewAlertsModel(conn),
		watch:   screens.NewWatchlistModel(conn),
		ticker:  screens.NewTickerModel(conn),
		jobs:    screens.NewJobsModel(conn),
	}
	if _, err := tea.NewProgram(r, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dashboard:", err)
		os.Exit(1)
	}
}
```

**Note:** `dashboard/main.go` references `screens.NewWatchlistModel`, `NewTickerModel`, `NewJobsModel` — those are created in Task 15. Keep this file compiling-broken for now; don't build `dashboard` until Task 15 is complete. The alerts screen test is self-contained.

- [ ] **Step 6: Run the alerts screen test**

```bash
go test ./dashboard/internal/screens/... -run TestAlertsModel
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add dashboard/ go.mod go.sum
git commit -m "feat(dashboard): alerts screen + root scaffolding (watchlist/ticker/jobs stubs to come)"
```

---

### Task 15: Watchlist, Ticker detail, and Jobs screens

**Files:**
- Create: `dashboard/internal/screens/watchlist.go`, `dashboard/internal/screens/ticker.go`, `dashboard/internal/screens/jobs.go`

- [ ] **Step 1: Implement `dashboard/internal/screens/watchlist.go`**

```go
package screens

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nkim500/market-digest/dashboard/internal/theme"
	"github.com/nkim500/market-digest/internal/data"
)

type WatchlistModel struct {
	Conn   *sql.DB
	Rows   []data.WatchlistRow
	Cursor int
	Error  string
}

type watchlistLoadedMsg struct {
	Rows []data.WatchlistRow
	Err  error
}

func NewWatchlistModel(conn *sql.DB) WatchlistModel {
	return WatchlistModel{Conn: conn}
}

func (m WatchlistModel) Init() tea.Cmd {
	return func() tea.Msg {
		rows, err := data.Watchlist(context.Background(), m.Conn)
		return watchlistLoadedMsg{rows, err}
	}
}

func (m WatchlistModel) Update(msg tea.Msg) (WatchlistModel, tea.Cmd) {
	switch msg := msg.(type) {
	case watchlistLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
		} else {
			m.Rows = msg.Rows
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Rows)-1 {
				m.Cursor++
			}
		case "r":
			return m, m.Init()
		}
	}
	return m, nil
}

func (m WatchlistModel) View() string {
	var b strings.Builder
	b.WriteString(theme.Header.Render("Watchlist") + "\n\n")
	if m.Error != "" {
		return b.String() + "ERROR: " + m.Error + "\n"
	}
	if len(m.Rows) == 0 {
		b.WriteString("  (empty — edit config/watchlist.yml and run `jobs migrate`)\n")
		return b.String()
	}
	for i, r := range m.Rows {
		line := fmt.Sprintf("  %-6s  alerts(30d):%-3d  trades(30d):%-3d  added:%s  %s",
			r.Ticker, r.AlertCount, r.TradesCount,
			time.Unix(r.AddedTS, 0).Format("2006-01-02"), r.Note)
		if i == m.Cursor {
			line = theme.Cursor.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + theme.Footer.Render("r reload · enter → ticker detail (not yet wired) · 1-4 screens · q quit") + "\n")
	return b.String()
}

func (m WatchlistModel) SelectedTicker() string {
	if m.Cursor >= len(m.Rows) {
		return ""
	}
	return m.Rows[m.Cursor].Ticker
}
```

- [ ] **Step 2: Implement `dashboard/internal/screens/ticker.go`**

```go
package screens

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nkim500/market-digest/dashboard/internal/theme"
	"github.com/nkim500/market-digest/internal/data"
)

type TickerModel struct {
	Conn   *sql.DB
	Ticker string
	Rows   []data.InsiderTradeRow
	Error  string
}

type tickerLoadedMsg struct {
	Rows []data.InsiderTradeRow
	Err  error
}

func NewTickerModel(conn *sql.DB) TickerModel {
	return TickerModel{Conn: conn}
}

// SetTicker lets the root model pass a ticker in when switching screens.
func (m *TickerModel) SetTicker(t string) { m.Ticker = t }

func (m TickerModel) Init() tea.Cmd {
	if m.Ticker == "" {
		return nil
	}
	ticker := m.Ticker
	conn := m.Conn
	return func() tea.Msg {
		rows, err := data.RecentInsiderTrades(context.Background(), conn, ticker, 50)
		return tickerLoadedMsg{rows, err}
	}
}

func (m TickerModel) Update(msg tea.Msg) (TickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickerLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
		} else {
			m.Rows = msg.Rows
		}
	}
	return m, nil
}

func (m TickerModel) View() string {
	var b strings.Builder
	title := "Ticker detail"
	if m.Ticker != "" {
		title = "Ticker detail — " + m.Ticker
	}
	b.WriteString(theme.Header.Render(title) + "\n\n")
	if m.Ticker == "" {
		b.WriteString("  (no ticker selected — pick one from Watchlist screen)\n")
		return b.String()
	}
	if m.Error != "" {
		return b.String() + "ERROR: " + m.Error + "\n"
	}
	b.WriteString("  Price pane: momentum mode not yet implemented.\n\n")
	b.WriteString("  Recent insider trades:\n")
	if len(m.Rows) == 0 {
		b.WriteString("    (none)\n")
	}
	for _, r := range m.Rows {
		b.WriteString(fmt.Sprintf("    %s  %-20s  %-7s  $%d-$%d\n",
			time.Unix(r.TransactionTS, 0).Format("2006-01-02"),
			r.Filer, r.Side, r.AmountLow, r.AmountHigh,
		))
	}
	return b.String()
}
```

- [ ] **Step 3: Implement `dashboard/internal/screens/jobs.go`**

```go
package screens

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nkim500/market-digest/dashboard/internal/theme"
	"github.com/nkim500/market-digest/internal/data"
)

type JobsModel struct {
	Conn  *sql.DB
	Rows  []data.JobRunRow
	Error string
}

type jobsLoadedMsg struct {
	Rows []data.JobRunRow
	Err  error
}

func NewJobsModel(conn *sql.DB) JobsModel {
	return JobsModel{Conn: conn}
}

func (m JobsModel) Init() tea.Cmd {
	return func() tea.Msg {
		rows, err := data.RecentJobRuns(context.Background(), m.Conn, 20)
		return jobsLoadedMsg{rows, err}
	}
}

func (m JobsModel) Update(msg tea.Msg) (JobsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case jobsLoadedMsg:
		if msg.Err != nil {
			m.Error = msg.Err.Error()
		} else {
			m.Rows = msg.Rows
		}
	case tea.KeyMsg:
		if msg.String() == "r" {
			return m, m.Init()
		}
	}
	return m, nil
}

func (m JobsModel) View() string {
	var b strings.Builder
	b.WriteString(theme.Header.Render("Jobs") + "\n\n")
	if m.Error != "" {
		return b.String() + "ERROR: " + m.Error + "\n"
	}
	for _, r := range m.Rows {
		dur := "(running)"
		if r.FinishedTS > 0 {
			dur = time.Unix(r.FinishedTS, 0).Sub(time.Unix(r.StartedTS, 0)).Truncate(time.Second).String()
		}
		errTxt := ""
		if r.Error != "" {
			errTxt = " ERR=" + truncate(r.Error, 60)
		}
		line := fmt.Sprintf("  %s  %-18s  %-6s  %5s  in=%-5d new=%-5d%s",
			time.Unix(r.StartedTS, 0).Format("2006-01-02 15:04"),
			r.Job, r.Status, dur, r.RowsIn, r.RowsNew, errTxt)
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + theme.Footer.Render("r reload · 1-4 screens · q quit") + "\n")
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
```

- [ ] **Step 4: Build dashboard**

```bash
go build -o bin/dashboard ./dashboard
```

Expected: success, no compile errors.

- [ ] **Step 5: Smoke-run the dashboard**

```bash
./bin/dashboard
```

Expected: opens alt screen, shows "Alerts" header, `1-4` switches screens, `q` exits. At least one `job_runs` row should be visible on screen 4 (from Task 11's smoke test). Press `q` to exit cleanly.

- [ ] **Step 6: Commit**

```bash
git add dashboard/internal/screens/
git commit -m "feat(dashboard): watchlist, ticker-detail, and jobs screens"
```

---

### Task 16: Cross-screen navigation (watchlist → ticker) and polish

**Files:**
- Modify: `dashboard/main.go`, `dashboard/internal/screens/watchlist.go`

- [ ] **Step 1: Wire `enter` on Watchlist to set ticker and switch screens**

Modify `dashboard/main.go`'s key handler to translate `enter` when `current == screenWatchlist`:

Replace the `Update` method's `tea.KeyMsg` case with:

```go
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return r, tea.Quit
		case "1":
			r.current = screenAlerts
		case "2":
			r.current = screenWatchlist
		case "3":
			r.current = screenTicker
			return r, r.ticker.Init()
		case "4":
			r.current = screenJobs
		case "enter":
			if r.current == screenWatchlist {
				if t := r.watch.SelectedTicker(); t != "" {
					r.ticker.SetTicker(t)
					r.current = screenTicker
					return r, r.ticker.Init()
				}
			}
		}
```

- [ ] **Step 2: Rebuild and smoke-test**

```bash
go build -o bin/dashboard ./dashboard
./bin/dashboard
```

Manual test:
1. Press `2` — see Watchlist screen with NVDA/CEG (from example YAML).
2. Press `enter` — switches to Ticker detail for NVDA, shows recent trades.
3. Press `1` — back to Alerts.
4. Press `q` — exits.

- [ ] **Step 3: Commit**

```bash
git add dashboard/main.go
git commit -m "feat(dashboard): watchlist → ticker-detail navigation"
```

---

## Phase 5 — Modes + Claude skill

### Task 17: `CLAUDE.md`, `FORK_NOTES.md`, `CLAUDE.local.md.template`

**Files:**
- Create: `CLAUDE.md`, `FORK_NOTES.md`, `CLAUDE.local.md.template`

- [ ] **Step 1: Write `CLAUDE.md`**

```markdown
# System Context — market-digest

<!-- This file is tracked. It contains shared rules that apply to every
     Claude Code session in this repo. User-specific overrides go in
     CLAUDE.local.md (gitignored, see CLAUDE.local.md.template). -->

## Purpose

market-digest is a personal, fork-friendly ideation tool for US equities. It pulls free public data, stores it in SQLite, and lets Claude synthesize the parts that benefit from reasoning. It is **not** a trading bot and does **not** place orders.

## Sources of Truth

| What | Where | When to read |
|------|-------|--------------|
| User profile | `config/profile.yml` (falls back to `profile.example.yml`) | Every mode |
| Watchlist | `config/watchlist.yml` | Every mode that cares about specific tickers |
| Data sources & rules | `config/sources.yml` | Every mode that fetches data |
| User narrative + risk posture | `modes/_profile.md` (gitignored) | Every mode — user customizations override defaults |
| Shared rules | `modes/_shared.md` | Every mode — loaded FIRST |

## Mode Invocation

A "mode" is a markdown file in `modes/` that describes a workflow. Invoke via `/digest <mode>` (wired through `.claude/skills/market-digest/SKILL.md`).

- **Run modes as subagents** (`Agent(subagent_type="general-purpose", ...)`) to preserve main-session context.
- **Always load `modes/_shared.md` first**, then `modes/<mode>.md`, then `_profile.md`.
- **Jobs do fetching, Claude does synthesis.** If a mode needs data, it runs the relevant `jobs <subcommand>` rather than WebFetching URLs.

## Alert Severities

- `info` — recorded, not surfaced loudly.
- `watch` — worth looking at today/this week.
- `act` — flagged for immediate review. Never a directive to place a trade.

Thresholds are defined in `config/profile.yml:reporting.dollar_thresholds`.

## Ethical Framing

- Modes report *observations*, not *advice*.
- Never frame output as "you should buy/sell X." Use "things to look at" framing.
- All sources are public (Senate/House disclosures, SEC filings, etc.). Never suggest shortcuts that rely on non-public information.
- Every generated report includes a disclaimer at the bottom.

## Agent Plan-Board Protocol

When executing a written plan in `implementation_plan.md` (gitignored):

- **Before starting a phase:** set its header to `[IN PROGRESS — <session id> — <ISO timestamp>]`.
- **After finishing:** set to `[DONE — <PR URL or "local">]` plus a one-line note of what actually landed (spec drift is expected).
- **When opening a PR for a phase:** link the PR description back to the phase heading; update the line with the PR URL.
- **If blocked:** `[BLOCKED — <reason>]` plus a paragraph. Never silently revert to PENDING.

This prevents parallel agents (subagents, other Claude Code sessions) from double-claiming the same work, and gives a human operator a single file to `cat` to see status.

## Fork Notes

See `FORK_NOTES.md` for how this fork diverges from any upstream (empty at repo creation — filled as divergence happens).

## Legal

See `README.md` for the full disclaimer. Summary: educational/ideational tool, not investment advice, public data only.
```

- [ ] **Step 2: Write `FORK_NOTES.md`**

```markdown
# Fork Notes

This file is the authoritative record of how this repo diverges from any upstream it tracks. Read before every upstream sync.

## Invariants

<!-- Rules that override upstream when they conflict. Add entries as the fork diverges. -->

_None yet — this is the initial commit._

## Local Features Log

Features and substantive changes local to this fork. Append newest-first.

| Date | Merge SHA | Feature | Notes |
|------|-----------|---------|-------|
| 2026-04-23 | _initial_ | Initial skeleton | Per `docs/superpowers/specs/2026-04-23-market-digest-skeleton-design.md`. |

## Upstream Sync Decision Log

One entry per sync. Records accepted, skipped, conflicted, and how resolved.

<!-- next sync entry goes here -->

_No syncs yet._
```

- [ ] **Step 3: Write `CLAUDE.local.md.template`**

```markdown
<!-- Copy this file to CLAUDE.local.md (gitignored) for per-machine overrides.
     Typical uses:
       - A personal API key or User-Agent for a data source
       - Notes only relevant to your machine setup
       - Pointers to external reference material you use locally -->

# Local Overrides

## Paths
DIGEST_HOME: /path/to/your/checkout

## Data source credentials
<!-- Do NOT commit real values. Keep them here only if you're sure this file
     won't leak. Prefer env vars in a shell rc file where possible. -->
```

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md FORK_NOTES.md CLAUDE.local.md.template
git commit -m "docs: CLAUDE.md, FORK_NOTES.md, local override template"
```

---

### Task 18: Mode files (`_shared`, wired and stub modes)

**Files:**
- Create: `modes/_shared.md`, `modes/_profile.template.md`, `modes/insiders.md`, `modes/alerts.md`, `modes/momentum.md`, `modes/sector.md`, `modes/macro.md`

- [ ] **Step 1: Write `modes/_shared.md`**

```markdown
# System Context — modes/_shared.md

Loaded first by every mode. Auto-updatable; user-specific customizations belong in `modes/_profile.md`.

## Read in order

1. `CLAUDE.md` (repo root)
2. This file
3. `modes/_profile.md` (user overrides — if present)
4. The specific mode file you were invoked as

## Data access

- **DB:** `data/digest.db` (SQLite). Run queries directly via `sqlite3` when you need them; don't shell out from inside the DB.
- **Config:** `config/profile.yml` (or `profile.example.yml` fallback), `config/watchlist.yml`, `config/sources.yml`.
- **Fetching:** run the relevant `./bin/jobs <subcommand>` rather than WebFetching URLs yourself. See `jobs --help` for the full command list.
- **Job freshness:** check `job_runs` for a today's row before re-running — `SELECT job, status, DATE(started_ts,'unixepoch') AS d FROM job_runs WHERE job='fetch-insiders' ORDER BY id DESC LIMIT 1`.

## Output conventions

- Reports → `data/reports/<mode>-YYYY-MM-DD.md`.
- New alerts → insert into `alerts` only when something crosses a threshold that isn't already covered by the job's built-in rules.
- Always end a generated report with the disclaimer line from `CLAUDE.md`'s Legal section.

## Severity reminders

- `info` — context only; don't lead with it.
- `watch` — investigate this week.
- `act` — investigate today. Never equivalent to "place a trade."
```

- [ ] **Step 2: Write `modes/_profile.template.md`**

```markdown
<!-- Copy to modes/_profile.md and fill in. This file is gitignored so your
     personal context stays local. -->

# My Trading Context

## Who I am
- Role/capacity: (e.g., "personal investor, ~20 hrs/week, separate day-job")
- Time horizon: (e.g., "swing: weeks to months; occasional multi-year positions")

## How I decide
- Entry criteria: (narrative rules you use)
- Exit criteria: (when do you cut losers / take profits?)
- Avoid: (themes or tickers you won't touch and why)

## What modes should emphasize
- Sectors I care about: (e.g., "semis, grid, biotech — deprioritize consumer discretionary")
- Context I usually want: (e.g., "always include sector-level analog, always check macro tilt")

## Notes to Claude
- (Any standing instructions — e.g., "always flag if a trade would push me over 5% position size")
```

- [ ] **Step 3: Write `modes/insiders.md` (WIRED)**

```markdown
# Mode: insiders — Politician & Corporate Insider Trade Digest

Surfaces meaningful recent insider activity on the watchlist and across the filing universe.

## Recommended execution

Run as a subagent (`subagent_type="general-purpose"`) to keep main-session context clean.

## Inputs
- CLI args: none by default. Optional: `--since 14d` (default 7 days).
- Reads: `data/digest.db` (`insider_trades`, `alerts`, `watchlist`), `config/profile.yml`, `config/watchlist.yml`.
- Depends on: `jobs fetch-insiders` having run today.

## Workflow

1. **Freshness check.** Query `SELECT status, DATE(started_ts,'unixepoch') AS d FROM job_runs WHERE job='fetch-insiders' ORDER BY id DESC LIMIT 1`. If no row for today, run `./bin/jobs fetch-insiders` and capture its output.

2. **Load context.** Read `config/profile.yml` (thresholds, interests) and `config/watchlist.yml`.

3. **Queries.**
   - Watchlist hits: `SELECT * FROM insider_trades WHERE ticker IN (<watchlist>) AND transaction_ts >= strftime('%s','now','-14 days') ORDER BY transaction_ts DESC`.
   - High-severity alerts: `SELECT * FROM alerts WHERE source='insiders' AND severity IN ('watch','act') AND created_ts >= strftime('%s','now','-14 days')`.
   - Cluster candidates: tickers with ≥3 distinct filers in the last 7 days.

4. **Group + synthesize.** Group trades by ticker, by filer, by sector (use your own domain knowledge — no SIC lookup in v1). Look for:
   - Same-direction clustering (multiple buys or sells on the same ticker)
   - Timing anomalies (insider sold days before a guidance cut, etc. — note, don't assert)
   - Novel names that aren't on the watchlist

5. **Produce report.** Markdown with these sections:
   - **TL;DR** — 2–4 bullet points
   - **Watchlist hits** — one block per ticker with hits
   - **Cluster signals** — the ≥3-filer cases
   - **Notable non-watchlist activity** — the top 3–5 tickers where activity looks material
   - **Proposed watchlist adds** — with reasoning. Mark as proposals, not instructions.

6. **Write to `data/reports/insiders-YYYY-MM-DD.md`** (create the dir if needed).

7. **Optional alert.** If your synthesis turns up something `act`-severity that the rule engine missed (e.g., qualitative read of a filing), insert a row into `alerts` with `source='insiders'` and a `payload` describing your reasoning.

## Output

- `data/reports/insiders-YYYY-MM-DD.md`
- Optional: new `alerts` rows for novel `act`-severity findings.

## Ethics

Observations, not advice. See `CLAUDE.md` Ethical Framing. End the report with the disclaimer.
```

- [ ] **Step 4: Write `modes/alerts.md` (WIRED)**

```markdown
# Mode: alerts — Triage the Unseen

Triages the current `alerts` queue: groups, prioritizes, recommends dismiss vs. investigate.

## Recommended execution

Subagent.

## Inputs
- Reads `alerts` table (especially unseen).

## Workflow

1. `./bin/jobs list-alerts --unseen` (or equivalent SQL).
2. Group by source, then by severity, then by ticker.
3. For each group, produce:
   - One-line summary
   - Recommended action: **investigate**, **dismiss**, or **monitor**
   - Reasoning (one sentence)
4. For alerts recommended as **dismiss**, shell out to `./bin/jobs mark-seen <id>`.
5. Leave **investigate** and **monitor** alerts unseen so they stay on the Alerts screen.

## Output

Plaintext summary printed to the session. No file written (transient triage).

## Ethics

Dismissing is a housekeeping action, not a trading decision. Err on the side of leaving ambiguous alerts for the user to decide.
```

- [ ] **Step 5: Write the stub mode files**

Create `modes/momentum.md`:

```markdown
# Mode: momentum — Watchlist Momentum Signals (STUB)

**Status:** not implemented. This file is a placeholder for the next real mode.

## What this mode should do (design intent)

Compute short-term momentum signals over the watchlist from EOD prices, and surface breakouts.

## What's needed to implement

1. `jobs fetch-prices` command that pulls EOD OHLC from a free source (Stooq or Tiingo free tier; 1000 req/day is enough for ≤50 tickers).
2. `0003_prices.sql` migration adding a `prices` table keyed on (ticker, date).
3. `jobs compute-momentum` that runs on new price rows and writes alerts when:
   - 5d-return crosses the threshold stored in `config/profile.yml:reporting.momentum.*`
   - Volume anomaly (today's volume > 2× 20d average)
4. Mode workflow mirroring `modes/insiders.md`.

## Why it's stubbed in v1

To keep the skeleton small and validate the insiders-mode pattern before cloning it.
```

Create `modes/sector.md`:

```markdown
# Mode: sector — Industry & Sector Snapshot (STUB)

**Status:** not implemented.

## Design intent

Generate a sector-level synthesis (semis, energy, biotech, etc.) pulling from:
- Recent earnings call transcripts (company IR pages — free)
- Public SEC filings (10-Ks, 10-Qs via EDGAR JSON)
- Macro context from FRED (see modes/macro.md)
- Relevant alerts from `alerts` table

Output: `data/reports/sector-<name>-YYYY-MM-DD.md`.

## Why it's stubbed

Depends on price data (momentum) and macro data (macro mode) being wired up first.
```

Create `modes/macro.md`:

```markdown
# Mode: macro — Macroeconomic Digest (STUB)

**Status:** not implemented.

## Design intent

Pull macro indicators from FRED (St. Louis Fed) — e.g., 10Y yield, 2Y yield, DXY, CPI YoY, unemployment — and narrate what they imply for equity positioning.

## What's needed

1. FRED API key (free registration) stored in `FRED_API_KEY` env var (never committed).
2. `jobs fetch-macro` pulling a curated set of series to a `macro_series` table.
3. `modes/macro.md` workflow that reads the most recent observations and produces a short brief with sections for rates, dollar, inflation, labor.

## Why it's stubbed

FRED integration + macro rubric is its own spec → plan cycle.
```

- [ ] **Step 6: Commit**

```bash
git add modes/
git commit -m "docs(modes): _shared + wired (insiders, alerts) + stub modes"
```

---

### Task 19: Claude Code skill file

**Files:**
- Create: `.claude/skills/market-digest/SKILL.md`, `.claude/settings.json`

- [ ] **Step 1: Write `.claude/skills/market-digest/SKILL.md`**

```markdown
---
name: market-digest
description: Use when the user types /digest <mode> in this repo, or asks for insider-trade / watchlist / market-ideation synthesis. Routes to a markdown mode file under modes/, loads shared context first, runs fetch jobs as needed, and produces a report. Read CLAUDE.md and modes/_shared.md before any mode.
---

# market-digest — Mode dispatcher

## What this repo is

A personal, fork-friendly equity-ideation tool. Go runtime (`./bin/jobs`, `./bin/dashboard`) does deterministic work (fetching, storing, alerting). Claude (via this skill and `modes/*.md`) does synthesis.

## How to invoke a mode

When the user says `/digest <name>`, `/market-digest <name>`, or asks for one of:
- "what's new on insider trades", "refresh insiders", "anything worth acting on" → **insiders**
- "triage my alerts", "what should I dismiss" → **alerts**
- "momentum", "sector", "macro" → tell the user the mode is stubbed and point at `modes/<name>.md`

## Required preamble for every mode

1. Read `CLAUDE.md` (repo root) — shared rules, ethical framing, plan-board protocol.
2. Read `modes/_shared.md` — data access conventions.
3. Read `modes/_profile.md` if it exists (user overrides).
4. Read the specific `modes/<name>.md`.
5. Run the mode workflow as a subagent when the mode file recommends it.

## Plan-board protocol

If `implementation_plan.md` exists at the repo root, follow the protocol in `CLAUDE.md` §"Agent Plan-Board Protocol":
- Mark the phase you're working on `[IN PROGRESS — <session> — <ts>]` before starting.
- Update to `[DONE — <PR or local>]` after finishing.
- `[BLOCKED — ...]` if blocked.

## Useful commands

```bash
./bin/jobs --help              # list all subcommands
./bin/jobs doctor              # check DB, config, source reachability
./bin/jobs fetch-insiders      # refresh Senate + House trade data
./bin/jobs list-alerts --unseen --since 168h
./bin/jobs mark-seen <id>
./bin/dashboard                # open the TUI
```

## Ethical framing

This skill produces *observations*, not *advice*. Every report ends with the disclaimer from `CLAUDE.md`. All sources are public.
```

- [ ] **Step 2: Write `.claude/settings.json`** (shared, minimal — permissions for the tools every mode uses)

```json
{
  "permissions": {
    "allow": [
      "Bash(./bin/jobs:*)",
      "Bash(./bin/dashboard:*)",
      "Bash(sqlite3:*)",
      "Bash(make:*)",
      "Read",
      "Write(data/reports/*)",
      "Edit(modes/*)"
    ]
  }
}
```

- [ ] **Step 3: Commit**

```bash
git add .claude/
git commit -m "feat(claude): market-digest skill and shared settings"
```

---

## Phase 6 — Scheduling, docs, and polish

### Task 20: launchd plist, install script, crontab example

**Files:**
- Create: `scripts/launchd/com.market-digest.fetch-insiders.plist`, `scripts/install-launchd.sh`, `scripts/crontab.example`

- [ ] **Step 1: Write the plist template**

Create `scripts/launchd/com.market-digest.fetch-insiders.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.market-digest.fetch-insiders</string>

  <key>ProgramArguments</key>
  <array>
    <string>__DIGEST_HOME__/bin/jobs</string>
    <string>fetch-insiders</string>
  </array>

  <key>EnvironmentVariables</key>
  <dict>
    <key>DIGEST_HOME</key>
    <string>__DIGEST_HOME__</string>
    <key>PATH</key>
    <string>/usr/local/bin:/usr/bin:/bin</string>
  </dict>

  <key>WorkingDirectory</key>
  <string>__DIGEST_HOME__</string>

  <key>StartCalendarInterval</key>
  <dict>
    <key>Hour</key><integer>7</integer>
    <key>Minute</key><integer>30</integer>
  </dict>

  <key>StandardErrorPath</key>
  <string>/tmp/market-digest-fetch-insiders.err.log</string>
  <key>StandardOutPath</key>
  <string>/tmp/market-digest-fetch-insiders.out.log</string>
</dict>
</plist>
```

- [ ] **Step 2: Write `scripts/install-launchd.sh`**

```bash
#!/usr/bin/env bash
# Install market-digest launchd agents. Substitutes the placeholder path
# in each *.plist under scripts/launchd/ with the absolute path of this
# repo and loads the result into ~/Library/LaunchAgents.

set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$HOME/Library/LaunchAgents"
mkdir -p "$DEST"

for plist in "$REPO"/scripts/launchd/*.plist; do
  name="$(basename "$plist")"
  target="$DEST/$name"
  sed "s|__DIGEST_HOME__|$REPO|g" "$plist" > "$target"
  launchctl unload "$target" >/dev/null 2>&1 || true
  launchctl load "$target"
  echo "loaded: $target"
done

echo "Done. Check logs in /tmp/market-digest-*.log"
```

- [ ] **Step 3: Write `scripts/crontab.example`**

```
# market-digest — crontab example. Copy relevant lines to `crontab -e`.
# Replace /PATH/TO/MARKET-DIGEST with your actual checkout path.

30 7  * * *  cd /PATH/TO/MARKET-DIGEST && ./bin/jobs fetch-insiders >> /tmp/market-digest.log 2>&1
# 0  */2 * * *  cd /PATH/TO/MARKET-DIGEST && ./bin/jobs compute-momentum >> /tmp/market-digest.log 2>&1   # when implemented
```

- [ ] **Step 4: Make the script executable and verify**

```bash
chmod +x scripts/install-launchd.sh
./scripts/install-launchd.sh
```

Expected: prints `loaded: ~/Library/LaunchAgents/com.market-digest.fetch-insiders.plist`. Check with `launchctl list | grep market-digest`.

(Optional rollback: `launchctl unload ~/Library/LaunchAgents/com.market-digest.fetch-insiders.plist`.)

- [ ] **Step 5: Commit**

```bash
git add scripts/
git commit -m "feat(scripts): launchd plist + install + crontab example"
```

---

### Task 21: Docs

**Files:**
- Create: `docs/getting-started.md`, `docs/adding-a-mode.md`, `docs/adding-a-data-source.md`

- [ ] **Step 1: Write `docs/getting-started.md`**

```markdown
# Getting Started

## Prerequisites
- Go 1.24+
- macOS or Linux (launchd/cron for scheduling)
- Claude Code CLI installed (for the modes layer)

## First run

```bash
git clone <this-repo> market-digest && cd market-digest
make build
cp config/profile.example.yml config/profile.yml
cp config/watchlist.example.yml config/watchlist.yml
./bin/jobs migrate
./bin/jobs doctor
./bin/jobs fetch-insiders
./bin/dashboard     # press 1-4 to switch screens; q to quit
```

## What's in the box

- `./bin/jobs` — one-shot subcommands: `fetch-insiders`, `list-alerts`, `mark-seen`, `doctor`, `migrate`, and stubs for upcoming modes.
- `./bin/dashboard` — TUI over `data/digest.db`. Alerts, Watchlist, Ticker detail, Jobs.
- `modes/` — markdown workflows invoked by Claude Code via `/digest <mode>`.
- `config/` — user-editable YAML. `*.example.yml` is tracked; `*.yml` is gitignored.
- `data/digest.db` — single SQLite file. Back this up, lose it, re-create it — no secrets live here.

## Schedule a daily fetch (macOS)

```bash
./scripts/install-launchd.sh
```

Schedules `fetch-insiders` at 07:30 local. Edit the `.plist` in `scripts/launchd/` if you want different times or additional jobs.

## Schedule a daily fetch (Linux)

```bash
crontab -e
# then add a line from scripts/crontab.example (with your path)
```

## Using modes

In Claude Code, after opening this repo:

```
/digest insiders
/digest alerts
```

(Or just describe what you want — the `market-digest` skill routes you.)

## Troubleshooting

- `jobs doctor` → shows DB path, config status, and per-source HTTP HEAD results.
- `./bin/dashboard` screen 4 → shows `job_runs` history with errors.
- `sqlite3 data/digest.db` → ad-hoc queries.
```

- [ ] **Step 2: Write `docs/adding-a-mode.md`**

```markdown
# Adding a mode

A mode is a markdown file in `modes/` describing a workflow Claude runs.

## Minimal checklist

1. Copy `modes/alerts.md` as a template — it's the smallest wired mode.
2. Rewrite **Inputs**, **Workflow**, and **Output** sections for your new mode.
3. If your mode needs new data, **add a fetcher first**: a new subcommand in `jobs/cmd/`, a migration in `migrations/` if you need a new table, and tests.
4. Wire the fetcher → DB → alerts path. Then write the mode file that consumes it.
5. Update `.claude/skills/market-digest/SKILL.md` so the dispatcher knows about the mode.
6. Commit.

## Anti-patterns

- Don't have the mode WebFetch URLs if you could instead put the fetcher in Go. Cheaper, more reliable.
- Don't have the mode skip the freshness check. Always verify `job_runs` before reasoning over stale data.
- Don't output trading advice. Observations + proposals only. See `CLAUDE.md` Ethical Framing.
```

- [ ] **Step 3: Write `docs/adding-a-data-source.md`**

```markdown
# Adding a data source

Every new data source follows the same 5 steps.

## 1. Add an entry to `config/sources.yml`

```yaml
<mode_name>:
  <source_name>:
    url: "https://..."
    enabled: true
    user_agent: "market-digest <your-email>"   # only if the source requires it
```

If the source needs an API key: add `api_key_env: MYAPI_KEY` and read `os.Getenv("MYAPI_KEY")` in your fetcher. Never put the key in the YAML.

## 2. Extend `internal/config/config.go`

Add typed fields under `Sources` matching your new YAML shape.

## 3. Write a migration (if it needs its own table)

`migrations/000N_<name>.sql` — `jobs migrate` picks it up by numeric prefix.

## 4. Add a fetcher under `internal/<domain>/`

Follow the pattern from `internal/insiders/` — `fetch.go` (HTTP + parse + normalize + hash), `store.go` (INSERT OR IGNORE on hash), `rules.go` (alert evaluation against newly inserted rows), and tests for each.

## 5. Wire a subcommand in `jobs/cmd/`

Use `jobrun.Track(ctx, conn, "<name>", ...)` so the run shows up in the TUI Jobs screen.

Add it to a schedule (launchd plist or crontab) only after you're confident it works from `./bin/jobs <name>` by hand.
```

- [ ] **Step 4: Commit**

```bash
git add docs/
git commit -m "docs: getting-started, adding-a-mode, adding-a-data-source"
```

---

### Task 22: End-to-end smoke + v0.1.0 tag

**Files:**
- Create: `scripts/smoke.sh`

- [ ] **Step 1: Write `scripts/smoke.sh`**

```bash
#!/usr/bin/env bash
# End-to-end smoke. Builds, migrates, fetches (over httptest? no — against real
# sources), reports counts. Used to validate a fork/clone.

set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> go build ./..."
go build ./...

echo "==> go test ./..."
go test ./...

echo "==> make build"
make build

if [ ! -f config/profile.yml ]; then
  cp config/profile.example.yml config/profile.yml
fi
if [ ! -f config/watchlist.yml ]; then
  cp config/watchlist.example.yml config/watchlist.yml
fi

echo "==> jobs migrate"
./bin/jobs migrate

echo "==> jobs doctor"
./bin/jobs doctor || true

echo "==> jobs fetch-insiders (first run — may take a moment)"
./bin/jobs fetch-insiders

echo "==> counts"
sqlite3 data/digest.db "SELECT 'trades', COUNT(*) FROM insider_trades UNION ALL SELECT 'alerts', COUNT(*) FROM alerts"

echo "==> jobs list-alerts --since 168h (sample)"
./bin/jobs list-alerts --since 168h | head -20

echo
echo "Smoke OK. Run ./bin/dashboard to open the TUI."
```

- [ ] **Step 2: Make executable and run**

```bash
chmod +x scripts/smoke.sh
./scripts/smoke.sh
```

Expected: builds, all tests pass, migration applies, doctor reports ok, fetch-insiders fetches ~thousands of rows, list-alerts shows recent entries.

- [ ] **Step 3: Verify help output for completeness**

```bash
./bin/jobs --help
./bin/dashboard --help 2>&1 | head || true   # dashboard has no flags, ok
```

Expected: `jobs --help` lists migrate, doctor, fetch-insiders, fetch-prices (stub), compute-momentum (stub), report-sector (stub), list-alerts, mark-seen.

- [ ] **Step 4: Final polish — update README with actual smoke-test command**

In `README.md`, under "Quick start", append:

```markdown
Or run the smoke test:

```bash
./scripts/smoke.sh
```
```

- [ ] **Step 5: Commit and tag**

```bash
git add scripts/smoke.sh README.md
git commit -m "chore: end-to-end smoke script"
git tag -a v0.1.0 -m "v0.1.0: market-digest skeleton with wired insiders mode"
```

- [ ] **Step 6: (Optional) Rename the working directory**

If you're still inside `day-trader-digest/`, you can rename:

```bash
cd ..
mv day-trader-digest market-digest
cd market-digest
```

The Go module name is already `market-digest`; the directory rename is cosmetic.

---

## Self-review notes (for the plan author)

- **Spec coverage:** every §1–12 requirement in the spec is covered by a numbered task. v1 use case #1 (insiders) is end-to-end; use cases #2–5 are stubbed per spec §2. Dashboard §7 screens all have tasks. Config §9 all three files tracked. Scheduling §10 has launchd plist + crontab. Plan-board §10 encoded in CLAUDE.md + skill. Ethical framing §8 encoded in `_shared.md` and mode files.
- **Placeholder scan:** mode stubs (`momentum.md`, `sector.md`, `macro.md`) and `stubs.go` subcommands are intentionally labeled as stubs with clear follow-up breadcrumbs — these are not plan failures, they are the explicit v1 boundary in the spec.
- **Type consistency:** `insiders.Trade` used identically across `fetch.go`, `store.go`, `rules.go`; `data.AlertRow` used identically across `data.go` and `screens/alerts.go`; `jobrun.Fn` signature matches caller site in `fetch_insiders.go`.
- **Known small imperfections tolerated:** `list-alerts --since 7d` requires Go's `time.ParseDuration` which doesn't understand `d`; documented as "use `168h`." Cluster alert dedup is per UTC day (documented in the test). Real-source smoke in Task 22 hits GitHub; acceptable for a personal tool.
