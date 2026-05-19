#!/usr/bin/env bash
set -euo pipefail
SERVICE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
mkdir -p "$SERVICE_DIR"
cp systemd/orion-mcp-health@.service "$SERVICE_DIR/orion-mcp-health@.service"
systemctl --user daemon-reload
systemctl --user enable --now "orion-mcp-health@$(whoami).service"
systemctl --user status --no-pager "orion-mcp-health@$(whoami).service" || true
