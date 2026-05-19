# Mattermost environment profiles

## Files
- `.env.local.example` — local Podman/Colima test profile using `http://localhost:8065`
- `.env.production.example` — production VPS profile using `https://chat.yourdomain.com`

## Usage
```bash
cd deploy/mattermost
cp .env.local.example .env
# or
cp .env.production.example .env
```

Then edit secrets before startup, especially:
- `MM_PG_PASSWORD`
- `MM_DB_DSN`

## Compose variables
The compose stack consumes:
- `MM_SITE_URL`
- `MM_LISTEN_ADDRESS`
- `MM_PG_DB`
- `MM_PG_USER`
- `MM_PG_PASSWORD`
- `MM_DB_DSN`
- `CADDY_DOMAIN`
