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
