#!/bin/sh
# Writes the admin's runtime settings to config.js from the container's
# environment (the server's .env, via env_file) -- see src/config/appEnv.ts.
# Runs at container start: the nginx image executes /docker-entrypoint.d/*.sh
# before nginx starts.
#
# Only the public settings listed below are published: the container
# receives the whole .env, secrets included, and config.js is served to any
# browser. Blank or unset settings are left out (the build-time value applies).
#
# Usage: write-config.sh [output]   (default /usr/share/nginx/html/config.js)
set -eu

out=${1:-/usr/share/nginx/html/config.js}
json=""
# COMMERCEOS_VERSION is the deployed release (deploy.sh writes it to .env
# before the containers start), shown in the sidebar.
for key in VITE_API_BASE_URL VITE_LM2_LOGIN_URL VITE_LM2_CALLBACK_PATH VITE_STOREFRONT_URL VITE_DESIGN_DEFAULT COMMERCEOS_VERSION; do
	eval "value=\${$key:-}"
	[ -n "$value" ] || continue
	escaped=$(printf '%s' "$value" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g')
	json="$json${json:+,}\"$key\":\"$escaped\""
done
printf 'window.__COMMERCEOS_CONFIG__ = {%s}\n' "$json" >"$out"
