# soham WhatsApp knowledge base

Reference for `servers/soham`: the WhatsApp relay server, the WhatsApp client, and the code they share. It keeps the design decisions and the limits both sides must agree on.

## What it is

- **`whatsapp-server`**: a Fiber HTTP server. Callers post send requests to the API (`/send_message`, `/send_base64_media`, `/send_web_media`, `/send_local_media`). The server forwards each request over a websocket to the client that owns the `from` number and waits for its reply.
- **`whatsapp-client`**: runs next to the WhatsApp session. It keeps one websocket open to the server (`/whatsapp-client/ws`), sends each request through WhatsApp, and replies with the per-recipient result.
- **`common`**: message types, request structs, error codes and the shared limits in `limits.go`.

## Code map

| Path | Role |
|------|------|
| `common/communication.interface.go` | `MessageType`, `WhatsappClientMessage`, request structs |
| `common/limits.go` | `MaxWebsocketMessageSize`, send timeouts, `SendTimeout` |
| `common/req.keys.go` | context keys and error codes (`limits.go` adds `ERROR_MESSAGE_TOO_LARGE = 210`) |
| `whatsapp-server/main.go` | Fiber app, body limit, TLS, routes |
| `whatsapp-server/env/index.go` | env loading, `WebsocketConnection` (serialized writes, size check) |
| `whatsapp-server/services/whatsapp.serivce.go` | connection registry, request/reply routing, `sendMessage` wait |
| `whatsapp-server/websocket/index.go` | websocket handler: read loop, pings, reconnect handling |
| `whatsapp-client/websocket/index.go` | client websocket: read limit, handlers, send timeouts, local media checks |

## Feature history (PRs on `rpsoftech/GolangServers`)

| PR | Change |
|----|--------|
| #66 | client: token-safe logs, tmp dir, start route, clean shutdown |
| #67 | client: websocket handler hardening (read limit, per-recipient send timeouts, handler cap) |
| #68 | client: local media from any path, client secret files blocked |
| #80 | server: write locking, reply routing, reconnects, pings, env loading |
| #81 | shared send timeout, websocket message size limit, request context (issues #73, #74, #76) |

## Limits both sides share

Keep these in `common/limits.go`. Never copy them into the client or the server.

- **Send timeout.** Base per recipient: text 10s, base64 and local media 30s, web media 45s. `SendTimeout(messageType, recipients)` multiplies the base by the recipient count (minimum 1) and caps it at 5 minutes. An unknown type gets the cap.
- **Server reply wait.** The server waits `SendTimeout + replyMargin` (15s). The client's own result, or its timeout error, must reach the server before the server gives up. A fixed 30s wait (before #81) returned 408 for many recipients while the client was still sending, and the real result was dropped.
- **Websocket message size.** `MaxWebsocketMessageSize` is 64 MiB. The client uses it as its read limit and closes the connection on a larger message. So the server encodes each message once and rejects one over the limit before writing (`ErrMessageTooLarge`, HTTP 413). A rejected message writes nothing, so the connection stays up for other requests.
- **HTTP body limit.** `BodyLimit` in `main.go` must stay below the websocket limit, or large requests reach the size check instead of being refused early.

## Design decisions

- **One writer per websocket.** `WebsocketConnection.SendMessage` holds `writeMu` for the write. Encoding happens before the lock, so a large payload does not block other senders while it is marshalled. Writes use `WriteMessage(TextMessage, data)`.
- **Write deadline.** Each write has `WriteWait` (2 minutes), so a client that stopped reading cannot block senders forever.
- **Request context.** `sendMessage` waits on `context.WithTimeout(ctx, replyTimeout)`. Deadline gives 408 (`ERROR_CLIENT_REQ_TIMEOUT`), cancellation gives 499 (`ERROR_REQUEST_CONNECTION_DROPPED`). fasthttp, and so Fiber's `c.Context()`, does not cancel the context when the HTTP client disconnects. Do not expect 499 from a dropped HTTP connection; the wait ends by its timeout.
- **One reply per request.** `RouteResponse` removes the waiting entry when it delivers, so duplicate and late replies are ignored. The reply channel is buffered and never closed.
- **Reconnects.** A new connection for a number replaces and closes the old one. The old handler's late status updates and removal are ignored.

## Known gaps

- Web media: the client downloads the file before its send timeout starts. A slow download can use up the 15s margin, so the server returns 408 while the client still sends.

## Testing

- `go build ./...`, `go vet ./...` and `go test -race ./...` in `servers/soham`.
- Server tests replace the websocket with a fake `WSConn` (`services/whatsapp_service_test.go`). Package variables (`sendTimeout`, `replyMargin`, `env.MaxMessageSize`, `env.WriteWait`) let tests shorten timeouts and limits.
- Existing `.go` files in `servers/soham` use CRLF. Keep CRLF on edited files and check that `git diff --stat` is not whole-file. `gofmt -l` lists CRLF files; that is harmless.
