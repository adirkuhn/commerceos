#!/usr/bin/env bash
# Tests deploy.sh with a fake `docker` on PATH that records its calls.
# Usage: deploy/prod/deploy_test.sh
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
fail=0
tmpdirs=()
trap 'rm -rf "${tmpdirs[@]}"' EXIT
check() { if [[ "$2" == "$3" ]]; then echo "ok   $1"; else echo "FAIL $1: expected '$2' got '$3'"; fail=1; fi; }

# setup: fresh deploy dir with deploy.sh, a .env on 1.0.0 (previous 0.9.0)
# and a fake docker that logs "<args>@<COMMERCEOS_VERSION>".
#   FAKE_FAIL=<word>      fails any call containing that word (e.g. pull, up)
#   FAKE_BAD_VERSION=<v>  fails `compose up` only while starting version v
setup() {
	T=$(mktemp -d)
	tmpdirs+=("$T")
	mkdir "$T/bin"
	cp "$HERE/deploy.sh" "$T/"
	printf 'API_DOMAIN=x.test\nCOMMERCEOS_VERSION=1.0.0\nCOMMERCEOS_PREVIOUS_VERSION=0.9.0\nJWT_SECRET=secret\n' >"$T/.env"
	cat >"$T/bin/docker" <<'FAKE'
#!/usr/bin/env bash
echo "$*@${COMMERCEOS_VERSION-}" >>"$CALLS"
[[ -n "${FAKE_FAIL:-}" && " $* " == *" $FAKE_FAIL "* ]] && exit 1
[[ -n "${FAKE_BAD_VERSION:-}" && " $* " == *" up "* && "${COMMERCEOS_VERSION-}" == "$FAKE_BAD_VERSION" ]] && exit 1
exit 0
FAKE
	chmod +x "$T/bin/docker"
	export CALLS=$T/calls
	: >"$CALLS"
}
deploy() { PATH="$T/bin:$PATH" "$T/deploy.sh" "$@" >"$T/out" 2>&1; }
env_value() { grep "^$1=" "$T/.env" | cut -d= -f2; }
version() { env_value COMMERCEOS_VERSION; }
previous() { env_value COMMERCEOS_PREVIOUS_VERSION; }
calls() { tr '\n' '|' <"$CALLS"; }
UP="compose up -d --no-build --wait --wait-timeout 300"

# --- deploy a release ---
setup
deploy 1.2.3
check "deploys a release version"         0 $?
check "sets COMMERCEOS_VERSION"           1.2.3 "$(version)"
check "remembers the version it replaced" 1.0.0 "$(previous)"
check "keeps other settings"              secret "$(env_value JWT_SECRET)"
check "pulls then starts the new version" "compose pull@1.2.3|$UP@1.2.3|" "$(calls)"
check "logs the deploy in history"        1 "$(grep -c ' deployed 1.2.3 (was 1.0.0)$' "$T/.deploy-history")"

setup
deploy 1.0.0
check "redeploying the running version keeps previous" 0.9.0 "$(previous)"

setup
PATH="$T/bin:$PATH" SSH_ORIGINAL_COMMAND=2.0.0-rc.1 "$T/deploy.sh" >"$T/out" 2>&1
check "reads version from SSH forced command" 0 $?
check "accepts pre-release"               2.0.0-rc.1 "$(version)"

# --- rollback ---
setup
deploy rollback
check "rollback succeeds"                 0 $?
check "rollback runs the previous version" 0.9.0 "$(version)"
check "rollback can be undone"            1.0.0 "$(previous)"
check "rollback pulls then starts"        "compose pull@0.9.0|$UP@0.9.0|" "$(calls)"
check "logs the rollback in history"      1 "$(grep -c ' rolled back to 0.9.0 (was 1.0.0)$' "$T/.deploy-history")"

setup
PATH="$T/bin:$PATH" SSH_ORIGINAL_COMMAND=rollback "$T/deploy.sh" >"$T/out" 2>&1
check "rollback over SSH forced command"  0.9.0 "$(version)"

setup
FAKE_FAIL=pull deploy rollback
check "rollback works with registry down" 0 $?
check "registry down: starts local images" "compose pull@0.9.0|$UP --pull never@0.9.0|" "$(calls)"

setup
sed -i '/^COMMERCEOS_PREVIOUS_VERSION=/d' "$T/.env"
deploy rollback
check "rollback without previous fails"   2 $?
check "rollback without previous runs no docker" "" "$(calls)"

# --- failures ---
setup
FAKE_FAIL=pull deploy 1.2.3
check "pull failure fails the deploy"     1 $?
check "pull failure keeps old version"    1.0.0 "$(version)"
check "pull failure keeps previous"       0.9.0 "$(previous)"
check "pull failure does not restart"     "compose pull@1.2.3|" "$(calls)"

setup
FAKE_BAD_VERSION=1.2.3 deploy 1.2.3
check "failed start fails the deploy"     1 $?
check "failed start rolls back automatically" 1.0.0 "$(version)"
check "auto rollback keeps the bad version as previous" 1.2.3 "$(previous)"
check "auto rollback restarts the old local images" 1 "$(grep -c "^$UP --pull never@1.0.0$" "$CALLS")"
check "failed start prints status"        1 "$(grep -c '^compose ps -a@' "$CALLS")"
check "logs the auto rollback"            1 "$(grep -c ' auto rolled back to 1.0.0 (1.2.3 failed to start)$' "$T/.deploy-history")"

setup
FAKE_FAIL=up deploy 1.2.3
check "old version failing too still fails" 1 $?
check "says both failed"                  1 "$(grep -c 'rollback to 1.0.0 failed too' "$T/out")"

setup
FAKE_BAD_VERSION=0.9.0 deploy rollback
check "failed rollback does not bounce back" 1 $?
check "failed rollback starts once"       1 "$(grep -c '^compose up' "$CALLS")"

# --- input and setup ---
for bad in "" "latest" "v1.2.3" "1.2" "1.2.3; rm -rf /" "1.2.3 --build" '$(id)' "../1.2.3" "rollback now"; do
	setup
	deploy "$bad"
	check "rejects '$bad'"                  2 $?
	check "'$bad' leaves .env untouched"    1.0.0 "$(version)"
	check "'$bad' runs no docker"           "" "$(calls)"
done

setup
chmod 660 "$T/.env"
deploy 1.2.3
check ".env keeps group read/write"       660 "$(stat -c %a "$T/.env")"

setup
chmod 664 "$T/.env"
deploy 1.2.3
check ".env loses access for others"      660 "$(stat -c %a "$T/.env")"

setup
exec 9>"$T/.deploy.lock"; flock 9
deploy 1.2.3
check "refuses concurrent deploy"         3 $?
exec 9>&-
check "concurrent deploy changes nothing" 1.0.0 "$(version)"

setup
rm "$T/.env"
deploy 1.2.3
check "missing .env fails"                2 $?

exit $fail
