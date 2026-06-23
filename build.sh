#!/bin/sh

cd frontend
npm i
npm run build

cd ..
echo "Backend"

mkdir -p web/html
rm -fr web/html/*
cp -R frontend/dist/* web/html/

BUILD_TAGS="with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_musl,badlinkname,tfogo_checklinkname0,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_ccm,with_ocm,with_oomkiller"

# Embed the release artifact platform suffix so the in-panel self-update knows
# which `s-ui-linux-<platform>.tar.gz` asset to fetch (config.ResolveArtifactPlatform).
ARCH="$(go env GOARCH)"
case "$ARCH" in
  arm) ARTIFACT_PLATFORM="armv$(go env GOARM 2>/dev/null || echo 7)" ;;
  *)   ARTIFACT_PLATFORM="$ARCH" ;;
esac
LDFLAGS="-w -s -checklinkname=0 -extldflags \"-Wl,-no_warn_duplicate_libraries\" -X github.com/deposist/s-ui-x-extended/config.ArtifactPlatform=${ARTIFACT_PLATFORM}"
go build -ldflags "$LDFLAGS" -tags "$BUILD_TAGS" -o sui main.go
