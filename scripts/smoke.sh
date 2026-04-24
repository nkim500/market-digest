#!/usr/bin/env bash
# End-to-end smoke. Builds, migrates, fetches (over httptest? no — against real
# sources), reports counts. Used to validate a fork/clone.

set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> go build ./..."
go build ./...

echo "==> go test ./..."
go test ./...

echo "==> make build"
make build

if [ ! -f config/profile.yml ]; then
  cp config/profile.example.yml config/profile.yml
fi
if [ ! -f config/watchlist.yml ]; then
  cp config/watchlist.example.yml config/watchlist.yml
fi

echo "==> jobs migrate"
./bin/jobs migrate

echo "==> jobs doctor"
./bin/jobs doctor || true

echo "==> jobs fetch-insiders (first run — may take a moment)"
./bin/jobs fetch-insiders

echo "==> counts"
sqlite3 data/digest.db "SELECT 'trades', COUNT(*) FROM insider_trades UNION ALL SELECT 'alerts', COUNT(*) FROM alerts"

echo "==> jobs list-alerts --since 168h (sample)"
./bin/jobs list-alerts --since 168h | head -20 || true

echo
echo "Smoke OK. Run ./bin/dashboard to open the TUI."
