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
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestUpdateToNilRelease(t *testing.T) {
	up, _ := NewUpdater(Config{Platform: Platform{OS: "linux", Arch: "amd64"}})
	err := up.UpdateTo(context.Background(), nil, "/tmp/test")
	assert.True(t, errors.Is(err, ErrInvalidRelease))

}

func TestUpdateToWithValidation(t *testing.T) {
	src := &mockSource{
		assets: map[int64]string{1: "binary content"},
	}
	validationCalled := false
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
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
		Asset:		Asset{ID: 1, Name: "app", URL: "https://example.com/app"},
		repository:	NewRepositorySlug("test", "repo"),
	}
	err := up.UpdateTo(context.Background(), rel, "/tmp/test")
	require.Nil(t, err)

	assert.True(t, validationCalled)

}

func TestUpdateToValidationFailure(t *testing.T) {
	src := &mockSource{
		assets: map[int64]string{1: "binary"},
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		Validate: func(rel *Release, data []byte) error {
			return fmt.Errorf("bad signature")
		},
		Install: func(r io.Reader, path string) error {
			t.Error("install should not be called after validation failure")
			return nil
		},
	})

	rel := &Release{
		Asset:		Asset{ID: 1, Name: "app", URL: "https://example.com/app"},
		repository:	NewRepositorySlug("test", "repo"),
	}
	err := up.UpdateTo(context.Background(), rel, "/tmp/test")
	assert.NotNil(t, err)

}

func TestUpdateToDownloadError(t *testing.T) {
	src := &mockSource{
		assets: map[int64]string{},	// empty, will error
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	rel := &Release{
		Asset:		Asset{ID: 999, Name: "app", URL: "https://example.com/app"},
		repository:	NewRepositorySlug("test", "repo"),
	}
	err := up.UpdateTo(context.Background(), rel, "/tmp/test")
	assert.NotNil(t, err)

}

func TestUpdateToWithDecompression(t *testing.T) {
	tarGz := makeTarGz(t, map[string][]byte{"myapp": []byte("new binary")})
	src := &mockSource{
		assets: map[int64]string{1: string(tarGz)},
	}

	var installed []byte
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			var err error
			installed, err = io.ReadAll(r)
			return err
		},
	})

	rel := &Release{
		Asset:		Asset{ID: 1, Name: "app_linux_amd64.tar.gz", URL: "https://example.com/app.tar.gz"},
		repository:	NewRepositorySlug("test", "repo"),
	}
	err := up.UpdateTo(context.Background(), rel, "/usr/local/bin/myapp")
	require.Nil(t, err)

	assert.Equal(t, "new binary", string(installed))

}

func TestUpdateCommandNoRelease(t *testing.T) {
	src := &mockSource{releases: []SourceRelease{}}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old"), 0o755)

	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	rel, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	assert.Equal(t, "1.0.0", rel.Version.Version)

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
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	rel, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	assert.Equal(t, "1.0.0", rel.Version.Version)

}

func TestUpdateCommandBadVersion(t *testing.T) {
	up, _ := NewUpdater(Config{Platform: Platform{OS: "linux", Arch: "amd64"}})
	_, err := up.UpdateCommand(context.Background(), "/tmp/x", "not-a-version", NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)

}

func TestUpdateCommandFileNotFound(t *testing.T) {
	up, _ := NewUpdater(Config{Platform: Platform{OS: "linux", Arch: "amd64"}})
	_, err := up.UpdateCommand(context.Background(), "/nonexistent/path", "1.0.0", NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)

}

func TestUpdateCommandPerformsUpdate(t *testing.T) {
	tarGz := makeTarGz(t, map[string][]byte{"myapp": []byte("new binary v2")})
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64.tar.gz", size: 100, url: "https://example.com/app.tar.gz"}),
		},
		assets:	map[int64]string{1: string(tarGz)},
	}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old binary"), 0o755)

	var installedContent []byte
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			var err error
			installedContent, err = io.ReadAll(r)
			return err
		},
	})

	rel, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	assert.Equal(t, "2.0.0", rel.Version.Version)

	assert.Equal(t, "new binary v2", string(installedContent))

}

func TestDecompressAndInstall(t *testing.T) {
	var installPath string
	up, _ := NewUpdater(Config{
		Platform:	Platform{OS: "linux", Arch: "amd64"},
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
	require.Nil(t, err)

	assert.Equal(t, "/usr/local/bin/myapp", installPath)

}

func TestDownload(t *testing.T) {
	src := &mockSource{
		assets: map[int64]string{42: "downloaded bytes"},
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	rel := &Release{repository: NewRepositorySlug("test", "repo")}
	data, err := up.download(context.Background(), rel, 42)
	require.Nil(t, err)

	assert.Equal(t, "downloaded bytes", string(data))

}

func TestDownloadError(t *testing.T) {
	src := &mockSource{assets: map[int64]string{}}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	rel := &Release{repository: NewRepositorySlug("test", "repo")}
	_, err := up.download(context.Background(), rel, 999)
	assert.NotNil(t, err)

}

func TestUpdateCommandCustomCompare(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64.tar.gz", size: 100, url: "https://example.com/app.tar.gz"}),
		},
		assets:	map[int64]string{1: "binary"},
	}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old"), 0o755)

	// Custom compare that says nothing is newer
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		CompareVersions: func(current, candidate Version) bool {
			return false
		},
	})

	rel, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	// Should return latest but not update
	assert.Equal(t, "2.0.0", rel.Version.Version)

}

func TestUpdateCommandSourceError(t *testing.T) {
	src := &mockSource{err: fmt.Errorf("network error")}

	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "myapp")
	os.WriteFile(cmdPath, []byte("old"), 0o755)

	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	_, err := up.UpdateCommand(context.Background(), cmdPath, "1.0.0", NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)

}

func TestPackageLevelDownloadReleaseAssetFromURL(t *testing.T) {
	// Just test the error path since we can't easily test success without a real server
	_, err := downloadReleaseAssetFromURL(context.Background(), "http://[invalid-url")
	assert.NotNil(t, err)

}

func TestUpdateSelfRefusesDirty(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64.tar.gz", size: 100, url: "https://example.com/app.tar.gz"}),
		},
		assets:	map[int64]string{1: "binary"},
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	_, err := up.UpdateSelf(context.Background(), "v0.0.1+dirty", NewRepositorySlug("test", "repo"))
	require.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "dirty"))

}

func TestDecompressAndInstallError(t *testing.T) {
	up, _ := NewUpdater(Config{
		Platform:	Platform{OS: "linux", Arch: "amd64"},
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
	assert.NotNil(t, err)

}
