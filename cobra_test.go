package selfupdate

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestNewVersionCommandUpToDate(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("myapp_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	root := &cobra.Command{Use: "myapp"}
	cmd := NewVersionCommand(&CobraConfig{
		Version:    "1.0.0",
		Repository: NewRepositorySlug("test", "repo"),
		Updater:    up,
	})
	root.AddCommand(cmd)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	err := root.Execute()
	require.Nil(t, err)

	out := buf.String()
	assert.Contains(t, out, "myapp version 1.0.0")
	assert.Contains(t, out, "Up to date")
}

func TestNewVersionCommandUpdateAvailable(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", newTestAsset("myapp_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	root := &cobra.Command{Use: "myapp"}
	cmd := NewVersionCommand(&CobraConfig{
		Version:    "1.0.0",
		Repository: NewRepositorySlug("test", "repo"),
		Updater:    up,
	})
	root.AddCommand(cmd)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	err := root.Execute()
	require.Nil(t, err)

	out := buf.String()
	assert.Contains(t, out, "myapp version 1.0.0")
	assert.Contains(t, out, "Update available: 2.0.0")
	assert.Contains(t, out, "Run 'myapp update' to update")
}

func TestNewUpdateCommandAlreadyUpToDate(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("myapp_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	root := &cobra.Command{Use: "myapp"}
	cmd := NewUpdateCommand(&CobraConfig{
		Version:    "1.0.0",
		Repository: NewRepositorySlug("test", "repo"),
		Updater:    up,
	})
	root.AddCommand(cmd)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"update"})
	err := root.Execute()
	require.Nil(t, err)

	assert.Contains(t, buf.String(), "Already up to date")
}

func TestAddVersionFlag(t *testing.T) {
	root := &cobra.Command{Use: "myapp"}
	AddVersionFlag(root, "1.2.3")
	assert.Equal(t, "1.2.3", root.Version)
}

func TestCobraConfigDefaultUpdater(t *testing.T) {
	defaultUpdater = nil
	defer func() { defaultUpdater = nil }()

	cfg := &CobraConfig{Version: "1.0.0"}
	up := cfg.updater()
	assert.NotNil(t, up)
}
