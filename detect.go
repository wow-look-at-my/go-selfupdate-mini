package selfupdate

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var reVersion = regexp.MustCompile(`\d+\.\d+\.\d+`)

// parseVersion extracts a Version from a release tag name.
func parseVersion(tagName string) (Version, bool) {
	verText := tagName
	indices := reVersion.FindStringIndex(verText)
	if indices == nil {
		return Version{}, false
	}
	if indices[0] > 0 {
		verText = verText[indices[0]:]
	}

	sv, err := semver.NewVersion(verText)
	if err != nil {
		return Version{}, false
	}

	return Version{
		Original:     tagName,
		Version:      sv.String(),
		Major:        int(sv.Major()),
		Minor:        int(sv.Minor()),
		Patch:        int(sv.Patch()),
		Prerelease:   sv.Prerelease(),
		IsPrerelease: sv.Prerelease() != "",
	}, true
}

// defaultCompareVersions uses semver to determine if candidate is newer than current.
func defaultCompareVersions(current, candidate Version) bool {
	cur, err := semver.NewVersion(current.Version)
	if err != nil {
		return false
	}
	cand, err := semver.NewVersion(candidate.Version)
	if err != nil {
		return false
	}
	return cand.GreaterThan(cur)
}

// parseCurrentVersion parses a version string (like "1.2.3") into a Version.
// Non-semver strings (e.g. "dev") are treated as 0.0.0 so any released
// version will compare as newer and the update will proceed.
func parseCurrentVersion(current string) (Version, error) {
	sv, err := semver.NewVersion(current)
	if err != nil {
		return Version{Original: current, Version: "0.0.0"}, nil
	}
	return Version{
		Original:     current,
		Version:      sv.String(),
		Major:        int(sv.Major()),
		Minor:        int(sv.Minor()),
		Patch:        int(sv.Patch()),
		Prerelease:   sv.Prerelease(),
		IsPrerelease: sv.Prerelease() != "",
	}, nil
}

// DetectLatest tries to get the latest version from the source provider.
func (up *Updater) DetectLatest(ctx context.Context, repository Repository) (release *Release, found bool, err error) {
	return up.DetectVersion(ctx, repository, "")
}

// DetectVersion tries to get the given version from the source provider.
func (up *Updater) DetectVersion(ctx context.Context, repository Repository, version string) (release *Release, found bool, err error) {
	rels, err := up.source.ListReleases(ctx, repository)
	if err != nil {
		return nil, false, err
	}

	rel, asset, ver, found := up.findReleaseAndAsset(rels, version)
	if !found {
		return nil, false, nil
	}

	log.Printf("Successfully fetched release %s, name: %s, URL: %s, asset: %s",
		rel.GetTagName(), rel.GetName(), rel.GetURL(),
		asset.GetBrowserDownloadURL(),
	)

	release = &Release{
		Version:  ver,
		Platform: up.platform,
		Asset: Asset{
			ID:   asset.GetID(),
			Name: asset.GetName(),
			URL:  asset.GetBrowserDownloadURL(),
			Size: asset.GetSize(),
		},
		ReleaseID:    rel.GetID(),
		URL:          rel.GetURL(),
		ReleaseNotes: rel.GetReleaseNotes(),
		Name:         rel.GetName(),
		PublishedAt:  rel.GetPublishedAt(),
		repository:   repository,
	}
	return release, true, nil
}

func (up *Updater) findReleaseAndAsset(rels []SourceRelease, targetVersion string) (SourceRelease, SourceAsset, Version, bool) {
	for _, arch := range getAdditionalArch(up.platform.Arch, up.platform.Arm, up.universalArch) {
		release, asset, version, found := up.findReleaseAndAssetForArch(arch, rels, targetVersion)
		if found {
			return release, asset, version, found
		}
	}
	return nil, nil, Version{}, false
}

