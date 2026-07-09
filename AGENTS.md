# AGENTS.md — fog-next monorepo

## Repo structure

| Directory | What | Go module |
|-----------|------|-----------|
| `server/` | Network boot + imaging server (Go, single binary `fog`) | `github.com/ckAdmins/fog-next` |
| `agent/` | PXE initramfs imaging agent (Go, PID 1 in Alpine+OpenRC initramfs) | `github.com/ckAdmins/fog-next/agent` |
| `web/` | React/TS frontend (Bun + Vite + TanStack Router) | — |
| `pixie/` | Docker build environment for the agent initramfs + kernel | — |
| `docs/` | Developer guide, API reference, architecture, install | — |

The agent talks exclusively to the server's REST API (boot endpoints). No shared Go code between agent and server — they have separate `go.mod` files.

## Build

All targets are **mise tasks** — run `mise run <task>` (or `mise <task>` for shorthand).

| Task | What |
|------|------|
| `mise run server` | `go build` from `server/cmd/fog` → `build/fog` |
| `mise run web` | `bun install && bun run build` → copies to `server/internal/api/static/` |
| `mise run agent` | Build agent kernel + initramfs via pixie Docker → `pixie/output/` |
| `mise run server-docker` | `docker build -f server/deploy/docker/Dockerfile` → `fog-next:latest` |
| `mise run all` | web → server → server-docker → agent |
| `mise run clean` | Remove `build/`, `web/dist/`, `pixie/output/` |
| `mise run test` | `go test -race` from `server/` |
| `mise run lint` | `golangci-lint run` from `server/` |
| `mise run run` | `go run ./cmd/fog serve` from `server/` |
| `mise run docker-up` | `docker compose up` postgres + fog-next |
| `mise run docker-down` | `docker compose down` |
| `mise run fetch-ipxe` | Download iPXE boot files to `/tftpboot` |

## Toolchain

Root `mise.toml`: `bun = "latest"`, `go = "latest"`. Run `mise install` at the repo root.

- Server and agent both build with Go. The server Dockerfile uses `golang:1.26-alpine`.
- Frontend uses Bun. Run `cd web && bun run dev` for HMR during development.
- The pixie Docker build uses an Alpine container with `go` + `git` installed.

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
# Server Go tests (requires PostgreSQL)
mise run test      # runs from server/ with -race -timeout 120s

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

Migrations live in `server/internal/database/migrations/` and use `golang-migrate`.

## CLI

```
fog serve              Start the HTTP server and all background services
fog install            Interactive first-run setup wizard
fog migrate up/down    Apply or rollback database migrations
fog fetch-kernels      Download the agent kernel (bzImage) + initramfs (init.xz)
fog version            Print version
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
mise run docker-up     # starts postgres:18 + fog-next
mise run docker-down   # stops both
```

Config volume: `server/deploy/docker/config.yaml` (create from `server/deploy/config.example.yaml`).
