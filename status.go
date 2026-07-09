package selfupdate

import (
	"context"
	"math"
	"time"
)

// UpdateStatus holds the result of checking for updates.
type UpdateStatus struct {
	// CurrentVersion is the parsed version of the running binary.
	CurrentVersion Version
	// LatestRelease is the latest release found, or nil if none was found.
	LatestRelease *Release
	// UpdateAvailable is true when the latest release is newer than current.
	UpdateAvailable bool
}

// DaysOutOfDate returns the number of whole days between the latest release's
// publish date and now. Returns 0 when no update is available or the publish
// time is unknown.
func (s *UpdateStatus) DaysOutOfDate() int {
	if !s.UpdateAvailable || s.LatestRelease == nil {
		return 0
	}
	return DaysOutOfDate(s.LatestRelease.PublishedAt)
}

// DaysOutOfDate returns the number of whole days between the given time and now.
// Returns 0 if t is zero or in the future.
func DaysOutOfDate(t time.Time) int {
	if t.IsZero() {
		return 0
	}
	d := time.Since(t)
	if d < 0 {
		return 0
	}
	return int(math.Floor(d.Hours() / 24))
}

// CheckUpdate queries the source for the latest release and reports whether an
// update is available for the given current version.
func (up *Updater) CheckUpdate(ctx context.Context, current string, repository Repository) (*UpdateStatus, error) {
	currentVer, err := parseCurrentVersion(current)
	if err != nil {
		return nil, err
	}

	rel, ok, err := up.DetectLatest(ctx, repository)
	if err != nil {
		return nil, err
	}

	status := &UpdateStatus{
		CurrentVersion: currentVer,
	}

	if !ok {
		return status, nil
	}

	status.LatestRelease = rel

	compare := up.compareVersions
	if compare == nil {
		compare = defaultCompareVersions
	}
	status.UpdateAvailable = compare(currentVer, rel.Version)

	return status, nil
}

// CheckUpdate queries the default updater for the latest release and reports
// whether an update is available for the given current version.
func CheckUpdate(ctx context.Context, current string, repository Repository) (*UpdateStatus, error) {
	return DefaultUpdater().CheckUpdate(ctx, current, repository)
}