func (up *Updater) findReleaseAndAssetForArch(arch string, rels []SourceRelease, targetVersion string) (SourceRelease, SourceAsset, Version, bool) {
	var bestVer Version
	var bestAsset SourceAsset
	var bestRelease SourceRelease
	found := false

	log.Printf("Searching for a possible candidate for os %q and arch %q", up.platform.OS, arch)

	compare := up.compareVersions
	if compare == nil {
		compare = defaultCompareVersions
	}

	for _, rel := range rels {
		if a, v, ok := up.findAssetFromRelease(rel, up.getSuffixes(arch), targetVersion); ok {
			if !found || compare(bestVer, v) {
				bestVer = v
				bestAsset = a
				bestRelease = rel
				found = true
			}
		}
	}

	if !found {
		log.Printf("Could not find any release for os %q and arch %q", up.platform.OS, arch)
	}
	return bestRelease, bestAsset, bestVer, found
}

func (up *Updater) findAssetFromRelease(rel SourceRelease, suffixes []string, targetVersion string) (SourceAsset, Version, bool) {
	if rel == nil {
		return nil, Version{}, false
	}
	if targetVersion != "" && targetVersion != rel.GetTagName() {
		log.Printf("Skip %s not matching to specified version %s", rel.GetTagName(), targetVersion)
		return nil, Version{}, false
	}

	if rel.GetDraft() && !up.versionFilter.Draft && targetVersion == "" {
		log.Printf("Skip draft version %s", rel.GetTagName())
		return nil, Version{}, false
	}
	if rel.GetPrerelease() && !up.versionFilter.Prerelease && targetVersion == "" {
		log.Printf("Skip pre-release version %s", rel.GetTagName())
		return nil, Version{}, false
	}

	ver, ok := parseVersion(rel.GetTagName())
	if !ok {
		log.Printf("Skip version not adopting semver: %s", rel.GetTagName())
		return nil, Version{}, false
	}

	for _, asset := range rel.GetAssets() {
		name := strings.ToLower(asset.GetName())
		if up.hasFilters() {
			if up.assetMatchFilters(name) {
				return asset, ver, true
			}
		} else {
			if up.assetMatchSuffixes(name, suffixes) {
				return asset, ver, true
			}
		}

		name = strings.ToLower(asset.GetBrowserDownloadURL())
		if up.hasFilters() {
			if up.assetMatchFilters(name) {
				return asset, ver, true
			}
		} else {
			if up.assetMatchSuffixes(name, suffixes) {
				return asset, ver, true
			}
		}
	}

	log.Printf("No suitable asset was found in release %s", rel.GetTagName())
	return nil, Version{}, false
}

func (up *Updater) hasFilters() bool {
	return len(up.filters) > 0
}

func (up *Updater) assetMatchFilters(name string) bool {
	for _, filter := range up.filters {
		if filter.MatchString(name) {
			log.Printf("Selected filtered asset: %s", name)
			return true
		}
		log.Printf("Skipping asset %q not matching filter %v\n", name, filter)
	}
	return false
}

func (up *Updater) assetMatchSuffixes(name string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func (up *Updater) getSuffixes(arch string) []string {
	suffixes := make([]string, 0)
	for _, sep := range []rune{'_', '-'} {
		for _, ext := range []string{".zip", ".tar.gz", ".tgz", ".gzip", ".gz", ".tar.xz", ".xz", ".bz2", ""} {
			suffix := fmt.Sprintf("%s%c%s%s", up.platform.OS, sep, arch, ext)
			suffixes = append(suffixes, suffix)
			if up.platform.OS == "windows" {
				suffix = fmt.Sprintf("%s%c%s.exe%s", up.platform.OS, sep, arch, ext)
				suffixes = append(suffixes, suffix)
			}
		}
	}

	// also check extensions from user-registered decompressors
	for ext := range up.decompressors {
		found := false
		for _, existing := range suffixes {
			if strings.HasSuffix(existing, ext) {
				found = true
				break
			}
		}
		if !found {
			for _, sep := range []rune{'_', '-'} {
				suffixes = append(suffixes, fmt.Sprintf("%s%c%s%s", up.platform.OS, sep, arch, ext))
			}
		}
	}

	return suffixes
}

// versionEqual returns true if two versions are equal.
func versionEqual(a, b Version) bool {
	return a.Major == b.Major && a.Minor == b.Minor && a.Patch == b.Patch && a.Prerelease == b.Prerelease
}
