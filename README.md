# go-selfupdate-mini

A minimal Go library for self-updating binaries via GitHub Releases.

## Features

- Detect latest or specific versions from GitHub Releases
- Download and atomically replace the running binary with rollback on failure
- Multi-platform support with automatic OS/architecture detection
- ARM variant fallback (armv7 -> armv6 -> armv5 -> arm) and macOS universal binary support
- Built-in decompression for `.zip`, `.tar.gz`, `.tgz`, `.gz`, `.bz2`
- GitHub Enterprise support
- Pluggable source, decompressor, validator, and installer interfaces
- No dependency on `github.com/google/go-github` -- uses the GitHub REST API directly

## Install

```sh
go get github.com/wow-look-at-my/go-selfupdate-mini
```

## Usage

### Detect the latest release

```go
import selfupdate "github.com/wow-look-at-my/go-selfupdate-mini"

rel, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug("owner/repo"))
if err != nil {
    log.Fatal(err)
}
if !found {
    log.Println("no release found")
    return
}
fmt.Printf("Latest: %s\n", rel.Version.Version)
```

### Update the running binary

```go
// Pass the running binary's version, or "" to auto-detect via CurrentVersion().
rel, err := selfupdate.UpdateSelf(ctx, "", selfupdate.ParseSlug("owner/repo"))
if err != nil {
    log.Fatal(err)
}
if rel == nil {
    fmt.Println("already up to date")
} else {
    fmt.Printf("updated to %s\n", rel.Version.Version)
}
```

### Cobra integration with auto-detected version

`RegisterCommands` wires up `version` and `update` subcommands plus the
`--version` flag. The current version is auto-detected, so most apps need nothing more:

```go
selfupdate.RegisterCommands(rootCmd, selfupdate.ParseSlug("owner/repo"))
```

To override the auto-detected version explicitly, pass `WithVersion`:

```go
selfupdate.RegisterCommands(rootCmd, repo, selfupdate.WithVersion("1.0.0"))
```

To optionally add an `install` command (for downloading and installing arbitrary releases
to a destination path), call `RegisterInstallCommand` separately:

```go
selfupdate.RegisterInstallCommand(rootCmd, selfupdate.ParseSlug("owner/repo"))
```

## Version embedding

`selfupdate.CurrentVersion()` resolves the binary's version from the first source available:

1. The package-level `selfupdate.EmbeddedVersion` variable, settable via ldflags or directly:

   ```sh
   go build -ldflags "-X github.com/wow-look-at-my/go-selfupdate-mini.EmbeddedVersion=1.2.3"
   ```

   ```go
   selfupdate.EmbeddedVersion = appVersion
   ```

2. The main module version from `runtime/debug.ReadBuildInfo`. This is populated automatically by

   ```sh
   go install github.com/owner/repo@v1.2.3
   ```

3. The VCS commit time rendered as the `wow-look-at-my/go-toolchain` "autorelease" tag
   `v0.0.<unix-seconds>` (with a `+dirty` suffix when the working tree was modified at build time),
   when `vcs.time` is recorded in the build info. `UpdateSelf` refuses to overwrite a `+dirty`
   build, since no released artifact corresponds to a dirty working tree.

4. A short VCS revision (with `+dirty` suffix when the working tree was modified) -- for ad-hoc
   `go build` from a checkout without VCS time information.

5. `(devel)` as a final fallback.

Because `CurrentVersion()` does the work, you do not need to specify the current version anywhere
when calling `RegisterCommands`, `UpdateSelf`, or `UpdateCommand` -- pass `""` to auto-detect.

### Custom updater

```go
src, _ := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{
    APIToken: os.Getenv("GITHUB_TOKEN"),
})

updater, _ := selfupdate.NewUpdater(selfupdate.Config{
    Source:  src,
    Filters: []string{`^myapp_`}, // only consider assets matching this pattern
})

rel, err := updater.UpdateSelf(ctx, currentVersion, selfupdate.ParseSlug("owner/repo"))
```

### GitHub Enterprise

```go
src, _ := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{
    APIToken:          "your-token",
    EnterpriseBaseURL: "https://github.example.com/api/v3",
})
```

## Authentication

The GitHub API token is resolved in order:

1. `GitHubConfig.APIToken` field
2. `$GITHUB_TOKEN` environment variable

A token is not required for public repositories, but recommended to avoid rate limits.

## Asset naming

Release assets are matched by OS and architecture suffixes. For example, a binary named `myapp` with a release for Linux amd64 should have an asset like:

```
myapp_linux_amd64.tar.gz
myapp_linux_x86_64.zip
```

## License

MIT
