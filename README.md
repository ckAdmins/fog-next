# FOG Next

A modern rewrite of the **FOG Project** network boot and computer imaging server.

FOG Next replaces the PHP/Apache stack with a single statically-linked Go binary that serves the REST API, web UI, and TFTP all at once.

---

## Features

- **iPXE-based network boot** — deploys images over HTTP; chains from DHCP option 67
- **Image management** — capture, deploy, multicast, replicate across storage nodes
- **Task scheduling** — queue capture/deploy/multicast/snapin tasks per host or group
- **Snapin support** — deploy software packages to hosts after imaging
- **Live task progress** — WebSocket feed; no page refresh needed
- **REST API** — full JSON API with JWT auth, suitable for scripting and integrations
- **Single binary** — embeds the React web UI; no separate web server required

---

## Quick Start (Docker Compose)

**Prerequisites:** Docker Engine 24+ and Docker Compose v2.

```bash
# 1. Clone the repo
git clone https://github.com/nemvince/fog-next.git
cd fog-next

# 2. Create a config file from the example
cp config.example.yaml server/deploy/docker/config.yaml
#    Edit server/deploy/docker/config.yaml and set a strong jwt_secret and DB password

# 3. Start
make docker-up

# 4. Run the install wizard to create your admin account
docker compose -f server/deploy/docker/docker-compose.yml exec fog fog install
```

The web UI is available at **http://localhost** after the first `fog serve` starts.

> **Tip:** To tear down (preserving volumes): `make docker-down`
> To destroy everything including data: `docker compose -f server/deploy/docker/docker-compose.yml down -v`

---

## Build from Source

**Prerequisites:** Go 1.25+, Bun 1.x, git.

```bash
# Single entry point for all builds
./build.sh server     # Go binary → build/fog
./build.sh web        # React frontend (embeds into server binary)
./build.sh all        # web → server → server-docker → agent
./build.sh agent      # Agent initramfs + kernel (via pixie Docker)
```

Also available as `make build` / `make test` / `make lint` for server-only workflows.

### Development mode (hot reload)

Start the backend and frontend in separate terminals:

```bash
# Terminal 1 — Go backend (use a non-privileged port during development)
export FOG_DATABASE_HOST=localhost FOG_DATABASE_USER=fog \
       FOG_DATABASE_PASSWORD=fog   FOG_DATABASE_NAME=fog \
       FOG_AUTH_JWT_SECRET=dev-secret FOG_SERVER_HTTP=:8080
cd server && go run ./cmd/fog serve -c config.yaml

# Terminal 2 — Vite dev server with HMR (proxies /api/* → :8080)
cd web && bun run dev
# UI available at http://localhost:5173
```

See **[docs/developing.md](docs/developing.md)** for the full developer guide:
running tests, linting, migrations, E2E tests, writing plugins, and more.

---

## First-Run Setup

After building, run the interactive installer once:

```bash
sudo ./build/fog install
```

This will:
1. Prompt for database connection details
2. Write `/etc/fog/config.yaml` with a generated JWT secret
3. Apply all database migrations
4. Create the initial admin user

Then start the server:

```bash
sudo ./build/fog serve
# or via systemd:
sudo systemctl enable --now fog
```

---

## Configuration

Configuration is loaded from (in priority order):
1. `/etc/fog/config.yaml`
2. `$HOME/.fog/config.yaml`
3. `./config.yaml`

See **[config.example.yaml](config.example.yaml)** for the full reference with inline comments.

### Key settings

| Key | Default | Description |
|-----|---------|-------------|
| `server.http` | `:80` | HTTP listen address |
| `server.base_url` | `http://localhost` | Externally reachable URL (used in iPXE scripts) |
| `database.host` | `localhost` | PostgreSQL host |
| `auth.jwt_secret` | — | **Must be set.** Sign JWT tokens. |
| `storage.base_path` | `/opt/fog/images` | Image storage root |
| `tftp.root_dir` | `/tftpboot` | TFTP root directory |

---

## CLI Reference

```
fog serve              Start the HTTP server and all background services
fog install            Interactive first-run setup wizard
fog migrate up         Apply all pending database migrations
fog migrate down       Roll back the most recent migration
fog migrate status     Print current migration schema version
fog fetch-kernels      Download (or re-download) the fos-next kernel and initramfs
fog version            Print the fog-next version
```

All commands accept `-c /path/to/config.yaml` to override the config file path.

---

## API

The REST API is available at `/fog/api/v1/`. Authentication uses JWT Bearer tokens obtained from `POST /fog/api/v1/auth/login`.

See **[docs/api.md](docs/api.md)** for the full endpoint reference.

---

## Documentation

| Document | Description |
|----------|-------------|
| [docs/developing.md](docs/developing.md) | Dev servers, tests, linting, plugins, project layout |
| [docs/install.md](docs/install.md) | Detailed installation guide (bare-metal and Docker) |
| [docs/architecture.md](docs/architecture.md) | Component overview and request flow |
| [docs/api.md](docs/api.md) | REST API endpoint reference |
| [docs/upgrading.md](docs/upgrading.md) | Upgrading between releases |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

FOG Next is released under the [GNU General Public License v3](LICENSE).
