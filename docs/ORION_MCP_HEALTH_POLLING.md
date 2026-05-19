# ORION MCP health polling

## Install
```bash
bash scripts/install_orion_mcp_health_service.sh
```

## Status
```bash
systemctl --user status orion-mcp-health@$(whoami).service
journalctl --user -u orion-mcp-health@$(whoami).service -f
tail -f ~/.orion/logs/orion-mcp-health.log
```

## Behavior
- reruns `scripts/discover_mcp_paths.sh` every 900 seconds by default
- appends results to `~/.orion/logs/orion-mcp-health.log`
- intended as a lightweight readiness/visibility loop until exact install paths stabilize
