package selfupdate

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

// mockRelease implements SourceRelease for testing.
type mockRelease struct {
	id		int64
	tagName		string
	name		string
	draft		bool
	prerelease	bool
	publishedAt	time.Time
	releaseNotes	string
	url		string
	assets		[]SourceAsset
}

func (r *mockRelease) GetID() int64			{ return r.id }
func (r *mockRelease) GetTagName() string		{ return r.tagName }
func (r *mockRelease) GetName() string			{ return r.name }
func (r *mockRelease) GetDraft() bool			{ return r.draft }
func (r *mockRelease) GetPrerelease() bool		{ return r.prerelease }
func (r *mockRelease) GetPublishedAt() time.Time	{ return r.publishedAt }
func (r *mockRelease) GetReleaseNotes() string		{ return r.releaseNotes }
func (r *mockRelease) GetURL() string			{ return r.url }
func (r *mockRelease) GetAssets() []SourceAsset		{ return r.assets }

// mockAsset implements SourceAsset for testing.
type mockAsset struct {
	id	int64
	name	string
	size	int
	url	string
}

func (a *mockAsset) GetID() int64			{ return a.id }
func (a *mockAsset) GetName() string			{ return a.name }
func (a *mockAsset) GetSize() int			{ return a.size }
func (a *mockAsset) GetBrowserDownloadURL() string	{ return a.url }

// mockSource implements Source for testing.
type mockSource struct {
	releases	[]SourceRelease
	err		error
	assets		map[int64]string	// assetID -> content
}

func (s *mockSource) ListReleases(_ context.Context, _ Repository) ([]SourceRelease, error) {
	return s.releases, s.err
}

func (s *mockSource) DownloadReleaseAsset(_ context.Context, _ *Release, assetID int64) (io.ReadCloser, error) {
	if content, ok := s.assets[assetID]; ok {
		return io.NopCloser(strings.NewReader(content)), nil
	}
	return nil, fmt.Errorf("asset %d not found", assetID)
}

func newTestRelease(tag string, assets ...SourceAsset) *mockRelease {
	return &mockRelease{
		id:		1,
		tagName:	tag,
		name:		tag,
		url:		"https://github.com/test/repo/releases/" + tag,
		assets:		assets,
	}
}

func newTestAsset(name string) *mockAsset {
	return &mockAsset{
		id:	1,
		name:	name,
		size:	1024,
		url:	"https://github.com/test/repo/releases/download/" + name,
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		tag	string
		wantOK	bool
		version	string
		major	int
		minor	int
		patch	int
		pre	string
	}{
		{"v1.2.3", true, "1.2.3", 1, 2, 3, ""},
		{"1.2.3", true, "1.2.3", 1, 2, 3, ""},
		{"v1.0.0-beta", true, "1.0.0-beta", 1, 0, 0, "beta"},
		{"release-2.0.1", true, "2.0.1", 2, 0, 1, ""},
		{"not-a-version", false, "", 0, 0, 0, ""},
		{"v1.2.3-rc.1+build.123", true, "1.2.3-rc.1+build.123", 1, 2, 3, "rc.1"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			v, ok := parseVersion(tt.tag)
			require.Equal(t, tt.wantOK, ok)

			if !ok {
				return
			}
			assert.Equal(t, tt.version, v.Version)

			assert.False(t, v.Major != tt.major || v.Minor != tt.minor || v.Patch != tt.patch)

			assert.Equal(t, tt.pre, v.Prerelease)

			assert.Equal(t, tt.tag, v.Original)

			assert.Equal(t, (tt.pre != ""), v.IsPrerelease)

		})
	}
}

func TestDefaultCompareVersions(t *testing.T) {
	v1, _ := parseVersion("v1.0.0")
	v2, _ := parseVersion("v2.0.0")
	v1beta, _ := parseVersion("v1.0.0-beta")

	assert.True(t, defaultCompareVersions(v1, v2))

	assert.False(t, defaultCompareVersions(v2, v1))

	assert.False(t, defaultCompareVersions(v1, v1))

	assert.True(t, defaultCompareVersions(v1beta, v1))

	bad := Version{Version: "not-valid"}
	assert.False(t, defaultCompareVersions(bad, v1))

	assert.False(t, defaultCompareVersions(v1, bad))

}

func TestParseCurrentVersion(t *testing.T) {
	v, err := parseCurrentVersion("1.2.3")
	require.Nil(t, err)
	assert.False(t, v.Major != 1 || v.Minor != 2 || v.Patch != 3)
}

func TestParseCurrentVersionNonSemver(t *testing.T) {
	// Non-semver strings like "dev" must not return an error; they are treated
	// as 0.0.0 so any real release will compare as newer.
	for _, input := range []string{"dev", "bad", "(devel)", "abcdef012345"} {
		v, err := parseCurrentVersion(input)
		assert.Nil(t, err, "input %q should not error", input)
		assert.Equal(t, "0.0.0", v.Version, "input %q should yield 0.0.0", input)
		assert.Equal(t, input, v.Original, "Original should preserve the raw input")
	}
}

func TestDetectLatest(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("app_linux_amd64.tar.gz")),
			newTestRelease("v2.0.0", newTestAsset("app_linux_amd64.tar.gz")),
			newTestRelease("v1.5.0", newTestAsset("app_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	rel, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

	assert.Equal(t, "2.0.0", rel.Version.Version)

}

func TestDetectVersion(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("app_linux_amd64.tar.gz")),
			newTestRelease("v2.0.0", newTestAsset("app_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
	})

	rel, found, err := up.DetectVersion(context.Background(), NewRepositorySlug("test", "repo"), "v1.0.0")
	require.Nil(t, err)

	require.True(t, found)

	assert.Equal(t, "1.0.0", rel.Version.Version)

}

