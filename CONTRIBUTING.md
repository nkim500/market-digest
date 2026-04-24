# Contributing

Thanks for looking at market-digest.

## Philosophy

This is a personal, fork-friendly tool. The intent is for others to fork and adapt, not necessarily to contribute upstream — though PRs are welcome for bug fixes, new data sources, new modes, and TUI polish.

## Before you open a PR

- Run `go test -race ./...` and `go vet ./...`.
- If you added a new data source, update `docs/adding-a-data-source.md`.
- If you added a new mode, update `docs/adding-a-mode.md` and `.claude/skills/market-digest/SKILL.md`.
- Don't commit real API keys, personal watchlists, or real email addresses in example configs.
- PRs that add trading-advice output (vs. observations) will be declined — see `CLAUDE.md` §"Ethical Framing".

## Commit style

Conventional commits (`feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`) are preferred because `release-please` reads them to generate CHANGELOG entries automatically.

## Running the smoke test

```bash
./scripts/smoke.sh
```

Exercises build + test + migrate + real fetch against Senate Stock Watcher + counts.

## License

By contributing you agree your work is licensed under the repo's LICENSE.
