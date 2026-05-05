package selfupdate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

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
	cmd := newUpdateCommand(repo, WithConfig(cfg), WithVersion("1.0.0"))
	cmd.SetArgs([]string{"2.0.0"})
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
	cmd := newUpdateCommand(repo, WithConfig(cfg), WithVersion("1.0.0"))
	cmd.SetArgs([]string{"9.9.9"})
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
	cmd := newUpdateCommand(repo, WithConfig(cfg), WithVersion("1.0.0"))
	cmd.SetArgs([]string{"2.0.0"})
	cmd.SetContext(context.Background())

	err := cmd.Execute()
	assert.NotNil(t, err)
}

func TestApplyOptions(t *testing.T) {
	cfg := applyOptions(nil)
	assert.Nil(t, cfg.config)
	// currentVersion always resolves to something non-empty (CurrentVersion fallback).
	assert.NotEqual(t, "", cfg.currentVersion)

	custom := Config{Platform: Platform{OS: "darwin", Arch: "arm64"}}
	cfg = applyOptions([]CommandOption{WithConfig(custom), WithVersion("3.2.1")})
	require.NotNil(t, cfg.config)
	assert.Equal(t, "darwin", cfg.config.Platform.OS)
	assert.Equal(t, "3.2.1", cfg.currentVersion)
}

func TestNewVersionCommandBare(t *testing.T) {
	repo := NewRepositorySlug("test", "myapp")
	cmd := newVersionCommand(repo, WithVersion("1.0.0"))
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
	cmd := newVersionCommand(repo, WithConfig(cfg), WithVersion("1.0.0"))
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
	cmd := newVersionCommand(repo, WithConfig(cfg), WithVersion("1.0.0"))
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
	cmd := newVersionCommand(repo, WithConfig(cfg), WithVersion("1.0.0"))
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	assert.Equal(t, "version: 1.0.0\n", out.String())
}

func TestNewVersionCommandAutoDetect(t *testing.T) {
	repo := NewRepositorySlug("test", "myapp")
	cmd := newVersionCommand(repo)
	cmd.SetArgs([]string{"--bare"})
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	assert.NotEqual(t, "\n", out.String())
	assert.NotEqual(t, "", out.String())
}

func TestNewVersionCommandUsesPackageVersion(t *testing.T) {
	prev := EmbeddedVersion
	EmbeddedVersion = "9.9.9-test"
	t.Cleanup(func() { EmbeddedVersion = prev })

	repo := NewRepositorySlug("test", "myapp")
	cmd := newVersionCommand(repo)
	cmd.SetArgs([]string{"--bare"})
	cmd.SetContext(context.Background())

	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	require.Nil(t, err)
	assert.Equal(t, "9.9.9-test\n", out.String())
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
	RegisterCommands(rootCmd, repo, WithConfig(cfg), WithVersion("1.0.0"))

	assert.Equal(t, "1.0.0", rootCmd.Version)

	cmd, _, err := rootCmd.Find([]string{"version"})
	require.Nil(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "version", cmd.Name())

	cmd, _, err = rootCmd.Find([]string{"update"})
	require.Nil(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "update", cmd.Name())

	_, _, err = rootCmd.Find([]string{"install"})
	assert.NotNil(t, err)
}

func TestRegisterCommandsAutoDetectVersion(t *testing.T) {
	prev := EmbeddedVersion
	EmbeddedVersion = "7.7.7"
	t.Cleanup(func() { EmbeddedVersion = prev })

	rootCmd := &cobra.Command{Use: "myapp"}
	repo := NewRepositorySlug("test", "myapp")
	RegisterCommands(rootCmd, repo)

	assert.Equal(t, "7.7.7", rootCmd.Version)
}
