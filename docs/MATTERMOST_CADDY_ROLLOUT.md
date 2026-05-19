# Mattermost + Caddy rollout for ORION Nexus v3.7

## Locked decisions
- Testing URL: `http://localhost:8065`
- Production URL: `https://chat.yourdomain.com`
- Reverse proxy: **Caddy**
- Runtime preference: Podman-first / Colima local validation

## Immediate blocker workarounds

### 1. Mattermost
- Validate local stack on port `8065`
- Keep production proxy config ready in `deploy/caddy/Caddyfile.mattermost`
- Confirm websockets survive proxying

### 2. MCP health check on Mini
Use `scripts/discover_mcp_paths.sh` to discover:
- binary/script path
- service file name
- working directory
- env/config file locations

### 3. Zo cold worker workaround
Use `scripts/zo_warm_pool.sh` as a lightweight guard loop to:
- keep a warm probe cadence
- detect `SSH :2288` failures early
- provide a deterministic insertion point for a Modal wake hook

## Acceptance criteria
- Mattermost local responds on `localhost:8065`
- Caddyfile present for production
- MCP services either found or explicitly reported absent
- Zo guard loop logs failures with timestamps