# Architecture

This document describes the high-level design of FOG Next.

---

## Overview

FOG Next is a single Go binary that combines five historically separate subsystems:

```
┌─────────────────────────────────────────────────────────────┐
│                        fog binary                           │
│                                                             │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐ │
│  │  HTTP/HTTPS │  │  TFTP server │  │ Background services│ │
│  │  server     │  │  (:69 UDP)   │  │ (7 goroutines)     │ │
│  └──────┬──────┘  └──────────────┘  └────────────────────┘ │
│         │                                                   │
│  ┌──────▼──────────────────────────────────────────────┐   │
│  │  Chi router                                         │   │
│  │  /api/v1/*  ── REST handlers                        │   │
│  │  /api/v1/ws ── WebSocket hub                        │   │
│  │  /fog/*     ── Boot + kernel endpoints                   │   │
│  │  /*         ── Embedded React SPA                   │   │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Store layer (interfaces + postgres implementations) │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
              ┌────────────▼────────────┐
              │   PostgreSQL database   │
              └─────────────────────────┘
```

---

## Directory Structure

```
cmd/fog/                  CLI entry point (cobra commands)
internal/
  api/
    handlers/             One file per resource (hosts, images, tasks, …)
    middleware/           Auth (JWT), rate limiter, logger
    response/             JSON helpers (OK, Created, BadRequest, …)
    server.go             Chi router and HTTP server lifecycle
    static/               Embedded React build (populated by mise run web)
  auth/                   JWT sign/verify, bcrypt password helpers
  config/                 Viper-based config loading and defaults
  database/               sqlx wrapper + golang-migrate integration
  models/                 Domain types shared by every layer
  pxe/                    iPXE script template generator
  services/               Background goroutines (7 services)
  store/
    store.go              Repository interfaces (HostStore, ImageStore, …)
    postgres/             PostgreSQL implementations
  tftp/                   TFTP server (serves /tftpboot)
  ws/                     WebSocket hub + HTTP upgrade handler
migrations/               SQL migration files (up + down)
web/                      React frontend (Vite + TypeScript)
deploy/
  docker/                 Dockerfile + docker-compose.yml
  systemd/                fog.service unit file
  config.example.yaml     Documented config template
```

---

## Request Flow

### UI / API request

```
Browser  →  HTTP  →  Chi router
                        │
                   middleware stack
                   1. RequestID
                   2. RealIP
                   3. Logger
                   4. Recoverer
                   5. CORS
                   6. Authenticate (JWT) — authenticated routes only
                        │
                   handler (e.g. handlers.Hosts.List)
                        │
                   store.HostStore.ListHosts()
                        │
                   PostgreSQL
```

### iPXE boot request

```
Client PXE ROM  →  DHCP option 66/67  →  TFTP
                                           │
                                     undionly.kpxe / ipxe.efi
                                           │
                                     chainloads HTTP
                                           │
GET /fog/boot?mac=aa:bb:cc:dd:ee:ff  →  handlers.Boot.ServeHTTP
                                           │
                                     pxe.Generator.Script(host)
                                           │
                                     iPXE script sent to client
```

### Live task progress (WebSocket)

```
Client browser  →  GET /api/v1/ws (Bearer token)
                        │
                   ws.Hub.Register(conn)
                        │
                   background service (TaskScheduler)
                        │  emits ws.Event{Type: "task.progress", …}
                        │
                   ws.Hub.Broadcast(event) → all registered clients
```

---

## Background Services

All services run as goroutines managed by `services.Manager`. They respect context cancellation on shutdown.

| Service | Interval | Responsibility |
|---------|----------|---------------|
| `TaskScheduler` | 60s | Picks queued tasks, transitions them to active, triggers imaging |
| `ImageReplicator` | 10m | Copies images to secondary storage nodes |
| `SnapinReplicator` | 10m | Copies snapin files to secondary storage nodes |
| `SnapinHash` | 30m | Recalculates SHA-256 hashes for snapin files |
| `ImageSize` | 1h | Updates image disk-usage statistics |
| `PingHosts` | 5m | ICMP-pings all enabled hosts to update `last_contact` |
| `MulticastManager` | 10s | Manages active multicast imaging sessions |

---

## Authentication

All `/api/v1/` routes (except `POST /auth/login` and `POST /auth/refresh`) require a `Authorization: Bearer <access_token>` header.

Tokens are JWTs signed with HMAC-SHA256 using `auth.jwt_secret`. They contain:

| Claim | Description |
|-------|-------------|
| `uid` | User UUID |
| `sub` | Username |
| `role` | `admin` or `readonly` |
| `exp` | Expiry (default 15 minutes) |

Refresh tokens are long-lived (default 7 days) and are used to obtain new access tokens without re-entering credentials.

---

## Store Layer

The store layer is entirely interface-based:

```go
type Store interface {
    Hosts()   HostStore
    Images()  ImageStore
    Tasks()   TaskStore
    Groups()  GroupStore
    Snapins() SnapinStore
    Storage() StorageStore
    Users()   UserStore
    Settings() SettingStore
}
```

Handlers depend only on the interfaces. The `postgres` package provides the production implementation. This makes unit testing handlers straightforward with mock stores.

---

## Database Migrations

Migrations live in `migrations/` as numbered SQL files (`000001_init.up.sql`, etc.). They are embedded in the binary and applied automatically on `fog serve` startup, or manually with `fog migrate up`.

The migration engine is `golang-migrate/migrate` with the `postgres` driver.

---

## Frontend

The React SPA is built with:

- **Vite 8** — bundler
- **React 19 + TypeScript** — UI framework
- **Tailwind CSS v4** — utility CSS
- **TanStack Query v5** — server state caching
- **TanStack Table v8** — headless tables
- **Zustand v5** — client-side state (auth token, toast queue)
- **React Router v7** — SPA routing
- **Radix UI** — accessible primitives (Dialog, Toast)
- **lucide-react** — icons

`mise run web` runs `bun install && bun run build` and copies the output to `internal/api/static/`, which is embedded into the binary via `//go:embed static`. The `spaHandler` in `server.go` serves `index.html` for any path that doesn't match a real file, enabling React Router's history mode.
