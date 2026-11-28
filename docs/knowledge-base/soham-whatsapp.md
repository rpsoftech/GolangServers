# soham WhatsApp knowledge base

Reference for `servers/soham`: the WhatsApp relay server, the WhatsApp client, and the code they share. It keeps the design decisions and the limits both sides must agree on.

## What it is

- **`whatsapp-server`**: a Fiber HTTP server. Callers post send requests to `/api/send_message`, `/api/send_base64_media`, `/api/send_web_media` and `/api/send_local_media` with a JWT. The server forwards each request over a websocket to the client that owns the `from` number and waits for its reply.
- **`whatsapp-client`**: runs next to the WhatsApp session on the customer's machine. It keeps one websocket open to the server (`/whatsapp-client/ws`), sends each request through WhatsApp (shared core in `functions/whatsapp`), and replies with the per-recipient result.
- **`common`**: message types, request structs, error codes and the shared limits in `limits.go`.

## Code map

| Path | Role |
|------|------|
| `common/communication.interface.go` | `MessageType`, `WhatsappClientMessage`, request structs (`to` needs at least one entry) |
| `common/limits.go` | `MaxWebsocketMessageSize`, send timeouts, `SendTimeout`, `ERROR_MESSAGE_TOO_LARGE = 210` |
| `common/req.keys.go` | header and context keys, error codes |
| `whatsapp-server/main.go` | startup: `env.Init()`, routes, TLS, listen |
| `whatsapp-server/server.go` | `bodyLimit`, `errorHandler`, `loadTLSConfig` |
| `whatsapp-server/env/index.go` | env loading, `WebsocketConnection` (serialized writes, size check) |
| `whatsapp-server/middleware/whatsapp-client.validate.go` | client websocket auth (`UUID_AUTH` + `NUMBER_AUTH`) |
| `whatsapp-server/middleware/token-validator.go` | JWT auth for `/api` |
| `whatsapp-server/services/whatsapp.serivce.go` | connection registry, request/reply routing, `sendMessage` wait |
| `whatsapp-server/websocket/index.go` | websocket handler: read loop, pings, read deadline, reconnect handling |
| `whatsapp-client/websocket/index.go` | client websocket: read limit, handlers, send timeouts, local media checks |
| `whatsapp-client/apis/start.go` | `POST /v1/start` |

## Feature history (PRs on `rpsoftech/GolangServers`)

| PR | Change |
|----|--------|
| #66 | client: token-safe logs, tmp dir in the exe folder, `/v1/start`, `DisconnectAll` on shutdown, graceful OTA exit (issue #64) |
| #67 | client: websocket hardening (pings, read limit, handler cap, error replies, per-recipient timeouts, goroutine cleanup) (issue #63; replaced #65 after a branch rename) |
| #68 | client: local media from any path, client secret files blocked (replaced the `LOCAL_MEDIA_DIR` setting from #67) |
| #80 | server: write locking, reply routing, reconnects, pings and read deadline, recover middleware, env loading (issues #69-#72, #78) |
| #81 | shared send timeout, websocket message size limit, request context (issues #73, #74, #76) |
| #82 | server: error handler, TLS loading, client auth checks, non-empty `to` (issues #75, #77, #79) |

## Limits both sides share

Keep these in `common/limits.go`. Never copy them into the client or the server.

- **Send timeout.** Base per recipient: text 10s, base64 and local media 30s, web media 45s. `SendTimeout(messageType, recipients)` multiplies the base by the recipient count (minimum 1) and caps it at 5 minutes. An unknown type gets the cap.
- **Server reply wait.** The server waits `SendTimeout + replyMargin` (15s). The client's own result, or its timeout error, must reach the server before the server gives up. A fixed 30s wait (before #81) returned 408 for many recipients while the client was still sending, and the real result was dropped.
- **Websocket message size.** `MaxWebsocketMessageSize` is 64 MiB. The client uses it as its read limit and closes the connection on a larger message. So the server encodes each message once and rejects one over the limit before writing (`ErrMessageTooLarge`, HTTP 413). A rejected message writes nothing, so the connection stays up for other requests.
- **HTTP body limit.** `bodyLimit` in `server.go` is 64 MiB, the same as the websocket limit. Larger bodies get Fiber's 413 before any work. Keep it at or below `MaxWebsocketMessageSize`.
- **Keepalive.** The client pings every 50s. The server pings every 45s and drops a connection after 90s without any message, ping or pong.

## Design decisions

### Server

