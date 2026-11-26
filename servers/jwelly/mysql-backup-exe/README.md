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

## Backing up over a remote link (optional)

Backing up from another machine is much slower than on the database server,
because every row crosses the network uncompressed. Set `"compress": true` on a
database to ask MySQL to compress that traffic (`--compress` for `mysqldump`,
`compress=true` for the built-in engine). It is off by default and brings
nothing when the tool runs on the database server itself. Running the tool on
the server, with `MYSQL_HOST` set to `127.0.0.1`, is still the fastest setup.

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
