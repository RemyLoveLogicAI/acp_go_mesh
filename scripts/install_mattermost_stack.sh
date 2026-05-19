#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/deploy/mattermost"
if [ ! -f .env ]; then
  echo 'Missing deploy/mattermost/.env. Copy from .env.local.example or .env.production.example first.' >&2
  exit 1
fi
podman compose up -d
