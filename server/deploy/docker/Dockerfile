FROM oven/bun:1-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=web-builder /web/dist ./internal/api/static/
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /fog ./cmd/fog

# Download iPXE boot files.
# undionly.kpxe  — legacy BIOS PXE
# ipxe.efi       — UEFI x86-64
# snponly.efi    — UEFI ARM64 (via snponly which avoids driver conflicts)
FROM alpine:3.20 AS ipxe-fetcher
RUN apk add --no-cache curl
RUN mkdir -p /tftpboot
RUN curl -fsSL -o /tftpboot/undionly.kpxe https://boot.ipxe.org/undionly.kpxe
RUN curl -fsSL -o /tftpboot/ipxe.efi      https://boot.ipxe.org/x86_64-efi/ipxe.efi
RUN curl -fsSL -o /tftpboot/snponly.efi   https://boot.ipxe.org/x86_64-efi/snponly.efi

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /fog /usr/local/bin/fog
COPY --from=ipxe-fetcher /tftpboot /tftpboot
RUN mkdir -p /etc/fog /opt/fog/images /opt/fog/snapins /opt/fog/kernels
EXPOSE 8080 8443 6969/udp
ENTRYPOINT ["/usr/local/bin/fog"]
CMD ["serve"]
