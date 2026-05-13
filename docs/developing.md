# Developing FOG Next

Deep-dive developer guide. For build commands and gotchas see [AGENTS.md](../AGENTS.md).

---

## Prerequisites

| Tool | Minimum version | Install hint |
|------|----------------|--------------|
| Go | 1.25 | `mise install` |
| Bun | 1.x | `mise install` |
| PostgreSQL | 15 | Docker is easiest (see below) |
| golangci-lint | 1.57 | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| Playwright browsers | — | `cd web && bunx playwright install --with-deps` |

---

## Spinning up a development database

```bash
docker run -d \
  --name fog-postgres \
  -e POSTGRES_USER=fog \
  -e POSTGRES_PASSWORD=fog \
  -e POSTGRES_DB=fog \
  -p 5432:5432 \
  postgres:16-alpine
```

Apply the schema:

```bash
cd server && go run ./cmd/fog migrate up
```

---

## Running the backend dev server

```bash
cp config.example.yaml config.yaml
# Edit config.yaml …
cd server && go run ./cmd/fog serve -c config.yaml
```

The API is now at **http://localhost:8080/fog/api/v1/**.

> Use `FOG_SERVER_HTTP=:8080` to avoid privileged ports during development.

---

## Running the frontend dev server

The React app (Vite + TanStack Router) has HMR and proxies `/fog/api/*` to `:8080`.

```bash
cd web
bun install          # only needed once
bun run dev
```

UI at **http://localhost:5173**. The proxy config is in [`web/vite.config.ts`](../web/vite.config.ts).

---

## Running tests

### Go unit tests

```bash
make test                                         # all tests, race detector
cd server && go test -v ./internal/api/handlers/...  # single package
cd server && go test -v -run TestHosts_Create ./internal/api/handlers/...  # single test
```

Requires PostgreSQL on `localhost:5432` with `FOG_DATABASE_*` env vars set.

### Playwright E2E tests

E2E tests require a running backend.

**Local dev servers:**

```bash
# Terminal 1: backend
cd server && go run ./cmd/fog serve

# Terminal 2: Playwright (starts Vite automatically)
cd web && bunx playwright test
```

**Against a staging server:**

```bash
FOG_E2E_BASE_URL=https://staging.example.com \
FOG_E2E_USER=admin \
FOG_E2E_PASS=secret \
  cd web && bunx playwright test
```

Useful flags: `--headed`, `--debug`, `--ui`, `auth.spec.ts` (single file).

---

## Linting

```bash
make lint                    # golangci-lint (server)
cd web && bun run lint       # eslint (frontend)
```

---

## Database migrations

Migrations live in `server/internal/database/migrations/` and use [golang-migrate](https://github.com/golang-migrate/migrate).

```bash
./build/fog migrate up        # apply all pending
./build/fog migrate down      # roll back one step
./build/fog migrate status    # show current version
```

To add a new migration:

```bash
touch server/internal/database/migrations/000002_description.up.sql
touch server/internal/database/migrations/000002_description.down.sql
```

---

## Project layout

```
server/
  cmd/fog/                  CLI (cobra): serve, install, migrate, migrate-legacy
  internal/
    api/
      handlers/             One file per resource (hosts.go, images.go, tasks.go …)
      middleware/            JWT auth, boot auth, rate limiter, request logger
      response/              JSON helper functions (OK, Created, Error, …)
      server.go              Chi router + HTTP/HTTPS lifecycle
      static/                Embedded React build output (git-ignored)
    auth/                    JWT sign/verify, bcrypt helpers, boot tokens
    config/                  Config struct + Viper loader
    database/                PostgreSQL connect + golang-migrate runner
    legacymigrate/           FOG 1.x MySQL → PostgreSQL migration
    plugins/                 Compile-time hook interfaces and Registry
    pxe/                     iPXE script template generator
    services/                Background goroutines (scheduler, replicator, multicast, …)
    tftp/                    UDP TFTP server
    ws/                      WebSocket hub for live task progress + agent logs
  ent/                       Ent ORM generated code (schemas in ent/schema/)
  deploy/
    docker/                  docker-compose + Dockerfile
  api/
    openapi.yaml             OpenAPI 3.1 specification

agent/
  cmd/fos-agent/             Agent entrypoint (PID 1 in initramfs)
  internal/
    actions/                 Capture, deploy, wipe, register, debug dispatch
    api/                     HTTP client for fog-next boot API
    cmdline/                 /proc/cmdline parser
    disk/                    Block device enumeration, partition device helpers
    imaging/                 partclone wrapper, filesystem shrink/expand, NTFS ops
    inventory/               Hardware inventory collection
    netup/                   Network readiness poller
    partition/               GPT/MBR backup/restore, UUID management, ExpandLast
    tui/                     Bubble Tea TUI dashboard for imaging progress
    version/                 Build-time version stamps

web/
  src/
    lib/api.ts               Typed API client (all fetch calls live here)
    components/              Reusable React components + shadcn/ui primitives
    routes/                  TanStack Router file-based routes
    store/                   Zustand auth store
    hooks/                   Custom React hooks (WebSocket, keyboard shortcuts)
    types/                   Shared TypeScript interfaces

pixie/
  build.sh                   Initramfs assembly script (runs inside Docker)
  Dockerfile                 Alpine build container
  overlay/                   Rootfs overlay (init scripts, OpenRC services)
  output/                    Build artifacts (bzImage, init.xz)
```

---

## Making a plugin

Implement one or more hook interfaces from `server/internal/plugins` and register in an `init()` function:

```go
package myplugin

import (
    "context"
    "github.com/nemvince/fog-next/internal/plugins"
)

func init() {
    plugins.Register(&AuditPlugin{})
}

type AuditPlugin struct{ plugins.Noop }

func (AuditPlugin) BeforeTaskCreate(ctx context.Context, task *ent.Task) error {
    log.Printf("task being created: type=%s host=%s", task.Type, task.HostID)
    return nil // return an error to reject the task
}
```

See `server/internal/plugins/plugins.go` for all available hook interfaces.
