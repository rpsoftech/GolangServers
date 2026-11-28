# #97: `deploy` command (git + ssh)

Status: in review
Branch: feat/cli-deploy
PR: (see below)
Issue: https://github.com/rpsoftech/GolangServers/issues/97
Size: M-L | Lane: cli | Needs first: none (YAML #92 soft)
Hot files: new `cli/deploy.go`, `common/deploy.go`

> The session working on #97 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

- 2026-11-29: `pm2go deploy [-f file] <env> <cmd>`; `deploy <env>` alone means update. Why: issue syntax has no file argument, ecosystem default is `ecosystem.json`.
- 2026-11-29: `common.Ecosystem` gets `Deploy map[string]DeployConfig`; `common.LoadDeploy` reads JSON only, through the package variable `common.ConvertEcosystem`. Why: #92 can replace that variable to feed YAML through the JSON path without touching deploy code. Deploy fields have json tags only.
- 2026-11-29: release = `git archive` of the resolved commit into `releases/<UTC ts>/` (no git worktree, no rsync). Why: no metadata left in `source/`, pruning is a plain `rm -rf`.
- 2026-11-29: no local git call; clone and fetch run in the remote script. Tests use a fake `ssh` that can execute the script locally with real bash and git. Why: this exercises the real script instead of only recording it.
- 2026-11-29: `--dry-run` also skips `pre_deploy_local`. Why: it has side effects.

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-29: implemented `common/deploy.go`, `cli/deploy.go`, `cli/deploy_test.go`, doc `pm2go-deploy-command.md`. Verified with the fake ssh (args, stdin, local execution) and by hand against a throwaway local sshd on port 22222 with the real ssh binary (setup, update, curr, list, exec; a wrong known_hosts entry made the deploy fail). The doc example lives in `pm2go-deploy-command.md` (issue text asked for `pm2go.md`; kept in the new file per the work rules).


## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- #97 deploy command, all acceptance lines (docs line: example ecosystem is in `pm2go-deploy-command.md`).
