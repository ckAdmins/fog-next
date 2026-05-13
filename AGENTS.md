# AGENTS.md — imaging monorepo

## Repo structure

| Directory | What | Go module |
|-----------|------|-----------|
| `server/` | Network boot + imaging server (Go, single binary `fog`) | `github.com/nemvince/fog-next` |
| `agent/` | PXE initramfs agent (Go, PID 1 in Buildroot initramfs) | `github.com/nemvince/fos-next` |
| `web/` | React/TS frontend (Bun + Vite + TanStack Router) | — |
| `pixie/` | Docker build environment for the agent initramfs + kernel | — |
| `docs/` | Developer guide, API reference, architecture, install | — |

The agent talks exclusively to the server's REST API (boot endpoints). No shared code.

## Build

Single entry point: **`./build.sh`** (also `make` for server-only workflows).

| Command | What |
|---------|------|
| `./build.sh agent` | Build agent kernel + initramfs via pixie Docker → `pixie/output/` |
| `./build.sh server` | `go build` from `server/` → `build/fog` |
| `./build.sh server-docker` | `docker build -f server/deploy/docker/Dockerfile` → `fog-next:latest` |
| `./build.sh web` | `bun install && bun run build` → copies to `server/internal/api/static/` |
| `./build.sh all` | web → server → server-docker → agent |
| `./build.sh clean` | Remove `build/`, `web/dist/`, `pixie/output/` |

Equivalent make targets: `make build`, `make run`, `make test`, `make lint`, `make docker-build`, `make docker-up`.

## Toolchain

Root `mise.toml`: `bun = "latest"`, `go = "latest"`. Run `mise install` at the repo root.

- Server and agent both build with Go. The server Dockerfile uses `golang:1.26-alpine`.
- Frontend uses Bun. Run `cd web && bun run dev` for HMR during development.
- The pixie Docker build uses an Alpine container with `go` installed.

## Dev workflow (server)

```bash
# Terminal 1 — Go backend
FOG_DATABASE_HOST=localhost FOG_DATABASE_USER=fog \
FOG_DATABASE_PASSWORD=fog FOG_DATABASE_NAME=fog \
FOG_AUTH_JWT_SECRET=dev-secret FOG_SERVER_HTTP=:8080 \
go run ./cmd/fog serve -c config.yaml
# ^ from server/ directory

# Terminal 2 — Vite dev server (proxies /api/* → :8080)
cd web && bun run dev
```

## Tests

```bash
# Server Go tests
make test          # runs from server/ with -race -timeout 120s

# Frontend E2E (requires PostgreSQL + running server)
cd web && bunx playwright install --with-deps
cd web && bunx playwright test

# Agent has no automated tests — test via full-stack QEMU VM.
```

## Migrations

```bash
./build/fog migrate up      # apply pending
./build/fog migrate down    # rollback latest
./build/fog migrate status  # current version
```

## Agent/imaging gotchas

These are hard-earned — the agent was ported from legacy FOS and had many subtle bugs.

### Partition numbers are NOT always consecutive

Cloud images (Debian, Ubuntu) often have partitions `[1, 14, 15]` not `[1, 2, 3]`. Always use **actual kernel partition numbers** from `disk.PartitionNumber(dev)`, never assume `i+1` is the kernel number.

- **Capture**: store actual partition numbers in metadata (`PartNumbers []int`).
- **Deploy**: use `partNumFromMeta(resp.PartNumbers, idx)` to compute device paths. The sequential index `part` (1..PartCount) is only for file slot naming.
- **FixedSizePartitions** stores actual kernel partition numbers. On deploy, look up with `fixedSet[num]` (actual number), not `fixedSet[part]` (sequential index).

### partclone flag format

Use **short flags only**. Long flags (`--source`, `--output`) are silently ignored or parsed as positional args.

- **Restore**: `partclone.ext4 -r -s - -O /dev/sda1 -F -f 1`
- **Clone**: `partclone.ext4 -c -s /dev/sda1 -o - -F -f 1`

`-O` combines output+overwrite (matching legacy `partclone.restore -O ${target}`). `-o` alone does not overwrite.

### Deploy: partclone binary selection

On deploy the partition is empty before restore, so `detectFilesystem` returns `"dd"`. **Must use `resp.PartTypes[partN]` (from capture metadata) as the `fs` argument to `imaging.Restore()`**, not the result of local detection. Only fall back to local detection if `PartTypes` is empty or `"?"`.

### Filesystem detection

`detectFilesystem` uses `blkid -po udev <dev>` (probe mode, reads directly from disk bypassing cache) as the first method — matches legacy `fsTypeSetting`. After partclone writes data, call `probeFilesystem(dev)` which does `blockdev --flushbufs <dev>` before probing.

### GPT restore must batch UUID operations

After `sgdisk --zap-all` + `sgdisk --load-backup`, the `restoreGPTUUIDs` function collects all `-U`, `-t`, `-u` flags and runs them in a **single sgdisk invocation**. Running 7 separate sgdisk calls causes kernel re-read races → partition devices show 0 bytes → partclone fails with "Target partition size(0 MB)".

### Partition device readiness

After partition table restore, `disk.WaitForPartitions(disk, partNumbers, 10*time.Second)` polls `blockdev --getsize64 <dev>` until every partition reports non-zero. Without this, partclone may open the device before the kernel finishes sizing it.

### Capture shrink is unconditional

Shrink happens regardless of `resp.ImageType`. The `ImageType` field ("fixed" vs "resizable") only controls whether deploy expands to fill the target disk. Default capture produces `"resizable"` unless `resp.ImageType == "fixed"`.

### Agent kernel cmdline

The iPXE script passes kernel parameters: `fog_server=<url> fog_action=<capture|deploy> fog_host=<uuid>`. The agent parses these via `/proc/cmdline`. `fog_server` must be the server's externally reachable URL (not localhost).

## No COMPAT.md

There is no version-compatibility file. The agent and server are built and released together from this repo. The agent discovers its action via the boot handshake — the server dictates whether to capture or deploy.

## Docker Compose (server)

```bash
make docker-up     # starts postgres:16 + fog-next
make docker-down   # stops both
```

Config volume: `server/deploy/docker/config.yaml` (create from `server/deploy/config.example.yaml`).
