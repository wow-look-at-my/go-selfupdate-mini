# go-selfupdate-mini

Minimal Go library for self-updating binaries via GitHub Releases. No dependency on `go-github`.

## Build & Test

```sh
go-toolchain
```

Do not use `go build`, `go test`, or other bare `go` commands. Always use `go-toolchain` which handles mod tidy, testing, coverage, and building.

## Architecture

- **package.go** -- High-level convenience functions (`DetectLatest`, `UpdateSelf`, etc.) using a global `DefaultUpdater` singleton
- **updater.go** -- Core `Updater` struct, platform detection, ARM version detection
- **config.go** -- `Config` struct and extension point interfaces (`Decompressor`, `VersionFilter`)
- **detect.go** -- Release detection, version parsing, asset matching by platform suffixes or regex filters
- **update.go** -- Download, decompress, validate, and install flow
- **install.go** -- Atomic binary replacement with rollback
- **decompress.go** -- Built-in decompressors for zip, tar.gz, gz, bz2
- **source.go** -- `Source` interface for pluggable release providers
- **github_source.go** -- GitHub REST API implementation of `Source`
- **github_release.go** -- GitHub API JSON response models
- **repository.go** / **repository_slug.go** -- `Repository` interface and `owner/repo` slug implementation
- **release.go** -- `Release`, `Version`, `Platform`, `Asset` data types
- **arch.go** -- Architecture fallback logic (ARM variants, x86_64 alias, universal)
- **arm.go** -- ARM version extraction from binary via `debug/buildinfo`
- **token.go** -- Domain matching for token scope
- **log.go** -- Optional logger interface (silent by default)
- **errors.go** -- Sentinel errors

## Conventions

- All files are in a single package `selfupdate` at the repo root
- Tests use `github.com/wow-look-at-my/testify` (assert/require)
- Tests use `net/http/httptest` for HTTP mocking -- no external test dependencies
- Errors are exported as sentinel variables in `errors.go`
- The `Source` interface allows non-GitHub providers without changing core logic
