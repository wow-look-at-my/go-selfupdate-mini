package selfupdate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestDefaultInstall(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "myapp")
	os.WriteFile(target, []byte("old binary"), 0o755)

	install := defaultInstall("")
	err := install(strings.NewReader("new binary"), target)
	require.Nil(t, err)

	data, err := os.ReadFile(target)
	require.Nil(t, err)

	assert.Equal(t, "new binary", string(data))

	// old file should be cleaned up
	oldPath := filepath.Join(tmpDir, ".myapp.old")
	_, err = os.Stat(oldPath)
	assert.NotNil(t, err)

}

func TestDefaultInstallWithOldSavePath(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "myapp")
	oldSave := filepath.Join(tmpDir, "myapp.backup")
	os.WriteFile(target, []byte("old binary"), 0o755)

	install := defaultInstall(oldSave)
	err := install(strings.NewReader("new binary"), target)
	require.Nil(t, err)

	// new binary in place
	data, _ := os.ReadFile(target)
	assert.Equal(t, "new binary", string(data))

	// old binary saved
	data, err = os.ReadFile(oldSave)
	require.Nil(t, err)

	assert.Equal(t, "old binary", string(data))

}

func TestDefaultInstallTargetNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "nonexistent")

	install := defaultInstall("")
	err := install(strings.NewReader("new binary"), target)
	assert.NotNil(t, err)

}

func TestDefaultInstallBadDirectory(t *testing.T) {
	install := defaultInstall("")
	err := install(strings.NewReader("data"), "/nonexistent/dir/myapp")
	assert.NotNil(t, err)

}
