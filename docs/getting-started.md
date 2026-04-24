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
