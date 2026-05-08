package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// NpmConfig configures an [NpmSource].
type NpmConfig struct {
	// Registry is the npm registry base URL, including a trailing slash.
	// Example: "https://git.pazer.us/api/packages/wow-look-at-my/npm/"
	Registry string
	// Token is an optional Bearer token for private registries.
	Token string
}

// NpmRepository identifies an npm package for self-update.
// Scope is the npm scope (e.g. "@wow-look-at-my") and Name is the
// unscoped package name (e.g. "go-toolchain").
type NpmRepository struct {
	Scope string
	Name  string
}

// GetSlug returns (scope, name).
func (r NpmRepository) GetSlug() (string, string, error) {
	return r.Scope, r.Name, nil
}

var _ Repository = NpmRepository{}

// NpmRepositoryFromBuildInfo derives the npm scope and package name from the
// running binary's Go module path via [runtime/debug.ReadBuildInfo].
//
// Given a module path like "github.com/wow-look-at-my/go-toolchain" it
// returns NpmRepository{Scope: "@wow-look-at-my", Name: "go-toolchain"}.
// The module host (e.g. "github.com") is ignored; the second path component
// becomes the npm scope and the third becomes the package name.
//
// This lets a binary self-identify its npm package without any hard-coded
// slug: just pass the registry base URL to [NewNpmSource] and pass the
// returned repository to [Updater.DetectLatest].
func NpmRepositoryFromBuildInfo() (NpmRepository, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return NpmRepository{}, fmt.Errorf("runtime/debug build info not available; set EmbeddedVersion or build with module support")
	}
	modPath := info.Main.Path
	if modPath == "" || modPath == "command-line-arguments" {
		return NpmRepository{}, fmt.Errorf("module path %q is not usable for npm package auto-detection", modPath)
	}
	// Module paths are at least "host/org/name"; major-version suffixes (e.g.
	// "/v2") are extra components after [2] and are intentionally ignored.
	parts := strings.SplitN(modPath, "/", 4)
	if len(parts) < 3 {
		return NpmRepository{}, fmt.Errorf("module path %q has fewer than 3 slash-separated components", modPath)
	}
	return NpmRepository{
		Scope: "@" + parts[1],
		Name:  parts[2],
	}, nil
}

// NpmSource implements [Source] using an npm registry that publishes
// per-platform binary packages following the go-toolchain npm convention:
//
//	@scope/name              (wrapper, optionalDependencies on platform pkgs)
//	@scope/name-OS-ARCH      (one per platform, bin/ contains the binary)
//
// Each version appears as a synthetic [SourceRelease] whose single asset
// is the platform package tarball. The existing tar.gz decompressor in
// [decompressCommand] extracts the binary by matching its base name.
type NpmSource struct {
	registry   string
	token      string
	httpClient *http.Client

	mu        sync.Mutex
	nextID    int64
	assetURLs map[int64]string
}

// NewNpmSource creates a new NpmSource.
func NewNpmSource(config NpmConfig) (*NpmSource, error) {
	reg := config.Registry
	if !strings.HasSuffix(reg, "/") {
		reg += "/"
	}
	return &NpmSource{
		registry:   reg,
		token:      config.Token,
		httpClient: &http.Client{},
		assetURLs:  make(map[int64]string),
	}, nil
}

var goToNpmOS = map[string]string{
	"linux":   "linux",
	"darwin":  "darwin",
	"windows": "win32",
	"freebsd": "freebsd",
	"openbsd": "openbsd",
}

var goToNpmArch = map[string]string{
	"amd64": "x64",
	"arm64": "arm64",
	"386":   "ia32",
	"arm":   "arm",
}

type npmPackageMeta struct {
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]npmVersionMeta  `json:"versions"`
	Time     map[string]string          `json:"time"`
}

type npmVersionMeta struct {
	Version string      `json:"version"`
	Dist    npmDistMeta `json:"dist"`
}

type npmDistMeta struct {
	Tarball string `json:"tarball"`
}

