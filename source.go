package selfupdate

import (
	"context"
	"io"
	"time"
)

// Source interface to load the releases from.
type Source interface {
	ListReleases(ctx context.Context, repository Repository) ([]SourceRelease, error)
	DownloadReleaseAsset(ctx context.Context, rel *Release, assetID int64) (io.ReadCloser, error)
}

// SourceRelease represents a release from the source provider.
type SourceRelease interface {
	GetID() int64
	GetTagName() string
	GetDraft() bool
	GetPrerelease() bool
	GetPublishedAt() time.Time
	GetReleaseNotes() string
	GetName() string
	GetURL() string
	GetAssets() []SourceAsset
}

// SourceAsset represents a downloadable asset attached to a release.
type SourceAsset interface {
	GetID() int64
	GetName() string
	GetSize() int
	GetBrowserDownloadURL() string
}
