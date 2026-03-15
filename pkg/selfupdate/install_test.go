package selfupdate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultInstall(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "myapp")
	os.WriteFile(target, []byte("old binary"), 0o755)

	install := defaultInstall("")
	err := install(strings.NewReader("new binary"), target)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new binary" {
		t.Errorf("expected new binary, got %q", data)
	}

	// old file should be cleaned up
	oldPath := filepath.Join(tmpDir, ".myapp.old")
	if _, err := os.Stat(oldPath); err == nil {
		t.Error("old file should have been removed")
	}
}

func TestDefaultInstallWithOldSavePath(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "myapp")
	oldSave := filepath.Join(tmpDir, "myapp.backup")
	os.WriteFile(target, []byte("old binary"), 0o755)

	install := defaultInstall(oldSave)
	err := install(strings.NewReader("new binary"), target)
	if err != nil {
		t.Fatal(err)
	}

	// new binary in place
	data, _ := os.ReadFile(target)
	if string(data) != "new binary" {
		t.Errorf("expected new binary, got %q", data)
	}

	// old binary saved
	data, err = os.ReadFile(oldSave)
	if err != nil {
		t.Fatal("old save file should exist")
	}
	if string(data) != "old binary" {
		t.Errorf("expected old binary in save path, got %q", data)
	}
}

func TestDefaultInstallTargetNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "nonexistent")

	install := defaultInstall("")
	err := install(strings.NewReader("new binary"), target)
	if err == nil {
		t.Error("expected error when target doesn't exist for rename")
	}
}

func TestDefaultInstallBadDirectory(t *testing.T) {
	install := defaultInstall("")
	err := install(strings.NewReader("data"), "/nonexistent/dir/myapp")
	if err == nil {
		t.Error("expected error for bad directory")
	}
}
