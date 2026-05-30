# dev.sh — Unified Development Script

## Overview

`dev.sh` is a single script at the repo root providing quick-iteration workflows
for both the server+web stack and the agent+OS stack.

Three subcommands:

| Command             | What it does |
|---------------------|-------------|
| `./dev.sh server`   | Postgres → config → migrations → air (Go hot reload) + vite (web HMR) in one terminal. Ctrl+C cleans up. |
| `./dev.sh agent`    | Compile agent locally → repack initramfs from cached rootfs → boot in QEMU with graphical window. `--full` flag rebuilds rootfs via Docker first. |
| `./dev.sh all`      | Runs server in background, then boots agent QEMU in foreground. |

## File Structure

```
repo-root/
  dev.sh                     # The script (POSIX shell, standalone)
  .dev/                      # Generated at runtime, gitignored
    air.toml                 # Air config for Go hot reload
    fos-agent                # Cross-compiled agent binary
    init.xz                  # Fast-repacked initramfs
    rootfs.tar.gz            # Cached rootfs from full pixie build
    dev-disk.raw             # 10 GiB raw disk image for QEMU
  server/
    config.dev.yaml          # Generated dev server config (in server/ so air finds it)
```

`.dev/` is in `.gitignore`. The script creates it on first run.

## Dependencies

| Tool         | How acquired                                        |
|--------------|-----------------------------------------------------|
| go           | `mise install` (from root `mise.toml`)             |
| bun          | `mise install` (from root `mise.toml`)             |
| docker       | Must be pre-installed. Checked with `command -v`.   |
| qemu-system-x86_64 | Must be pre-installed. Checked with `command -v`. |
| air          | `go install github.com/air-verse/air@latest` on first `dev.sh server` run |

The script runs `mise install` at the top to ensure go and bun are present.
If mise is not installed it warns but continues (the user may have go/bun on PATH).

## Bootstrap (shared)

Every invocation starts with:

1. Determine `ROOT` from script location.
2. Create `.dev/` directory.
3. Run `mise install` (warn if mise not found, continue if go/bun on PATH).
4. Source any environment from `mise` if available.

## `dev.sh server`

### Step-by-step

1. **Check prerequisites**: docker, go, bun.
2. **Start Postgres**:
   ```bash
   docker rm -f fog-dev-db 2>/dev/null || true
   docker run -d --name fog-dev-db \
     --mount source=fog-dev-data,target=/var/lib/postgresql/data \
     -e POSTGRES_DB=fog \
     -e POSTGRES_USER=fog \
     -e POSTGRES_PASSWORD=fog \
     -p 5432:5432 \
     postgres:18-alpine
   ```
   Wait until `docker exec fog-dev-db pg_isready -U fog` succeeds (timeout 30s).

3. **Generate dev config** (`server/config.dev.yaml`) if it does not exist:
   ```yaml
   server:
     http: ":8080"
     base_url: "http://localhost:8080"
   database:
     host: "localhost"
     port: 5432
     name: "fog"
     user: "fog"
     password: "fog"
     sslmode: "disable"
   auth:
     jwt_secret: "dev-secret"
   storage:
     base_path: "/tmp/fog-dev/images"
     snapin_path: "/tmp/fog-dev/snapins"
     kernel_path: "/tmp/fog-dev/kernels"
   tftp:
     enabled: false
   log:
     level: "debug"
     format: "text"
   fos:
     release_url: ""
   ```
   If the file already exists, skip generation (preserve user customizations).

4. **Create storage dirs**: `mkdir -p /tmp/fog-dev/{images,snapins,kernels}`, `mkdir -p /tftpboot`.

5. **Run migrations**:
   ```bash
   cd server && FOG_CONFIG_FILE=config.dev.yaml go run ./cmd/fog migrate up
   ```

6. **Install air** if not on PATH:
   ```bash
   go install github.com/air-verse/air@latest
   ```

7. **Generate `.dev/air.toml`** if it does not exist:
   ```toml
   root = "server"
   tmp_dir = "/tmp"
   [build]
     cmd = "go build -o /tmp/fog-dev ./cmd/fog"
     bin = "/tmp/fog-dev serve -c config.dev.yaml"
     include_ext = ["go", "tmpl", "html"]
     exclude_dir = ["ent", "build"]
     delay = 500
     stop_on_error = true
   [log]
     time = true
   [color]
     main = "magenta"
     watcher = "cyan"
     build = "yellow"
     runner = "green"
   ```

8. **Start watchers as background children**:
   - PID 1: `air -c .dev/air.toml` (from repo root, watches `server/`)
   - PID 2: `cd web && bun run dev` (Vite HMR on port 5173, proxies /api to :8080)

   Each child's stdout/stderr is line-buffered and prefixed with `[server]` or `[web]`
   for readability via a simple `awk` or `sed` wrapper in a pipe.

9. **Trap SIGINT/SIGTERM**:
   - Kill both child processes (SIGTERM, then SIGKILL after 2s).
   - `docker stop fog-dev-db && docker rm fog-dev-db`.
   - Print "dev server stopped."

10. **Wait** for children to exit.

### Postgres data persistence

The named Docker volume `fog-dev-data` persists across script restarts. To wipe:
```bash
docker volume rm fog-dev-data
```

## `dev.sh agent`

### Step-by-step

1. **Check prerequisites**: go, docker, qemu-system-x86_64.

