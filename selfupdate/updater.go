package selfupdate

import (
	"fmt"
	"io"
	"regexp"
	"runtime"
)

// Updater is responsible for managing the context of self-update.
type Updater struct {
	source          Source
	platform        Platform
	universalArch   string
	versionFilter   VersionFilter
	filters         []*regexp.Regexp
	compareVersions func(current, candidate Version) bool
	validate        func(release *Release, data []byte) error
	install         func(assetReader io.Reader, cmdPath string) error
	decompressors   map[string]Decompressor
}

var defaultUpdater *Updater

// NewUpdater creates a new updater instance.
// If you don't specify a source in the config object, GitHub will be used.
func NewUpdater(config Config) (*Updater, error) {
	source := config.Source
	if source == nil {
		source, _ = NewGitHubSource(GitHubConfig{})
	}

	filtersRe := make([]*regexp.Regexp, 0, len(config.Filters))
	for _, filter := range config.Filters {
		re, err := regexp.Compile(filter)
		if err != nil {
			return nil, fmt.Errorf("could not compile regular expression %q for filtering releases: %w", filter, err)
		}
		filtersRe = append(filtersRe, re)
	}

	platform := config.Platform
	if platform.OS == "" {
		platform.OS = runtime.GOOS
	}
	if platform.Arch == "" {
		platform.Arch = runtime.GOARCH
	}
	if platform.Arm == 0 && platform.Arch == "arm" {
		exe, _ := getExecutablePath()
		platform.Arm = getGOARM(exe)
	}

	universalArch := ""
	if platform.OS == "darwin" && config.UniversalArch != "" {
		universalArch = config.UniversalArch
	}

	// merge decompressors: built-ins first, then user overrides
	decompressors := builtinDecompressors(platform.OS, platform.Arch)
	for ext, d := range config.Decompressors {
		decompressors[ext] = d
	}

	install := config.Install
	if install == nil {
		install = defaultInstall(config.OldSavePath)
	}

	return &Updater{
		source:          source,
		platform:        platform,
		universalArch:   universalArch,
		versionFilter:   config.Version,
		filters:         filtersRe,
		compareVersions: config.CompareVersions,
		validate:        config.Validate,
		install:         install,
		decompressors:   decompressors,
	}, nil
}

// DefaultUpdater creates a new updater instance with default configuration.
// Every call returns the same instance.
func DefaultUpdater() *Updater {
	if defaultUpdater != nil {
		return defaultUpdater
	}
	defaultUpdater, _ = NewUpdater(Config{})
	return defaultUpdater
}
