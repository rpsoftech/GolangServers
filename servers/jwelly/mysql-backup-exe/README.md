# mysql-backup-exe

Backs up the databases listed in `config.json` to gzipped SQL files. See
`config.json.example`, `config.once.json.example` and
`config.cron.linux.json.example`.

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
