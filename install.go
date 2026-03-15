package selfupdate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// defaultInstall performs an atomic binary replacement:
// 1. Write new binary to <target>.new
// 2. Rename current to <target>.old (or OldSavePath)
// 3. Rename .new to target
// 4. Remove .old (unless OldSavePath is set)
// 5. Rollback on failure
func defaultInstall(oldSavePath string) func(io.Reader, string) error {
	return func(src io.Reader, targetPath string) error {
		newBytes, err := io.ReadAll(src)
		if err != nil {
			return err
		}

		dir := filepath.Dir(targetPath)
		filename := filepath.Base(targetPath)

		newPath := filepath.Join(dir, fmt.Sprintf(".%s.new", filename))
		fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}

		if _, err = fp.Write(newBytes); err != nil {
			fp.Close()
			return err
		}
		fp.Close()

		removeOld := oldSavePath == ""
		oldPath := oldSavePath
		if removeOld {
			oldPath = filepath.Join(dir, fmt.Sprintf(".%s.old", filename))
		}

		// remove any stale .old file (necessary on Windows)
		_ = os.Remove(oldPath)

		if err = os.Rename(targetPath, oldPath); err != nil {
			return err
		}

		if err = os.Rename(newPath, targetPath); err != nil {
			// rollback
			if rerr := os.Rename(oldPath, targetPath); rerr != nil {
				return fmt.Errorf("update failed (%w) and rollback also failed (%v)", err, rerr)
			}
			return err
		}

		if removeOld {
			_ = os.Remove(oldPath)
		}

		return nil
	}
}
