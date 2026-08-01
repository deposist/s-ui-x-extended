#!/usr/bin/env bash
# Build and readiness-smoke the platforms actually published for Compose.
#
# Repository Linux VM/release matrix and OCI mapping:
#   amd64 -> linux/amd64       arm64 -> linux/arm64
#   armv7 -> linux/arm/v7      armv6 -> linux/arm/v6
#   386   -> linux/386         armv5 -> release artifact only
#   s390x -> release artifact only
# GHCR/Compose provider images are published only for linux/amd64 and
# linux/arm64. The other mappings describe release artifacts or optional local
# Dockerfile feasibility; they must not be implied to exist in the image index.

set -Eeuo pipefail

readonly ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
readonly COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"
readonly NFTABLES_FILE="$ROOT_DIR/docker-compose.nftables.yml"
readonly DOCKERFILE="$ROOT_DIR/Dockerfile"
readonly LINUX_VM_MATRIX="amd64 arm64 armv7 armv6 armv5 386 s390x"
readonly PROVIDER_PLATFORMS_DEFAULT="linux/amd64 linux/arm64"
readonly EMULATION_MODE="${SUI_DOCKER_EMULATION:-required}"
readonly SMOKE_TIMEOUT="${SUI_DOCKER_SMOKE_TIMEOUT:-180}"
readonly LOG_DIR="${SUI_DOCKER_LOG_DIR:-${TMPDIR:-/tmp}}"
readonly IMAGE_PREFIX="${SUI_DOCKER_TEST_IMAGE:-s-ui-compose-smoke}"

# Set SUI_DOCKER_FOCUSED=1 for a non-gating local run that may override/skip
# platforms. The default gate rejects those shortcuts and must smoke both
# published provider platforms.
readonly FOCUSED_MODE="${SUI_DOCKER_FOCUSED:-0}"
[[ $FOCUSED_MODE == 0 || $FOCUSED_MODE == 1 ]] || { printf 'ERROR: SUI_DOCKER_FOCUSED must be 0 or 1\n' >&2; exit 1; }
if [[ $FOCUSED_MODE == 1 ]]; then
    platform_input=${SUI_DOCKER_PLATFORMS:-$PROVIDER_PLATFORMS_DEFAULT}
else
    [[ -z ${SUI_DOCKER_PLATFORMS:-} ]] || { printf 'ERROR: SUI_DOCKER_PLATFORMS requires SUI_DOCKER_FOCUSED=1\n' >&2; exit 1; }
    [[ $EMULATION_MODE == required ]] || { printf 'ERROR: a gating run requires SUI_DOCKER_EMULATION=required\n' >&2; exit 1; }
    platform_input=$PROVIDER_PLATFORMS_DEFAULT
fi
platform_input=${platform_input//,/ }
read -r -a provider_platforms <<<"$platform_input"

current_override=
current_project=
current_container=
current_image=
current_image_created=false

die() {
    printf 'ERROR: %s\n' "$*" >&2
    exit 1
}

cleanup() {
    local status=$?
    set +e
    if [[ -n $current_override && -f $current_override ]]; then
        if [[ $status -ne 0 && -n $current_project ]]; then
            docker compose --project-name "$current_project" \
                --file "$COMPOSE_FILE" --file "$current_override" logs --no-color >&2
        fi
        if [[ -n $current_project ]]; then
            docker compose --project-name "$current_project" \
                --file "$COMPOSE_FILE" --file "$current_override" down --volumes --remove-orphans >/dev/null 2>&1
        fi
        rm -f -- "$current_override"
    fi
    if [[ $current_image_created == true && ${SUI_DOCKER_KEEP_IMAGES:-0} != 1 && -n $current_image ]]; then
        docker image rm --force "$current_image" >/dev/null 2>&1
    fi
    exit "$status"
}
trap cleanup EXIT

command -v docker >/dev/null 2>&1 || die "docker is required"
command -v grep >/dev/null 2>&1 || die "grep is required"
command -v tee >/dev/null 2>&1 || die "tee is required"
[[ $SMOKE_TIMEOUT =~ ^[1-9][0-9]*$ ]] || die "SUI_DOCKER_SMOKE_TIMEOUT must be a positive integer"
case "$EMULATION_MODE" in
    required|auto|off) ;;
    *) die "SUI_DOCKER_EMULATION must be required, auto, or off" ;;
esac
((${#provider_platforms[@]} > 0)) || die "no Docker provider platforms selected"

native_platform=
case "$(uname -m)" in
    x86_64|amd64) native_platform=linux/amd64 ;;
    aarch64|arm64) native_platform=linux/arm64 ;;
    armv7l) native_platform=linux/arm/v7 ;;
    armv6l) native_platform=linux/arm/v6 ;;
    i386|i486|i586|i686) native_platform=linux/386 ;;
    s390x) native_platform=linux/s390x ;;
