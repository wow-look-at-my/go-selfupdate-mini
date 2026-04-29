package selfupdate

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
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

func TestVersionFromBuildInfoAutoreleaseClean(t *testing.T) {
	// 2026-04-27T09:52:22Z == unix 1777283542 (the wow-cli v0.0.1777283542 release).
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.modified", Value: "false"},
			{Key: "vcs.time", Value: "2026-04-27T09:52:22Z"},
		},
	}
	assert.Equal(t, "v0.0.1777283542", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoAutoreleaseDirty(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.modified", Value: "true"},
			{Key: "vcs.time", Value: "2026-04-27T09:52:22Z"},
		},
	}
	assert.Equal(t, "v0.0.1777283542+dirty", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoAutoreleasePrefersMainVersion(t *testing.T) {
	// Main.Version takes priority -- the autorelease branch must not preempt
	// a properly recorded module version (e.g. from "go install ...@v1.2.3").
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.modified", Value: "false"},
			{Key: "vcs.time", Value: "2026-04-27T09:52:22Z"},
		},
	}
	assert.Equal(t, "1.2.3", versionFromBuildInfo(info))
}

func TestVersionFromBuildInfoAutoreleaseBadTime(t *testing.T) {
	// Unparseable vcs.time falls through to the short-revision branch.
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.modified", Value: "false"},
			{Key: "vcs.time", Value: "not-a-date"},
		},
	}
	assert.Equal(t, "abcdef012345", versionFromBuildInfo(info))
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

// TestEmbeddedVersionViaLDFlags is the end-to-end check for the documented
// "ready-to-go" embedding mechanism: it builds a real binary that imports the
// library, injects a value into [EmbeddedVersion] via -ldflags "-X ...", and
// verifies the running binary surfaces that exact value through
// [CurrentVersion].
func TestEmbeddedVersionViaLDFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a real binary; skipped under -short")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go toolchain not on PATH: %v", err)
	}

	const wantVersion = "1.2.3-ldflags-test"

	binPath := filepath.Join(t.TempDir(), "versionprog")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	build := exec.Command(goBin, "build",
		"-o", binPath,
		"-ldflags", "-X github.com/wow-look-at-my/go-selfupdate-mini.EmbeddedVersion="+wantVersion,
		"./testdata/versionprog",
	)
	buildOut, err := build.CombinedOutput()
	require.Nilf(t, err, "go build failed: %s", buildOut)

	runOut, err := exec.Command(binPath).CombinedOutput()
	require.Nilf(t, err, "binary execution failed: %s", runOut)
	assert.Equal(t, wantVersion, string(runOut))
}
