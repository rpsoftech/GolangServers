# whatsapp-server knowledge base

Reference for `servers/whatsapp-server` and the shared WhatsApp code in `functions/whatsapp`, which `servers/soham/whatsapp-client` also uses. Keep design decisions and lessons here; see [soham-whatsapp.md](soham-whatsapp.md) for the soham relay.

## What it is

A Fiber HTTP server that holds one whatsmeow session per API token and sends WhatsApp messages for callers.

- `POST /v1/start` connects a token's number in the background (QR scan for a new device).
- `GET /v1/qr_code` and `GET /scan/:id` show the QR code to scan.
- `POST /v1/send_message`, `/v1/send_media` (multipart), `/v1/send_media_64`, `/v1/send_media_url`, `/v1/send_media_url_multiple` send messages. They need a logged-in session.
- Every `/v1` call needs the `X-Api-Token` header. Header `Headless: true` makes media sends return at once and run in the background.
- Tokens and their numbers live in `server.config.json` (`tokens`, `jid`), next to the executable. Env keys: `APP_ENV`, `PORT`, `ALLOW_LOCAL_NO_AUTH`, `AUTO_CONNECT_TO_WHATSAPP`, `OPEN_BROWSER_FOR_SCAN` (see `env_template`).

## Code map

| Path | Role |
|------|------|
| `servers/whatsapp-server/main.go` | startup, auto-connect, shutdown |
| `servers/whatsapp-server/server.go` | `setupFiberServer`: logger, routes, 404 |
| `servers/whatsapp-server/src/apis/index.go` | send handlers |
| `servers/whatsapp-server/src/apis/helpers.go` | `to`/`msg` parsing, recipient checks, timeouts, uploads, `isHeadless` |
| `servers/whatsapp-server/src/apis/start.number.go` | `/v1/start` |
| `servers/whatsapp-server/src/apis/qr-browser.api.go` | QR page for `/scan/:id` |
| `functions/whatsapp/core/interfaces.go` | `WhatsappConnection`: sends, uploads, log file |
| `functions/whatsapp/core/lifecycle.go` | `Reserve`, `ConnectEach`, `ConnectReserved`, `DisconnectAll` |
| `functions/whatsapp/core/index.go` | SQLite session store (`WhatsappSuperSecrete.db`) |
| `functions/whatsapp/middleware/token-validator.go` | `X-Api-Token` auth and logged-in check |
| `functions/whatsapp/logging/logger.go` | request logger that redacts tokens |
| `functions/whatsapp/config/index.go` | env and `server.config.json` loading |
| `interfaces/whatsapp` | config types, thread-safe token store |

## Feature history (PRs on `rpsoftech/GolangServers`)

| PR | Change |
|----|--------|
| #56 | core: daily log rotation, a result for every recipient, upload failure fails fast, audio caption sent as text, masked token logs |
| #57 | API: unique temp uploads, multipart `to`/`msg` formats, 400 on bad input, no `200 null`, scaled timeouts, every recipient checked |
| #58 | lifecycle: `/v1/start` returns at once, one client per token (`Reserve`), failed connects release the token, auto-connect per token, `DisconnectAll` on shutdown |
| #60 | token-safe request logs, duplicate middleware removed, tests for #56-#58, root `CLAUDE.md` |
| #66 | logger moved to shared `functions/whatsapp/logging`, error handler uses `errors.As` |

## Design decisions

- **One connection per token.** `ConnectionMap.Reserve` claims a token atomically by storing a nil entry. Callers of `ConnectionMap.Get` treat nil as "not found". `ConnectReserved` removes the entry when a connect fails or a QR scan times out, so `/start` can retry.
- **Background connects.** `/v1/start` and auto-connect never block on the QR channel. Auto-connect starts one goroutine per token.
- **Timeouts.** Sends allow 30s per recipient (times the URL count for `send_media_url_multiple`), capped at 15 minutes. Headless sends create their own timeout context inside the goroutine.
- **Recipients.** Every `to` entry is checked: spaces and one leading `+` are stripped, then at least 7 digits are required.
- **Uploads.** Each upload goes to a new `os.CreateTemp("./tmp", "upload-*<ext>")` file and is removed after the send, so two uploads with the same name never collide.
- **Log file.** `AppendToOutPutFile` recomputes `<midnight-unix>.log.csv` on each write, so the file rotates daily.
- **No tokens in logs.** The request logger uses an explicit format (no header, query or body tags). `/scan/<token>` logs as `/scan/***`, and the token is removed from error text. Do not switch to `logger.DefaultFormat`: Fiber writes it through a fast path that ignores custom tags, so the masking would stop working. A test guards this.
- **Local no-auth.** `ALLOW_LOCAL_NO_AUTH=true` skips the token only for loopback clients. Spoofed `X-Forwarded-For` is ignored unless a proxy is trusted.
- **Outbound URLs.** Web media is fetched with `utility_functions.FetchUserURL`, which blocks loopback, private, link-local, metadata and CGNAT addresses, also after redirects, and caps the size.
- **Shared code.** `functions/whatsapp` is also built into the soham client. Keep exported signatures stable, or update both servers in one PR.

## Known gaps

- If `client.Connect()` fails for a new device before a socket opens, `ConnectAndGetQRCode` may wait on the QR channel forever, so that token stays reserved. Fix it by returning the `Connect` error or giving `GetQRChannel` a context with a timeout.
- Nothing from #56-#60 was tested against a live WhatsApp account before merge.

## Release

`whatsapp-server` is a release component in `utility/deploy/components.go`. A push to `main` that changes any package it imports (including `functions/whatsapp`), `go.mod` or the deploy tool auto-releases it to STAGING. PRODUCTION is manual.

## Testing

- `go test -race ./functions/whatsapp/... ./interfaces/whatsapp/... ./utility/functions/... ./servers/whatsapp-server/...`.
- Handler tests use Fiber `app.Test` with a stub connection (`getConnection` package var). Core send tests use a fake `mediaClient`. The clock in core is `var now = time.Now` for log rotation tests.
