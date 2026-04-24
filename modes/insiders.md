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
