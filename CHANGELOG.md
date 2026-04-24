# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1](https://github.com/nkim500/market-digest/compare/v0.1.0...v0.1.1) (2026-04-24)


### Bug Fixes

* **ci:** idempotent SBOM release upload ([2cd66df](https://github.com/nkim500/market-digest/commit/2cd66df23851dc7c2db9442f8a2aff321f8a2fb6))

## [0.1.0] - 2026-04-23

### Added
- Initial skeleton: Go TUI dashboard + jobs binary + SQLite + Claude modes layer
- Wired `insiders` mode end-to-end: fetch from Senate Stock Watcher, dedup, write alerts
- Stub modes: `momentum`, `sector`, `macro`
- Four dashboard screens: Alerts, Watchlist, Ticker detail, Jobs
- CLI subcommands: `migrate`, `doctor`, `fetch-insiders`, `list-alerts`, `mark-seen`, plus stubs
- macOS launchd + Linux crontab examples for scheduling
- Design spec and implementation plan under `docs/superpowers/`
- Claude Code skill at `.claude/skills/market-digest/`
