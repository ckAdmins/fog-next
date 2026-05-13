#!/bin/sh

set -eux

ROOTFS=/tmp/rootfs
OUT=$1

rm -rf $ROOTFS
mkdir -p $ROOTFS
mkdir -p $OUT

# repositories
cat > /etc/apk/repositories <<EOF
https://dl-cdn.alpinelinux.org/alpine/latest-stable/main
https://dl-cdn.alpinelinux.org/alpine/latest-stable/community
EOF

# install root filesystem
apk \
  --root $ROOTFS \
  --initdb \
  --keys-dir /etc/apk/keys \
  --repositories-file /etc/apk/repositories \
  add \
    alpine-base \
    busybox \
    bash \
    curl \
    e2fsprogs \
    partclone \
    kmod \
    btrfs-progs \
    coreutils \
    eudev \
    f2fs-tools \
    gptfdisk \
    sgdisk \
    iproute2 \
    ntfs-3g \
    openrc \
    parted \
    util-linux \
    xfsprogs \
    zstd \
    file \
    e2fsprogs-extra \
    --no-cache

# build the agent
cd /work/agent
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -buildvcs=false \
  -ldflags "-s -w \
    -X github.com/nemvince/fos-agent/internal/version.Version=$(git describe --tags --always 2>/dev/null || echo dev) \
    -X github.com/nemvince/fos-agent/internal/version.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
    -X github.com/nemvince/fos-agent/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o fos-agent ./cmd/fos-agent

cp fos-agent $ROOTFS/bin/fos-agent

cd /work

# copy overlay files
rsync -a /work/pixie/overlay/ $ROOTFS/

# make wrappers and init scripts executable
chmod +x $ROOTFS/bin/fos-autologin
chmod +x $ROOTFS/etc/init.d/fos-net 2>/dev/null || true

# set up OpenRC runlevels
# sysinit runlevel — udev device management
mkdir -p $ROOTFS/etc/runlevels/sysinit
ln -sf /etc/init.d/udev $ROOTFS/etc/runlevels/sysinit/udev 2>/dev/null || true
ln -sf /etc/init.d/devfs $ROOTFS/etc/runlevels/sysinit/devfs 2>/dev/null || true
ln -sf /etc/init.d/dmesg $ROOTFS/etc/runlevels/sysinit/dmesg 2>/dev/null || true

# boot runlevel — basic system services
mkdir -p $ROOTFS/etc/runlevels/boot
ln -sf /etc/init.d/hostname $ROOTFS/etc/runlevels/boot/hostname 2>/dev/null || true
ln -sf /etc/init.d/loopback $ROOTFS/etc/runlevels/boot/loopback 2>/dev/null || true
ln -sf /etc/init.d/modules $ROOTFS/etc/runlevels/boot/modules 2>/dev/null || true
ln -sf /etc/init.d/mountinfo $ROOTFS/etc/runlevels/boot/mountinfo 2>/dev/null || true
ln -sf /etc/init.d/procfs $ROOTFS/etc/runlevels/boot/procfs 2>/dev/null || true

# default runlevel — networking and services
mkdir -p $ROOTFS/etc/runlevels/default
ln -sf /etc/init.d/fos-net $ROOTFS/etc/runlevels/default/fos-net 2>/dev/null || true
ln -sf /etc/init.d/udev-postmount $ROOTFS/etc/runlevels/default/udev-postmount 2>/dev/null || true

# create minimal init — bootstrap kernel, then hand off to OpenRC as PID 1
cat > $ROOTFS/init <<'INITEOF'
#!/bin/sh

set -x

mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t devtmpfs devtmpfs /dev

# Load kernel modules for common hardware
modprobe -a \
  virtio_blk virtio_net virtio_scsi virtio_pci \
  ata_piix ahci sd_mod sr_mod \
  nvme xen_blkfront \
  usb_storage uas \
  2>/dev/null || true

echo "Starting OpenRC init system..."

# Hand off to OpenRC as PID 1
export PATH=/bin:/sbin:/usr/bin:/usr/sbin
exec /sbin/init
INITEOF

chmod +x $ROOTFS/init

# kernel version
KVER=$(ls /lib/modules)

# copy kernel modules
mkdir -p $ROOTFS/lib/modules
cp -a /lib/modules/$KVER $ROOTFS/lib/modules/

# generate module deps
depmod -b $ROOTFS $KVER

# device nodes
mkdir -p $ROOTFS/dev

mknod -m 622 $ROOTFS/dev/console c 5 1
mknod -m 666 $ROOTFS/dev/null c 1 3

# build initramfs
(
cd $ROOTFS

find . \
  -print0 \
| cpio --null -ov --format=newc \
| xz -T0 -6 --check=crc32 \
> "$OUT/init.xz"
)

# copy kernel
cp /boot/vmlinuz-lts $OUT/bzImage
