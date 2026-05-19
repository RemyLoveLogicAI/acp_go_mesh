#!/usr/bin/env bash
set -euo pipefail

pass() { echo "[PASS] $*"; }
warn() { echo "[WARN] $*"; }
fail() { echo "[FAIL] $*"; }

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

check_file() {
  local p="$1"
  if [ -e "$p" ]; then pass "exists: $p"; else fail "missing: $p"; fi
}

check_service() {
  local name="$1"
  if systemctl --user is-enabled "$name" >/dev/null 2>&1; then pass "enabled: $name"; else warn "not enabled: $name"; fi
  if systemctl --user is-active "$name" >/dev/null 2>&1; then pass "active: $name"; else warn "not active: $name"; fi
}

echo '== files =='
check_file deploy/caddy/Caddyfile.mattermost
check_file deploy/mattermost/Caddyfile
check_file deploy/mattermost/compose.yml
check_file deploy/mattermost/.env.local.example
check_file deploy/mattermost/.env.production.example
check_file scripts/discover_mcp_paths.sh
check_file scripts/zo_warm_pool.sh
check_file scripts/orion_mcp_health_poll.sh
check_file scripts/check_mattermost_stack.sh

echo '== env =='
if [ -f deploy/mattermost/.env ]; then pass 'exists: deploy/mattermost/.env'; else warn 'missing: deploy/mattermost/.env'; fi

echo '== local http =='
if curl -fsSI http://localhost:8065 >/dev/null 2>&1; then pass 'mattermost reachable on localhost:8065'; else warn 'mattermost not reachable on localhost:8065'; fi

echo '== services =='
check_service "zo-warm-pool@$(whoami).service"
check_service "orion-mcp-health@$(whoami).service"

echo '== scripts executable =='
for s in scripts/discover_mcp_paths.sh scripts/zo_warm_pool.sh scripts/orion_mcp_health_poll.sh scripts/check_mattermost_stack.sh scripts/install_mattermost_stack.sh scripts/install_mattermost_stack_service.sh scripts/install_zo_warm_pool_service.sh scripts/install_orion_mcp_health_service.sh; do
  if [ -x "$s" ]; then pass "executable: $s"; else warn "not executable: $s"; fi
done

echo '== done =='
