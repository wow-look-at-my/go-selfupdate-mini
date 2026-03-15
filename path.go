package selfupdate

import (
	"os"
	"path/filepath"
)

// getExecutablePath returns the path of the current executable with symlinks resolved.
func getExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}
