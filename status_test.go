package selfupdate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestDaysOutOfDate(t *testing.T) {
	assert.Equal(t, 0, DaysOutOfDate(time.Time{}))
	assert.Equal(t, 0, DaysOutOfDate(time.Now().Add(24*time.Hour)))
	assert.Equal(t, 0, DaysOutOfDate(time.Now()))
	assert.Equal(t, 3, DaysOutOfDate(time.Now().Add(-3*24*time.Hour-time.Hour)))
	assert.Equal(t, 10, DaysOutOfDate(time.Now().Add(-10*24*time.Hour)))
}

func TestUpdateStatusDaysOutOfDate(t *testing.T) {
	// Not available => 0
	s := &UpdateStatus{UpdateAvailable: false}
	assert.Equal(t, 0, s.DaysOutOfDate())

	// Available but nil release => 0
	s = &UpdateStatus{UpdateAvailable: true, LatestRelease: nil}
	assert.Equal(t, 0, s.DaysOutOfDate())

	// Available with publish time
	s = &UpdateStatus{
		UpdateAvailable: true,
		LatestRelease: &Release{
			PublishedAt: time.Now().Add(-5 * 24 * time.Hour),
		},
	}
	assert.Equal(t, 5, s.DaysOutOfDate())
}

func TestCheckUpdateUpToDate(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("myapp_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	status, err := up.CheckUpdate(context.Background(), "1.0.0", NewRepositorySlug("test", "repo"))
	require.Nil(t, err)
	assert.False(t, status.UpdateAvailable)
	assert.Equal(t, "1.0.0", status.CurrentVersion.Version)
	assert.NotNil(t, status.LatestRelease)
}

func TestCheckUpdateAvailable(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v2.0.0", newTestAsset("myapp_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	status, err := up.CheckUpdate(context.Background(), "1.0.0", NewRepositorySlug("test", "repo"))
	require.Nil(t, err)
	assert.True(t, status.UpdateAvailable)
	assert.Equal(t, "2.0.0", status.LatestRelease.Version.Version)
}

func TestCheckUpdateNoRelease(t *testing.T) {
	src := &mockSource{releases: []SourceRelease{}}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	status, err := up.CheckUpdate(context.Background(), "1.0.0", NewRepositorySlug("test", "repo"))
	require.Nil(t, err)
	assert.False(t, status.UpdateAvailable)
	assert.Nil(t, status.LatestRelease)
}

func TestCheckUpdateBadVersion(t *testing.T) {
	up, _ := NewUpdater(Config{Platform: Platform{OS: "linux", Arch: "amd64"}})
	_, err := up.CheckUpdate(context.Background(), "bad", NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)
}

func TestCheckUpdateSourceError(t *testing.T) {
	src := &mockSource{err: fmt.Errorf("network error")}
	up, _ := NewUpdater(Config{
		Source:   src,
		Platform: Platform{OS: "linux", Arch: "amd64"},
	})

	_, err := up.CheckUpdate(context.Background(), "1.0.0", NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)
}

func TestPackageLevelCheckUpdate(t *testing.T) {
	// Just verify it delegates to DefaultUpdater (will fail on source but that's OK)
	defaultUpdater = nil
	defer func() { defaultUpdater = nil }()

	_, err := CheckUpdate(context.Background(), "bad-version", NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)
}
