# Release pipeline knowledge base

Reference for `utility/deploy` and `.github/workflows/release.yml`.

## Build order

`releaseAll` in `utility/deploy/main.go` runs in three phases:

1. **Resolve versions**: one lookup per component, in parallel.
2. **Build**: targets (GOOS/GOARCH) run one after another. Go caches compiled packages per GOOS/GOARCH, so the first component of a target builds alone and compiles the shared libraries; the remaining components of that target then build in parallel (up to `-parallel`, default 3) on the warm cache.
3. **Publish**: only if every build succeeded. Uploads and KV writes run up to 4 at a time.

A failed build publishes nothing. Before this change, a failure on a late target left earlier targets live at the new version.

## Limits

Only `linux_amd64` and `windows_amd64` have several components. The other five targets belong to `soham-whatsapp-client` alone, so they build one at a time and each recompiles its dependencies. A warm build cache across runs (`actions/setup-go` cache) helps more than any ordering.

## Notes

- `-ldflags` only affects linking, so different `-X main.version` values per component do not invalidate cached compiled packages.
- Build output is captured per build and printed in one piece so parallel builds do not interleave.
- GitHub runners have 4 vCPUs (public repo) or 2 (private). `go build` already uses all cores, so keep `-parallel` at 3 or lower.
