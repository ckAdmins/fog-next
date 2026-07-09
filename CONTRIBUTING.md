# Contributing

## Setup

```bash
git clone https://github.com/ckAdmins/fog-next.git
cd fog-next
mise install          # Installs Go + Bun
mise run server       # Build the server
mise run web          # Build the frontend
```

## Development workflow

Make changes in a feature branch. Keep commits focused. Follow existing code conventions.

**Before submitting:**
```bash
mise run lint          # golangci-lint on server
mise run test          # Go tests (needs PostgreSQL on localhost:5432)
cd web && bun run lint && bun run build   # Frontend
```

## Commit messages

Use conventional commits (e.g. `feat:`, `fix:`, `refactor:`). Reference issues when applicable.

## CI

CI runs on every push and PR. It validates:
- Go vet, build, staticcheck, test (server + agent)
- Frontend lint, typecheck, build
- Agent initramfs + kernel build via pixie Docker

On tags, the server Docker image is pushed to GHCR and the agent artifacts are attached to the release.
