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
