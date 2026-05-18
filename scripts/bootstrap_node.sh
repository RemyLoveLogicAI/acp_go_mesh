#!/bin/bash
# LoveLogicAI Pi Mesh Node Bootstrap (Dockerless)
# Usage: ./bootstrap_node.sh [node_type]

set -euo pipefail

NODE_TYPE="${1:-worker}"
HOSTNAME=$(hostname -s)
TAILSCALE_IP=$(tailscale ip -4 2>/dev/null || echo "unknown")

echo "Bootstrapping LoveLogicAI Pi Mesh node..."
echo "   Node type: $NODE_TYPE"
echo "   Hostname: $HOSTNAME"
echo "   Tailscale IP: $TAILSCALE_IP"

command -v brew >/dev/null 2>&1 || { echo "Homebrew required"; exit 1; }
command -v git >/dev/null 2>&1 || { echo "git required"; exit 1; }

echo "Installing core services..."
brew list node >/dev/null 2>&1 || brew install node
brew list python@3.14 >/dev/null 2>&1 || brew install python@3.14
brew list nats-server >/dev/null 2>&1 || brew install nats-server
brew list postgresql@16 >/dev/null 2>&1 || brew install postgresql@16
brew list ollama >/dev/null 2>&1 || brew install ollama

echo "Installing Pi packages..."
npm install -g pi-coding-agent 2>/dev/null || true
pi install npm:@ollama/pi-web-search 2>/dev/null || true
pi install npm:pi-subagents 2>/dev/null || true
pi install npm:pi-mcp-adapter 2>/dev/null || true
pi install npm:pi-web-access 2>/dev/null || true
pi install npm:@runfusion/fusion 2>/dev/null || true
pi install npm:glimpseui 2>/dev/null || true
pi install npm:pi-agent-flow 2>/dev/null || true
pi install npm:pi-crew 2>/dev/null || true
pi install npm:pi-studio 2>/dev/null || true
pi install npm:@companion-ai/feynman 2>/dev/null || true
pi install npm:@llblab/pi-telegram 2>/dev/null || true
pi install npm:pi-intercom 2>/dev/null || true
pi install npm:@howaboua/pi-codex-conversion 2>/dev/null || true

MESH_DIR="/Users/lovelogic/.config/lovelogic/pi-mesh"
mkdir -p "$MESH_DIR"

echo "Registering native services..."
if [ "$NODE_TYPE" == "control" ]; then
    brew services start nats-server
    brew services start postgresql@16
    brew services start ollama
fi
brew services start nats-server
brew services start ollama

if [ ! -f "$HOME/.ssh/id_ed25519" ]; then
    echo "Generating SSH key for mesh sync..."
    ssh-keygen -t ed25519 -C "pi-mesh@$HOSTNAME" -f "$HOME/.ssh/id_ed25519" -N ""
    echo "Add this public key to your git forge:"
    cat "$HOME/.ssh/id_ed25519.pub"
fi

echo ""
echo "Bootstrap complete for $HOSTNAME ($NODE_TYPE)"
echo "Services:"
brew services list | grep -E "nats|postgresql|ollama" || true
echo ""
echo "Next steps:"
echo "  1. Configure ~/.config/lovelogic/pi-mesh/fusion_nodes.json"
echo "  2. Start Fusion: npx fusion daemon"
echo "  3. Open war room: npx glimpseui ~/.config/lovelogic/pi-mesh/war-room.html"
echo "  4. Check mesh: curl http://localhost:8000/topology"
