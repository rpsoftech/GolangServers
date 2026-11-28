# pm2go deployment

How to publish the hub (web dashboard) behind Caddy and how to put the agent and daemon on Linux hosts. Design and behaviour are in `pm2go.md`; what is done and open is in `pm2go-status.md`.

Files live in `servers/pm2go/deploy/`: `Dockerfile`, `docker-compose.yml`, `Caddyfile.host`, `Caddyfile.container`, `.env.example`, `Dockerfile.dockerignore`.

> The Docker and Caddy files were written on a laptop without Docker or Caddy and have not been run by hand. The `pm2go` GitHub workflow (`.github/workflows/pm2go.yml`) builds the image and smoke-tests `/healthz` and the health check on every pull request that touches pm2go, but a real deployment (Caddy, DNS, Telegram) is still unverified: do the verification steps at the end the first time (issue #106).

## Released artifacts

Pushing a tag `pm2go-v<version>` runs the `pm2go` workflow: tests, then a GitHub release with Linux binaries and a published hub image.

```bash
git tag pm2go-v0.1.0 && git push origin pm2go-v0.1.0
```

Pull the hub image (multi-arch amd64 and arm64):

```bash
docker pull ghcr.io/rpsoftech/pm2go-hub:pm2go-v0.1.0   # or :latest
docker run -d --name pm2go-hub -p 127.0.0.1:8080:8080 -v pm2go-hub-data:/data ghcr.io/rpsoftech/pm2go-hub:latest
```

The image tag equals the git tag. If the package is private, run `docker login ghcr.io` with a token that has `read:packages` first.

Download a binary for a monitored host and verify it:

```bash
TAG=pm2go-v0.1.0; ARCH=amd64    # or arm64
BASE=https://github.com/rpsoftech/GolangServers/releases/download/$TAG
curl -fLO $BASE/pm2go-linux-$ARCH && curl -fLO $BASE/SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing
sudo install -m 0755 pm2go-linux-$ARCH /usr/local/bin/pm2go
```

Pull request CI (same workflow): `gofmt -l` must be empty, `go vet`, `go test -race -count=1`, Linux build, image build and smoke test (container started, `/healthz` answers `ok`, first-run keys printed, Docker health status `healthy`).

## What you need

| Where | Needs |
|-------|-------|
| Hub server | Docker with the compose plugin, a DNS name pointing at the server, ports 80 and 443 open to the internet (Caddy gets the certificate on port 80/443) |
| Caddy | Already installed on the platform (option A), or the compose Caddy container (option B) |
| Each monitored host | Linux, systemd, the `pm2go` binary, outbound HTTPS to the hub. No inbound port |
| Telegram | A bot token from @BotFather and the chat ids to notify |

## 1. Build and start the hub

On the hub server, from a checkout of this repository:

```bash
cd servers/pm2go/deploy
cp .env.example .env        # set PM2GO_DOMAIN and PM2GO_VERSION
docker compose up -d --build
docker compose logs hub     # first start prints the admin password and agent key ONCE
```

The hub keeps everything (config, secrets, SQLite database) in the `hub-data` volume (`/data` in the container). Print the keys again any time:

```bash
docker compose exec hub pm2go hub --show-keys
```

## 2. Publish it with Caddy

### Option A: Caddy already runs on the host (your case)

The compose file binds the hub to `127.0.0.1:8080` only, so nothing is exposed until Caddy proxies it. Add the block from `Caddyfile.host` to the host Caddyfile (usually `/etc/caddy/Caddyfile`), change the domain, and reload:

```bash
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

If Caddy runs in its own Docker container instead, put it on the `pm2go` compose network and use `reverse_proxy hub:8080`.

### Option B: no Caddy on the host

```bash
docker compose --profile caddy up -d --build
```

Caddy gets a Let's Encrypt certificate for `PM2GO_DOMAIN` and listens on 80 and 443 (`Caddyfile.container`).

### Why the hub runs with `PMGO_TRUST_PROXY=1`

The image sets it. The hub then trusts `X-Forwarded-For` and `X-Forwarded-Proto` that Caddy sends: the login lockout counts the real client address instead of Caddy's, the command audit log shows the real IP, and the session cookie gets the `Secure` flag. **Never** expose the hub port directly with this flag on, because any client could forge its address. The compose file binds to localhost for this reason.

WebSockets (agents at `/agent/ws`, live logs at `/api/logs/ws`) need no extra Caddy settings.

## Without Caddy: built-in TLS

The hub can terminate TLS itself. It is off by default and does not change the Caddy setup. Do not combine it with `PMGO_TRUST_PROXY`; the hub warns at startup if both are set.

Own certificate files (for example from certbot or a private CA):

```bash
pm2go hub --listen :443 --tls-cert /etc/pm2go/fullchain.pem --tls-key /etc/pm2go/privkey.pem
```

The hub re-reads both files when their modification time changes (checked at most every 10 seconds, on a new connection) and on `SIGHUP`. A broken renewal is logged and the old certificate stays in use, so no restart is needed after renewal.

Automatic Let's Encrypt certificates (`golang.org/x/crypto/acme/autocert`):

```bash
pm2go hub --listen :443 --acme-domain pm2.example.com      # also listens on :80
```

The domain must resolve to this host and ports 80 and 443 must be reachable. Certificates are cached in `<hub home>/acme`. `--acme-http ADDR` changes the port 80 listener, which answers HTTP-01 challenges and redirects everything else to HTTPS. With certificate files, `--acme-http` adds a redirect-only listener. All options can also be set in `config.json` as `tls_cert`, `tls_key`, `acme_domain`, `acme_http`; flags win.

When TLS is on: the session cookie is `Secure`, every response carries `Strict-Transport-Security`, and the startup log line states the mode. The hub warns when it listens on a non-loopback address in plain HTTP without `--trust-proxy`.

Private CA: on each host, point the agent at the CA that signed the hub certificate. The path is stored in `link.json` (`ca_file`) and added to the system roots:

```bash
pm2go link --hub https://pm2.example.com --key <agent key> --hub-ca /etc/pm2go/ca.pem
```

> Verified: certificate files, reload on renewal, redirect, HSTS, Secure cookie and `wss://` with a self-signed CA (unit tests and a local run). **Not verified**: the ACME path against Let's Encrypt (staging or production); only the setup and the redirect handler are unit tested. Do a staging check before relying on it (issue #106).

## 3. First login

Open `https://<your domain>/`, sign in with the admin password, then:

1. Settings, Telegram: paste the bot token, add chats with a minimum level, press "Send test".
2. Alerts: review the 13 default rules and adjust thresholds.
3. Settings shows the exact `pm2go link` command for hosts.

Rotate the admin password or agent key by editing `config.json` in the volume and restarting the hub:

```bash
docker compose exec hub vi /data/config.json   # or edit through a volume mount
docker compose restart hub
```

Changing the agent key disconnects every agent until they are re-linked with the new key.

## 4. Put pm2go on a monitored host

Build the Linux binary (on any machine with Go) and copy it over:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o pm2go ./servers/pm2go
scp pm2go host:/usr/local/bin/pm2go
```

On the host:

```bash
pm2go start ./myapp --name myapp --max-restarts 10      # or: pm2go start ecosystem.json
pm2go save                                              # remember the process list
sudo pm2go startup daemon                               # systemd unit, runs "pm2go daemon --resurrect"
pm2go link --hub https://pm2.example.com --key <agent key> --name <machine-name>
```

`startup` needs root to write `/etc/systemd/system/pm2go-daemon.service`; without root it prints the unit and the commands to install it. Remove it with `sudo pm2go unstartup daemon`. The unit runs as the user who ran `startup` and sets `PMGO_HOME` to that user's state directory.

Machine names must be unique per hub. A second host with the same name replaces the first connection.

## 5. Verify

- `curl -s https://<domain>/healthz` prints `ok`.
- `docker compose ps` shows `pm2go-hub` as healthy.
- The host appears under Servers within a few seconds of `pm2go link`.
- Kill a managed process twice quickly (`kill -9 <pid>`) and confirm a restart event, then trigger a test Telegram message from Settings.
- Stop the daemon on a host (`pm2go kill`) and confirm the "Server offline" alert fires after about a minute.

## 6. Operate

| Task | Command |
|------|---------|
| Update the hub | `git pull && docker compose up -d --build` (data volume is kept) |
| Logs | `docker compose logs -f hub` |
| Back up | `docker run --rm -v pm2go_hub-data:/data -v "$PWD":/backup alpine tar czf /backup/pm2go-hub.tgz -C /data .` (volume name is `<compose project>_hub-data`) |
| Restore | stop the hub, extract the archive back into the volume, start the hub |
| Change retention | edit `retention_days` in `config.json`, restart (default 7 days) |

## Security notes

- One shared admin password, no per-user accounts or audit trail beyond the command log line. Keep the hub on a network you trust or behind a VPN or Caddy `basicauth` if it is public.
- The agent key lets any holder push data and receive commands for a machine name. Treat it like a password.
- Remote commands can restart, stop, delete and start processes on any linked host. Starting a process runs an arbitrary command as the daemon's user.
- The Telegram bot token is stored unencrypted in the hub database; the API never returns it.
