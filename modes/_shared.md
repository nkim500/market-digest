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
