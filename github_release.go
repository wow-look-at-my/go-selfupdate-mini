package selfupdate

import "time"

// githubRelease maps directly to GitHub's JSON API response for a release.
type githubRelease struct {
	ID          int64         `json:"id"`
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	Assets      []githubAsset `json:"assets"`
}

func (r *githubRelease) GetID() int64              { return r.ID }
func (r *githubRelease) GetTagName() string         { return r.TagName }
func (r *githubRelease) GetDraft() bool             { return r.Draft }
func (r *githubRelease) GetPrerelease() bool        { return r.Prerelease }
func (r *githubRelease) GetPublishedAt() time.Time  { return r.PublishedAt }
func (r *githubRelease) GetReleaseNotes() string    { return r.Body }
func (r *githubRelease) GetName() string            { return r.Name }
func (r *githubRelease) GetURL() string             { return r.HTMLURL }
func (r *githubRelease) GetAssets() []SourceAsset {
	assets := make([]SourceAsset, len(r.Assets))
	for i := range r.Assets {
		assets[i] = &r.Assets[i]
	}
	return assets
}

// githubAsset maps directly to GitHub's JSON API response for a release asset.
type githubAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int    `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (a *githubAsset) GetID() int64                  { return a.ID }
func (a *githubAsset) GetName() string               { return a.Name }
func (a *githubAsset) GetSize() int                  { return a.Size }
func (a *githubAsset) GetBrowserDownloadURL() string { return a.BrowserDownloadURL }

var (
	_ SourceRelease = &githubRelease{}
	_ SourceAsset   = &githubAsset{}
)
