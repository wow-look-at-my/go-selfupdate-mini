package selfupdate

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var pseudoVersionRE = regexp.MustCompile(`-\d{14}-[A-Fa-f0-9]{12}(\+|$)`)

// EmbeddedVersion is the current version of the running binary. Consuming
// applications populate it in any of three ways:
//
//  1. ldflags injection at build time:
//
//     go build -ldflags "-X github.com/wow-look-at-my/go-selfupdate-mini.EmbeddedVersion=1.2.3"
//
//  2. Direct assignment from main, e.g. when the application keeps its own
//     version variable:
//
//     selfupdate.EmbeddedVersion = appVersion
//
//  3. Leave it empty -- [CurrentVersion] then derives a value from
//     [debug.ReadBuildInfo], which Just Works for binaries installed via
//     "go install <module>@<version>" or built from a VCS checkout.
//
// Note: this is a value (not the [Version] struct, which models a parsed
// release tag).
var EmbeddedVersion string

var (
	detectVersionOnce sync.Once
	detectedVersion   string
)

// CurrentVersion returns the binary's current version string in priority order:
//
//  1. The package-level [EmbeddedVersion] variable, if non-empty.
//  2. The main module version reported by [debug.ReadBuildInfo] -- populated
//     automatically by "go install <module>@<version>".
//  3. The VCS commit time rendered as the wow-look-at-my "autorelease" tag
//     "v0.0.<unix-seconds>" (with a "+dirty" suffix when the working tree was
//     modified at build time), when [debug.BuildInfo.Settings] carries a
//     parseable "vcs.time" entry.
//  4. A short VCS revision (with a "+dirty" suffix when the working tree was
//     modified at build time) extracted from [debug.BuildInfo.Settings].
//  5. "(devel)" as a final fallback.
//
// The returned string is opaque: paths 2 and 3 are guaranteed to be parseable
// release tags. Self-update flows that compare versions (e.g.
// [Updater.UpdateSelf]) require a parseable semver, so callers using a
// short-revision/(devel) fallback should pass an explicit "--version" or set
// [EmbeddedVersion] via ldflags before shipping.
func CurrentVersion() string {
	if EmbeddedVersion != "" {
		return EmbeddedVersion
	}
	detectVersionOnce.Do(detectEmbeddedVersion)
	return detectedVersion
}

func detectEmbeddedVersion() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		detectedVersion = "(devel)"
		return
	}
	detectedVersion = versionFromBuildInfo(info)
}

// versionFromBuildInfo applies the prioritisation rules of [CurrentVersion] to
// a concrete [debug.BuildInfo]. Split out so tests can drive every branch
// without relying on the ambient build context.
func versionFromBuildInfo(info *debug.BuildInfo) string {
	if info == nil {
		return "(devel)"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" && !pseudoVersionRE.MatchString(v) {
		return strings.TrimPrefix(v, "v")
	}

	var revision, vcsTime string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		case "vcs.time":
			vcsTime = s.Value
		}
	}

	// wow-look-at-my "autorelease" tag scheme: every push to the default
	// branch is tagged "v0.0.<unix-seconds>" where the suffix is the commit
	// time. vcs.time is the same value rendered as RFC3339, so converting
	// back to unix seconds yields the exact tag a binary built from this
	// commit was published under.
	if vcsTime != "" {
		if t, err := time.Parse(time.RFC3339, vcsTime); err == nil {
			v := fmt.Sprintf("v0.0.%d", t.Unix())
			if modified {
				v += "+dirty"
			}
			return v
		}
	}

	if revision == "" {
		return "(devel)"
	}
	short := revision
	if len(short) > 12 {
		short = short[:12]
	}
	if modified {
		return fmt.Sprintf("%s+dirty", short)
	}
	return short
}

// resetDetectedVersion is a test helper that clears the memoised detection
// result so tests can re-exercise [detectEmbeddedVersion] under different
// conditions.
func resetDetectedVersion() {
	detectVersionOnce = sync.Once{}
	detectedVersion = ""
}