- **One writer per websocket.** `WebsocketConnection.SendMessage` holds `writeMu` for the write. A concurrent write panics in the websocket library (`concurrent write to websocket connection`), and before #80 that killed the process. Encoding happens before the lock, so a large payload does not block other senders. Writes use `WriteMessage(TextMessage, data)` and ping/pong use `WriteControl`, which is safe next to other writes.
- **Write deadline.** Each write has `WriteWait` (2 minutes), so a client that stopped reading cannot block senders forever.
- **Panics.** Fiber's `recover` middleware is on, so a handler panic returns 500 instead of stopping the server.
- **Request context.** `sendMessage` waits on `context.WithTimeout(ctx, replyTimeout)`. Deadline gives 408 (`ERROR_CLIENT_REQ_TIMEOUT`), cancellation gives 499 (`ERROR_REQUEST_CONNECTION_DROPPED`). fasthttp, and so Fiber's `c.Context()`, does not cancel the context when the HTTP client disconnects. Do not expect 499 from a dropped HTTP connection; the wait ends by its timeout.
- **One reply per request.** `RouteResponse` removes the waiting entry and sends without blocking while it holds the lock. Duplicate and late replies are ignored. The reply channel is buffered and never closed (closing it caused `send on closed channel` before #80).
- **Reconnects.** A new connection for a number replaces and closes the old one. The old handler's late status updates and removal only apply to its own connection, so they are ignored.
- **Error handler.** `errorHandler` uses `errors.As`. `*interfaces.RequestError` keeps its status. `*fiber.Error` below 500 keeps its code and message (404, 413, bad JSON). Anything else is logged on the server and returned as a generic 500 without the raw error.
- **TLS.** If `ssl.config.json` exists next to the executable, `loadTLSConfig` loads only its `cert` and `key` (the cert file may hold the full chain, leaf first), TLS 1.2 minimum. A bad config or missing file is fatal at startup. Before #82 the code read a CA file from a developer's home folder and quietly never started on other machines.
- **Client auth.** The client sends `NUMBER_AUTH` and `UUID_AUTH`. `UUID_AUTH` must be the UUIDv5 of the number under `BASE_UUID`, compared in constant time. The number must be canonical (`7`, not `007` or `+7`) and positive.
- **Env.** `env.Init()` runs first in `main` (not `init()`, so packages can be tested). It reads `whatsapp-server.env` from the executable's folder and falls back to the working folder. Keys: `APP_ENV`, `PORT`, `BASE_UUID`, `ACCESS_TOKEN_KEY` (at least 10 characters). Local and develop environments listen on `127.0.0.1` only.
- **Logging.** Never log `UUID_AUTH` or JWTs. Log the number only.

### Client

- **Local media from any path.** `SEND_FILE_PATH_MEDIA` may read a file anywhere on the machine, by customer requirement. The client refuses its own secret files: `WhatsappSuperSecrete.db` and its `-wal`, `-shm`, `-journal` files, `whatsapp-client.env` and `server.config.json`. It also refuses any `.env`, `.env.*`, `.db*` or `.sqlite*` file, folders and non-regular files. The check runs on the requested name, the name after symlinks, and file identity, so symlinks and hard links are caught. A renamed copy of a secret file is not caught.
- **`ws://` stays allowed** for `SOCKET_URL`, by customer requirement. Do not force `wss://`.
- **Web media** is fetched through the SSRF guard `utility_functions.FetchUserURL`.
- **Tmp folder** is `<exe folder>/tmp`, created before anything else starts.
- **Logs** go through `functions/whatsapp/logging`, which redacts `/scan/<token>` and `X-Api-Token`.

## Deliberately not changed

- **JWT expiry.** The `/api` JWT parser does not require `exp`, so a token without `exp` never expires. Left as is by owner decision (review finding 8 of the server review). Revisit if tokens are ever issued without `exp`.

## Known gaps

- Web media: the client downloads the file before its send timeout starts. A slow download can use up the 15s margin, so the server returns 408 while the client still sends.
- None of #66-#68 or #80-#82 was tested against a live client and server pair before merge.

## Release and deploy

- The client is a release component (`soham-whatsapp-client` in `utility/deploy/components.go`, legacy key `soham_go_wbot_<os>_<arch>`). A push to `main` that changes any package it imports (found with `go list -deps`, so also `servers/soham/common` and `functions/whatsapp`), `go.mod` or the deploy tool auto-releases it to STAGING.
- The soham server is not a release component. Build and deploy it by hand.
- After #80-#82 the server needs no new env keys. Check `ssl.config.json` paths before deploying, because a bad config now stops startup.

## Testing

- `go build ./...`, `go vet ./servers/soham/...` and `go test -race ./servers/soham/...`.
- Server unit tests replace the websocket with a fake `WSConn` (`services/whatsapp_service_test.go`). `websocket/index_test.go` runs a real Fiber server with a gorilla client (pings, silent drop, reconnect). Package variables (`pongWait`, `pingPeriod`, `replyMargin`, `env.MaxMessageSize`, `env.WriteWait`) let tests shorten timeouts and limits. The handler copies them at connection start, so restoring them in test cleanup does not race.
- Most `.go` files under `servers/soham/whatsapp-server` and `servers/soham/common` use CRLF; the client files use LF. Keep each file's line endings and check that `git diff --stat` is not whole-file. `gofmt -l` lists CRLF files; that is harmless.
