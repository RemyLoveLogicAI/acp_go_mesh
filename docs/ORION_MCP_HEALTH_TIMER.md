# ORION MCP Health Timer

The timer-driven variant avoids a long-running sleep loop by launching the MCP health poller on a schedule.

## Install
```bash
bash scripts/install_orion_mcp_health_timer.sh
```

## Behavior
- Starts 2 minutes after user session boot
- Re-runs every 15 minutes
- Uses `orion-mcp-health@%u.service`
- Appends output to `~/.orion/logs/orion-mcp-health.log` via the poller script

## Check status
```bash
systemctl --user status orion-mcp-health.timer
systemctl --user list-timers | grep orion-mcp-health
journalctl --user -u orion-mcp-health@$(whoami).service -f
```