#!/usr/bin/env bash
set -euo pipefail

echo '== whoami =='
whoami
echo '== hostname =='
hostname || true
echo '== date =='
date

echo '== which =='
which mcp-super-server 2>/dev/null || true
which zo-pika-self-agent 2>/dev/null || true

echo '== command -v =='
command -v mcp-super-server 2>/dev/null || true
command -v zo-pika-self-agent 2>/dev/null || true

echo '== filesystem matches =='
find "$HOME" /opt /usr/local /etc/systemd /lib/systemd /usr/lib/systemd -maxdepth 5 \
  \( -path '*/.git' -o -path '*/node_modules' -o -path '*/.venv' -o -path '*/venv' \) -prune -o \
  \( -iname '*mcp*super*server*' -o -iname '*zo*pika*self*agent*' \) -print 2>/dev/null | sort -u

echo '== systemd units =='
(systemctl list-unit-files --type=service 2>/dev/null || true) | grep -Ei 'mcp|pika|zo' || true

echo '== unit details =='
for u in $((systemctl list-unit-files --type=service 2>/dev/null || true) | awk '{print $1}' | grep -Ei 'mcp|pika|zo'); do
  echo "-- $u --"
  systemctl cat "$u" 2>/dev/null || true
  systemctl show "$u" -p FragmentPath -p WorkingDirectory -p EnvironmentFiles -p ExecStart -p User 2>/dev/null || true
done

echo '== process matches =='
ps aux | grep -Ei 'mcp-super-server|zo-pika-self-agent' | grep -v grep || true