esac

printf '==> Linux VM/release architecture matrix: %s\n' "$LINUX_VM_MATRIX"
printf '==> Published Compose/GHCR provider platforms: %s\n' "$PROVIDER_PLATFORMS_DEFAULT"
printf '==> Host platform: %s; emulation policy: %s\n' "${native_platform:-unknown}" "$EMULATION_MODE"

docker compose version >/dev/null
base_config=$(docker compose --file "$COMPOSE_FILE" config)
if grep -Eq '^[[:space:]]+platform:' <<<"$base_config"; then
    die "base Compose must select its native image manifest without a platform override"
fi
grep -Fq 'healthcheck:' <<<"$base_config" || die "base Compose service has no healthcheck"
for mountpoint in /app/db /app/cert /app/logs; do
    grep -Fq "target: $mountpoint" <<<"$base_config" || die "base Compose does not persist $mountpoint"
done
grep -Fq 'no-new-privileges:true' <<<"$base_config" || die "base Compose lacks no-new-privileges"
grep -Eq '^[[:space:]]+cap_drop:$' <<<"$base_config" || die "base Compose has no cap_drop policy"
grep -Eq '^[[:space:]]+- ALL$' <<<"$base_config" || die "base Compose does not drop all capabilities"
if grep -Eq '(^|[[:space:]])privileged:[[:space:]]*true|^[[:space:]]*cap_add:|^[[:space:]]*user:[[:space:]]*(root|0)([[:space:]]|$)' <<<"$base_config"; then
    die "base Compose regressed to privileged/root/capability execution"
fi

nftables_config=$(docker compose --file "$COMPOSE_FILE" --file "$NFTABLES_FILE" --profile nftables config)
nft_cap_add=$(grep -A1 -E '^[[:space:]]+cap_add:$' <<<"$nftables_config" || true)
[[ $(grep -Ec '^[[:space:]]+- ' <<<"$nft_cap_add") == 1 ]] || die "nftables opt-in must add exactly one capability"
grep -Eq '^[[:space:]]+- NET_ADMIN$' <<<"$nft_cap_add" || die "nftables opt-in capability is not NET_ADMIN"
if grep -Eq '(^|[[:space:]])privileged:[[:space:]]*true|SYS_ADMIN|NET_RAW|^[[:space:]]*user:[[:space:]]*(root|0)([[:space:]]|$)' <<<"$nftables_config"; then
    die "nftables opt-in grants more privilege than NET_ADMIN"
fi
printf '==> Compose base and nftables opt-in configurations are valid\n'

builder_platforms=$(docker buildx inspect --bootstrap 2>/dev/null | grep -E '^Platforms:' || true)

platform_is_published() {
    case "$1" in
        linux/amd64|linux/arm64) return 0 ;;
        *) return 1 ;;
    esac
}

builder_has_platform() {
    local escaped=${1//\//\\/}
    grep -Eq "(^|[[:space:],])${escaped}(\\*|[[:space:],]|$)" <<<"$builder_platforms"
}

wait_for_health() {
    local container=$1 deadline=$((SECONDS + SMOKE_TIMEOUT)) health
    while ((SECONDS < deadline)); do
        health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' "$container" 2>/dev/null || true)
        case "$health" in
            healthy) return 0 ;;
            unhealthy|missing)
                docker inspect --format '{{range .State.Health.Log}}{{println .Output}}{{end}}' "$container" >&2 2>/dev/null || true
                return 1
                ;;
        esac
        sleep 2
    done
    docker inspect --format '{{range .State.Health.Log}}{{println .Output}}{{end}}' "$container" >&2 2>/dev/null || true
    return 1
}

