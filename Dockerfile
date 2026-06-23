# Base images pinned by digest (multi-arch manifest list) for reproducibility and
# supply-chain integrity; refresh digests (and any version bump) via Dependabot.
FROM --platform=$BUILDPLATFORM node:alpine@sha256:3ad34ca6292aec4a91d8ddeb9229e29d9c2f689efd0dd242860889ac71842eba AS front-builder
WORKDIR /app
COPY frontend/ ./
# npm ci (not install) so the image is built from the exact, audited
# package-lock.json — matches CI/release and fails closed on lockfile drift.
RUN npm ci && npm run build

FROM golang:1.26.4-alpine@sha256:7a3e50096189ad57c9f9f865e7e4aa8585ed1585248513dc5cda498e2f41812c AS backend-builder
WORKDIR /app
ARG TARGETARCH
ARG TARGETVARIANT
ARG CRONET_GO_VERSION=2faf34666c2cc8234f10f2ab6d4c4d6104d34ae2
# Immutable cronet-go release tag for the prebuilt native library (decoupled from
# the source pin upstream; kept in sync manually). Pinning the tag + verifying the
# sha256 replaces the mutable releases/latest fetch.
ARG CRONET_GO_ASSET_TAG=v148.0.7778.96-1
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
ENV GOARCH=$TARGETARCH

RUN apk update && apk add --no-cache \
    gcc \
    musl-dev \
    libc-dev \
    make \
    git \
    wget \
    unzip \
    bash \
    curl

ENV CC=gcc

# Download the prebuilt cronet native library from a PINNED, immutable release
# tag and verify its sha256 before use. The .so is dlopen'd in-process by the
# root sing-box core (with_purego), so it is the most privileged code in the
# image and must be integrity-checked, never fetched from the mutable
# releases/latest. Asset digests captured from the GitHub release API for
# ${CRONET_GO_ASSET_TAG}; update both the tag and the hashes together.
RUN CRONET_ARCH="$TARGETARCH" && \
    case "$CRONET_ARCH" in \
      amd64) CRONET_SHA256="dc7293a929dffa695aae1a89555e7366158fa0a3f40bbe3012d445bc05c99672" ;; \
      arm64) CRONET_SHA256="1518e73270c7b49694592bc0448ba1033a80ff4084bfb92cfa5baacec627bd9f" ;; \
      arm)   CRONET_SHA256="40deac370a3257deff8d348382ce59a3948600e3d9f211215b0c453bab5d3657" ;; \
      386)   CRONET_SHA256="0ddbd9575ce8f5b39a13115e2b7d9f60d578d4fb1a84c7baca10d89f920392d0" ;; \
      *) echo "no pinned libcronet sha256 for arch ${CRONET_ARCH}" >&2; exit 1 ;; \
    esac; \
    CRONET_URL="https://github.com/SagerNet/cronet-go/releases/download/${CRONET_GO_ASSET_TAG}/libcronet-linux-${CRONET_ARCH}.so"; \
    echo "cronet-go source pin: ${CRONET_GO_VERSION}; pinned asset tag: ${CRONET_GO_ASSET_TAG}" && \
    echo "Downloading $CRONET_URL" && \
    wget -q -O ./libcronet.so "$CRONET_URL" && \
    echo "${CRONET_SHA256}  ./libcronet.so" | sha256sum -c - && \
    chmod 755 ./libcronet.so

COPY . .
COPY --from=front-builder /app/dist/ /app/web/html/

RUN if [ "$TARGETARCH" = "arm" ]; then export GOARM=7; [ "$TARGETVARIANT" = "v6" ] && export GOARM=6; fi; \
    go build -ldflags="-w -s" \
    -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_ccm,with_ocm,with_oomkiller" \
    -o sui main.go

FROM alpine:latest@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4
# Match defaultValueMap["timeLocation"] in service settings.
ENV TZ=Europe/Moscow
WORKDIR /app
RUN set -ex && apk add --no-cache --upgrade bash tzdata ca-certificates nftables
COPY --from=backend-builder /app/sui /app/libcronet.so /app/
COPY entrypoint.sh /app/
RUN chmod +x /app/entrypoint.sh
ENTRYPOINT [ "./entrypoint.sh" ]
