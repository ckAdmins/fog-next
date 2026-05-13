#!/bin/sh
set -eu

ROOT=$(cd "$(dirname "$0")" && pwd)
BUILD_DIR="$ROOT/build"
SERVER_DIR="$ROOT/server"
WEB_DIR="$ROOT/web"
AGENT_DIR="$ROOT/agent"
PIXIE_DIR="$ROOT/pixie"
OUTPUT_DIR="$PIXIE_DIR/output"

VERSION=${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}
COMMIT=${COMMIT:-$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)}
BUILD_DATE=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%S%z)}

# ─── helpers ──────────────────────────────────────────────────────────────────

info()  { echo "  [build] $*"; }
step()  { echo "==> $*"; }
die()   { echo "ERROR: $*" >&2; exit 1; }

# ─── agent: initramfs + kernel via pixie Docker ───────────────────────────────

build_agent() {
    step "Building agent (kernel + initramfs) via pixie Docker..."
    if ! command -v docker >/dev/null 2>&1; then
        die "docker is required to build the agent"
    fi
    info "building pixie container image..."
    docker build -t alpine-image-builder "$PIXIE_DIR"
    info "running build container (mounting repo root)..."
    mkdir -p "$OUTPUT_DIR"
    docker run --rm \
        --privileged \
        -v "$ROOT:/work" \
        alpine-image-builder \
        sh /work/pixie/build.sh $OUTPUT_DIR
    info "agent build complete → $OUTPUT_DIR/bzImage, $OUTPUT_DIR/init.xz"
}

# ─── server: Go binary ────────────────────────────────────────────────────────

build_server() {
    step "Building server Go binary..."
    mkdir -p "$BUILD_DIR"
    cd "$SERVER_DIR"
    go build -ldflags "-s -w" -o "$BUILD_DIR/fog" ./cmd/fog
    info "server binary → $BUILD_DIR/fog"
}

# ─── server-docker: Docker image ──────────────────────────────────────────────

build_server_docker() {
    step "Building server Docker image..."
    if ! command -v docker >/dev/null 2>&1; then
        die "docker is required to build the server image"
    fi
    docker build \
        -f "$SERVER_DIR/deploy/docker/Dockerfile" \
        -t fog-next:latest \
        "$ROOT"
    info "server Docker image → fog-next:latest"
}

# ─── web: React frontend ──────────────────────────────────────────────────────

build_web() {
    step "Building web frontend..."
    if ! command -v bun >/dev/null 2>&1; then
        die "bun is required to build the frontend (https://bun.sh)"
    fi
    cd "$WEB_DIR"
    bun install --frozen-lockfile
    bun run build
    # Copy built assets into the server's embedded static directory.
    find "$SERVER_DIR/internal/api/static" -mindepth 1 ! -name '.gitkeep' -delete 2>/dev/null || true
    cp -r "$WEB_DIR/dist/." "$SERVER_DIR/internal/api/static/"
    info "frontend built → $SERVER_DIR/internal/api/static/"
}

# ─── all ──────────────────────────────────────────────────────────────────────

build_all() {
    build_web
    build_server
    build_server_docker
    build_agent
}

# ─── clean ────────────────────────────────────────────────────────────────────

clean() {
    step "Cleaning build artifacts..."
    rm -rf "$BUILD_DIR"
    rm -rf "$WEB_DIR/dist"
    rm -f "$OUTPUT_DIR/bzImage" "$OUTPUT_DIR/init.xz" 2>/dev/null || true
    info "clean complete"
}

# ─── dispatch ─────────────────────────────────────────────────────────────────

usage() {
    cat <<EOF
Usage: $0 <target>

Targets:
  agent          Build fos-agent initramfs + kernel (via pixie Docker)
  server         Build the fog server Go binary
  server-docker  Build the fog server Docker image (fog-next:latest)
  web            Build the React frontend
  all            Build everything (web → server → server-docker → agent)
  clean          Remove all build artifacts

Environment:
  VERSION        Override version string (default: git describe)
  COMMIT         Override commit hash (default: git rev-parse HEAD)
  BUILD_DATE     Override build date (default: current UTC)
EOF
}

case "${1:-}" in
    agent)          build_agent ;;
    server)         build_server ;;
    server-docker)  build_server_docker ;;
    web)            build_web ;;
    all)            build_all ;;
    clean)          clean ;;
    -h|--help|help) usage ;;
    "")             usage; exit 1 ;;
    *)              die "unknown target: $1 (use -h for help)" ;;
esac