smoked=0
for platform in "${provider_platforms[@]}"; do
    platform_is_published "$platform" || die "$platform is not in the published Compose/GHCR provider matrix"

    if [[ $platform != "$native_platform" ]]; then
        if [[ $EMULATION_MODE == off ]]; then
            printf '==> Skipping non-native %s because emulation is disabled\n' "$platform"
            continue
        fi
        if ! builder_has_platform "$platform"; then
            if [[ $EMULATION_MODE == auto ]]; then
                printf '==> Skipping %s: the current builder does not advertise it\n' "$platform"
                continue
            fi
            die "$platform needs preconfigured BuildKit/QEMU emulation; refusing to install it with privileged containers"
        fi
        printf '==> %s will be built and run through preconfigured emulation\n' "$platform"
    fi

    suffix=${platform//\//-}
    current_image="${IMAGE_PREFIX}:${suffix}"
    build_log="$LOG_DIR/s-ui-docker-build-${suffix}.log"
    mkdir -p -- "$LOG_DIR"
    printf '==> Building %s as %s\n' "$platform" "$current_image"
    docker buildx build \
        --platform "$platform" \
        --load \
        --tag "$current_image" \
        --file "$DOCKERFILE" \
        --progress=plain \
        "$ROOT_DIR" 2>&1 | tee "$build_log"
    current_image_created=true
    [[ $(docker image inspect --format '{{.Config.User}}' "$current_image") == sui ]] || \
        die "$platform image default user is not sui"
    if [[ $platform != "$native_platform" ]]; then
        if ! docker run --rm --platform "$platform" --entrypoint /bin/true "$current_image"; then
            if [[ $EMULATION_MODE == auto ]]; then
                printf '==> Skipping %s smoke: runtime emulation is unavailable\n' "$platform"
                if [[ ${SUI_DOCKER_KEEP_IMAGES:-0} != 1 ]]; then
                    docker image rm --force "$current_image" >/dev/null
                fi
                current_image=
                current_image_created=false
                continue
            fi
            die "$platform image cannot execute; configure runtime emulation outside this unprivileged script"
        fi
    fi

    current_project="sui-smoke-${suffix//[^a-zA-Z0-9]/-}-$$"
    current_container="${current_project}-app"
    current_override=$(mktemp "${TMPDIR:-/tmp}/s-ui-compose-smoke.XXXXXX.yml")
    cat >"$current_override" <<EOF
services:
  s-ui:
    image: $current_image
    container_name: $current_container
    platform: $platform
volumes:
  s-ui-db:
    name: ${current_project}-db
  s-ui-certs:
    name: ${current_project}-certs
  s-ui-logs:
    name: ${current_project}-logs
EOF

    docker compose --project-name "$current_project" \
        --file "$COMPOSE_FILE" --file "$current_override" config >/dev/null
    docker compose --project-name "$current_project" \
        --file "$COMPOSE_FILE" --file "$current_override" up --detach --no-build --pull never
    wait_for_health "$current_container" || die "$platform application did not become ready"

    [[ $(docker inspect --format '{{.Config.User}}' "$current_container") == sui ]] || die "smoke container is not running as sui"
    [[ $(docker inspect --format '{{.HostConfig.Privileged}}' "$current_container") == false ]] || die "smoke container is privileged"
    cap_add=$(docker inspect --format '{{json .HostConfig.CapAdd}}' "$current_container")
    [[ $cap_add == null || $cap_add == '[]' ]] || die "default smoke container has added capabilities: $cap_add"
    cap_drop=$(docker inspect --format '{{json .HostConfig.CapDrop}}' "$current_container")
    grep -Fq 'ALL' <<<"$cap_drop" || die "default smoke container does not drop all capabilities: $cap_drop"
    security_opts=$(docker inspect --format '{{json .HostConfig.SecurityOpt}}' "$current_container")
    grep -Fq 'no-new-privileges:true' <<<"$security_opts" || die "smoke container lost no-new-privileges"

    docker compose --project-name "$current_project" \
        --file "$COMPOSE_FILE" --file "$current_override" exec --no-TTY s-ui sh -eu -c \
        'for dir in /app/db /app/cert /app/logs; do test -w "$dir"; : >"$dir/.compose-smoke"; done'
    # Prove data lives in volumes rather than the container writable layer: tear
    # the container down, retain volumes, recreate, and wait for readiness again.
    docker compose --project-name "$current_project" \
        --file "$COMPOSE_FILE" --file "$current_override" down --remove-orphans
    docker compose --project-name "$current_project" \
        --file "$COMPOSE_FILE" --file "$current_override" up --detach --no-build --pull never
    wait_for_health "$current_container" || die "$platform application was not ready after container recreation"
    docker compose --project-name "$current_project" \
        --file "$COMPOSE_FILE" --file "$current_override" exec --no-TTY s-ui sh -eu -c \
        'for dir in /app/db /app/cert /app/logs; do test -f "$dir/.compose-smoke"; done'

    printf '==> %s readiness, persistence, and least-privilege smoke passed (log: %s)\n' "$platform" "$build_log"
    docker compose --project-name "$current_project" \
        --file "$COMPOSE_FILE" --file "$current_override" down --volumes --remove-orphans
    rm -f -- "$current_override"
    current_override=
    current_project=
    current_container=
    if [[ ${SUI_DOCKER_KEEP_IMAGES:-0} != 1 ]]; then
        docker image rm --force "$current_image" >/dev/null
    fi
    current_image_created=false
    current_image=
    ((smoked += 1))
done

if [[ $FOCUSED_MODE == 1 ]]; then
    ((smoked > 0)) || die "no provider platform was smoke tested"
    printf '==> Focused local run completed %d provider-platform readiness smoke(s)\n' "$smoked"
else
    [[ $smoked == ${#provider_platforms[@]} ]] || die "gating run did not smoke every published provider platform"
    printf '==> Gating run completed both provider-platform readiness smokes without added capabilities\n'
fi
