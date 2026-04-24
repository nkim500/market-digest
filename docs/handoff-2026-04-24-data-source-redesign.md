# Handoff: Data Source Redesign for Insider/Politician Trade Scanning

**Date:** 2026-04-24
**Status:** v0.1.1 shipped. All three insider data sources in `config/sources.yml` now disabled. Local `data/digest.db` wiped. Fresh session picks up from here.
**Owner:** Whoever picks this next — paste this file into the new Claude Code session as the starting brief.

---

## What this repo is

`market-digest` is a Go-based skeleton for ideating on US equity positions using free public data, with Claude Code as the synthesis layer. Published at https://github.com/nkim500/market-digest (public, MIT, v0.1.1).

Architecture in two sentences: a Go `jobs` binary (Cobra CLI) fetches and normalizes data into a single SQLite file, a Go `dashboard` binary (Bubbletea TUI) renders it, and markdown mode files under `modes/` are invoked via Claude Code's `/digest <mode>` slash command to read the DB and produce narrative reports. All three layers are wired end-to-end.

Full design spec: [`docs/superpowers/specs/2026-04-23-market-digest-skeleton-design.md`](superpowers/specs/2026-04-23-market-digest-skeleton-design.md)

## What's already built (reusable)

### Go packages (all tested, pure-Go, no cgo)

| Package | Purpose |
|---|---|
| `internal/db` | SQLite open (WAL, busy_timeout=5000, MaxOpenConns=1) + migrations runner reading `migrations/NNNN_*.sql` |
| `internal/config` | YAML loader with `*.yml` → `*.example.yml` fallback (profile, watchlist, sources) |
| `internal/jobrun` | `jobrun.Track(ctx, conn, name, fn)` wraps any job body in a `job_runs` row with panic recovery + `ErrNoop` sentinel |
| `internal/alert` | Typed `Alert` struct + `Insert()` with severity validation (info/watch/act) and JSON payload serialization |
| `internal/insiders` | **Was the old anchor.** `Trade` type, `Hash()` dedup key, `NewClient()` HTTP fetcher with retry, `StoreInserts()` with `INSERT OR IGNORE`, `EvaluateRules()` alert engine |
| `internal/data` | Typed read helpers for the TUI (`RecentAlerts`, `Watchlist`, `RecentInsiderTrades`, `RecentJobRuns`, `LastJobRun`, `MarkSeen`) |
| `jobs/cmd` | Cobra subcommands: `migrate`, `doctor`, `fetch-insiders`, `list-alerts`, `mark-seen`, plus stubs for `fetch-prices`, `compute-momentum`, `report-sector` |
| `dashboard/internal/screens` | Bubbletea models: `AlertsModel`, `WatchlistModel`, `TickerModel`, `JobsModel` |

### Tables (schema in `migrations/`)

- `watchlist` — user's tickers, synced from `config/watchlist.yml` on each fetch run
- `alerts` — canonical surface for everything worth surfacing; rows have `severity` in {info, watch, act} and `seen_ts`
- `job_runs` — one row per job invocation; shown on the TUI's Jobs screen
- `insider_trades` — the old insider data table (UNIQUE hash for dedup); **keep the table**, just not the source that filled it

### CI/CD (all green on `main`)

`test.yml`, `codeql.yml`, `dependency-review.yml`, `labeler.yml`, `release.yml` (release-please), `sbom.yml`, `stale.yml`, `welcome.yml`, plus dependabot + coderabbit. Vulnerability alerts + automated security updates enabled on the repo.

## The problem (why you're here)

The `insiders` anchor was wired to `timothycarambat/senate-stock-watcher-data`. That repo was abandoned upstream on **2020-12-05** and contains only 2012–2020 data. It was shipped into v0.1.x without recency verification. The TUI showed these 5+-year-old trades with `alerts.created_ts = today` (when the fetcher inserted the alert), which made them look current at a glance.

Two concrete bugs uncovered:

1. **Stale source.** `timothycarambat/senate-stock-watcher-data/aggregate/all_transactions.json` has not been updated in 5+ years. Nothing after 2020-12-02 is in there.
2. **UI bug.** `dashboard/internal/screens/alerts.go` displays `r.CreatedTS` (alert creation time) as the row's time, never `transaction_ts` from `insider_trades`. Even with a fresh source, the Alerts screen needs to surface transaction time so stale-but-today-inserted rows aren't misread as current.

