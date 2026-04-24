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
