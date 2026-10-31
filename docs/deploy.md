# Deploying CommerceOS

Production runs images that CI builds and publishes. Nothing is built on the server.

```
git tag v1.2.3 ──► CI: tests ──► arm64 images ──► GHCR ──► approval ──► ssh deploy@server 1.2.3
                                                                          └─ deploy.sh: pull, migrate, start, wait
```

- **CI** (`.github/workflows/ci.yml`) runs on every push to `master`, every pull request and every `v*` tag:
  - backend: vet, generated-code check, unit tests and integration tests
  - admin frontend and storefront: i18n check, tests, type check and build
  - `deploy.sh` tests and a check that the production compose file is valid
- **Images:** only code that passed those checks gets built. CI builds `commerceos-backend`, `commerceos-frontend` and `commerceos-storefront` for `linux/arm64`, the Oracle Ampere server. Pushes and tags publish them to `ghcr.io/adirkuhn/`, tagged with the version (`1.2.3`, `1.2`), the branch (`master`) and the commit (`sha-…`).
- **Deploy** (`.github/workflows/deploy.yml`) runs for a release tag after the images are published. It waits for approval in the `production` environment, then connects over SSH with a key that can only run `deploy/prod/deploy.sh`.
- **`deploy.sh <version>`:**
  1. pulls the images
  2. records the version in `.env` and keeps the version it replaced
  3. starts the stack: `migrate` first, then the app services only if migrate succeeded
  4. waits until everything is up

  If the new version fails to start, the previous version is started again automatically.

## Releasing

```bash
git tag v1.2.3
git push origin v1.2.3
```

Then go to GitHub → Actions → the CI run → **Review deployments** → approve `production`.

## Rolling back

Any of these works:

- **GitHub:** Actions → **Deploy** → Run workflow → version `rollback`, to go back to the version the current one replaced. You can also enter any published release, e.g. `1.2.2`.
- **On the server:** `cd /var/www/commerceos && ./deploy.sh rollback`, or `./deploy.sh 1.2.2`.
- **Automatic:** a deploy whose new version fails to start puts the previous version back by itself and reports the deploy as failed.

A rollback starts from images already on the server if the registry can't be reached. Every deploy and rollback is logged in `.deploy-history`.

**Migrations are never rolled back.** The older image's `goose up` finds nothing to apply on the newer database and simply continues. So every migration must keep working with the previous release:

- add columns or tables; don't drop or rename them in the same release
- when something has to be dropped or renamed, stop using it in release N and remove it in release N+1

## One-time setup

### GitHub

1. **Settings → Environments → New environment** `production`.
   - Required reviewers: yourself.
   - Deployment branches and tags: tags matching `v*`.
2. **Secrets for the `production` environment:**

   | Secret | Value |
   |---|---|
   | `DEPLOY_HOST` | server hostname or IP |
   | `DEPLOY_PORT` | SSH port (optional, default 22) |
   | `DEPLOY_USER` | `deploy` |
   | `DEPLOY_SSH_KEY` | private key of a new key pair made only for this: `ssh-keygen -t ed25519 -f commerceos-deploy -N ''` |
   | `DEPLOY_KNOWN_HOSTS` | output of `ssh-keyscan -p <port> <host>` |

3. **Settings → Secrets and variables → Actions → Variables** (repository level). Use the same values as the `VITE_*` lines in the server's current `.env`. They're compiled into the admin frontend image.
   - `VITE_API_BASE_URL`
   - `VITE_LM2_LOGIN_URL`
   - `VITE_LM2_CALLBACK_PATH`
   - `VITE_STOREFRONT_URL`
   - `VITE_DESIGN_DEFAULT` (optional)

### Server

These steps work on any Docker host with Traefik (network `traefik-public`), Postgres (container `postgres` on network `database`) and MinIO (network `minio`) already running.

`/var/www/commerceos` becomes the deploy folder: `docker-compose.yml` (the production one), `deploy.sh`, `nats.conf` and `.env` side by side. Compose reads `.env` from the compose file's own folder, so they must stay together (`deploy.sh` refuses to run otherwise). The server needs no source checkout: CI builds the images.

