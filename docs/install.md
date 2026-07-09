# Installation Guide

This guide covers installing FOG Next on a bare-metal server or VM running a Debian/Ubuntu or RHEL/Fedora/AlmaLinux system, as well as the Docker Compose path.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Docker Compose (recommended)](#docker-compose-recommended)
3. [Bare-metal installation](#bare-metal-installation)
4. [DHCP configuration](#dhcp-configuration)
5. [TFTP setup](#tftp-setup)
6. [Systemd service](#systemd-service)
7. [Upgrading](#upgrading)

---

## Prerequisites

### Hardware / OS

- A dedicated Linux server (physical or VM) with **two network interfaces** recommended: one for management/internet, one on the imaging VLAN
- Ubuntu 22.04 LTS / Debian 12 / AlmaLinux 9 / RHEL 9 (other distros work, but these are tested)
- At least 4 GB RAM, 20 GB OS disk, and sufficient storage for images (typically hundreds of GBs)

### Software

| Component | Minimum version | Notes |
|-----------|----------------|-------|
| PostgreSQL | 14 | 16 recommended |
| Go toolchain | 1.23 | Only needed to build from source |
| Bun | 1.x | Only needed to build the frontend |
| Docker Engine | 24 | Only required for Docker Compose path |

---

## Docker Compose (recommended)

The quickest path to a running server.

```bash
git clone https://github.com/ckAdmins/fog-next.git
cd fog-next

# Copy and edit the config
cp server/deploy/config.example.yaml server/deploy/docker/config.yaml
```

Edit `server/deploy/docker/config.yaml` at minimum:
- `server.base_url` — set to the server's IP or hostname (used in iPXE scripts)
- `auth.jwt_secret` — replace with the output of `openssl rand -hex 32`

```bash
# Pull images and start
docker compose -f server/deploy/docker/compose.yml up -d

# Wait for postgres to become healthy, then run the wizard
docker compose -f server/deploy/docker/compose.yml exec fog fog install
```

The wizard prompts for database connection details (use the values from your `config.yaml`) and creates the initial admin user.

Navigate to **http://\<server-ip\>** to access the web UI.

---

## Bare-metal installation

### 1. Install PostgreSQL

**Debian/Ubuntu:**
```bash
sudo apt-get install -y postgresql postgresql-contrib
sudo systemctl enable --now postgresql
```

**RHEL/AlmaLinux:**
```bash
sudo dnf install -y postgresql-server postgresql-contrib
sudo postgresql-setup --initdb
sudo systemctl enable --now postgresql
```

Create the database and user:
```bash
sudo -u postgres psql <<'SQL'
CREATE USER fog WITH PASSWORD 'change_me_strong_password';
CREATE DATABASE fog OWNER fog;
SQL
```

### 2. Build fog-next

```bash
# Install Go (if needed)
# See https://go.dev/doc/install

# Install Bun (if needed)
curl -fsSL https://bun.sh/install | bash

# Clone and build
git clone https://github.com/ckAdmins/fog-next.git
cd fog-next
mise run server

# Install the binary
sudo install -m 0755 build/fog /usr/local/bin/fog
```

### 3. Create directories

```bash
sudo mkdir -p /opt/fog/images /opt/fog/snapins /opt/fog/kernels /tftpboot
sudo useradd -r -s /usr/sbin/nologin fog
sudo chown -R fog:fog /opt/fog /tftpboot
```

### 4. Run the installer

```bash
sudo fog install
```

The wizard will:
- Prompt for PostgreSQL connection details
- Generate a random JWT secret
- Write `/etc/fog/config.yaml`
- Apply database migrations
- Create the admin user you specify

### 5. Kernel and iPXE files

FOG Next reads boot files from `tftp.root_dir` (default `/tftpboot`) and kernels/initrds from `storage.kernel_path` (default `/opt/fog/kernels`).

Download the standard iPXE binaries using the provided Makefile target:

```bash
mise run fetch-ipxe            # writes to /tftpboot by default
# or to a custom directory:
IPXE_DIR=/srv/tftp mise run fetch-ipxe
```

This downloads:

| File | Purpose |
|------|---------|
| `undionly.kpxe` | Legacy BIOS / UNDI PXE boot |
| `ipxe.efi` | UEFI x86-64 |
| `arm64-efi/snponly.efi` | UEFI ARM64 |

Build the agent kernel and initramfs using the pixie Docker build:

```bash
mise run agent                   # → pixie/output/bzImage + pixie/output/init.xz
sudo cp pixie/output/bzImage /opt/fog/kernels/bzImage
sudo cp pixie/output/init.xz /opt/fog/kernels/init.xz
```

---

## DHCP configuration

Clients discover the FOG server via DHCP options. You need to point DHCP **option 66** (TFTP server) and **option 67** (boot filename) at your FOG server.

### ISC DHCP (dhcpd)

```
# /etc/dhcp/dhcpd.conf (inside the relevant subnet block)
next-server 192.168.1.10;          # FOG server IP
filename "undionly.kpxe";          # BIOS clients
# For UEFI clients, use a class-based approach:
if exists vendor-class-identifier and
   option vendor-class-identifier = "PXEClient:Arch:00007" {
    filename "ipxe.efi";
}
```

### Dnsmasq

```
# /etc/dnsmasq.conf
dhcp-boot=undionly.kpxe,,192.168.1.10
```

### UEFI + BIOS dual-boot (dnsmasq)

```
dhcp-match=set:efi-x86_64,option:client-arch,7
dhcp-match=set:efi-x86_64,option:client-arch,9
dhcp-boot=tag:efi-x86_64,ipxe.efi,,192.168.1.10
dhcp-boot=tag:!efi-x86_64,undionly.kpxe,,192.168.1.10
```

---

## TFTP setup

FOG Next includes its own TFTP server (enabled by default on `:69`). If you already run `tftpd-hpa` or `dnsmasq` as a TFTP server, either disable FOG's built-in TFTP (`tftp.enabled: false` in config) and point your TFTP root at `/tftpboot`, or let FOG's server handle it.

### Populating /tftpboot

For **Docker Compose** deployments the iPXE binaries are baked into the image automatically at build time — nothing extra is required.

For **bare-metal** deployments run:

```bash
mise run fetch-ipxe
```

Or manually:

```bash
sudo mkdir -p /tftpboot/arm64-efi
sudo curl -fsSL -o /tftpboot/undionly.kpxe         https://boot.ipxe.org/undionly.kpxe
sudo curl -fsSL -o /tftpboot/ipxe.efi              https://boot.ipxe.org/ipxe.efi
sudo curl -fsSL -o /tftpboot/arm64-efi/snponly.efi https://boot.ipxe.org/arm64-efi/snponly.efi
```

Verify TFTP is reachable from a client:
```bash
tftp 192.168.1.10 -c get undionly.kpxe /tmp/test.kpxe && echo "TFTP OK"
```

---

## Systemd service

A systemd unit is provided at `server/deploy/systemd/fog.service`:

```bash
sudo cp deploy/systemd/fog.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now fog
sudo systemctl status fog
```

To view logs:
```bash
journalctl -u fog -f
```

---

## Upgrading

See [upgrading.md](upgrading.md) for updating between fog-next releases.

For minor updates within fog-next:

```bash
git pull
mise run build
sudo install -m 0755 build/fog /usr/local/bin/fog
sudo systemctl restart fog
```

Database migrations run automatically on startup, or you can run them manually:

```bash
sudo fog migrate up
```
