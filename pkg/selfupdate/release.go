package selfupdate

import "time"

// Version holds parsed version information extracted from a release tag.
type Version struct {
	Original     string // raw tag name, e.g. "v1.2.3-beta"
	Version      string // extracted version, e.g. "1.2.3-beta"
	Major        int
	Minor        int
	Patch        int
	Prerelease   string // e.g. "beta", "" if stable
	IsPrerelease bool
}

// Platform describes the target OS/architecture.
type Platform struct {
	OS   string // "linux", "darwin", "windows"
	Arch string // "amd64", "arm64"
	Arm  uint8  // 5, 6, 7 if Arch is "arm", 0 otherwise
}

// Asset describes a downloadable release asset.
type Asset struct {
	ID   int64
	Name string // filename, e.g. "myapp_linux_amd64.tar.gz"
	URL  string // browser download URL
	Size int
}

// Release represents a GitHub release matched to the current platform.
type Release struct {
	Version      Version
	Platform     Platform
	Asset        Asset
	ReleaseID    int64
	URL          string // HTML URL of the release page
	ReleaseNotes string
	Name         string
	PublishedAt  time.Time

	repository Repository // private, for downloads
}