1. **Deploy user.** One `deploy` user can serve several projects. Each project gets its **own SSH key**, and each key is locked to that project's `deploy.sh` (step 4). If the user already exists (e.g. for akache), skip to the group lines. The `docker` group is root-equivalent on that host.
   ```bash
   sudo adduser --disabled-password --gecos '' deploy   # only if it doesn't exist yet
   sudo usermod -aG docker deploy
   sudo groupadd -f commerceos && sudo usermod -aG commerceos deploy && sudo usermod -aG commerceos ubuntu
   ```
2. **Deploy folder.** Move the old source checkout out of the way, then fill the folder from it. `-L` copies `nats.conf` instead of the repo's symlink.
   ```bash
   sudo mv /var/www/commerceos /var/www/commerceos-src
   cd /var/www/commerceos-src && git pull
   sudo install -d -o deploy -g commerceos -m 2770 /var/www/commerceos
   cp -L deploy/prod/docker-compose.yml deploy/prod/deploy.sh deploy/prod/nats.conf /var/www/commerceos/
   cp .env /var/www/commerceos/.env          # the existing production settings
   cd /var/www/commerceos
   echo 'COMMERCEOS_VERSION=1.0.0' >> .env   # the first release you tag
   chgrp commerceos .env deploy.sh docker-compose.yml nats.conf && chmod 660 .env
   ```
   Compare `.env` with `deploy/prod/.env.example`: `HTTP_PORT` must be `80`, and anything the example lists must be set.
3. **GHCR login as `deploy`.** The images are private. A classic GitHub token with only `read:packages` covers every package you own, so if `deploy` is already logged in to `ghcr.io` for another of your projects, skip this.
   ```bash
   sudo -iu deploy docker login ghcr.io -u adirkuhn   # paste the token as the password
   ```
4. **Forced command: a second key, next to the other projects' lines.** Make a key just for CommerceOS (`ssh-keygen -t ed25519 -f commerceos-deploy -N ''`), put the private key in this repo's `DEPLOY_SSH_KEY` secret, and **add** a line to `/home/deploy/.ssh/authorized_keys` (directory `700`, file `600`, owned by `deploy`). Keep the existing lines:
   ```
   restrict,command="/var/www/akache/deploy.sh" ssh-ed25519 AAAA…(akache key) github-actions-deploy
   restrict,command="/var/www/commerceos/deploy.sh" ssh-ed25519 AAAA…(commerceos-deploy.pub) github-actions-commerceos
   ```
   SSH picks the line by the key that connects, so each repo's key can only run its own `deploy.sh <version>` or `deploy.sh rollback`.

### Switching over from the source-built stack

`deploy/prod/docker-compose.yml` uses the same project name (`commerceos`) as the old checkout. Volumes (`nats_data`, `media_uploads`), networks and Traefik routes carry over, and the containers are replaced in place.

1. Tag the first release: `git tag v1.0.0 && git push origin v1.0.0`. Let CI publish the images. Don't approve that run's deployment yet.
2. Deploy by hand once:
   ```bash
   sudo -iu deploy sh -c 'cd /var/www/commerceos && ./deploy.sh 1.0.0'
   ```
3. Check the site. From then on, never run `docker compose` in `/var/www/commerceos-src`: it would build from source over the published images. That checkout is only where updated deploy files come from (or delete it and copy them from GitHub).

## Day to day

- **The deploy files changed** (`docker-compose.yml`, `nats.conf` or `deploy.sh` in `deploy/prod/`): copy them from `/var/www/commerceos-src` (after `git pull`) to `/var/www/commerceos` before the release that needs them. `deploy.sh` only ever changes the version.
- **One-off commands** (repair or backfill programs in the backend image):
  ```bash
  cd /var/www/commerceos
  docker compose run --rm --no-deps backend /repair-unmanaged-woo-stock
  ```
- **What's running:** `grep COMMERCEOS_ .env`, `tail .deploy-history`, `docker compose ps`.