Both need to be fixed. The source problem is the bigger one.

## What was done to leave this clean

- `config/sources.yml` — `senate`, `house`, `sec_form4` all set to `enabled: false` with comments pointing at this doc.
- `data/digest.db` wiped (only `.gitkeep` remains in `data/`).
- `internal/insiders/` code is UNTOUCHED and still correct — hash, store, rules, fetcher scaffolding. A new fetcher can plug into the same flow.
- `modes/insiders.md` is UNCHANGED — it describes the mode workflow; the workflow is fine, the data underneath is what needs replacing.
- The dashboard binary still builds and runs; it just has nothing to show until a fresh data source lands.

## The open design question

**How do we actually get fresh insider + politician trade data?** Concrete options, ranked by my read:

### 1. SEC EDGAR Form 4 (corporate insiders — directors, officers, 10% owners)

- **Source:** `https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&type=4&output=atom` (the index) + per-filing `ownership.xml`
- **Freshness:** Real-time. Filings post within 2 business days of the trade.
- **Cost:** Free. SEC requires a `User-Agent` header with contact info — already scaffolded via `config/sources.yml:insiders.sec_form4.user_agent`.
- **Complexity:** Medium. Atom feed → list of filings → fetch each filing's XML → parse `<nonDerivativeTransaction>` elements. The data is structured and documented.
- **Coverage:** Corporate insider buying/selling. **Not politicians.**

This is almost certainly the highest-value first source to wire up. It's official, fresh, structured, free, and covers the "insider trades" half of the original brief.

### 2. Senate EFD (official Senate disclosures — politicians)

- **Source:** `https://efdsearch.senate.gov/search/` — the official Senate Periodic Transaction Report search
- **Freshness:** Updated as senators file (required within 30–45 days of the trade)
- **Cost:** Free
- **Complexity:** **High.** The site has a consent-gate form (checkbox + button) you have to POST through to establish a session cookie, THEN you can hit the search results. Results are HTML tables; per-filing detail is a PDF. PDF parsing is fragile.
- **Why the community JSON mirrors abandoned it:** the consent gate + PDF parsing is annoying to maintain, and senators have complained about scrapers.

Possible, but a real engineering lift. Realistic v0.2 scope if you want politician data from the source of truth.

### 3. US House Clerk disclosures (official House — politicians)

- **Source:** `https://disclosures-clerk.house.gov/FinancialDisclosure/Download` — annual ZIP files containing XML
- **Freshness:** Updated as representatives file
- **Cost:** Free
- **Complexity:** Medium. Download ZIP, extract XML, parse PTR entries. Each PTR is then a separate linked PDF for trade-level detail — same PDF-parsing problem as Senate at the leaf.
- **Year-boundary caveat:** data is sharded by year, so the fetcher needs to poll the current year's ZIP periodically.

### 4. Paid data vendors (if budget is acceptable)

- **QuiverQuant** — has a Congressional-trading API; paid after a very limited free tier (~150 req/day). Well-maintained, current, JSON.
- **Finnhub** — has Congressional trades on its free tier but coverage + freshness is weaker than Quiver. 60 req/min limit.
- **CapitolTrades.com** — nice UI, has an undocumented JSON backend that's scrapable; ToS grey area.
- **Polygon.io** — fine stock data API, doesn't really cover politician trades.

For a personal tool, spending $10–25/mo for QuiverQuant is the lowest-effort path to fresh politician data if the scraping is too painful. For SEC Form 4, pay nothing — go direct.

## Recommended next-session plan

Written as a brief for the next Claude Code agent. Use/modify freely.

### Phase 1 — SEC Form 4 fetcher (anchor for v0.2.0)

1. Read this doc + [`docs/superpowers/specs/2026-04-23-market-digest-skeleton-design.md`](superpowers/specs/2026-04-23-market-digest-skeleton-design.md) + [`CLAUDE.md`](../CLAUDE.md)
2. Brainstorm the Form 4 fetcher design with the user (use `superpowers:brainstorming`). Key questions:
   - Filter to a watchlist, or pull broadly and filter at query time? (Latter is more general.)
   - How far back on the first run? (Suggest: last 90 days; subsequent runs resume from the latest `filing_ts`.)
   - Schema — add a new `sec_form4_trades` table, or extend `insider_trades` with a `source='sec-form4'` and new columns? (Latter keeps the dedup and rules engine reusable.)
