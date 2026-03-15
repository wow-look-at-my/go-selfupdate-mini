package selfupdate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateToNilRelease(t *testing.T) {
	up, _ := NewUpdater(Config{Platform: Platform{OS: "linux", Arch: "amd64"}})
	err := up.UpdateTo(context.Background(), nil, "/tmp/test")
	if !errors.Is(err, ErrInvalidRelease) {
		t.Errorf("expected ErrInvalidRelease, got %v", err)
	}
}

func TestUpdateToWithValidation(t *testing.T) {
	src := &mockSource{
		assets: map[int64]string{1: "binary content"},
	}
	validationCalled := false
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Validate: func(rel *Release, data []byte) error {
			validationCalled = true
			if string(data) != "binary content" {
				return fmt.Errorf("unexpected data")
			}
			return nil
		},
		Install: func(r io.Reader, path string) error {
			return nil
		},
	})

	rel := &Release{
		Asset:      Asset{ID: 1, Name: "app", URL: "https://example.com/app"},
		repository: NewRepositorySlug("test", "repo"),
	}
	err := up.UpdateTo(context.Background(), rel, "/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	if !validationCalled {
		t.Error("validation callback was not called")
	}
}

func TestUpdateToValidationFailure(t *testing.T) {
	src := &mockSource{
		assets: map[int64]string{1: "binary"},
	}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Validate: func(rel *Release, data []byte) error {
			return fmt.Errorf("bad signature")
		},
		Install: func(r io.Reader, path string) error {
			t.Error("install should not be called after validation failure")
			return nil
		},
	})

	rel := &Release{
		Asset:      Asset{ID: 1, Name: "app", URL: "https://example.com/app"},
		repository: NewRepositorySlug("test", "repo"),
	}
	err := up.UpdateTo(context.Background(), rel, "/tmp/test")
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestUpdateToDownloadError(t *testing.T) {
	src := &mockSource{
		assets: map[int64]string{}, // empty, will error
	}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	rel := &Release{
		Asset:      Asset{ID: 999, Name: "app", URL: "https://example.com/app"},
		repository: NewRepositorySlug("test", "repo"),
	}
	err := up.UpdateTo(context.Background(), rel, "/tmp/test")
	if err == nil {
		t.Error("expected download error")
	}
}

func TestUpdateToWithDecompression(t *testing.T) {
	tarGz := makeTarGz(t, map[string][]byte{"myapp": []byte("new binary")})
	src := &mockSource{
		assets: map[int64]string{1: string(tarGz)},
	}

	var installed []byte
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			var err error
			installed, err = io.ReadAll(r)
			return err
		},
	})

	rel := &Release{
		Asset:      Asset{ID: 1, Name: "app_linux_amd64.tar.gz", URL: "https://example.com/app.tar.gz"},
		repository: NewRepositorySlug("test", "repo"),
	}
	err := up.UpdateTo(context.Background(), rel, "/usr/local/bin/myapp")
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != "new binary" {
		t.Errorf("unexpected installed content: %q", installed)
	}
}

func TestUpdateCommandNoRelease(t *testing.T) {
	src := &mockSource{releases: []SourceRelease{}}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old"), 0o755)

	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	rel, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version.Version != "1.0.0" {
		t.Errorf("expected current version returned, got %s", rel.Version.Version)
	}
}

func TestUpdateCommandAlreadyLatest(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("myapp_linux_amd64.tar.gz")),
		},
	}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old"), 0o755)

	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	rel, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version.Version != "1.0.0" {
		t.Errorf("expected 1.0.0 (already latest), got %s", rel.Version.Version)
	}
}

func TestUpdateCommandBadVersion(t *testing.T) {
	up, _ := NewUpdater(Config{Platform: Platform{OS: "linux", Arch: "amd64"}})
	_, err := up.UpdateCommand(context.Background(), "/tmp/x", "not-a-version", NewRepositorySlug("test", "repo"))
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestUpdateCommandFileNotFound(t *testing.T) {
	up, _ := NewUpdater(Config{Platform: Platform{OS: "linux", Arch: "amd64"}})
	_, err := up.UpdateCommand(context.Background(), "/nonexistent/path", "1.0.0", NewRepositorySlug("test", "repo"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestUpdateCommandPerformsUpdate(t *testing.T) {
	tarGz := makeTarGz(t, map[string][]byte{"myapp": []byte("new binary v2")})
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64.tar.gz", size: 100, url: "https://example.com/app.tar.gz"}),
		},
		assets: map[int64]string{1: string(tarGz)},
	}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old binary"), 0o755)

	var installedContent []byte
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			var err error
			installedContent, err = io.ReadAll(r)
			return err
		},
	})

	rel, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version.Version != "2.0.0" {
		t.Errorf("expected 2.0.0, got %s", rel.Version.Version)
	}
	if string(installedContent) != "new binary v2" {
		t.Errorf("unexpected installed content: %q", installedContent)
	}
}

func TestDecompressAndInstall(t *testing.T) {
	var installPath string
	up, _ := NewUpdater(Config{
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			installPath = path
			return nil
		},
	})

	err := up.decompressAndInstall(
		bytes.NewReader([]byte("raw binary")),
		"myapp",
		"https://example.com/myapp",
		"/usr/local/bin/myapp",
	)
	if err != nil {
		t.Fatal(err)
	}
	if installPath != "/usr/local/bin/myapp" {
		t.Errorf("unexpected install path: %s", installPath)
	}
}

func TestDownload(t *testing.T) {
	src := &mockSource{
		assets: map[int64]string{42: "downloaded bytes"},
	}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	rel := &Release{repository: NewRepositorySlug("test", "repo")}
	data, err := up.download(context.Background(), rel, 42)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "downloaded bytes" {
		t.Errorf("unexpected: %q", data)
	}
}

func TestDownloadError(t *testing.T) {
	src := &mockSource{assets: map[int64]string{}}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	rel := &Release{repository: NewRepositorySlug("test", "repo")}
	_, err := up.download(context.Background(), rel, 999)
	if err == nil {
		t.Error("expected error for missing asset")
	}
}

func TestUpdateCommandCustomCompare(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64.tar.gz", size: 100, url: "https://example.com/app.tar.gz"}),
		},
		assets: map[int64]string{1: "binary"},
	}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old"), 0o755)

	// Custom compare that says nothing is newer
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		CompareVersions: func(current, candidate Version) bool {
			return false
		},
	})

	rel, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	// Should return latest but not update
	if rel.Version.Version != "2.0.0" {
		t.Errorf("expected 2.0.0, got %s", rel.Version.Version)
	}
}

func TestUpdateCommandSourceError(t *testing.T) {
	src := &mockSource{err: fmt.Errorf("network error")}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old"), 0o755)

	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	_, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	if err == nil {
		t.Error("expected error from source")
	}
}

func TestPackageLevelDownloadReleaseAssetFromURL(t *testing.T) {
	// Just test the error path since we can't easily test success without a real server
	_, err := downloadReleaseAssetFromURL(context.Background(), "http://[invalid-url")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestDecompressAndInstallError(t *testing.T) {
	up, _ := NewUpdater(Config{
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			return nil
		},
	})

	// invalid tar.gz data
	err := up.decompressAndInstall(
		strings.NewReader("not valid"),
		"app.tar.gz",
		"https://example.com/app.tar.gz",
		"/usr/local/bin/app",
	)
	if err == nil {
		t.Error("expected decompression error")
	}
}
