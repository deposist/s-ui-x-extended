FROM --platform=$BUILDPLATFORM node:alpine AS front-builder
WORKDIR /app
COPY frontend/ ./
RUN npm install && npm run build

FROM golang:1.26.4-alpine AS backend-builder
WORKDIR /app
ARG TARGETARCH
ARG TARGETVARIANT
ARG CRONET_GO_VERSION=2faf34666c2cc8234f10f2ab6d4c4d6104d34ae2
ARG CRONET_GO_DOWNLOAD_DATE=2026-05-13
# Prebuilt cronet release that was current on CRONET_GO_DOWNLOAD_DATE (built from
# source commit CRONET_GO_VERSION; see .github/workflows/release.yml). Pinned by
# tag + sha256 below instead of the floating "releases/latest".
ARG CRONET_GO_RELEASE=v148.0.7778.96-1
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

# SagerNet/cronet-go does not publish assets keyed by commit SHA, so we pin the
# prebuilt release that was current on CRONET_GO_DOWNLOAD_DATE and verify its
# sha256 (digests captured from the GitHub release API). To bump cronet: change
# CRONET_GO_RELEASE / CRONET_GO_VERSION and refresh the sums below.
RUN CRONET_ARCH="$TARGETARCH"; \
    CRONET_FILE="libcronet-linux-${CRONET_ARCH}.so"; \
    CRONET_URL="https://github.com/SagerNet/cronet-go/releases/download/${CRONET_GO_RELEASE}/${CRONET_FILE}"; \
    echo "cronet-go release ${CRONET_GO_RELEASE} (source ${CRONET_GO_VERSION}, captured ${CRONET_GO_DOWNLOAD_DATE})"; \
    echo "Downloading $CRONET_URL"; \
    wget -q -O ./libcronet.so "$CRONET_URL"; \
    { \
      echo "0ddbd9575ce8f5b39a13115e2b7d9f60d578d4fb1a84c7baca10d89f920392d0  libcronet-linux-386.so"; \
      echo "dc7293a929dffa695aae1a89555e7366158fa0a3f40bbe3012d445bc05c99672  libcronet-linux-amd64.so"; \
      echo "40deac370a3257deff8d348382ce59a3948600e3d9f211215b0c453bab5d3657  libcronet-linux-arm.so"; \
      echo "1518e73270c7b49694592bc0448ba1033a80ff4084bfb92cfa5baacec627bd9f  libcronet-linux-arm64.so"; \
      echo "58e403be5e140fb8d1b36379a1d3489baef8ffed31329cbdaa7d2f23bc0c518a  libcronet-linux-loong64.so"; \
      echo "86c14a29528996dccd432c85953991feda19a086925ab18c8bf230d0e08e064d  libcronet-linux-mips64le.so"; \
      echo "6964f8f4e3313e13874214becebb690a8314bc29b5663703f6b3533967e0d2b9  libcronet-linux-mipsle.so"; \
      echo "4ca7dfc8fb909e624c3c61d1aa387e945e183c6dacb91b752b2c279f2b293b1e  libcronet-linux-riscv64.so"; \
    } > /tmp/cronet.sha256; \
    EXPECTED="$(grep "  ${CRONET_FILE}$" /tmp/cronet.sha256 | awk '{print $1}')"; \
    [ -n "$EXPECTED" ] || { echo "no pinned sha256 for ${CRONET_FILE}"; exit 1; }; \
    echo "${EXPECTED}  ./libcronet.so" | sha256sum -c -; \
    rm -f /tmp/cronet.sha256; \
    chmod 755 ./libcronet.so

COPY . .
COPY --from=front-builder /app/dist/ /app/web/html/

RUN if [ "$TARGETARCH" = "arm" ]; then export GOARM=7; [ "$TARGETVARIANT" = "v6" ] && export GOARM=6; fi; \
    go build -ldflags="-w -s" \
    -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_ccm,with_ocm,with_oomkiller" \
    -o sui main.go

FROM alpine
# Match defaultValueMap["timeLocation"] in service settings.
ENV TZ=Europe/Moscow
WORKDIR /app
RUN set -ex && apk add --no-cache bash tzdata ca-certificates nftables
COPY --from=backend-builder /app/sui /app/libcronet.so /app/
COPY entrypoint.sh /app/
ENTRYPOINT [ "./entrypoint.sh" ]
