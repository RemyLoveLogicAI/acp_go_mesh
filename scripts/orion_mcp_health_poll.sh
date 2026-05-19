#!/usr/bin/env bash
set -euo pipefail
INTERVAL="${INTERVAL:-900}"
LOG_DIR="${LOG_DIR:-$HOME/.orion/logs}"
LOG_FILE="$LOG_DIR/orion-mcp-health.log"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$LOG_DIR"
cd "$ROOT"
while true; do
  {
    echo "[$(date -Is)] begin mcp health poll"
    bash scripts/discover_mcp_paths.sh
    echo "[$(date -Is)] end mcp health poll"
  } >> "$LOG_FILE" 2>&1 || true
  sleep "$INTERVAL"
done
