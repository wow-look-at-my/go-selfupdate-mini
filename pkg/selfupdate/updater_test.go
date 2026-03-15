package selfupdate

import (
	"io"
	"testing"
)

func TestNewUpdaterDefaults(t *testing.T) {
	up, err := NewUpdater(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if up.platform.OS == "" {
		t.Error("OS should default to runtime.GOOS")
	}
	if up.platform.Arch == "" {
		t.Error("Arch should default to runtime.GOARCH")
	}
	if up.source == nil {
		t.Error("source should default to GitHubSource")
	}
	if up.install == nil {
		t.Error("install should have default handler")
	}
	if len(up.decompressors) == 0 {
		t.Error("decompressors should have builtins")
	}
}

func TestNewUpdaterCustomPlatform(t *testing.T) {
	up, err := NewUpdater(Config{
		Platform: Platform{OS: "darwin", Arch: "arm64"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if up.platform.OS != "darwin" {
		t.Errorf("expected darwin, got %s", up.platform.OS)
	}
	if up.platform.Arch != "arm64" {
		t.Errorf("expected arm64, got %s", up.platform.Arch)
	}
}

func TestNewUpdaterUniversalArch(t *testing.T) {
	up, _ := NewUpdater(Config{
		Platform:      Platform{OS: "darwin", Arch: "amd64"},
		UniversalArch: "universal",
	})
	if up.universalArch != "universal" {
		t.Error("universalArch should be set for darwin")
	}

	up, _ = NewUpdater(Config{
		Platform:      Platform{OS: "linux", Arch: "amd64"},
		UniversalArch: "universal",
	})
	if up.universalArch != "" {
		t.Error("universalArch should be empty for non-darwin")
	}
}

func TestNewUpdaterFilters(t *testing.T) {
	up, err := NewUpdater(Config{
		Filters: []string{"linux", "amd64"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(up.filters) != 2 {
		t.Errorf("expected 2 filters, got %d", len(up.filters))
	}
}

func TestNewUpdaterCustomDecompressors(t *testing.T) {
	custom := DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) {
		return src, nil
	})
	up, _ := NewUpdater(Config{
		Decompressors: map[string]Decompressor{".zst": custom},
	})
	if _, ok := up.decompressors[".zst"]; !ok {
		t.Error("custom decompressor should be registered")
	}
	// builtins should still exist
	if _, ok := up.decompressors[".zip"]; !ok {
		t.Error("builtin .zip should still exist")
	}
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
	if !called {
		t.Error("custom install should be used")
	}
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
	if !validateCalled {
		t.Error("validate callback should be stored")
	}

	up.compareVersions(Version{}, Version{})
	if !compareCalled {
		t.Error("compareVersions callback should be stored")
	}
}

func TestDefaultUpdater(t *testing.T) {
	// Reset singleton
	defaultUpdater = nil
	defer func() { defaultUpdater = nil }()

	up1 := DefaultUpdater()
	up2 := DefaultUpdater()
	if up1 != up2 {
		t.Error("DefaultUpdater should return same instance")
	}
}
