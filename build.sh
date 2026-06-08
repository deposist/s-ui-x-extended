#!/usr/bin/env bash
set -euo pipefail

cd frontend
npm i
npm run build

cd ..
echo "Backend"

mkdir -p web/html
rm -fr web/html/*
cp -R frontend/dist/* web/html/

BUILD_TAGS="with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_musl,badlinkname,tfogo_checklinkname0,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_ccm,with_ocm,with_oomkiller"
go build -ldflags '-w -s -checklinkname=0 -extldflags "-Wl,-no_warn_duplicate_libraries"' -tags "$BUILD_TAGS" -o sui main.go
