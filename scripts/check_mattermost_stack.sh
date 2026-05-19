#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/deploy/mattermost"
echo '== podman compose ps =='
podman compose ps || true
echo '== local mattermost =='
curl -fsSI http://127.0.0.1:8065 || true
echo '== local caddy =='
curl -kfsSI https://127.0.0.1 || true
