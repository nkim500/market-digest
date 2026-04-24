# Mode: momentum — Watchlist Momentum Signals (STUB)

**Status:** not implemented. This file is a placeholder for the next real mode.

## What this mode should do (design intent)

Compute short-term momentum signals over the watchlist from EOD prices, and surface breakouts.

## What's needed to implement

1. `jobs fetch-prices` command that pulls EOD OHLC from a free source (Stooq or Tiingo free tier; 1000 req/day is enough for ≤50 tickers).
2. `0003_prices.sql` migration adding a `prices` table keyed on (ticker, date).
3. `jobs compute-momentum` that runs on new price rows and writes alerts when:
   - 5d-return crosses the threshold stored in `config/profile.yml:reporting.momentum.*`
   - Volume anomaly (today's volume > 2× 20d average)
4. Mode workflow mirroring `modes/insiders.md`.

## Why it's stubbed in v1

To keep the skeleton small and validate the insiders-mode pattern before cloning it.
