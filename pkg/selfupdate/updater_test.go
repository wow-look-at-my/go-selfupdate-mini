package selfupdate

import (
	"io"
	"testing"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestNewUpdaterDefaults(t *testing.T) {
	up, err := NewUpdater(Config{})
	require.Nil(t, err)

	assert.NotEqual(t, "", up.platform.OS)

	assert.NotEqual(t, "", up.platform.Arch)

	assert.NotNil(t, up.source)

	assert.NotNil(t, up.install)

	assert.NotEqual(t, 0, len(up.decompressors))

}

func TestNewUpdaterCustomPlatform(t *testing.T) {
	up, err := NewUpdater(Config{
		Platform: Platform{OS: "darwin", Arch: "arm64"},
	})
	require.Nil(t, err)

	assert.Equal(t, "darwin", up.platform.OS)

	assert.Equal(t, "arm64", up.platform.Arch)

}

func TestNewUpdaterUniversalArch(t *testing.T) {
	up, _ := NewUpdater(Config{
		Platform:	Platform{OS: "darwin", Arch: "amd64"},
		UniversalArch:	"universal",
	})
	assert.Equal(t, "universal", up.universalArch)

	up, _ = NewUpdater(Config{
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		UniversalArch:	"universal",
	})
	assert.Equal(t, "", up.universalArch)

}

func TestNewUpdaterFilters(t *testing.T) {
	up, err := NewUpdater(Config{
		Filters: []string{"linux", "amd64"},
	})
	require.Nil(t, err)

	assert.Equal(t, 2, len(up.filters))

}

func TestNewUpdaterCustomDecompressors(t *testing.T) {
	custom := DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) {
		return src, nil
	})
	up, _ := NewUpdater(Config{
		Decompressors: map[string]Decompressor{".zst": custom},
	})
	_, ok := up.decompressors[".zst"]
	assert.True(t, ok)

	// builtins should still exist
	_, ok = up.decompressors[".zip"]
	assert.True(t, ok)

}

func TestNewUpdaterCustomInstall(t *testing.T) {
	called := false
	up, _ := NewUpdater(Config{
		Install: func(r io.Reader, path string) error {
			called = true
			return nil
		},
	})
	up.install(nil, "")
	assert.True(t, called)

}

func TestNewUpdaterCallbacks(t *testing.T) {
	validateCalled := false
	compareCalled := false
	up, _ := NewUpdater(Config{
		Validate: func(rel *Release, data []byte) error {
			validateCalled = true
			return nil
		},
		CompareVersions: func(current, candidate Version) bool {
			compareCalled = true
			return true
		},
	})

	up.validate(&Release{}, nil)
	assert.True(t, validateCalled)

	up.compareVersions(Version{}, Version{})
	assert.True(t, compareCalled)

}

func TestDefaultUpdater(t *testing.T) {
	// Reset singleton
	defaultUpdater = nil
	defer func() { defaultUpdater = nil }()

	up1 := DefaultUpdater()
	up2 := DefaultUpdater()
	assert.Equal(t, up2, up1)

}
