#!/usr/bin/env bash
set -euo pipefail
SERVICE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
mkdir -p "$SERVICE_DIR"
cp systemd/zo-warm-pool@.service "$SERVICE_DIR/zo-warm-pool@.service"
systemctl --user daemon-reload
systemctl --user enable --now "zo-warm-pool@$(whoami).service"
systemctl --user status --no-pager "zo-warm-pool@$(whoami).service" || true