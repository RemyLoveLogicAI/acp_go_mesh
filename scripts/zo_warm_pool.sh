#!/usr/bin/env bash
set -euo pipefail

HOST="${1:-100.73.154.25}"
PORT="${2:-2288}"
INTERVAL="${INTERVAL:-120}"
SSH_TIMEOUT="${SSH_TIMEOUT:-5}"
LOG_DIR="${LOG_DIR:-$HOME/.orion/logs}"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/zo-warm-pool.log"

probe_tcp() {
  nc -z -w "$SSH_TIMEOUT" "$HOST" "$PORT" >/dev/null 2>&1
}

probe_ssh() {
  ssh -o BatchMode=yes -o StrictHostKeyChecking=no -o ConnectTimeout="$SSH_TIMEOUT" -p "$PORT" root@"$HOST" 'echo ready' >/dev/null 2>&1
}

wake_hint() {
  echo "[$(date -Is)] WARN ssh unreachable for $HOST:$PORT -- trigger modal warm-up hook here" | tee -a "$LOG_FILE"
}

while true; do
  if probe_tcp && probe_ssh; then
    echo "[$(date -Is)] OK zo warm and reachable at $HOST:$PORT" >> "$LOG_FILE"
  else
    wake_hint
  fi
  sleep "$INTERVAL"
done