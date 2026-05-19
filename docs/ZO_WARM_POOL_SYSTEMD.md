# Zo warm-pool systemd user service

## Install
```bash
bash scripts/install_zo_warm_pool_service.sh
```

## Check status
```bash
systemctl --user status zo-warm-pool@$(whoami).service
journalctl --user -u zo-warm-pool@$(whoami).service -f
```

## Behavior
- Probes TCP and SSH against `100.73.154.25:2288`.
- Logs warnings when the Zo worker is cold or unreachable.
- Intended insertion point for a future Modal wake hook.