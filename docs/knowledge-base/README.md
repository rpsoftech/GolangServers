# Knowledge base

Reference notes per component: design decisions, limits, lessons and release notes. Read the matching file before changing a component, and update it in the same PR as the change.

| File | Covers |
|------|--------|
| [mysql-backup-exe.md](mysql-backup-exe.md) | `servers/jwelly/mysql-backup-exe` |
| [whatsapp-server.md](whatsapp-server.md) | `servers/whatsapp-server` and shared `functions/whatsapp` |
| [soham-whatsapp.md](soham-whatsapp.md) | `servers/soham` relay server, WhatsApp client and `common` |

Add a file when a component gets its first notes, and list it here.

## Repo-wide conventions

- No AI or tool attribution anywhere: commits, PRs, issues, branch names, tags, code comments. No `Co-Authored-By` trailers for tools. See the root `CLAUDE.md`.
- Branch names `feat/<topic>`, `fix/<topic>`, `docs/<topic>`, `chore/<topic>`. No random suffixes: renaming a pushed branch closes its PR (#65 had to be replaced by #67).
- PRs are squash-merged and need a review approval on `main`.
- After a squash merge, branches show as unmerged in `git branch --no-merged`. Check the PR state (`gh pr list --state all --head <branch>`) instead.
- Check that a PR's last commits landed in `main` before calling it done (#55 and #61 were squash-merged before their final commits were pushed and needed follow-up PRs).
- Stacked PRs: after the base PR is squash-merged, rebase each child with `git rebase --onto origin/main <old base commit>`, or the child carries the old commits and conflicts with its siblings.
- Parallel work on one component: split by files so no two PRs edit the same file.
- Before merging several related PRs, merge them together in a scratch worktree and run `go build ./...`, `go vet` and `go test -race` on the affected packages.

## Release process (all components)

A push to `main` auto-releases every component whose packages changed (found with `go list -deps`) to STAGING. A change to `go.mod` or `utility/deploy` releases all components. PRODUCTION releases are manual and need explicit approval. Auto-update only reaches builds on the same channel. Components are listed in `utility/deploy/components.go`; the soham relay server is not one of them.
