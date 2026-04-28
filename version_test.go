package selfupdate

import (
	"runtime/debug"
	"testing"

	"github.com/wow-look-at-my/testify/assert"
)

func TestCurrentVersionReturnsExplicitVersion(t *testing.T) {
	prev := EmbeddedVersion
	EmbeddedVersion = "1.2.3"
	t.Cleanup(func() { EmbeddedVersion = prev })

	assert.Equal(t, "1.2.3", CurrentVersion())
}

func TestCurrentVersionFallsBackWhenUnset(t *testing.T) {
	prev := EmbeddedVersion
	EmbeddedVersion = ""
	t.Cleanup(func() { EmbeddedVersion = prev })

	resetDetectedVersion()
	t.Cleanup(resetDetectedVersion)

	got := CurrentVersion()
	// Under `go test`, debug.ReadBuildInfo always succeeds and reports VCS
	// settings (or "(devel)"). Either way the result must be non-empty.
	assert.NotEqual(t, "", got)
}

func TestCurrentVersionPrefersEmbedded(t *testing.T) {
	prev := EmbeddedVersion
	t.Cleanup(func() { EmbeddedVersion = prev })

	EmbeddedVersion = "v9.9.9"
	assert.Equal(t, "v9.9.9", CurrentVersion())
}

func TestVersionFromBuildInfoNil(t *testing.T) {
	assert.Equal(t, "(devel)", versionFromBuildInfo(nil))
}

func TestVersionFromBuildInfoMainVersion(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v2.4.6"}}
	assert.Equal(t, "2.4.6", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoMainVersionNoPrefix(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "2.4.6"}}
	assert.Equal(t, "2.4.6", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoIgnoresDevelMain(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
		},
	}
	assert.Equal(t, "abcdef012345", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoVCSClean(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.modified", Value: "false"},
		},
	}
	assert.Equal(t, "abcdef012345", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoVCSDirty(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	assert.Equal(t, "abcdef012345+dirty", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoShortRevision(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef"},
		},
	}
	assert.Equal(t, "abcdef", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoNoVCSNoMain(t *testing.T) {
	info := &debug.BuildInfo{}
	assert.Equal(t, "(devel)", versionFromBuildInfo(info))
}

func TestDetectEmbeddedVersionPopulatesDetected(t *testing.T) {
	prev := EmbeddedVersion
	EmbeddedVersion = ""
	t.Cleanup(func() { EmbeddedVersion = prev })

	resetDetectedVersion()
	t.Cleanup(resetDetectedVersion)

	// Trigger the lazy path through CurrentVersion.
	got := CurrentVersion()
	assert.NotEqual(t, "", got)
	// Calling again returns the same memoised value without re-running
	// detectEmbeddedVersion (sync.Once contract).
	assert.Equal(t, got, CurrentVersion())
}
