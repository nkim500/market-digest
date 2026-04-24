# Mode: sector — Industry & Sector Snapshot (STUB)

**Status:** not implemented.

## Design intent

Generate a sector-level synthesis (semis, energy, biotech, etc.) pulling from:
- Recent earnings call transcripts (company IR pages — free)
- Public SEC filings (10-Ks, 10-Qs via EDGAR JSON)
- Macro context from FRED (see modes/macro.md)
- Relevant alerts from `alerts` table

Output: `data/reports/sector-<name>-YYYY-MM-DD.md`.

## Why it's stubbed

Depends on price data (momentum) and macro data (macro mode) being wired up first.
