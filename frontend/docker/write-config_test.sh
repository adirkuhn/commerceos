#!/usr/bin/env bash
# Tests write-config.sh. Usage: frontend/docker/write-config_test.sh
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
fail=0
T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
check() { if [[ "$2" == "$3" ]]; then echo "ok   $1"; else echo "FAIL $1: expected '$2' got '$3'"; fail=1; fi; }
# run VAR=value... -- runs the script with only these variables set.
run() { env -i PATH="$PATH" "$@" sh "$HERE/write-config.sh" "$T/config.js" >"$T/out" 2>&1; }

run VITE_API_BASE_URL=https://api.example/api/v1 VITE_LM2_LOGIN_URL=https://lm2.example/app-login \
	VITE_LM2_CALLBACK_PATH=/auth/callback VITE_STOREFRONT_URL=https://shop.example VITE_DESIGN_DEFAULT=classic
check "succeeds"                0 $?
check "writes every setting" \
	'window.__COMMERCEOS_CONFIG__ = {"VITE_API_BASE_URL":"https://api.example/api/v1","VITE_LM2_LOGIN_URL":"https://lm2.example/app-login","VITE_LM2_CALLBACK_PATH":"/auth/callback","VITE_STOREFRONT_URL":"https://shop.example","VITE_DESIGN_DEFAULT":"classic"}' \
	"$(cat "$T/config.js")"

run VITE_API_BASE_URL=https://api.example/api/v1 JWT_SECRET=s3cret S3_SECRET_ACCESS_KEY=k3y APP_DB_PASSWORD=pw
check "publishes only the public settings, never secrets" \
	'window.__COMMERCEOS_CONFIG__ = {"VITE_API_BASE_URL":"https://api.example/api/v1"}' "$(cat "$T/config.js")"

run
check "nothing set: empty config (build-time values apply)" 'window.__COMMERCEOS_CONFIG__ = {}' "$(cat "$T/config.js")"

run VITE_API_BASE_URL= VITE_LM2_CALLBACK_PATH=/cb
check "blank values are left out" 'window.__COMMERCEOS_CONFIG__ = {"VITE_LM2_CALLBACK_PATH":"/cb"}' "$(cat "$T/config.js")"

run 'VITE_STOREFRONT_URL=https://x.example/a"b\c'
check "quotes and backslashes are escaped" \
	'window.__COMMERCEOS_CONFIG__ = {"VITE_STOREFRONT_URL":"https://x.example/a\"b\\c"}' "$(cat "$T/config.js")"

exit $fail
