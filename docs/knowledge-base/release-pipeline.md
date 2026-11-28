# Release pipeline knowledge base

Reference for `utility/deploy` and `.github/workflows/release.yml`.

## Build order

`releaseAll` in `utility/deploy/main.go` runs in three phases:

1. **Resolve versions**: one lookup per component, in parallel.
2. **Build**: builds are grouped by target (GOOS/GOARCH). Go caches compiled packages per GOOS/GOARCH, so the components of one target build one after another and reuse the compiled libraries; up to `-parallel` (default 3) target groups build at once.
3. **Publish**: only if every build succeeded. Uploads and KV writes run up to 4 at a time.

A failed build publishes nothing. Before this change, a failure on a late target left earlier targets live at the new version.

## Limits

Only `linux_amd64` and `windows_amd64` have several components. The other five targets belong to `soham-whatsapp-client` alone, so each recompiles its dependencies. A warm build cache across runs helps more than any ordering.

Measured on an 8-core Mac, cold cache, `-component all -dry-run`: `-parallel 1` 130 s, targets in parallel (`-parallel 3`) 109 s. Building targets one at a time with components in parallel inside a target was slower (152 s vs 133 s): the CPU was already saturated.

## CI build cache

`release.yml` uses `actions/cache` with a key that ends in `github.run_id`, and `restore-keys` that load the newest earlier cache. `setup-go`'s own cache is off (`cache: false`).

Why: an `actions/cache` entry is saved only when its exact key does not exist yet, and `setup-go` works the same way. A push to main builds only the changed components. If such a run was the first under a given `go.sum`, the saved cache held only those targets, and later runs hit that key and never added the other targets. The Go cache itself keeps every GOOS/GOARCH side by side and never evicts one for another (checked locally: linux 18 s cold, 2 s again after windows and darwin builds).

## Notes

- `-ldflags` only affects linking, so different `-X main.version` values per component do not invalidate cached compiled packages.
- Build output is captured per build and printed in one piece so parallel builds do not interleave.
- GitHub runners have 4 vCPUs (public repo) or 2 (private). `go build` already uses all cores, so keep `-parallel` at 3 or lower.
