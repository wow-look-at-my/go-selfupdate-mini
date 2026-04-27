package selfupdate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
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

	home, err := os.UserHomeDir()
	require.Nil(t, err)

	err = cmd.Execute()
	require.Nil(t, err)
	assert.Equal(t, filepath.Join(home, ".local", "bin", "myapp"), installDst)
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
	home, herr := os.UserHomeDir()
	require.Nil(t, herr)
	assert.Equal(t, filepath.Join(home, ".local", "bin", "myapp"), path)
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

func TestNewVersionCommandBare(t *testing.T) {
	repo := NewRepositorySlug("test", "myapp")
	cmd := NewVersionCommand("1.0.0", repo)
	cmd.SetArgs([]string{"--bare"})
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	assert.Equal(t, "1.0.0\n", out.String())
}

func TestNewVersionCommandUpToDate(t *testing.T) {
	rel := newTestRelease("v1.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64", size: 100, url: "https://example.com/myapp"})
	rel.publishedAt = time.Now().Add(-21 * 24 * time.Hour)

	src := &mockSource{releases: []SourceRelease{rel}}
	cfg := Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}}

	repo := NewRepositorySlug("test", "myapp")
	cmd := NewVersionCommand("1.0.0", repo, WithConfig(cfg))
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	output := out.String()
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "latest")
	assert.Contains(t, output, "3 weeks ago")
}

func TestNewVersionCommandOutdated(t *testing.T) {
	rel := newTestRelease("v2.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64", size: 100, url: "https://example.com/myapp"})
	rel.publishedAt = time.Now().Add(-14 * 24 * time.Hour)

	src := &mockSource{releases: []SourceRelease{rel}}
	cfg := Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}}

	repo := NewRepositorySlug("test", "myapp")
	cmd := NewVersionCommand("1.0.0", repo, WithConfig(cfg))
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	output := out.String()
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "2.0.0")
	assert.Contains(t, output, "2 weeks ago")
}

func TestNewVersionCommandNetworkError(t *testing.T) {
	src := &mockSource{err: fmt.Errorf("network failure")}
	cfg := Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}}

	repo := NewRepositorySlug("test", "myapp")
	cmd := NewVersionCommand("1.0.0", repo, WithConfig(cfg))
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	assert.Equal(t, "version: 1.0.0\n", out.String())
}

func TestHumanizeAge(t *testing.T) {
	tests := []struct {
		days     int
		expected string
	}{
		{0, "today"},
		{1, "1 day ago"},
		{5, "5 days ago"},
		{13, "13 days ago"},
		{14, "2 weeks ago"},
		{21, "3 weeks ago"},
		{29, "4 weeks ago"},
		{30, "1 month ago"},
		{60, "2 months ago"},
		{90, "3 months ago"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			d := time.Duration(tt.days) * 24 * time.Hour
			assert.Equal(t, tt.expected, humanizeAge(d))
		})
	}
}

func TestRegisterCommands(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", &mockAsset{id: 1, name: "myapp_linux_amd64", size: 100, url: "https://example.com/myapp"}),
		},
	}
	cfg := Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}}

	rootCmd := &cobra.Command{Use: "myapp"}
	repo := NewRepositorySlug("test", "myapp")
	RegisterCommands(rootCmd, "1.0.0", repo, WithConfig(cfg))

	// Check that Version is set
	assert.Equal(t, "1.0.0", rootCmd.Version)

	// Check that commands are registered
	cmd, _, err := rootCmd.Find([]string{"version"})
	require.Nil(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "version", cmd.Name())

	cmd, _, err = rootCmd.Find([]string{"update"})
	require.Nil(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "update", cmd.Name())

	cmd, _, err = rootCmd.Find([]string{"install"})
	require.Nil(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "install", cmd.Name())
}
