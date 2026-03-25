# go-selfupdate-mini

A minimal Go library for self-updating binaries from GitHub releases, with ready-to-use [cobra](https://github.com/spf13/cobra) commands.

## Features

- **Self-update from GitHub releases** - detect, download, decompress, and atomically replace the running binary
- **Cobra integration** - drop-in `update` and `version` subcommands, plus a `--version` flag
- **Up-to-date detection** - check whether a newer version exists without installing it
- **Days-out-of-date** - see how many days behind the latest release you are
- **Multi-platform** - automatic OS/arch detection (Linux, macOS, Windows, ARM)
- **Flexible** - custom sources, version comparators, validators, decompressors, and install handlers

## Install

```sh
go get github.com/wow-look-at-my/go-selfupdate-mini
```

## Quick start with cobra

```go
package main

import (
    "fmt"
    "os"

    selfupdate "github.com/wow-look-at-my/go-selfupdate-mini"
    "github.com/spf13/cobra"
)

var version = "0.0.0-dev"

func main() {
    repo := selfupdate.ParseSlug("owner/repo")
    cfg := &selfupdate.CobraConfig{
        Version:    version,
        Repository: repo,
    }

    root := &cobra.Command{
        Use: "myapp",
    }

    // Adds `myapp update` and `myapp version` subcommands
    root.AddCommand(selfupdate.NewUpdateCommand(cfg))
    root.AddCommand(selfupdate.NewVersionCommand(cfg))

    // Adds `myapp --version` flag
    selfupdate.AddVersionFlag(root, version)

    if err := root.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

Running the commands:

```sh
$ myapp version
myapp version 1.0.0
Update available: 2.1.0 (released 12 day(s) ago)
Run 'myapp update' to update.

$ myapp update
Updating from 1.0.0 to 2.1.0...
Updated to version 2.1.0.

$ myapp --version
myapp version 1.0.0
```

## Library usage

### Check for updates without installing

```go
status, err := selfupdate.CheckUpdate(ctx, "1.0.0", selfupdate.ParseSlug("owner/repo"))
if err != nil {
    log.Fatal(err)
}
if status.UpdateAvailable {
    fmt.Printf("New version %s available (you are %d days behind)\n",
        status.LatestRelease.Version.Version,
        status.DaysOutOfDate())
}
```

### Self-update programmatically

```go
rel, err := selfupdate.UpdateSelf(ctx, "1.0.0", selfupdate.ParseSlug("owner/repo"))
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Updated to %s\n", rel.Version.Version)
```

### Detect latest release

```go
rel, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug("owner/repo"))
if err != nil {
    log.Fatal(err)
}
if found {
    fmt.Printf("Latest: %s published at %s\n", rel.Version.Version, rel.PublishedAt)
}
```

## Configuration

Use `selfupdate.NewUpdater(config)` for fine-grained control:

```go
up, err := selfupdate.NewUpdater(selfupdate.Config{
    // Use a custom source (default: GitHub)
    Source: myGitLabSource,

    // Override platform detection
    Platform: selfupdate.Platform{OS: "linux", Arch: "arm64"},

    // Include pre-releases
    Version: selfupdate.VersionFilter{Prerelease: true},

    // Filter assets by regex instead of OS/arch suffix matching
    Filters: []string{`myapp_.*\.tar\.gz$`},

    // Validate downloaded binary before install
    Validate: func(rel *selfupdate.Release, data []byte) error {
        return verifyChecksum(data)
    },

    // Custom version comparison
    CompareVersions: func(current, candidate selfupdate.Version) bool {
        return candidate.Major > current.Major
    },

    // Custom decompressor
    Decompressors: map[string]selfupdate.Decompressor{
        ".zst": myZstdDecompressor,
    },

    // Back up old binary
    OldSavePath: "/tmp/myapp.old",
})
```

## GitHub Enterprise / private repos

```go
source, _ := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{
    APIToken: os.Getenv("GITHUB_TOKEN"),
    BaseURL:  "https://github.example.com/api/v3",
})
up, _ := selfupdate.NewUpdater(selfupdate.Config{Source: source})
```

## Supported archive formats

Built-in: `.zip`, `.tar.gz`, `.tgz`, `.gz`, `.gzip`, `.bz2`

Add custom formats via `Config.Decompressors`.

## Asset naming convention

By default, assets are matched by suffix: `{os}_{arch}{ext}` (e.g., `myapp_linux_amd64.tar.gz`). Both `_` and `-` separators are supported. Use `Config.Filters` for custom naming schemes.
