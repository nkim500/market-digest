# Mode: macro — Macroeconomic Digest (STUB)

**Status:** not implemented.

## Design intent

Pull macro indicators from FRED (St. Louis Fed) — e.g., 10Y yield, 2Y yield, DXY, CPI YoY, unemployment — and narrate what they imply for equity positioning.

## What's needed

1. FRED API key (free registration) stored in `FRED_API_KEY` env var (never committed).
2. `jobs fetch-macro` pulling a curated set of series to a `macro_series` table.
3. `modes/macro.md` workflow that reads the most recent observations and produces a short brief with sections for rates, dollar, inflation, labor.

## Why it's stubbed

FRED integration + macro rubric is its own spec → plan cycle.
