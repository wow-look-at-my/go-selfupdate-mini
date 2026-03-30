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
rel, err := selfupdate.UpdateSelf(ctx, "1.0.0", selfupdate.ParseSlug("owner/repo"))
if err != nil {
    log.Fatal(err)
}
if rel == nil {
    fmt.Println("already up to date")
} else {
    fmt.Printf("updated to %s\n", rel.Version.Version)
}
```

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
