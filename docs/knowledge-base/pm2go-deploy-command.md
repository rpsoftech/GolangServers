# pm2go deploy command

`pm2go deploy` ships a git repository to one or more hosts over ssh, in the style of `pm2 deploy`: release directories, an atomic `current` symlink, hooks, and rollback. It is configured by a `deploy` block in the ecosystem file. Code: `servers/pm2go/common/deploy.go` (config, loader, validation) and `servers/pm2go/cli/deploy.go` (script builder, ssh runner).

## Usage

```bash
pm2go deploy [-f ecosystem.json] <env> setup            # create dirs, clone, run pre_setup / post_setup
pm2go deploy [-f ecosystem.json] <env> [update]         # fetch, build a release, switch, post_deploy, prune
pm2go deploy [-f ecosystem.json] <env> revert [n]       # go back n releases (default 1), re-run post_deploy
pm2go deploy [-f ecosystem.json] <env> curr|prev|list   # show current, previous, all releases
pm2go deploy [-f ecosystem.json] <env> exec "<cmd>"     # run a command on every host (in current/)
```

Flags: `-f` (default `ecosystem.json`), `--dry-run` (print the ssh command line and the exact script per host, run nothing, not even `pre_deploy_local`), `--serial` (one host at a time; the default is all hosts in parallel), `--fail-fast` (stop at the first failing host; with parallel hosts the other ssh processes are cancelled).

Output is prefixed per host (`[user@host] ...`). The exit code is non-zero when any host fails. Without `--fail-fast` the other hosts continue.

## Example ecosystem

```json
{
  "apps": [{ "name": "api", "script": "./api", "instances": 2 }],
  "deploy": {
    "production": {
      "user": "deploy",
      "host": ["web1.example.com", "web2.example.com"],
      "repo": "git@github.com:acme/api.git",
      "ref": "origin/main",
      "path": "/var/www/api",
      "ssh_options": ["-i", "/home/me/.ssh/deploy_key", "-p", "2222"],
      "env": { "GOFLAGS": "-mod=readonly" },
      "keep_releases": 5,
      "pre_deploy_local": "go test ./...",
      "pre_setup": "mkdir -p shared/logs",
      "post_setup": "echo ready",
      "pre_deploy": "go build -o api ./cmd/api",
      "post_deploy": "pm2go reload ecosystem.json"
    }
  }
}
```

| Field | Meaning |
|-------|---------|
| `user`, `host` | ssh destination(s); `host` is a string or a list |
| `repo`, `ref` | what to clone and which ref to deploy (`ref` default `origin/main`; a branch as `origin/<branch>`, a tag, or a commit) |
| `path` | remote base directory (`~/x` expands to the remote home; a relative path is relative to the remote home) |
| `ssh_options` | extra ssh arguments, one list element per argument (`-p`, `2222`, `-o`, `StrictHostKeyChecking=...`) |
| `pre_setup`, `post_setup` | run on the host in `path` during `setup` |
| `pre_deploy_local` | run on this machine (cwd: the ecosystem file's directory) before `update`; failure aborts everything |
| `pre_deploy` | run on the host in the new release directory before the switch; failure aborts that host and `current` is not touched |
| `post_deploy` | run on the host in `current` after the switch (and after `revert`) |
| `env` | exported on the host for every hook |
| `keep_releases` | releases kept, default 5; the current one is never pruned |

Hooks run as `bash -e -c '<hook>'` and see `PM2GO_PATH`, `PM2GO_SHARED`, `PM2GO_REF`, plus `PM2GO_RELEASE` and `PM2GO_COMMIT` during `update`. Remote hosts need `bash`, `git`, `tar` and (for `post_deploy` as above) `pm2go` installed.

## Remote layout

```
<path>/source/            git clone (no checkout)
<path>/releases/<UTC YYYYmmddHHMMSS>/   one tree per release (git archive of the commit) plus .pm2go-commit
<path>/current            symlink to the active release
<path>/shared/            persistent data; link it from hooks
```

`update` builds the release in its own directory, runs `pre_deploy`, and only then creates a temporary symlink and renames it over `current` (`mv -T`, falling back to `mv -h` on BSD), so `current` never points at a half-built release. A failed `pre_deploy` removes the new directory. A failed `post_deploy` leaves the new release active and reports the host as failed; use `revert` to go back.

## Security model

- Uses the system `ssh` binary through `os/exec`: your keys, agent, `~/.ssh/config` and ControlMaster apply. Always passes `-o BatchMode=yes`; host key checking is whatever your ssh config says (nothing is weakened by default; `StrictHostKeyChecking=accept-new` only if you put it in `ssh_options`).
- Config values never reach a command line. The remote side is one script sent on stdin to `ssh -- <dest> bash -s`; every value (`path`, `ref`, `repo`, `env`, hooks) is assigned as a single-quoted shell word (`'` becomes `'\''`), so quotes, `;`, `$()`, backticks and newlines stay data. Hooks are code by design, but each runs as the argument of its own `bash -e -c` and cannot break out of the script.
- `host` and `user` are validated against a strict character set, and `repo` and `ref` must not start with `-` (option injection). NUL bytes are rejected.
- The remote `git` runs with `GIT_TERMINAL_PROMPT=0`, so a missing credential fails instead of hanging.

## Tests

`cli/deploy_test.go` puts a fake `ssh` first in `PATH`. It records its arguments and stdin; in exec mode it also runs the received script with a local `bash`, so the real git and filesystem steps run against a local repository (setup, update, prune, curr/prev/list, revert, exec, failed hooks, injection payloads in `ref`, `path`, `repo`, `env` and hooks that must not create a marker file). No fake `git` is needed: git only runs inside the received script. The real ssh path was also checked by hand against a throwaway local `sshd` on a high port (setup, update, curr, list, exec, and a failing host key check).

## Not done

- YAML ecosystem files: the loader goes through `common.ConvertEcosystem`, which the YAML change can replace to feed YAML through the JSON path. The deploy fields only have json tags.
- Per-host overrides, `rsync` of build artifacts, and `post_deploy` rolling across hosts one by one with health checks.