func TestDetectLatestNoReleases(t *testing.T) {
	src := &mockSource{releases: nil}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}})

	_, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	assert.False(t, found)

}

func TestDetectLatestSourceError(t *testing.T) {
	src := &mockSource{err: fmt.Errorf("API error")}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}})

	_, _, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)

}

func TestDetectSkipsDrafts(t *testing.T) {
	draft := newTestRelease("v3.0.0", newTestAsset("app_linux_amd64.tar.gz"))
	draft.draft = true
	src := &mockSource{
		releases: []SourceRelease{
			draft,
			newTestRelease("v1.0.0", newTestAsset("app_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}})

	rel, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

	assert.Equal(t, "1.0.0", rel.Version.Version)

}

func TestDetectIncludesDraftsWhenEnabled(t *testing.T) {
	draft := newTestRelease("v3.0.0", newTestAsset("app_linux_amd64.tar.gz"))
	draft.draft = true
	src := &mockSource{
		releases: []SourceRelease{
			draft,
			newTestRelease("v1.0.0", newTestAsset("app_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		Version:	VersionFilter{Draft: true},
	})

	rel, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

	assert.Equal(t, "3.0.0", rel.Version.Version)

}

func TestDetectSkipsPrereleases(t *testing.T) {
	pre := newTestRelease("v3.0.0-beta", newTestAsset("app_linux_amd64.tar.gz"))
	pre.prerelease = true
	src := &mockSource{
		releases: []SourceRelease{
			pre,
			newTestRelease("v1.0.0", newTestAsset("app_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}})

	rel, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

	assert.Equal(t, "1.0.0", rel.Version.Version)

}

func TestDetectSkipsNonSemver(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("nightly", newTestAsset("app_linux_amd64.tar.gz")),
		},
	}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}})

	_, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	assert.False(t, found)

}

func TestDetectWithFilters(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0",
				newTestAsset("app-special-build.tar.gz"),
				newTestAsset("app_linux_amd64.tar.gz"),
			),
		},
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		Filters:	[]string{"special"},
	})

	rel, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

	assert.Equal(t, "app-special-build.tar.gz", rel.Asset.Name)

}

func TestDetectFilterMatchesBrowserURL(t *testing.T) {
	asset := &mockAsset{
		id:	1,
		name:	"generic-name",
		size:	100,
		url:	"https://example.com/download/special-linux-amd64.tar.gz",
	}
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", asset),
		},
	}
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		Filters:	[]string{"special"},
	})

	_, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

}

func TestDetectNoMatchingAsset(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("app_windows_amd64.zip")),
		},
	}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}})

	_, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	assert.False(t, found)

}

func TestDetectNilRelease(t *testing.T) {
	src := &mockSource{releases: []SourceRelease{nil}}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}})

	_, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	assert.False(t, found)

}

func TestDetectWindowsSuffixes(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("app_windows_amd64.exe.zip")),
		},
	}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "windows", Arch: "amd64"}})

	rel, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

	assert.Equal(t, "app_windows_amd64.exe.zip", rel.Asset.Name)

}

func TestDetectMatchesDownloadURL(t *testing.T) {
	asset := &mockAsset{
		id:	1,
		name:	"some-id-12345",
		size:	100,
		url:	"https://example.com/app_linux_amd64.tar.gz",
	}
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", asset),
		},
	}
	up, _ := NewUpdater(Config{Source: src, Platform: Platform{OS: "linux", Arch: "amd64"}})

	_, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

}

func TestDetectCustomCompareVersions(t *testing.T) {
	src := &mockSource{
		releases: []SourceRelease{
			newTestRelease("v1.0.0", newTestAsset("app_linux_amd64.tar.gz")),
			newTestRelease("v2.0.0", newTestAsset("app_linux_amd64.tar.gz")),
		},
	}
	// custom compare that prefers lower versions
	up, _ := NewUpdater(Config{
		Source:		src,
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		CompareVersions: func(current, candidate Version) bool {
			return candidate.Major < current.Major
		},
	})

	rel, found, err := up.DetectLatest(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.True(t, found)

	assert.Equal(t, "1.0.0", rel.Version.Version)

}

func TestGetSuffixesIncludesCustomDecompressors(t *testing.T) {
	up, _ := NewUpdater(Config{
		Platform:	Platform{OS: "linux", Arch: "amd64"},
		Decompressors: map[string]Decompressor{
			".tar.zst": DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) {
				return src, nil
			}),
		},
	})

	suffixes := up.getSuffixes("amd64")
	found := false
	for _, s := range suffixes {
		if strings.HasSuffix(s, ".tar.zst") {
			found = true
			break
		}
	}
	assert.True(t, found)

}

func TestVersionEqual(t *testing.T) {
	a := Version{Major: 1, Minor: 2, Patch: 3, Prerelease: ""}
	b := Version{Major: 1, Minor: 2, Patch: 3, Prerelease: ""}
	c := Version{Major: 2, Minor: 0, Patch: 0, Prerelease: ""}
	d := Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "beta"}

	assert.True(t, versionEqual(a, b))

	assert.False(t, versionEqual(a, c))

	assert.False(t, versionEqual(a, d))

}

func TestNewUpdaterInvalidFilter(t *testing.T) {
	_, err := NewUpdater(Config{Filters: []string{"[invalid"}})
	assert.NotNil(t, err)

}
