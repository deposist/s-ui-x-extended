#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
REPO_ROOT=$SCRIPT_DIR
FRONTEND_DIR=$REPO_ROOT/frontend
FRONTEND_DIST=$FRONTEND_DIR/dist
WEB_DIR=$REPO_ROOT/web
WEB_HTML=$WEB_DIR/html
STAGE_DIR=
BACKUP_DIR=

die() {
  echo "Error: $*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "$1 is not installed or not in PATH"
}

cleanup() {
  status=$?
  trap - EXIT HUP INT TERM

  if [ -n "$STAGE_DIR" ] && { [ -e "$STAGE_DIR" ] || [ -L "$STAGE_DIR" ]; }; then
    rm -rf "$STAGE_DIR" || :
  fi

  if [ -n "$BACKUP_DIR" ] && { [ -e "$BACKUP_DIR" ] || [ -L "$BACKUP_DIR" ]; }; then
    if { [ -e "$BACKUP_DIR/html" ] || [ -L "$BACKUP_DIR/html" ]; } && ! { [ -e "$WEB_HTML" ] || [ -L "$WEB_HTML" ]; }; then
      mv "$BACKUP_DIR/html" "$WEB_HTML" || echo "Error: failed to restore prior web assets from $BACKUP_DIR/html" >&2
    fi
    if ! { [ -e "$BACKUP_DIR/html" ] || [ -L "$BACKUP_DIR/html" ]; }; then
      rm -rf "$BACKUP_DIR" || :
    else
      echo "Warning: prior web assets were preserved at $BACKUP_DIR/html" >&2
    fi
  fi

  exit "$status"
}

require_command go
require_command node
require_command npm
require_command mktemp

cd "$REPO_ROOT"

echo "Building frontend..."
(
  cd "$FRONTEND_DIR"
  npm ci
  npm run lint -- --max-warnings=0
  npm run test
  npm run build
  npm run verify:dist
)

# Copy the verified production output into a sibling staging directory first.
# The embedded assets are not touched unless every frontend gate and this copy
# succeed.
mkdir -p "$WEB_DIR"
STAGE_DIR=$(mktemp -d "$WEB_DIR/.html-stage.XXXXXX") || die "failed to create frontend staging directory"
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
cp -R "$FRONTEND_DIST"/. "$STAGE_DIR"/ || die "failed to stage frontend production assets"

if [ -e "$WEB_HTML" ] || [ -L "$WEB_HTML" ]; then
  BACKUP_DIR=$(mktemp -d "$WEB_DIR/.html-backup.XXXXXX") || die "failed to create web asset backup directory"
  mv "$WEB_HTML" "$BACKUP_DIR/html" || die "failed to preserve prior web assets"
fi


if ! mv "$STAGE_DIR" "$WEB_HTML"; then
  die "failed to replace embedded web assets"
fi
STAGE_DIR=

if [ -n "$BACKUP_DIR" ]; then
  rm -rf "$BACKUP_DIR" || die "failed to remove prior web asset backup"
  BACKUP_DIR=
fi

trap - EXIT HUP INT TERM

echo "Building backend..."
BUILD_TAGS="with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_musl,badlinkname,tfogo_checklinkname0,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_call,with_ccm,with_ocm,with_oomkiller"

# Embed the exact release artifact platform suffix so the in-panel self-update
# selects the artifact for the architecture actually requested through Go's
# target environment.
ARCH=$(go env GOARCH) || die "failed to determine GOARCH"
case "$ARCH" in
  amd64|arm64|386|s390x)
    ARTIFACT_PLATFORM=$ARCH
    ;;
  arm)
    ARM_VERSION=$(go env GOARM) || die "failed to determine GOARM"
    ARM_VERSION=${ARM_VERSION%%,*}
    case "$ARM_VERSION" in
      5|6|7) ARTIFACT_PLATFORM=armv$ARM_VERSION ;;
      *) die "unsupported GOARM value: $ARM_VERSION (expected 5, 6, or 7)" ;;
    esac
    ;;
  *)
    die "unsupported GOARCH value: $ARCH"
    ;;
esac

LDFLAGS="-w -s -checklinkname=0 -extldflags \"-Wl,-no_warn_duplicate_libraries\" -X github.com/deposist/s-ui-x-extended/config.ArtifactPlatform=${ARTIFACT_PLATFORM}"
go build -ldflags "$LDFLAGS" -tags "$BUILD_TAGS" -o "$REPO_ROOT/sui" "$REPO_ROOT/main.go"
