package selfupdate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestNewInstallCommandLatest(t *testing.T) {
	tarGz := makeTarGz(t, map[string][]byte{"repo": []byte("binary-v1")})
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", &mockAsset{id: 1, name: "repo_linux_amd64.tar.gz", size: 100, url: "https://example.com/a.tar.gz"}),
		},
		assets: map[int64]string{1: string(tarGz)},
	}

	var installed []byte
	cfg := Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			var err error
			installed, err = io.ReadAll(r)
			return err
		},
	}

	repo := NewRepositorySlug("test", "repo")
	cmd := NewInstallCommand(repo, WithConfig(cfg))

	dest := filepath.Join(t.TempDir(), "repo")
	cmd.SetArgs([]string{dest})
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	assert.Equal(t, "binary-v1", string(installed))
	assert.Contains(t, out.String(), "Installed 1.0.0")
}

func TestNewInstallCommandSpecificVersion(t *testing.T) {
	tarGz := makeTarGz(t, map[string][]byte{"repo": []byte("binary-v2")})
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", &mockAsset{id: 2, name: "repo_linux_amd64.tar.gz", size: 100, url: "https://example.com/a.tar.gz"}),
			newTestRelease("v1.0.0", &mockAsset{id: 1, name: "repo_linux_amd64.tar.gz", size: 100, url: "https://example.com/b.tar.gz"}),
		},
		assets: map[int64]string{2: string(tarGz)},
	}

	cfg := Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, _ string) error {
			_, err := io.ReadAll(r)
			return err
		},
	}

	repo := NewRepositorySlug("test", "repo")
	cmd := NewInstallCommand(repo, WithConfig(cfg))

	dest := filepath.Join(t.TempDir(), "repo")
	cmd.SetArgs([]string{"--version", "2.0.0", dest})
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	assert.Contains(t, out.String(), "2.0.0")
}

func TestNewInstallCommandVersionNotFound(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", &mockAsset{id: 1, name: "repo_linux_amd64.tar.gz", size: 100, url: "https://example.com/a.tar.gz"}),
		},
	}

	cfg := Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	}

	repo := NewRepositorySlug("test", "repo")
	cmd := NewInstallCommand(repo, WithConfig(cfg))
	cmd.SetArgs([]string{"--version", "9.9.9", "/tmp/repo"})
	cmd.SetContext(context.Background())

	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "9.9.9 not found")
}

func TestNewInstallCommandNoRelease(t *testing.T) {
	src := &mockSource{releases: []SourceRelease{}}

	cfg := Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	}

	repo := NewRepositorySlug("test", "repo")
	cmd := NewInstallCommand(repo, WithConfig(cfg))
	cmd.SetArgs([]string{"/tmp/repo"})
	cmd.SetContext(context.Background())

	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no release found")
}

func TestNewInstallCommandDefaultPath(t *testing.T) {
	tarGz := makeTarGz(t, map[string][]byte{"myapp": []byte("bin")})
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64.tar.gz", size: 100, url: "https://example.com/a.tar.gz"}),
		},
		assets: map[int64]string{1: string(tarGz)},
	}

	var installDst string
	cfg := Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			installDst = path
			_, _ = io.ReadAll(r)
			return nil
		},
	}

	repo := NewRepositorySlug("test", "myapp")
	cmd := NewInstallCommand(repo, WithConfig(cfg))
	cmd.SetArgs([]string{})
	cmd.SetContext(context.Background())
	cmd.SetOut(&bytes.Buffer{})

	err := cmd.Execute()
	require.Nil(t, err)
	assert.Equal(t, "/usr/local/bin/myapp", installDst)
}

func TestNewUpdateCommandWithVersion(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64", size: 100, url: "https://example.com/myapp"}),
		},
		assets: map[int64]string{1: "new-binary"},
	}

	var installed []byte
	cfg := Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
		Install: func(r io.Reader, path string) error {
			var err error
			installed, err = io.ReadAll(r)
			return err
		},
	}

	repo := NewRepositorySlug("test", "myapp")
	cmd := NewUpdateCommand(repo, "1.0.0", WithConfig(cfg))
	cmd.SetArgs([]string{"--version", "2.0.0"})
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	assert.Equal(t, "new-binary", string(installed))
	assert.Contains(t, out.String(), "Updated to 2.0.0")
}

func TestNewUpdateCommandVersionNotFound(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64.tar.gz", size: 100, url: "https://example.com/a.tar.gz"}),
		},
	}

	cfg := Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	}

	repo := NewRepositorySlug("test", "myapp")
	cmd := NewUpdateCommand(repo, "1.0.0", WithConfig(cfg))
	cmd.SetArgs([]string{"--version", "9.9.9"})
	cmd.SetContext(context.Background())

	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "9.9.9 not found")
}

func TestNewUpdateCommandSourceError(t *testing.T) {
	src := &mockSource{err: fmt.Errorf("network failure")}

	cfg := Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	}

	repo := NewRepositorySlug("test", "myapp")
	cmd := NewUpdateCommand(repo, "1.0.0", WithConfig(cfg))
	cmd.SetArgs([]string{"--version", "2.0.0"})
	cmd.SetContext(context.Background())

	err := cmd.Execute()
	assert.NotNil(t, err)
}

func TestInstallPathWithArgs(t *testing.T) {
	repo := NewRepositorySlug("test", "myapp")
	path, err := installPath(repo, []string{"/custom/path"})
	require.Nil(t, err)
	assert.Equal(t, "/custom/path", path)
}

func TestInstallPathDefault(t *testing.T) {
	repo := NewRepositorySlug("test", "myapp")
	path, err := installPath(repo, nil)
	require.Nil(t, err)
	assert.Equal(t, "/usr/local/bin/myapp", path)
}

func TestInstallPathInvalidSlug(t *testing.T) {
	repo := RepositorySlug{}
	_, err := installPath(repo, nil)
	assert.NotNil(t, err)
}

func TestApplyOptions(t *testing.T) {
	cfg := applyOptions(nil)
	assert.Nil(t, cfg.config)

	custom := Config{Platform: Platform{OS: "darwin", Arch: "arm64"}}
	cfg = applyOptions([]CommandOption{WithConfig(custom)})
	require.NotNil(t, cfg.config)
	assert.Equal(t, "darwin", cfg.config.Platform.OS)
}
