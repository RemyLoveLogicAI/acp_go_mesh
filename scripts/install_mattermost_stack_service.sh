#!/usr/bin/env bash
set -euo pipefail
SERVICE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
mkdir -p "$SERVICE_DIR"
cp systemd/orion-mattermost-stack.service "$SERVICE_DIR/orion-mattermost-stack.service"
systemctl --user daemon-reload
systemctl --user enable --now orion-mattermost-stack.service
systemctl --user status --no-pager orion-mattermost-stack.service || true
