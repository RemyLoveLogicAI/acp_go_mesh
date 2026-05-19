#!/usr/bin/env bash
set -euo pipefail
PROFILE="${1:-local}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_DIR="$ROOT/deploy/mattermost"
cd "$ROOT"

case "$PROFILE" in
  local) SRC="$ENV_DIR/.env.local.example" ;;
  production) SRC="$ENV_DIR/.env.production.example" ;;
  *) echo "Usage: $0 [local|production]"; exit 1 ;;
esac

if [ ! -f "$ENV_DIR/.env" ]; then
  cp "$SRC" "$ENV_DIR/.env"
  echo "Created $ENV_DIR/.env from $SRC"
else
  echo "Keeping existing $ENV_DIR/.env"
fi

bash scripts/install_zo_warm_pool_service.sh || true
bash scripts/install_orion_mcp_health_service.sh || true
bash scripts/install_mattermost_stack_service.sh || true
bash scripts/orion_ops_smoke.sh || true
