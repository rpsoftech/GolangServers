# GolangServers

Go monorepo (module `github.com/rpsoftech/golang-servers`).

## Project structure

- `servers/` - deployable servers (whatsapp-server, telegram-server, bullion, soham, jwelly, pm2go process manager + dashboard, ...)
- `apps/` - client apps
- `functions/` - shared feature code (for example `functions/whatsapp` core, config, middleware)
- `utility/` - shared helpers (deploy, updater, mysql, mongodb, redis, jwt, firebase, ...)
- `interfaces/`, `env/`, `validator/`, `middleware/`, `events/` - shared types, env loading, validation, middleware

## Git rules (final, always apply)

- Never mention Claude, Anthropic or any AI tool in commit messages, PR titles, PR descriptions, PR comments, issue text, branch names, tags or code comments.
- Never add `Co-Authored-By: Claude ...` trailers, "Generated with Claude Code" lines, or any other AI attribution.
- Never use `claude/` or any AI-related prefix in branch names. Use plain names such as `feat/<topic>` or `fix/<topic>`.
- Commits are authored only by the repository owner's configured git identity.

## Knowledge base

- `docs/knowledge-base/` holds reference notes per component (design decisions, lessons, release process). Start at `docs/knowledge-base/README.md`, which lists the files and the repo-wide conventions. Read the matching file before changing that component and update it with the change.
- `docs/knowledge-base/graph/` holds a generated code graph and knowledge graph for `servers/pm2go` (start with `GRAPH.md`). Read it before exploring that component; regenerate with `go run docs/knowledge-base/graph/gen.go`.
