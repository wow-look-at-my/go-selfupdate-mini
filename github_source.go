package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// GitHubConfig is the configuration for NewGitHubSource.
type GitHubConfig struct {
	// APIToken is a GitHub API token. If empty, $GITHUB_TOKEN is used.
	APIToken string
	// EnterpriseBaseURL is the base URL for GitHub Enterprise API.
	// Default: "https://api.github.com"
	EnterpriseBaseURL string
}

// GitHubSource loads release information from GitHub using the REST API.
type GitHubSource struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewGitHubSource creates a new GitHubSource from a config.
// Pass an empty GitHubConfig{} for default configuration.
func NewGitHubSource(config GitHubConfig) (*GitHubSource, error) {
	token := config.APIToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	baseURL := strings.TrimSuffix(config.EnterpriseBaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	return &GitHubSource{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}, nil
}

func (s *GitHubSource) do(ctx context.Context, url, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	return s.httpClient.Do(req)
}

// ListReleases returns all available releases.
func (s *GitHubSource) ListReleases(ctx context.Context, repository Repository) ([]SourceRelease, error) {
	owner, repo, err := repository.GetSlug()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases", s.baseURL, owner, repo)
	resp, err := s.do(ctx, url, "application/vnd.github.v3+json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Print("Repository or release not found")
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d for %s", resp.StatusCode, url)
	}

	var rels []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub releases: %w", err)
	}

	releases := make([]SourceRelease, len(rels))
	for i := range rels {
		releases[i] = &rels[i]
	}
	return releases, nil
}

// DownloadReleaseAsset downloads an asset from a release.
// It returns an io.ReadCloser: it is your responsibility to Close it.
func (s *GitHubSource) DownloadReleaseAsset(ctx context.Context, rel *Release, assetID int64) (io.ReadCloser, error) {
	if rel == nil {
		return nil, ErrInvalidRelease
	}
	owner, repo, err := rel.repository.GetSlug()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases/assets/%d", s.baseURL, owner, repo, assetID)
	resp, err := s.do(ctx, url, "application/octet-stream")
	if err != nil {
		return nil, fmt.Errorf("failed to download asset %d from %s/%s: %w", assetID, owner, repo, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download asset %d from %s/%s: HTTP %d", assetID, owner, repo, resp.StatusCode)
	}
	return resp.Body, nil
}

var _ Source = &GitHubSource{}
