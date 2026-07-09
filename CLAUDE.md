# go-selfupdate-mini

A minimal Go library for self-updating binaries from GitHub releases.

## Project structure

All source lives in the root package `selfupdate`. There are no sub-packages.

Key files:
- `updater.go` - `Updater` struct and `NewUpdater`/`DefaultUpdater` constructors
- `config.go` - `Config`, `VersionFilter`, `Decompressor` types
- `detect.go` - Version detection, asset matching, semver parsing
- `update.go` - `UpdateTo`, `UpdateCommand`, `UpdateSelf` methods
- `status.go` - `CheckUpdate`, `UpdateStatus`, `DaysOutOfDate` helpers
- `cobra.go` - Ready-to-use cobra commands (`NewUpdateCommand`, `NewVersionCommand`, `AddVersionFlag`)
- `release.go` - `Release`, `Version`, `Platform`, `Asset` types
- `source.go` - `Source`, `SourceRelease`, `SourceAsset` interfaces
- `github_source.go` - GitHub REST API implementation of `Source`
- `install.go` - Atomic binary replacement with rollback
- `decompress.go` - Built-in decompressors (zip, tar.gz, gzip, bz2)
- `package.go` - Package-level convenience functions

## Build and test

```sh
go build ./...
go test ./...
```

No special tooling required beyond the Go toolchain (1.22+).

## Dependencies

- `github.com/Masterminds/semver/v3` - Semantic version parsing and comparison
- `github.com/spf13/cobra` - CLI command framework (for cobra integration)

## Architecture notes

- The library uses a `Source` interface so it's not hard-coupled to GitHub. Implement `Source` to use GitLab, Gitea, etc.
- `Updater` is the core orchestrator. Package-level functions delegate to `DefaultUpdater()`.
- Asset matching works by suffix (OS + arch + extension) or regex filters.
- Installation is atomic: write `.new`, rename current to `.old`, rename `.new` to target, remove `.old`. Rollback on failure.
- All internal logging goes through the `Logger` interface (default: discard).