2. **Full build** (if `--full` flag or `.dev/rootfs.tar.gz` missing):
   ```bash
   ./build.sh agent
   ```
   This produces `pixie/output/rootfs.tar.gz` (see `pixie/build.sh` change below).
   Copy it to the cache:
   ```bash
   cp pixie/output/rootfs.tar.gz .dev/rootfs.tar.gz
   cp pixie/output/bzImage pixie/output/bzImage  # already in output/
   ```

   **Change to `pixie/build.sh`**: Add one line before the initramfs packing step
   (line ~134, before `(cd $ROOTFS ...`):
   ```bash
   tar -czf "$OUT/rootfs.tar.gz" -C $ROOTFS .
   ```
   This adds ~5s to the full build and writes a ~100 MB tarball to `pixie/output/`.

3. **Fast rebuild** (default):
   a. **Compile agent**:
      ```bash
      cd agent && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
        -ldflags "-s -w" \
        -o ../.dev/fos-agent ./cmd/fos-agent
      ```
   b. **Repack initramfs**:
      ```bash
      rm -rf /tmp/fog-dev-rootfs
      mkdir -p /tmp/fog-dev-rootfs
      tar -xzf .dev/rootfs.tar.gz -C /tmp/fog-dev-rootfs
      cp .dev/fos-agent /tmp/fog-dev-rootfs/bin/fos-agent
      cd /tmp/fog-dev-rootfs
      find . -print0 | cpio --null -ov --format=newc | xz -T0 -6 --check=crc32 > "$ROOT/.dev/init.xz"
      ```
   c. Total time: ~10-20s (vs 2-3 min for full Docker build).

4. **Ensure disk image**:
   ```bash
   if [ ! -f .dev/dev-disk.raw ]; then
     qemu-img create -f raw .dev/dev-disk.raw 10G
   fi
   ```

5. **Boot QEMU** (graphical window):
   ```bash
   qemu-system-x86_64 \
     -name "fog-agent-dev" \
     -machine q35,accel=kvm \
     -m 2048 \
     -smp 2 \
     -kernel pixie/output/bzImage \
     -initrd .dev/init.xz \
     -append "fog_server=http://10.0.2.2:8080 fog_action=deploy fog_host=dev-test console=ttyS0 console=tty0" \
     -drive file=.dev/dev-disk.raw,format=raw,if=virtio \
     -netdev user,id=net0 \
     -device virtio-net-pci,netdev=net0 \
     -device virtio-rng-pci \
     -vga virtio \
     -display gtk,gl=on \
     -usb -device usb-tablet
   ```

   - `-machine q35`: Modern PCIe chipset.
   - `-accel kvm`: Hardware acceleration if `/dev/kvm` is available.
   - `-smp 2`: Two CPU cores.
   - `-display gtk,gl=on`: Graphical GTK window with OpenGL.
   - `-vga virtio`: VirtIO GPU.
   - `console=ttyS0 console=tty0`: Kernel messages to both serial and VGA console.
   - Network: User-mode NAT, agent reaches host at `10.0.2.2`.
   - Disk: VirtIO block device, 10 GiB raw image.
   - USB tablet: Proper mouse cursor tracking.

   **Closing the VM**: Close the QEMU window or press Ctrl+C in terminal.
   The script exits when QEMU exits.

### Testing the agent

For the agent to successfully complete a deploy/capture, the dev server
(`./dev.sh server` in another terminal) must be running with:

- A host record with ID `dev-test` (or whatever `fog_host` is set to).
- An image assigned to that host with a pending deploy task.

The script prints a reminder before boot:
```
Agent will contact: http://10.0.2.2:8080
Host UUID: dev-test
Make sure the dev server is running and a task is queued for this host.
```

## `dev.sh all`

Starts both stacks. Extra flags (e.g. `--full`) are forwarded to the agent subcommand.

1. Launch `dev.sh server` as a background process, redirect output to `.dev/server.log`.
2. Wait for the server to be ready (poll `http://localhost:8080/api/health`).
3. Run `dev.sh agent "$@"` in the foreground (so `./dev.sh all --full` triggers a full agent rebuild).
4. On Ctrl+C (agent QEMU exit): kill the background server process and cleanup.

## Helper Functions

```sh
info()  { echo "  [dev] $*"; }
step()  { echo "==> $*"; }
die()   { echo "ERROR: $*" >&2; exit 1; }
warn()  { echo "WARN: $*" >&2; }

# Prefix each line of child stdout/stderr with a tag.
prefixed() {
  local tag="$1"
  shift
  "$@" 2>&1 | awk -v tag="$tag" '{printf "[%s] %s\n", tag, $0; fflush()}'
}

# Wait for a URL to return HTTP 2xx.
wait_for_http() {
  local url="$1" timeout="${2:-30}" elapsed=0
  while [ "$elapsed" -lt "$timeout" ]; do
    if curl -sf "$url" >/dev/null 2>&1; then return 0; fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
  return 1
}
```

## Edge Cases & Error Handling

- **mise not installed**: Warn, check for go/bun on PATH, die if missing.
- **Docker not running**: Die with "start Docker first".
- **QEMU not installed**: Die with "install qemu-system-x86_64".
- **Postgres already running** (leftover from crash): `docker rm -f fog-dev-db` before starting.
- **Port 5432/8080/5173 in use**: Die with message about what's occupying the port.
- **Air install fails**: Fall back to `go run` without hot reload, warn user.
- **Agent fast rebuild without cached rootfs**: Automatically do full build first.
- **KVM not available**: Still boot but slower (no HW accel), warn.
- **Script run from wrong directory**: `ROOT` is computed from script location, works from any CWD.

## .gitignore additions

```
.dev/
/tftpboot/
/tmp/fog-dev/
```
