#!/usr/bin/env bash
# Test Docker multi-platform build (linux/amd64, 386, arm64, arm/v7, arm/v6)

set -e
cd "$(dirname "$0")"

PLATFORMS="linux/amd64,linux/386,linux/arm64/v8,linux/arm/v7,linux/arm/v6"
echo "==> Testing Docker build platforms: $PLATFORMS"
docker buildx build \
  --platform "$PLATFORMS" \
  -f Dockerfile \
  --progress=plain \
  . 2>&1 | tee docker-build-test.log

echo "==> Done. See docker-build-test.log for full output."
