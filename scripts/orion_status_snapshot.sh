#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${LOG_DIR:-$HOME/.orion/logs}"
mkdir -p "$LOG_DIR"

say_section() { echo; echo "== $1 =="; }
svc_state() {
  local unit="$1"
  if systemctl --user list-unit-files 2>/dev/null | grep -q "^${unit}\b"; then
    echo "present"
    systemctl --user is-enabled "$unit" 2>/dev/null || true
    systemctl --user is-active "$unit" 2>/dev/null || true
  else
    echo "absent"
  fi
}

say_section 'context'
date -Is
hostname || true
whoami
echo "repo=$ROOT_DIR"

say_section 'mattermost files'
for p in \
"$ROOT_DIR/deploy/mattermost/compose.yml" \
"$ROOT_DIR/deploy/mattermost/Caddyfile" \
"$ROOT_DIR/deploy/mattermost/.env" \
"$ROOT_DIR/deploy/mattermost/.env.local.example" \
"$ROOT_DIR/deploy/mattermost/.env.production.example"; do
  if [ -e "$p" ]; then echo "OK $p"; else echo "MISS $p"; fi
done

say_section 'mattermost local reachability'
curl -I --max-time 5 http://localhost:8065 2>/dev/null | head -n 5 || echo 'unreachable'

say_section 'mattermost containers'
if command -v podman >/dev/null 2>&1; then
  podman ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' 2>/dev/null | grep -Ei 'mattermost|caddy|postgres' || echo 'no matching containers'
else
  echo 'podman not installed'
fi

say_section 'caddy configs'
for p in "$ROOT_DIR/deploy/caddy/Caddyfile.mattermost" "$ROOT_DIR/deploy/mattermost/Caddyfile"; do
  if [ -f "$p" ]; then echo "OK $p"; else echo "MISS $p"; fi
done

say_section 'zo service'
svc_state "zo-warm-pool@$(whoami).service"
tail -n 10 "$HOME/.orion/logs/zo-warm-pool.log" 2>/dev/null || echo 'no zo warm log yet'

say_section 'mcp services'
svc_state "orion-mcp-health@$(whoami).service"
svc_state "orion-mcp-health.timer"
tail -n 10 "$HOME/.orion/logs/orion-mcp-health.log" 2>/dev/null || echo 'no mcp health log yet'

say_section 'discovery commands'
command -v mcp-super-server 2>/dev/null || echo 'mcp-super-server not on PATH'
command -v zo-pika-self-agent 2>/dev/null || echo 'zo-pika-self-agent not on PATH'

say_section 'ports'
if command -v lsof >/dev/null 2>&1; then
  lsof -nP -iTCP:8065 -sTCP:LISTEN 2>/dev/null || echo 'nothing listening on 8065'
  lsof -nP -iTCP:2288 2>/dev/null || echo 'nothing visible on 2288 locally'
else
  echo 'lsof not installed'
fi