3. Write a spec + implementation plan using `superpowers:writing-plans`
4. Execute via `superpowers:subagent-driven-development`. Reuse existing package structure:
   - New `internal/insiders/edgar.go` alongside `fetch.go` (or a sibling package `internal/secedgar/` if you want cleaner separation)
   - `jobs/cmd/fetch_insiders.go` (or a new `fetch_sec_form4.go`) wires the fetcher into the existing `EvaluateRules` + `alert.Insert` flow
   - `config/sources.yml:insiders.sec_form4.enabled` flips to `true` by default

### Phase 2 — Fix the TUI display bug

Also in Phase 1 or as a sibling PR: `dashboard/internal/screens/alerts.go` should show BOTH dates in the Alerts screen:

```
  transaction: 2026-04-10  alert: 2026-04-24 07:31  insiders  watch  NVDA  ...
```

Or at minimum show `transaction_ts` via `alerts.payload.transaction_ts` (already stored in the JSON payload when `EvaluateRules` inserts the alert). This is a small surface change — mostly touching `screens/alerts.go`'s `View()` and `internal/data.AlertRow`.

### Phase 3 — Politician trade source (v0.3+, optional)

If Form 4 alone is enough for your ideation, stop here. If you specifically want politician trades:

- Easiest: subscribe to QuiverQuant, wire their API as a new source under `insiders:` in `sources.yml`, support `api_key_env: QUIVER_API_KEY`.
- Ambitious: build a proper EFD scraper (Senate) + Clerk FD.zip fetcher (House). Budget for ongoing maintenance.

## Guardrails the next agent should respect

All already encoded in `CLAUDE.md` and `modes/_shared.md` — but highlighting the critical ones:

1. **Go does fetching, Claude does synthesis.** Do not rewrite the fetcher to use WebFetch from a mode file. Keep the boundary.
2. **Never surface trades as advice.** Output stays observational. `modes/_shared.md` enforces this.
3. **SEC User-Agent is required.** Any fetcher hitting `sec.gov` MUST include `User-Agent: market-digest <email>`. Without it, SEC throttles or blocks.
4. **The hash function is load-bearing.** Changing `internal/insiders/Hash()`'s inputs invalidates all dedup. If Form 4 needs a different hash shape, make a new hash function, don't modify the existing one.
5. **Migrations are immutable once applied.** Don't edit `0001_init.sql` or `0002_insiders.sql`. Add `0003_*.sql` for new tables/columns.
6. **Never push straight to main without a PR going forward.** The v0.1.x commits landed on main because the repo was being bootstrapped; feature work should go through PRs so CI + dependency-review + codeql gate it.

## Known follow-ups already tracked

These are low-priority items the final code review flagged (see session history; won't block Phase 1):

- `jobrun.Track` writes finalize errors to stdout via `fmt.Printf` instead of stderr (cosmetic)
- `WindowSizeMsg` is not propagated to watchlist/ticker/jobs screens (only Alerts) — so terminal resize mid-session doesn't reflow those screens
- `AlertsModel.SelectedBody()` is exported but unused — intended for a future detail pane
- Watchlist subqueries for alert/trade counts are O(N) correlated; fine for ≤50 tickers
- `list-alerts --since` uses Go's `time.ParseDuration` which doesn't understand `7d` (must use `168h`)

## Files to read first, in order

1. This doc
2. [`CLAUDE.md`](../CLAUDE.md) — the system rules for any agent working in this repo
3. [`docs/superpowers/specs/2026-04-23-market-digest-skeleton-design.md`](superpowers/specs/2026-04-23-market-digest-skeleton-design.md) — original spec
4. [`docs/superpowers/plans/2026-04-23-market-digest-skeleton.md`](superpowers/plans/2026-04-23-market-digest-skeleton.md) — the 22-task plan that's already been executed
5. [`internal/insiders/`](../internal/insiders/) — the existing Senate/House fetcher, to use as a template for the new Form 4 fetcher
6. [`jobs/cmd/fetch_insiders.go`](../jobs/cmd/fetch_insiders.go) — the orchestration pattern (load config → migrate → fetch → store → evaluate rules → record job_runs)
7. [`config/sources.yml`](../config/sources.yml) — where the new source will be configured
