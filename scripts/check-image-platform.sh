#!/usr/bin/env bash
# Checks that a locally loaded CommerceOS image is built for the given
# platform and that its programs actually run there (each prints its own
# output, not "exec format error"). A foreign platform needs QEMU binfmt
# (docker/setup-qemu-action in CI).
# Usage: scripts/check-image-platform.sh backend|frontend|storefront <image:tag> linux/arm64
set -u
NAME=$1
IMAGE=$2
PLATFORM=$3
fail=0
check() { if [[ "$3" == *"$2"* ]]; then echo "ok   $1"; else echo "FAIL $1: expected '$2' in '$3'"; fail=1; fi; }
run() { docker run --rm --pull never --platform "$PLATFORM" --entrypoint "$1" "$IMAGE" "${@:2}" 2>&1 | head -3; }

check "$NAME image platform" "$PLATFORM" "$(docker image inspect "$IMAGE" --format '{{.Os}}/{{.Architecture}}' 2>&1)"
case $NAME in
backend)
	for bin in /api /relay /scheduler /webhook-processor /webhook-forwarder; do
		check "$bin runs" "required env var" "$(run "$bin")"
	done
	check "goose runs"       "goose version" "$(run /goose --version)"
	check "psql runs"        "psql (PostgreSQL)" "$(run psql --version)"
	check "has migrations"   "001_outbox.sql" "$(run ls /migrations)"
	;;
frontend)
	check "nginx config ok"  "successful" "$(run nginx -t)"
	;;
storefront)
	check "node runs"        "v24" "$(run node --version)"
	check "has SSR bundle"   "server" "$(run ls /app/dist)"
	;;
*)
	echo "unknown image $NAME" >&2
	exit 2
	;;
esac
exit $fail
