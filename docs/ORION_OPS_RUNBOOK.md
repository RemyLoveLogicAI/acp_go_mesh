# ORION Ops Runbook

This runbook covers the no-wait operational workflow for Mattermost, Zo warm-pool health, and MCP discovery/polling.

## Components
- Mattermost stack: `deploy/mattermost/compose.yml`
- Production proxy: `deploy/mattermost/Caddyfile` and `deploy/caddy/Caddyfile.mattermost`
- Zo warm-pool guard: `scripts/zo_warm_pool.sh`
- MCP discovery: `scripts/discover_mcp_paths.sh`
- MCP poller: `scripts/orion_mcp_health_poll.sh`
- Unified smoke test: `scripts/orion_ops_smoke.sh`
- Status snapshot: `scripts/orion_status_snapshot.sh`

## First-time bootstrap
```bash
bash scripts/bootstrap_orion_ops.sh local
# or
bash scripts/bootstrap_orion_ops.sh production
```

## Mattermost workflow
1. Choose env profile:
```bash
cd deploy/mattermost
cp .env.local.example .env
# or cp .env.production.example .env
```
2. Start the stack:
```bash
bash scripts/install_mattermost_stack.sh
```
3. Validate:
```bash
bash scripts/check_mattermost_stack.sh
curl -I http://localhost:8065
```
4. Optional user service:
```bash
bash scripts/install_mattermost_stack_service.sh
```

## Zo workflow
Install the persistent user service:
```bash
bash scripts/install_zo_warm_pool_service.sh
```
Inspect service/logs:
```bash
systemctl --user status zo-warm-pool@$(whoami).service
journalctl --user -u zo-warm-pool@$(whoami).service -f
tail -f ~/.orion/logs/zo-warm-pool.log
```

## MCP workflow
One-shot discovery:
```bash
bash scripts/discover_mcp_paths.sh
```
Install timer-driven polling:
```bash
bash scripts/install_orion_mcp_health_timer.sh
```
Check poller status:
```bash
systemctl --user status orion-mcp-health.timer
journalctl --user -u orion-mcp-health@$(whoami).service -f
tail -f ~/.orion/logs/orion-mcp-health.log
```

## Unified verification
```bash
bash scripts/orion_ops_smoke.sh
bash scripts/orion_status_snapshot.sh
```

## Troubleshooting quick hits
- `localhost:8065` unreachable: start Mattermost stack and inspect Podman logs.
- Zo service flaps: inspect `~/.orion/logs/zo-warm-pool.log` and add actual wake hook.
- MCP tools missing: run `discover_mcp_paths.sh` and inspect systemd unit definitions.
- TLS not issuing: confirm DNS for `chat.yourdomain.com` and inbound 80/443 reachability.

## Deferred items
- Real cloud wake integration for Zo worker
- Confirmed Mini MCP installation paths
- Branding asset consolidation after locating source repo