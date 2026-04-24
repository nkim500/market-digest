#!/usr/bin/env bash
# Install market-digest launchd agents. Substitutes the placeholder path
# in each *.plist under scripts/launchd/ with the absolute path of this
# repo and loads the result into ~/Library/LaunchAgents.

set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$HOME/Library/LaunchAgents"
mkdir -p "$DEST"

for plist in "$REPO"/scripts/launchd/*.plist; do
  name="$(basename "$plist")"
  target="$DEST/$name"
  sed "s|__DIGEST_HOME__|$REPO|g" "$plist" > "$target"
  launchctl unload "$target" >/dev/null 2>&1 || true
  launchctl load "$target"
  echo "loaded: $target"
done

echo "Done. Check logs in /tmp/market-digest-*.log"
