# ORION blocker workaround status pack

## Generated artifacts
- `deploy/caddy/Caddyfile.mattermost`
- `scripts/discover_mcp_paths.sh`
- `scripts/zo_warm_pool.sh`
- `docs/MATTERMOST_CADDY_ROLLOUT.md`

## Operator sequence
1. Run MCP discovery on Mini
2. Validate Mattermost at `localhost:8065`
3. Apply Caddyfile on VPS or proxy host
4. Start Zo warm-pool guard loop or convert it to systemd

## Suggested commands
```bash
bash scripts/discover_mcp_paths.sh
INTERVAL=120 bash scripts/zo_warm_pool.sh 100.73.154.25 2288
curl -I http://localhost:8065
```