func (s *NpmSource) fetchMeta(ctx context.Context, pkgName string) (*npmPackageMeta, error) {
	url := s.registry + strings.ReplaceAll(pkgName, "/", "%2F")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned HTTP %d for %s", resp.StatusCode, pkgName)
	}
	var meta npmPackageMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode npm metadata for %s: %w", pkgName, err)
	}
	return &meta, nil
}

func (s *NpmSource) storeAssetURL(url string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.assetURLs[s.nextID] = url
	return s.nextID
}

// ListReleases queries the npm registry for the platform-specific package
// and returns one synthetic release per version.
func (s *NpmSource) ListReleases(ctx context.Context, repository Repository) ([]SourceRelease, error) {
	scope, name, err := repository.GetSlug()
	if err != nil {
		return nil, err
	}

	npmOS, ok := goToNpmOS[runtime.GOOS]
	if !ok {
		return nil, fmt.Errorf("npm source: unsupported OS %q", runtime.GOOS)
	}
	npmArch, ok := goToNpmArch[runtime.GOARCH]
	if !ok {
		return nil, fmt.Errorf("npm source: unsupported arch %q", runtime.GOARCH)
	}

	platformPkg := scope + "/" + name + "-" + npmOS + "-" + npmArch
	meta, err := s.fetchMeta(ctx, platformPkg)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, nil
	}

	var releases []SourceRelease
	for ver, info := range meta.Versions {
		tarball := info.Dist.Tarball
		if tarball == "" {
			continue
		}

		assetID := s.storeAssetURL(tarball)
		assetName := fmt.Sprintf("%s_%s_%s.tgz", name, runtime.GOOS, runtime.GOARCH)

		var publishedAt time.Time
		if ts, ok := meta.Time[ver]; ok {
			publishedAt, _ = time.Parse(time.RFC3339, ts)
		}

		releases = append(releases, &npmRelease{
			tagName:     "v" + ver,
			publishedAt: publishedAt,
			assets: []*npmReleaseAsset{{
				id:   assetID,
				name: assetName,
				url:  tarball,
			}},
		})
	}
	return releases, nil
}

// DownloadReleaseAsset downloads the npm package tarball.
// The caller's decompressor pipeline extracts the binary from the .tgz.
func (s *NpmSource) DownloadReleaseAsset(ctx context.Context, _ *Release, assetID int64) (io.ReadCloser, error) {
	s.mu.Lock()
	url, ok := s.assetURLs[assetID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("npm source: unknown asset ID %d", assetID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("npm source: tarball download returned HTTP %d for %s", resp.StatusCode, url)
	}
	return resp.Body, nil
}

var _ Source = (*NpmSource)(nil)

// --- SourceRelease / SourceAsset implementations ---

type npmRelease struct {
	tagName     string
	publishedAt time.Time
	assets      []*npmReleaseAsset
}

func (r *npmRelease) GetID() int64             { return 0 }
func (r *npmRelease) GetTagName() string       { return r.tagName }
func (r *npmRelease) GetDraft() bool           { return false }
func (r *npmRelease) GetPrerelease() bool      { return false }
func (r *npmRelease) GetPublishedAt() time.Time { return r.publishedAt }
func (r *npmRelease) GetReleaseNotes() string  { return "" }
func (r *npmRelease) GetName() string          { return r.tagName }
func (r *npmRelease) GetURL() string           { return "" }
func (r *npmRelease) GetAssets() []SourceAsset {
	out := make([]SourceAsset, len(r.assets))
	for i, a := range r.assets {
		out[i] = a
	}
	return out
}

type npmReleaseAsset struct {
	id   int64
	name string
	url  string
}

func (a *npmReleaseAsset) GetID() int64                  { return a.id }
func (a *npmReleaseAsset) GetName() string               { return a.name }
func (a *npmReleaseAsset) GetSize() int                  { return 0 }
func (a *npmReleaseAsset) GetBrowserDownloadURL() string { return a.url }
