# ORION Mattermost stack systemd user service

## Install
```bash
bash scripts/install_mattermost_stack_service.sh
```

## Status
```bash
systemctl --user status orion-mattermost-stack.service
journalctl --user -u orion-mattermost-stack.service -f
```

## Notes
- Service uses `podman compose` with fallback to `podman-compose`.
- Working directory assumes this repo lives at `~/Documents/01-Active-Projects/acp_go_mesh`.
- Adjust the unit if installed elsewhere.