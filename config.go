package selfupdate

import "io"

// Decompressor extracts an executable from a compressed/archived reader.
type Decompressor interface {
	Decompress(src io.Reader, cmd string) (io.Reader, error)
}

// DecompressorFunc adapts a function to the Decompressor interface.
type DecompressorFunc func(src io.Reader, cmd string) (io.Reader, error)

func (f DecompressorFunc) Decompress(src io.Reader, cmd string) (io.Reader, error) {
	return f(src, cmd)
}

// VersionFilter controls which releases are considered during detection.
type VersionFilter struct {
	Draft      bool // include draft releases
	Prerelease bool // include prerelease versions
}

// Config represents the configuration of self-update.
type Config struct {
	// Source where to load releases from. Defaults to GitHubSource.
	Source Source

	// Platform targeting. Defaults to runtime.GOOS/GOARCH with auto-detected ARM version.
	Platform Platform

	// Version controls which releases are considered (drafts, prereleases).
	Version VersionFilter

	// UniversalArch is the architecture name for macOS universal binaries.
	// If set, the updater will pick the universal binary only when the
	// platform arch is not found.
	UniversalArch string

	// Filters are regexps for matching asset names. If set, suffix matching
	// is skipped and any asset matching a filter is selected.
	Filters []string

	// Validate, if set, validates downloaded asset bytes before install.
	// Return nil to proceed, error to abort the update.
	Validate func(release *Release, data []byte) error

	// CompareVersions returns true if candidate is newer than current.
	// If nil, semantic versioning comparison is used.
	CompareVersions func(current, candidate Version) bool

	// Install applies the update. Receives a reader for the new binary
	// and the path to the current executable. If nil, defaults to atomic
	// overwrite (write tmp, rename over current).
	Install func(assetReader io.Reader, cmdPath string) error

	// Decompressors maps file extensions to decompressor implementations.
	// Merged with built-in defaults (.zip, .tar.gz, .tgz, .gz, .bz2).
	// User entries take precedence over built-ins.
	Decompressors map[string]Decompressor

	// OldSavePath, if set, backs up old binary before replacing.
	// Only used by the default Install handler.
	OldSavePath string
}
