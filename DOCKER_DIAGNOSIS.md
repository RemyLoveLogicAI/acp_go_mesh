# Docker Desktop Virtualization Error - Diagnosis Report

**Date:** 2026-05-18
**System:** macOS Tahoe, Mac Mini M4
**Docker Version:** 29.5.0

## Error Summary
Docker Desktop encountered an unexpected error:
```
com.docker.virtualization: process terminated unexpectedly: use of closed network connection
```

## Root Cause Analysis

### 1. Settings File Issue (Initial)
- **Problem:** Extended attributes on `settings-store.json` prevented Docker from reading configuration
- **Location:** `/Users/lovelogic/Library/Group Containers/group.com.docker/settings-store.json`
- **Fix Applied:** Cleared extended attributes with `xattr -c`

### 2. VM Crash (Primary Issue)
Docker VM starts but immediately crashes with:
```
VM has stopped: Internal Virtualization error. The virtual machine stopped unexpectedly.
```

**Error Details from virtualization.log:**
```
[com.docker.virtualization][E] sending fd: dialing ethernet-fd.sock: connect: no such file or directory
[com.docker.virtualization][W] NetworkHandler: dialing ethernet-fd.sock: connect: no such file or directory
[com.docker.virtualization] VM has stopped: Internal Virtualization error
```

### 3. Network Socket Missing
The VM crashes because required network socket files don't exist when the VM tries to connect:
- `ethernet-fd.sock` - Missing
- `vpnkit-bridge-fd.sock` - Connection issues

### 4. Backend Process Pattern
Backend starts → VM launches → Network sockets fail → VM crashes → Backend shuts down

## System State

### Virtualization Support
- ✅ macOS Hypervisor support enabled: `kern.hv_support: 1`
- ✅ Rosetta for Linux enabled
- ✅ VM configuration valid
- ✅ Disk images accessible

### Running Processes
- ❌ Docker Desktop backend: NOT RUNNING (crashes immediately)
- ✅ `com.docker.vmnetd`: Running (privileged helper)
- ✅ `docker-agent`: Running (AI agent service)

### Socket Status
```
/Users/lovelogic/.docker/run/
  - Empty directory (should contain docker.sock)
```

## Attempted Fixes

1. ✅ Cleared extended attributes on settings file
2. ✅ Cleared VM cache: `~/Library/Containers/com.docker.docker/Data/vms/*`
3. ✅ Restarted Docker Desktop multiple times
4. ⚠️ Network service restart (requires sudo - not attempted)

## Recommended Fix

The issue requires restarting the Docker networking privileged helper:

```bash
# Stop all Docker processes
killall -9 Docker "com.docker.backend" "com.docker.supervisor" "docker-agent"

# Restart the privileged network helper
sudo launchctl stop com.docker.vmnetd
sudo launchctl start com.docker.vmnetd

# Restart Docker Desktop
open -a "Docker"
```

Alternatively, use Docker Desktop's built-in factory reset:
1. Open Docker Desktop settings
2. Go to "Troubleshoot"
3. Click "Reset to factory defaults"

## Log Locations

- Main: `~/Library/Containers/com.docker.docker/Data/log/host/Docker.log`
- Backend: `~/Library/Containers/com.docker.docker/Data/log/host/com.docker.backend.log`
- Virtualization: `~/Library/Containers/com.docker.docker/Data/log/host/com.docker.virtualization.log`
- VM Console: `~/Library/Containers/com.docker.docker/Data/log/vm/console.log`
- Monitor: `~/Library/Containers/com.docker.docker/Data/log/host/monitor.log`

## Impact on Pi Mesh

Docker is not critical for the Pi Mesh infrastructure:
- ✅ NATS broker on Zo: Running independently
- ✅ Orion Control Plane: Native Go binary
- ✅ Fusion daemon: Native process
- ✅ Ollama: Running natively
- ⚠️ Optional: Some development containers may be unavailable

## Next Steps

1. User can manually restart Docker with factory reset via GUI
2. Or restart networking services with sudo access (requires password)
3. Docker is optional for Pi Mesh core functionality
