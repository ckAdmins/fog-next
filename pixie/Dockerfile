FROM alpine:latest

RUN apk add --no-cache \
    alpine-sdk \
    apk-tools \
    bash \
    build-base \
    cpio \
    findutils \
    gzip \
    xz \
    zstd \
    rsync \
    squashfs-tools \
    linux-lts \
    mkinitfs \
    util-linux \
    go \
    git

WORKDIR /work
CMD ["sh", "/work/pixie/build.sh"]