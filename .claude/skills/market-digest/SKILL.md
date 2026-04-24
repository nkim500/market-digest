---
name: market-digest
description: Use when the user types /digest <mode> in this repo, or asks for insider-trade / watchlist / market-ideation synthesis. Routes to a markdown mode file under modes/, loads shared context first, runs fetch jobs as needed, and produces a report. Read CLAUDE.md and modes/_shared.md before any mode.
---

# market-digest — Mode dispatcher

## What this repo is

A personal, fork-friendly equity-ideation tool. Go runtime (`./bin/jobs`, `./bin/dashboard`) does deterministic work (fetching, storing, alerting). Claude (via this skill and `modes/*.md`) does synthesis.

## How to invoke a mode

When the user says `/digest <name>`, `/market-digest <name>`, or asks for one of:
- "what's new on insider trades", "refresh insiders", "anything worth acting on" → **insiders**
- "triage my alerts", "what should I dismiss" → **alerts**
- "momentum", "sector", "macro" → tell the user the mode is stubbed and point at `modes/<name>.md`

## Required preamble for every mode

1. Read `CLAUDE.md` (repo root) — shared rules, ethical framing, plan-board protocol.
2. Read `modes/_shared.md` — data access conventions.
3. Read `modes/_profile.md` if it exists (user overrides).
4. Read the specific `modes/<name>.md`.
5. Run the mode workflow as a subagent when the mode file recommends it.

## Plan-board protocol

If `implementation_plan.md` exists at the repo root, follow the protocol in `CLAUDE.md` §"Agent Plan-Board Protocol":
- Mark the phase you're working on `[IN PROGRESS — <session> — <ts>]` before starting.
- Update to `[DONE — <PR or local>]` after finishing.
- `[BLOCKED — ...]` if blocked.

## Useful commands

```bash
./bin/jobs --help              # list all subcommands
./bin/jobs doctor              # check DB, config, source reachability
./bin/jobs fetch-insiders      # refresh Senate + House trade data
./bin/jobs list-alerts --unseen --since 168h
./bin/jobs mark-seen <id>
./bin/dashboard                # open the TUI
```

## Ethical framing

This skill produces *observations*, not *advice*. Every report ends with the disclaimer from `CLAUDE.md`. All sources are public.
