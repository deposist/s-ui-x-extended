#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
REPO_ROOT=$SCRIPT_DIR

cd "$REPO_ROOT"
"$REPO_ROOT/build.sh"
SUI_DB_FOLDER="$REPO_ROOT/db" SUI_DEBUG=true exec "$REPO_ROOT/sui"
