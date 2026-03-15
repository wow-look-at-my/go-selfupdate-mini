package selfupdate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestGitHubSourceListReleases(t *testing.T) {
	releases := []githubRelease{
		{ID: 1, TagName: "v1.0.0", Name: "v1.0.0", HTMLURL: "https://github.com/test/repo/releases/v1.0.0",
			Assets:	[]githubAsset{{ID: 10, Name: "app_linux_amd64.tar.gz", Size: 1024, BrowserDownloadURL: "https://example.com/app.tar.gz"}}},
		{ID: 2, TagName: "v2.0.0", Name: "v2.0.0", HTMLURL: "https://github.com/test/repo/releases/v2.0.0"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/test/repo/releases", r.URL.Path)

		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	src, err := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	require.Nil(t, err)

	rels, err := src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	require.Equal(t, 2, len(rels))

	assert.Equal(t, "v1.0.0", rels[0].GetTagName())

	assets := rels[0].GetAssets()
	require.Equal(t, 1, len(assets))

	assert.Equal(t, "app_linux_amd64.tar.gz", assets[0].GetName())

}

func TestGitHubSourceListReleasesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	rels, err := src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))
	require.Nil(t, err)

	assert.Nil(t, rels)

}

func TestGitHubSourceListReleasesServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	_, err := src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)

}

func TestGitHubSourceListReleasesInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	_, err := src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))
	assert.NotNil(t, err)

}

func TestGitHubSourceListReleasesInvalidSlug(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{})
	_, err := src.ListReleases(context.Background(), NewRepositorySlug("", ""))
	assert.NotNil(t, err)

}

func TestGitHubSourceDownloadReleaseAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/repo/releases/assets/42" {
			w.Write([]byte("binary data"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	rel := &Release{repository: NewRepositorySlug("test", "repo")}

	rc, err := src.DownloadReleaseAsset(context.Background(), rel, 42)
	require.Nil(t, err)

	defer rc.Close()

	data := make([]byte, 100)
	n, _ := rc.Read(data)
	assert.Equal(t, "binary data", string(data[:n]))

}

func TestGitHubSourceDownloadNilRelease(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{})
	_, err := src.DownloadReleaseAsset(context.Background(), nil, 1)
	assert.Equal(t, ErrInvalidRelease, err)

}

func TestGitHubSourceDownloadInvalidSlug(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{})
	rel := &Release{repository: NewRepositorySlug("", "")}
	_, err := src.DownloadReleaseAsset(context.Background(), rel, 1)
	assert.NotNil(t, err)

}

func TestGitHubSourceDownloadServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	rel := &Release{repository: NewRepositorySlug("test", "repo")}
	_, err := src.DownloadReleaseAsset(context.Background(), rel, 1)
	assert.NotNil(t, err)

}

func TestGitHubSourceWithToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{APIToken: "test-token", EnterpriseBaseURL: srv.URL})
	src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))

	assert.Equal(t, "Bearer test-token", gotAuth)

}

func TestGitHubSourceDefaultBaseURL(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{})
	assert.Equal(t, "https://api.github.com", src.baseURL)

}

func TestGitHubSourceEnterpriseURL(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: "https://github.corp.com/api/v3/"})
	assert.Equal(t, "https://github.corp.com/api/v3", src.baseURL)

}

func TestGitHubReleaseInterface(t *testing.T) {
	r := &githubRelease{
		ID:	1, TagName: "v1.0.0", Name: "Release 1",
		Draft:	false, Prerelease: true, Body: "notes", HTMLURL: "https://example.com",
		Assets:	[]githubAsset{{ID: 10, Name: "asset", Size: 100, BrowserDownloadURL: "https://example.com/asset"}},
	}

	assert.Equal(t, int64(1), r.GetID())

	assert.Equal(t, "v1.0.0", r.GetTagName())

	assert.False(t, r.GetDraft())

	assert.True(t, r.GetPrerelease())

	assert.Equal(t, "notes", r.GetReleaseNotes())

	assert.Equal(t, "https://example.com", r.GetURL())

	assert.Equal(t, "Release 1", r.GetName())

	assets := r.GetAssets()
	require.Equal(t, 1, len(assets))

	a := assets[0]
	assert.False(t, a.GetID() != 10 || a.GetName() != "asset" || a.GetSize() != 100 || a.GetBrowserDownloadURL() != "https://example.com/asset")

}
