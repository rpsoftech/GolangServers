# mysql-backup-exe

Backs up the databases listed in `config.json` to gzipped SQL files. See
`config.json.example`, `config.once.json.example` and
`config.cron.linux.json.example`.

## Destinations

| type | needs | notes |
|------|-------|-------|
| `local` | `path` | `keepLast` / `keepDays` prune old backups |
| `fileServer` | `url`, `token`, optional `folderPath` | multipart upload |
| `r2` | `accountId` (or `endpoint`), `bucket`, `accessKeyId`, `secretAccessKey`, optional `folderPath` | Cloudflare R2; `keepLast` / `keepDays` prune old backups |

```json
{
  "type": "r2",
  "accountId": "your-cloudflare-account-id",
  "bucket": "db-backups",
  "accessKeyId": "...",
  "secretAccessKey": "...",
  "folderPath": "jwelly/main",
  "keepLast": 14
}
```

Create the R2 API token with access to that one bucket only. It needs read and
delete permission on objects when `keepLast` or `keepDays` is set, because
pruning lists and deletes old backups. Turn on versioning or Object Lock on the
bucket if you want a compromised backup machine to be unable to destroy history.
Uploads are retried (`retries`) and a transfer that sends no data for 10 minutes
is aborted. Pruning only touches `<name>-<timestamp>.sql.gz[.age]` objects of
the same database directly under `folderPath`.

## Telegram reports (optional)

Add a top-level `telegram` section next to `databases` to get a message for
every backup result, success or failure, from both quick runs and the daemon:

```json
{
  "telegram": { "botToken": "123456:ABC-from-BotFather", "chatId": -1001234567890 },
  "databases": [ ... ]
}
```

Create the bot with @BotFather, add it to the chat or group (or press Start in a
private chat), and use the chat id. `chatId` may be a number or a string such as
`"@mychannel"`. The message shows the database, size, engine, duration and each
destination, or the first line of each error. A failed send is logged and never
fails the backup. The bot token is a secret: keep `config.json` private. It is
not printed in logs.

## Updates

A published build updates itself. The daemon checks every 5 minutes and exits
after installing, so your service or scheduler must start it again. A quick run
(one-time or started by cron or Task Scheduler) checks once at the start; when
it installs a newer version it starts that version with the same arguments,
waits for it and returns its exit code, so the backup runs with the new
version. If the update server is unreachable the backup runs with the current
version. Use `--no-update` to skip the check. Builds made locally never update.

## Backing up over a remote link (optional)

Backing up from another machine is much slower than on the database server,
because every row crosses the network uncompressed. Set `"compress": true` on a
database to ask MySQL to compress that traffic (`--compress` for `mysqldump`,
`compress=true` for the built-in engine). It is off by default and brings
nothing when the tool runs on the database server itself. Running the tool on
the server, with `MYSQL_HOST` set to `127.0.0.1`, is still the fastest setup.

`mysqldump` also asks the server several questions for every table, one after
another, so a database with many tables is slow over a remote link even when it
holds little data (each question costs one network round trip). Compression does
not help with that. Instead set a top-level `"parallel"` number, next to
`databases`, to back up that many databases at the same time:

```json
{ "parallel": 4, "databases": [ ... ] }
```

Allowed values are 1 to 16. Unset (or 0) keeps the old behaviour: a quick run
goes one database after another, and the daemon puts no limit on schedules that
fire together. With `parallel` set, both a quick run and the daemon (schedules
and `runOnStart`) run at most that many backups at once. Each running backup is
one more connection to the database server, so raise it gradually. It makes the
whole run faster, not a single database. The summary and Telegram messages
still list the databases in config order.

## Encrypting backups (optional)

Set `encryptTo` on a database to one or more age public keys. The backup is
encrypted once, after the dump is verified, and **every** destination of that
database (local folder and file server) receives the same `.sql.gz.age` file.
Without `encryptTo` nothing changes.

```json
{ "name": "jwelly-main", "encryptTo": ["age1..."], "destinations": [ ... ] }
```

Make a key pair on a machine that is NOT the backup machine:

```
mysql-backup-exe keygen -o backup-key.txt
```

It prints the public key for `encryptTo` and writes the secret key to the file.
Keep the secret key offline and backed up: without it no encrypted backup can be
opened. Give two people a key each by listing both public keys in `encryptTo`;
either secret key opens the file.

Restore:

```
mysql-backup-exe decrypt --identity backup-key.txt jwelly-main-20260101-020000.sql.gz.age
gunzip jwelly-main-20260101-020000.sql.gz
```

`decrypt` refuses to overwrite an existing file and leaves nothing behind when
the key is wrong or the file is damaged.
