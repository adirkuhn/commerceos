#!/usr/bin/env bash
# Deploys a published CommerceOS version (backend, frontend and storefront
# images share one version). Lives next to docker-compose.yml and .env on the
# server (e.g. /var/www/commerceos) and runs from there.
#
#   deploy.sh 1.2.3          # deploy a release
#   deploy.sh rollback       # go back to the version this one replaced
#   ssh deploy@server 1.2.3  # from CI, via an authorized_keys forced command:
#     restrict,command="/var/www/commerceos/deploy.sh" ssh-ed25519 AAAA... github-actions-deploy
#
# Takes a version (or "rollback") only: the caller cannot change the compose
# file or run anything else.
#
# Pulls first (.env changes only after a good pull), records the version in
# .env -- COMMERCEOS_VERSION, and the one it replaced in
# COMMERCEOS_PREVIOUS_VERSION -- then starts it (migrate runs before the app
# services) and waits until they are up. If the new version fails to start,
# the previous one is started again automatically. Rollback starts from the
# images already on the server when the registry can't be reached. Every
# deploy is logged in .deploy-history.
#
# Rolling back never undoes migrations: an older image's `goose up` finds
# nothing to apply on a newer database. So migrations must stay backward
# compatible (add, don't drop/rename in the same release).
#
# Exit codes: 0 deployed, 1 pull/start failed, 2 invalid input or setup,
# 3 another deploy running.
set -euo pipefail

main() {
	local dir arg version current previous mode
	umask 007
	dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
	arg=${1-${SSH_ORIGINAL_COMMAND-}}
	ENV_FILE=$dir/.env

	if [[ $arg != rollback && ! $arg =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
		echo "deploy: invalid argument '$arg' (expected a release like 1.2.3, or rollback)" >&2
		return 2
	fi
	# Compose reads .env (and env_file) from the compose file's own folder, so
	# both must sit next to this script.
	local f
	for f in "$ENV_FILE" "$dir/docker-compose.yml"; do
		if [[ ! -f $f ]]; then
			echo "deploy: $f not found (docker-compose.yml, .env and deploy.sh must be in one folder)" >&2
			return 2
		fi
	done

	exec 9>"$dir/.deploy.lock"
	if ! flock -n 9; then
		echo "deploy: another deploy is running" >&2
		return 3
	fi
	cd "$dir"

	current=$(env_value COMMERCEOS_VERSION)
	previous=$(env_value COMMERCEOS_PREVIOUS_VERSION)
	if [[ $arg == rollback ]]; then
		mode=rollback
		version=$previous
		if [[ -z $version ]]; then
			echo "deploy: nothing to roll back to (COMMERCEOS_PREVIOUS_VERSION is not set)" >&2
			return 2
		fi
	else
		mode=deploy
		version=$arg
	fi
	# Redeploying the running version must not forget the real previous one.
	if [[ $version != "$current" ]]; then
		previous=$current
	fi

	local up_args=()
	log "pulling $version"
	# The environment overrides .env for compose interpolation, so .env
	# changes only after a good pull.
	if ! COMMERCEOS_VERSION=$version docker compose pull; then
		if [[ $mode != rollback ]]; then
			log "pull failed; still on $current"
			return 1
		fi
		log "pull failed; rolling back with the images already on this server"
		up_args=(--pull never)
	fi

	write_versions "$version" "$previous" || return 2

	log "starting $version"
	if start "$version" "${up_args[@]}"; then
		if [[ $mode == rollback ]]; then
			history "rolled back to $version (was $current)"
		else
			history "deployed $version (was $current)"
		fi
		log "deployed $version"
		return 0
	fi

	log "$version failed to start; services:"
	docker compose ps -a || true
	docker compose logs --no-color --tail 50 || true
	if [[ $mode == rollback || -z $current || $current == "$version" ]]; then
		log "not rolling back automatically; deploy a known good version by hand (deploy.sh <version>)"
		return 1
	fi

	log "rolling back to $current"
	write_versions "$current" "$version" || return 2
	if start "$current" --pull never; then
		history "auto rolled back to $current ($version failed to start)"
		log "rolled back to $current; $version did not deploy"
	else
		history "$version failed to start; rollback to $current failed too"
		log "rollback to $current failed too; deploy a known good version by hand (deploy.sh <version>)"
	fi
	return 1
}

start() {
	local version=$1
	shift
	COMMERCEOS_VERSION=$version docker compose up -d --no-build --wait --wait-timeout 300 "$@"
}

env_value() { grep "^$1=" "$ENV_FILE" | tail -1 | cut -d= -f2- || true; }

# write_versions replaces COMMERCEOS_VERSION/COMMERCEOS_PREVIOUS_VERSION in
# .env, keeping the file's owner group and mode (others never get access).
write_versions() {
	local tmp
	tmp=$(mktemp "$ENV_FILE.XXXXXX")
	grep -v -e '^COMMERCEOS_VERSION=' -e '^COMMERCEOS_PREVIOUS_VERSION=' "$ENV_FILE" >"$tmp" || true
	printf 'COMMERCEOS_VERSION=%s\n' "$1" >>"$tmp"
	[[ -n $2 ]] && printf 'COMMERCEOS_PREVIOUS_VERSION=%s\n' "$2" >>"$tmp"
	if ! { chgrp --reference="$ENV_FILE" "$tmp" && chmod --reference="$ENV_FILE" "$tmp" && chmod o-rwx "$tmp"; }; then
		rm -f "$tmp"
		log "cannot copy owner group/mode of $ENV_FILE to its replacement; nothing changed"
		return 1
	fi
	mv "$tmp" "$ENV_FILE"
}

log() { echo "$(date -u +%FT%TZ) deploy: $*"; }
history() { echo "$(date -u +%FT%TZ) $*" >>.deploy-history; }

# Parsed in full before running, so the script can be replaced while a deploy runs.
main "$@"
exit
