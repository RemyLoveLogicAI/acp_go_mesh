# ORION Mattermost production stack

## Files
- `compose.yml` — Mattermost, Postgres, and Caddy
- `Caddyfile` — production reverse proxy for `chat.yourdomain.com`
- `.env.example` — required environment variables and secrets template

## First-time setup
```bash
cd deploy/mattermost
cp .env.example .env
# edit .env and replace placeholder secrets
podman compose up -d
```

## Validation
```bash
bash scripts/check_mattermost_stack.sh
```

## Optional user service
```bash
bash scripts/install_mattermost_stack_service.sh
```

## Security notes
- Never commit real `.env` secrets.
- Replace the placeholder DB password before non-local use.
- Consider moving secrets to a secrets manager later.