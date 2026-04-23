# Market-Digest Skeleton — Design Spec

**Date:** 2026-04-23
**Status:** Draft — awaiting user review
**Author:** nick.mj.kim@gmail.com (with Claude Opus 4.7)

---

## 1. Context & Motivation

This repository is the starting skeleton for **market-digest** — a personal, fork-friendly tool for ideating around US equity positions using free public data and Claude as the synthesis engine. The structure intentionally mirrors [santifer/career-ops](https://github.com/santifer/career-ops) (a parallel "agentic workflow + Go TUI" project): markdown-defined Claude "modes" for higher-level reasoning, a Bubbletea terminal dashboard for reviewing state, and small utility scripts/jobs that fetch and normalize data.

The directory is currently called `day-trader-digest` on the author's machine. **The project name is `market-digest`** going forward — "day trader" overpromises (see §2 non-goals). The working directory will be renamed, and the Go module name is `github.com/<user>/market-digest`.

### Intended usage (author's own, representative of forkers)

The tool should make it easy to:

1. Track politician/insider trades and surface meaningful activity on a watchlist
2. Identify momentum building on watchlist stocks
3. Generate industry and sector development reports
4. Understand overall macroeconomic factors
5. Run scheduled + event-driven jobs that surface "you should look at this"

These are **use cases**, not v1 features. V1 ships the skeleton plus **one fully-wired end-to-end example** (insider trades) that establishes the pattern. The other use cases are stubbed modes that future work will flesh out one at a time.

## 2. Scope

### In scope for v1

- Repo layout, conventions, and docs (CLAUDE.md, FORK_NOTES.md, implementation_plan.md protocol)
- Go TUI dashboard with four screens (Alerts, Watchlist, Ticker Detail, Jobs)
- Go `jobs` binary with cobra-style subcommands
- SQLite schema + migrations runner
- `insiders` mode fully implemented end-to-end (fetch → dedup → alert → Claude narrative)
- Stubbed mode files for `momentum`, `sector`, `macro`
- Wired `alerts` mode (triage helper over whatever is in the DB)
- Config loader + three example YAML configs (profile, watchlist, sources)
- macOS launchd plist + Linux crontab example for scheduling
- `.claude/skills/market-digest/` skill file teaching Claude how to invoke modes

### Out of scope for v1 (explicitly deferred)

- **Intraday / real-time data.** Free sources are 15-min delayed. This is an *ideation* tool on swing/position horizons, not an execution tool.
- **Price fetching & momentum computation.** Stubbed only. Next mode to implement.
- **Sector & macro fetching.** Stubbed only. Future modes.
- **Charts/sparklines in the TUI.** High time-sink for marginal value. Add once real price data exists.
- **Any order-placement capability.** Never. See §11 non-goals.
- **`jobs daemon`** — no long-running scheduler process. OS cron/launchd only.
- **Paid data sources.** Skeleton assumes free-only. A source can be added under `config/sources.yml` with `api_key_env:` later without redesign.

### Non-user-visible guarantees

- `go build ./...` produces two binaries (`dashboard`, `jobs`) with **no cgo** (pure-Go SQLite driver `modernc.org/sqlite`). Forks work on any platform with a Go toolchain.
- Every job is idempotent: re-running produces 0 new rows until upstream data updates.
- Every write to the DB is observable via `job_runs`.

## 3. Architecture Overview

Two-layer split (chosen in brainstorming — see §12 for rationale):

```
  ┌──────────────────────────────┐
  │  Claude Code (modes + skill) │   reasoning / synthesis / narrative
  └──────────┬───────────────────┘
             │ invokes
             ▼
  ┌──────────────────────────────┐
  │  Go:  jobs binary            │   fetch · normalize · dedup · alert
  │  Go:  dashboard binary       │   render · triage
  └──────────┬───────────────────┘
             │ reads/writes
             ▼
  ┌──────────────────────────────┐
  │  SQLite (data/digest.db)     │   single file, WAL mode
  └──────────────────────────────┘
```

- **Claude** handles everything token-worthy: summarizing, clustering patterns, writing reports, proposing watchlist additions. Invoked by the user via slash commands wired up through `.claude/skills/market-digest/`.
- **Go** handles everything deterministic: HTTP, parsing, schema writes, alert-rule evaluation, TUI rendering. Runs without Claude.
- **SQLite** is the contract between the two layers. Everything both layers care about lives in the DB.

## 4. Repo Layout

```
market-digest/
├── CLAUDE.md                    # system rules (tracked)
├── CLAUDE.local.md              # per-machine overrides (gitignored, template shipped)
├── FORK_NOTES.md                # fork divergence doc (tracked)
├── README.md
├── Makefile                     # build, run, migrate, lint shortcuts
├── go.mod                       # single module at repo root
├── go.sum
├── .gitignore
├── .claude/
│   ├── settings.json            # tracked — shared hooks, permissions
│   ├── settings.local.json      # gitignored
│   └── skills/
│       └── market-digest/
│           └── SKILL.md         # teaches Claude how to invoke modes
├── modes/
│   ├── _shared.md               # auto-updatable, loaded by every mode
│   ├── _profile.template.md     # template (tracked)
│   ├── _profile.md              # user-edited (gitignored)
│   ├── insiders.md              # WIRED — anchor example
│   ├── momentum.md              # stub
│   ├── sector.md                # stub
│   ├── macro.md                 # stub
│   └── alerts.md                # WIRED — triage
├── dashboard/
│   ├── main.go
│   └── internal/
│       ├── data/                # DB reads for the TUI
│       ├── model/
│       ├── theme/
│       └── ui/screens/
│           ├── alerts.go
│           ├── watchlist.go
│           ├── ticker.go
│           └── jobs.go
├── jobs/
│   ├── main.go
│   └── cmd/
│       ├── migrate.go
│       ├── fetch_insiders.go    # WIRED
│       ├── fetch_prices.go      # stub
│       ├── compute_momentum.go  # stub
│       ├── report_sector.go     # stub
│       ├── doctor.go
│       ├── list_alerts.go
│       └── mark_seen.go
├── internal/                    # shared Go packages
│   ├── db/                      # sqlite open, migrations runner
│   ├── config/                  # yaml loaders (profile, watchlist, sources)
│   ├── jobrun/                  # job_runs bookkeeping helper
│   └── alert/                   # shared alert-write helpers
├── migrations/
│   ├── 0001_init.sql
│   └── 0002_insiders.sql
├── config/
│   ├── profile.example.yml
│   ├── watchlist.example.yml
│   └── sources.yml              # tracked (URLs/rules, no secrets)
├── data/
│   └── .gitkeep                 # digest.db lives here, gitignored
├── scripts/
│   ├── install-launchd.sh       # substitutes paths, copies plist
│   ├── launchd/
│   │   └── com.market-digest.fetch-insiders.plist
│   └── crontab.example
└── docs/
    ├── getting-started.md
    ├── adding-a-mode.md
    ├── adding-a-data-source.md
    └── superpowers/
        └── specs/
            └── 2026-04-23-market-digest-skeleton-design.md
```

### `.gitignore`

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

# Agent plan board (see §10)
implementation_plan.md

# OS noise
.DS_Store

# Go build artifacts
*.out
*.test
*.prof

# Reports written by modes (user-specific)
data/reports/
```

**Note:** `config/sources.yml` IS tracked (not user-specific) because URLs + rate limits are part of the shared config. API keys go in env vars, never in files.

## 5. Data Model (SQLite)

### `migrations/0001_init.sql`

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS watchlist (
  ticker      TEXT PRIMARY KEY,
  note        TEXT,
  added_ts    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS alerts (
  id          INTEGER PRIMARY KEY,
  created_ts  INTEGER NOT NULL,
  source      TEXT NOT NULL,          -- 'insiders' | 'momentum' | 'macro' | ...
  severity    TEXT NOT NULL,          -- 'info' | 'watch' | 'act'
  ticker      TEXT,                   -- nullable (macro alerts have no ticker)
  title       TEXT NOT NULL,
  body        TEXT,                   -- markdown, rendered in dashboard detail pane
  payload     TEXT,                   -- JSON, source-specific structured context
  seen_ts     INTEGER                 -- NULL until user presses 'x' in TUI
);
CREATE INDEX alerts_unseen ON alerts(seen_ts) WHERE seen_ts IS NULL;
CREATE INDEX alerts_created ON alerts(created_ts DESC);

CREATE TABLE IF NOT EXISTS job_runs (
  id          INTEGER PRIMARY KEY,
  job         TEXT NOT NULL,
  started_ts  INTEGER NOT NULL,
  finished_ts INTEGER,
  status      TEXT NOT NULL,          -- 'running' | 'ok' | 'error' | 'noop'
  rows_in     INTEGER,
  rows_new    INTEGER,
  error       TEXT
);
CREATE INDEX job_runs_job_started ON job_runs(job, started_ts DESC);

CREATE TABLE IF NOT EXISTS schema_version (
  version     INTEGER PRIMARY KEY,
  applied_ts  INTEGER NOT NULL
);
```

### `migrations/0002_insiders.sql`

```sql
CREATE TABLE IF NOT EXISTS insider_trades (
  id             INTEGER PRIMARY KEY,
  source         TEXT NOT NULL,        -- 'senate' | 'house' | 'sec-form4'
  filer          TEXT NOT NULL,        -- politician/insider name
  role           TEXT,                 -- 'Senator' | 'Representative' | 'Director' | ...
  ticker         TEXT,
  asset_desc     TEXT,                 -- raw description from source
  side           TEXT,                 -- 'buy' | 'sell' | 'exchange'
  amount_low     INTEGER,              -- disclosures are ranges; $1K-$15K, etc.
  amount_high    INTEGER,
  transaction_ts INTEGER,              -- when the trade happened
  filing_ts      INTEGER,              -- when it was reported
  raw_url        TEXT,                 -- link to the original filing
  hash           TEXT UNIQUE NOT NULL
);
CREATE INDEX insider_trades_ticker_ts ON insider_trades(ticker, transaction_ts DESC);
CREATE INDEX insider_trades_filer ON insider_trades(filer);
CREATE INDEX insider_trades_filing_ts ON insider_trades(filing_ts DESC);
```

**Dedup hash:** `sha1(source|filer|ticker|transaction_date|amount_low|amount_high|side)`. `UNIQUE(hash)` makes `INSERT OR IGNORE` the dedup primitive.

**Migrations runner:** `jobs migrate` reads files from `migrations/` in order, applies any whose number isn't in `schema_version`, records the version. No ORM, no external tool, plain `database/sql`.

## 6. Jobs Binary

Single binary. Subcommands via `github.com/spf13/cobra` (justified: discoverability, per-command flags, grows with modes).

### v1 command surface

```
jobs migrate                    apply pending migrations (idempotent)
jobs fetch-insiders             WIRED — fetch Senate + House JSON, dedup, write alerts
jobs fetch-prices               STUB — will hit Stooq/Tiingo in momentum follow-up
jobs compute-momentum           STUB
jobs report-sector <sector>     STUB — invoked by /digest sector
jobs doctor                     check DB, config, source URLs, print status
jobs list-alerts [--unseen] [--since 7d] [--source insiders]
jobs mark-seen <alert-id>
```

### Conventions every job follows

- Reads `DIGEST_HOME` env (defaults to CWD). Config + DB paths resolved relative to it.
- Opens SQLite with `_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)` — TUI can read concurrently.
- Wraps work in `internal/jobrun.Track(ctx, jobName, func() (rows_in, rows_new int, err error))` which:
  - Inserts `job_runs` row with `status='running'` on entry
  - Recovers panics → `status='error'`, stack in `error`
  - Updates `finished_ts`, `status`, counts on exit
- Logs structured JSON (slog) to stderr.
- Exit codes: **0** ok, **1** error, **2** no-op (nothing to do / source unchanged).

### `fetch-insiders` in detail (anchor example)

```
1. migrate (no-op if up to date)
2. jobrun.Track("fetch-insiders", ...)
3. For each enabled source in sources.yml:insiders:
     a. HTTP GET (with retry/backoff from sources.yml:http)
     b. Unmarshal JSON into []RawTrade
     c. Normalize → []InsiderTrade (tickers uppercased, side normalized to buy/sell/exchange)
     d. Compute hash per row
     e. INSERT OR IGNORE INTO insider_trades (...)
     f. Collect rowids of newly-inserted rows via RETURNING id
4. Evaluate alert rules over new rows (watchlist_hit, amount_over_1m, amount_over_500k):
     - INSERT alerts rows with payload = JSON of the trade
5. Cluster rule (post-dedup): SELECT ticker, COUNT(DISTINCT filer) FROM insider_trades
   WHERE transaction_ts > now-7d GROUP BY ticker HAVING COUNT(DISTINCT filer) >= 3.
   Emit one 'watch' alert per qualifying ticker (once per day; dedup via (source='insiders', title LIKE 'Cluster: ...', DATE(created_ts)=today)).
6. Return (rows_in, rows_new). jobrun closes out.
```

**Data sources (free, no auth):**

- Senate: `raw.githubusercontent.com/jeremiak/senate-stock-watcher-data/main/aggregate/all_transactions.json`
- House: `raw.githubusercontent.com/.../house-stock-watcher-data/main/transactions/all_transactions.json`
- SEC Form 4 (stubbed, `enabled: false` by default): `https://www.sec.gov/cgi-bin/browse-edgar?...type=4&output=atom`. Requires `User-Agent` with contact per SEC rules.

## 7. Dashboard TUI

Bubbletea + lipgloss (same stack as career-ops). Single binary, opens on Alerts.

### Screens

| # | Screen | Contents | Primary bindings |
|---|--------|----------|-------------------|
| 1 | **Alerts** | Unseen first, then recent seen. Columns: time, source, severity (colored), ticker, title. Right pane: markdown `body` of selected alert. | `↑/↓` move, `enter` toggle detail, `x` mark seen, `/` filter, `s` cycle severity filter, `1-4` switch screen |
| 2 | **Watchlist** | Tickers + note + counts (recent alerts, recent insider trades). | `a` add ticker, `d` remove, `enter` → ticker detail |
| 3 | **Ticker detail** | Recent insider trades table, price pane placeholder ("momentum not yet implemented"), alerts-for-this-ticker history. | `b` back, `o` open filing URL via `$BROWSER` |
| 4 | **Jobs** | Last 20 `job_runs`. Status, duration, counts, truncated error. | `enter` full error, `r` run job now (shells out), `b` back |

**Footer (always visible):** `digest.db @ <path>  ·  last fetch-insiders: 3h ago  ·  unseen: 7`

### Writes allowed from TUI

- Mark alert seen (`UPDATE alerts SET seen_ts=?`)
- Add/remove watchlist entry (writes both to `watchlist` table AND to `config/watchlist.yml` so the YAML stays authoritative; if the YAML write fails the DB write is rolled back).
- Trigger `jobs <name>` (spawns a subprocess, shows its stderr in a panel, reloads affected screens on exit).

No other business logic in the TUI — it's a view over the DB plus three imperative actions.

### What's explicitly NOT in v1

- Charts / sparklines.
- Vim-style command palette.
- Theme switcher (one default theme; users edit `internal/theme/theme.go`).

## 8. Modes (Claude Layer)

Each mode is a markdown file in `modes/`. The `.claude/skills/market-digest/SKILL.md` teaches Claude to invoke them via `/digest <mode>` (short, memorable).

### Mode file shape

```markdown
# Mode: <name> — <one-line purpose>

## Recommended execution
Run as a subagent (general-purpose) to avoid consuming main context.

## Inputs
- CLI args
- Data read (DB tables, config files, report files)
- Dependencies on other modes or jobs

## Workflow
Numbered steps. First step usually: "ensure data is fresh: run `jobs fetch-X` if job_runs shows nothing today."

## Output
Where results go (usually `data/reports/<mode>-<date>.md` + optional `alerts` row).
```

### v1 mode inventory

| Mode | Status | Purpose |
|------|--------|---------|
| `_shared.md` | auto-updatable | System rules, severity thresholds, ethical framing, pointers to `_profile.md` and `CLAUDE.md`. |
| `_profile.template.md` | template (tracked) | Commented skeleton of what the user should fill in. |
| `insiders.md` | WIRED | Refresh insider data, query last 7–14 days, narrate meaningful patterns, propose watchlist adds. |
| `momentum.md` | STUB | TODOs: fetcher → prices table → breakout detection. Clearly flagged as "next to implement." |
| `sector.md` | STUB | TODOs: per-sector synthesis using public filings + macro context. |
| `macro.md` | STUB | TODOs: FRED/Treasury/BLS ingestion + synthesis. |
| `alerts.md` | WIRED | Triage current unseen alerts: group, prioritize, recommend dismiss/act, call `jobs mark-seen` for dismissals. |

### `insiders.md` end-to-end detail

1. Read `modes/_shared.md` + `modes/_profile.md` + `config/watchlist.yml`.
2. Query `job_runs` WHERE job='fetch-insiders' AND DATE(started_ts,'unixepoch') = DATE('now'). If 0 rows or latest is error/noop, run `jobs fetch-insiders`.
3. SQL queries:
   - Recent `insider_trades` (last 14d) joined to `watchlist`.
   - All `severity IN ('watch','act')` alerts from `source='insiders'` in last 14d.
   - Top tickers by filer count (last 7d).
4. Group by ticker / by filer / by sector inferred from ticker.
5. Produce report with sections: TL;DR, Watchlist hits, Cluster signals, Notable non-watchlist activity, Proposed watchlist adds (with rationale).
6. Write to `data/reports/insiders-YYYY-MM-DD.md`.
7. If any finding warrants `severity='act'` but no alert row exists, INSERT one.

### Ethical framing (enforced in `_shared.md`)

- Report *observations*, not *advice*. "3 senators sold semis" not "you should short semis."
- Never narrate as if the user is about to execute; frame as "things to look at."
- No shortcuts that imply non-public information. All sources are public.
- Legal disclaimer at the bottom of every generated report.

## 9. Config

Three files. `*.example.yml` tracked; user copies to `*.yml` (gitignored, except `sources.yml` which is tracked).

### `config/profile.example.yml`

```yaml
user:
  display_name: "Your Name"
  timezone: "America/New_York"

risk:
  tolerance: "moderate"           # conservative | moderate | aggressive — free-form context
  max_position_pct: 5
  notes: "No leverage. No options outside watchlist."

interests:
  sectors: ["semis", "energy", "biotech"]
  themes: ["AI infra", "onshoring", "grid buildout"]
  avoid: ["crypto-adjacent penny stocks"]

reporting:
  dollar_thresholds:              # used by insiders alert rules
    watch: 500000
    act:   1000000
  cluster_window_days: 7

notifications:
  macos: false                    # if true & on darwin, osascript notification for 'act'
  webhook_url: ""                 # if set, POST JSON for 'act' alerts
```

### `config/watchlist.example.yml`

```yaml
tickers:
  - ticker: NVDA
    note: "Core AI infra hold"
    added: 2026-01-15
  - ticker: CEG
    note: "Grid/nuclear thesis"
    added: 2026-02-01
```

`jobs migrate` syncs this YAML → `watchlist` table. TUI edits write both. YAML is the authored source of truth; DB is the runtime mirror.

### `config/sources.yml` (tracked)

```yaml
insiders:
  senate:
    url: "https://raw.githubusercontent.com/jeremiak/senate-stock-watcher-data/main/aggregate/all_transactions.json"
    enabled: true
  house:
    url: "https://raw.githubusercontent.com/.../house-stock-watcher-data/main/transactions/all_transactions.json"
    enabled: true
  sec_form4:
    url: "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&type=4&output=atom"
    enabled: false
    user_agent: "market-digest <your-email>"   # SEC rule; required when enabling

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

Forkers extend `sources.yml` to add new data sources. Paid sources use `api_key_env: MYAPI_KEY` and read the value from env (never committed).

## 10. Scheduling, Alerts Delivery, and Agent Plan-Board

### Scheduling

Zero daemon. OS scheduler runs `jobs <command>`.

- **macOS:** `scripts/launchd/com.market-digest.fetch-insiders.plist`. Placeholder `/USERS/YOUR_USER/...` paths. `scripts/install-launchd.sh` does the substitution and `launchctl load`.
- **Linux:** `scripts/crontab.example` — user copies relevant lines into `crontab -e`.

Sample plist runs `fetch-insiders` daily at 07:30 local. Logs to `/tmp/digest-fetch-insiders.{out,err}.log`.

### Alerts delivery (beyond the TUI)

`alerts` table is canonical. Two optional push paths, both off by default:

1. **macOS notification** — if `profile.yml:notifications.macos: true` and GOOS==darwin, the job calls `osascript -e 'display notification "<title>" with title "market-digest"'` for each `severity='act'` row it inserts. Zero deps.
2. **Webhook** — if `profile.yml:notifications.webhook_url` is set, POST a JSON body (`{severity, ticker, title, body, source, alert_id}`). User wires to Slack/Discord/ntfy/whatever. No service-specific formatting.

### Agent plan-board protocol (the `implementation_plan.md` convention)

**Problem it solves:** When the user (or `superpowers:writing-plans`) produces a phased implementation plan, and multiple Claude sessions / subagents may work on it concurrently, we need a lightweight claim-board so no two agents work the same phase.

**Convention:**

- `implementation_plan.md` is gitignored (each worktree has its own state).
- When a plan is produced, it's written to this file with phases numbered and statuses set to `[PENDING]`.
- **Before starting a phase:** an agent edits the phase header to `[IN PROGRESS — <session-id or agent-name> — <ISO timestamp>]`.
- **After finishing:** `[DONE — <PR URL or "local">]` with a one-line note of what actually landed (spec drift is normal and expected).
- **When opening a PR for a phase:** the PR description links to the phase heading; the agent updates that phase line with the PR URL.
- **If blocked:** `[BLOCKED — <reason>]` + a comment paragraph. Never silently revert to pending.

`CLAUDE.md` teaches this protocol. The `.claude/skills/market-digest/SKILL.md` points to it. Because the file is gitignored, it travels with the worktree (not the repo), so each feature branch has independent claim state.

## 11. Non-Goals

Listed explicitly so scope creep has something to push back against:

- **Not a trading bot.** No order placement, no broker integration, no auto-anything.
- **Not real-time.** Free data is delayed; the tool is for ideation on swing/position horizons.
- **Not a backtester.** No historical simulation framework. If a mode wants to validate a signal, it writes a one-off query.
- **Not a replacement for licensed research.** Claude's synthesis over public data is a complement, not a substitute.
- **No user accounts, no multi-user support.** Single-user, single-machine.

## 12. Rationale for Major Choices

### Why unified Go (TUI + jobs)

One stack. `go build ./...` gives you everything. Scheduled jobs benefit from Go's retries/typed HTTP/concurrency more than from bash's terseness. Career-ops split Go + Node because it grew that way; a greenfield skeleton doesn't need to.

### Why SQLite (not CSV)

CSVs work for career-ops because the user hand-edits the pipeline. Here, the user's flow is filter-and-sort in the TUI over thousands of insider trades. Joins across tables (watchlist × insider_trades × alerts) are natural in SQL, miserable in CSV. Pure-Go driver (`modernc.org/sqlite`) avoids cgo and keeps the single-`go build` promise.

### Why OS cron (not `jobs daemon`)

A long-running daemon needs the same nohup/launchd/systemd wrapper anyway, so you've saved nothing by building one. Plus, one-shot processes with exit codes are trivial to monitor and don't leak resources.

### Why two-layer split (Go for fetching, Claude for synthesis)

Fetching is boring deterministic work — Go does it faster, cheaper (no tokens), more reliably. Synthesis is where Claude actually adds value. Also: scheduled fetchers cost $0 per run, so running hourly is fine.

### Why start with `insiders` as the anchor

Free pre-parsed JSON sources (Senate/House Stock Watcher) exist — no PDF parsing, no scraping, no auth. The end-to-end flow (fetch → dedup → alert → narrative) is simple and exercises every layer of the architecture. Any forker can run it on day 1 with no API keys.

## 13. Open Questions

None blocking. Deferred decisions:

- Whether to use `github.com/spf13/cobra` vs `github.com/urfave/cli/v2`. Cobra is recommended; final pick is an implementation detail.
- Exact column set for `prices` table (pending momentum mode design).
- Whether sector classification comes from a static map, SEC SIC codes, or a third-party API (pending sector mode design).

## 14. References

- [santifer/career-ops](https://github.com/santifer/career-ops) — parallel project this mirrors
- [jeremiak/senate-stock-watcher-data](https://github.com/jeremiak/senate-stock-watcher-data) — Senate data source
- [house-stock-watcher-data](https://github.com/house-stock-watcher) — House data source
- SEC EDGAR: `https://www.sec.gov/edgar/sec-api-documentation`
- `modernc.org/sqlite` — pure-Go SQLite driver
- Bubbletea: `github.com/charmbracelet/bubbletea`
