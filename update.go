package selfupdate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UpdateTo downloads an executable from the source provider and replaces the
// current binary with the downloaded one.
func (up *Updater) UpdateTo(ctx context.Context, rel *Release, cmdPath string) error {
	if rel == nil {
		return ErrInvalidRelease
	}

	data, err := up.download(ctx, rel, rel.Asset.ID)
	if err != nil {
		return fmt.Errorf("failed to read asset %q: %w", rel.Asset.Name, err)
	}

	if up.validate != nil {
		if err = up.validate(rel, data); err != nil {
			return fmt.Errorf("validation failed for %q: %w", rel.Asset.Name, err)
		}
	}

	return up.decompressAndInstall(bytes.NewReader(data), rel.Asset.Name, rel.Asset.URL, cmdPath)
}

// UpdateCommand updates a given command binary to the latest version.
// 'current' is used to check the latest version against the current version.
// When current is empty, it is resolved via [CurrentVersion] -- see the
// package-level [EmbeddedVersion] variable for ldflags-based version embedding.
func (up *Updater) UpdateCommand(ctx context.Context, cmdPath string, current string, repository Repository) (*Release, error) {
	if current == "" {
		current = CurrentVersion()
	}
	currentVer, err := parseCurrentVersion(current)
	if err != nil {
		return nil, err
	}

	if up.platform.OS == "windows" && !strings.HasSuffix(cmdPath, ".exe") {
		cmdPath = cmdPath + ".exe"
	}

	stat, err := os.Lstat(cmdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat '%s'. file may not exist: %s", cmdPath, err)
	}
	if stat.Mode()&os.ModeSymlink != 0 {
		p, err := filepath.EvalSymlinks(cmdPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve symlink '%s' for executable: %s", cmdPath, err)
		}
		cmdPath = p
	}

	rel, ok, err := up.DetectLatest(ctx, repository)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.Print("No release detected. Current version is considered up-to-date")
		return &Release{Version: currentVer}, nil
	}

	compare := up.compareVersions
	if compare == nil {
		compare = defaultCompareVersions
	}

	if !compare(currentVer, rel.Version) {
		log.Printf("Current version %s is the latest. Update is not needed", currentVer.Version)
		return rel, nil
	}

	log.Printf("Will update %s to the latest version %s", cmdPath, rel.Version.Version)
	if err := up.UpdateTo(ctx, rel, cmdPath); err != nil {
		return nil, err
	}
	return rel, nil
}

// UpdateSelf updates the running executable itself to the latest version.
// When current is empty, it is resolved via [CurrentVersion].
//
// UpdateSelf refuses to overwrite a binary whose version carries the "+dirty"
// suffix that [versionFromBuildInfo] appends for builds made from a modified
// working tree: there is no released artifact corresponding to that commit, so
// any "update" would destroy the local changes the running binary was built
// from. Callers that genuinely want to clobber a dirty build can strip the
// suffix themselves before invoking UpdateSelf, or pass an explicit semver via
// the current argument.
func (up *Updater) UpdateSelf(ctx context.Context, current string, repository Repository) (*Release, error) {
	if current == "" {
		current = CurrentVersion()
	}
	if strings.Contains(current, "+dirty") {
		return nil, fmt.Errorf("refusing to self-update a dirty build (%s)", current)
	}
	cmdPath, err := getExecutablePath()
	if err != nil {
		return nil, err
	}
	return up.UpdateCommand(ctx, cmdPath, current, repository)
}

func (up *Updater) decompressAndInstall(src io.Reader, assetName, assetURL, cmdPath string) error {
	_, cmd := filepath.Split(cmdPath)
	asset, err := decompressCommand(src, assetName, cmd, up.decompressors)
	if err != nil {
		return err
	}

	log.Printf("Will update %s to the latest downloaded from %s", cmdPath, assetURL)
	return up.install(asset, cmdPath)
}

func (up *Updater) download(ctx context.Context, rel *Release, assetID int64) (data []byte, err error) {
	var reader io.ReadCloser
	if reader, err = up.source.DownloadReleaseAsset(ctx, rel, assetID); err == nil {
		defer func() { _ = reader.Close() }()
		data, err = io.ReadAll(reader)
	}
	return
}
