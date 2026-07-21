package selfupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// DetectLatest detects the latest release from the repository.
// Shortcut for DefaultUpdater().DetectLatest.
func DetectLatest(ctx context.Context, repository Repository) (*Release, bool, error) {
	return DefaultUpdater().DetectLatest(ctx, repository)
}

// DetectVersion detects the given release from the repository.
func DetectVersion(ctx context.Context, repository Repository, version string) (*Release, bool, error) {
	return DefaultUpdater().DetectVersion(ctx, repository, version)
}

// UpdateTo downloads an executable from assetURL and replaces the current binary.
// This is a low-level API that downloads directly via HTTP (not via the source provider),
// so it is not available for private repositories.
func UpdateTo(ctx context.Context, assetURL, assetFileName, cmdPath string) error {
	up := DefaultUpdater()
	src, err := downloadReleaseAssetFromURL(ctx, assetURL)
	if err != nil {
		return err
	}
	defer src.Close()
	return up.decompressAndInstall(src, assetFileName, assetURL, cmdPath)
}

// UpdateCommand updates a given command binary to the latest version.
// Pass an empty string for current to auto-detect via [CurrentVersion].
func UpdateCommand(ctx context.Context, cmdPath string, current string, repository Repository) (*Release, error) {
	return DefaultUpdater().UpdateCommand(ctx, cmdPath, current, repository)
}

// UpdateSelf updates the running executable itself to the latest version.
// Pass an empty string for current to auto-detect via [CurrentVersion].
func UpdateSelf(ctx context.Context, current string, repository Repository) (*Release, error) {
	return DefaultUpdater().UpdateSelf(ctx, current, repository)
}

func downloadReleaseAssetFromURL(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download a release file from %s: %w", url, err)
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download a release file from %s: HTTP %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}
