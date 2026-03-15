package selfupdate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubSourceListReleases(t *testing.T) {
	releases := []githubRelease{
		{ID: 1, TagName: "v1.0.0", Name: "v1.0.0", HTMLURL: "https://github.com/test/repo/releases/v1.0.0",
			Assets: []githubAsset{{ID: 10, Name: "app_linux_amd64.tar.gz", Size: 1024, BrowserDownloadURL: "https://example.com/app.tar.gz"}}},
		{ID: 2, TagName: "v2.0.0", Name: "v2.0.0", HTMLURL: "https://github.com/test/repo/releases/v2.0.0"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/test/repo/releases" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Errorf("unexpected Accept header: %s", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	src, err := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	rels, err := src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rels) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(rels))
	}
	if rels[0].GetTagName() != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %s", rels[0].GetTagName())
	}
	assets := rels[0].GetAssets()
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	if assets[0].GetName() != "app_linux_amd64.tar.gz" {
		t.Errorf("unexpected asset name: %s", assets[0].GetName())
	}
}

func TestGitHubSourceListReleasesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	rels, err := src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))
	if err != nil {
		t.Fatal(err)
	}
	if rels != nil {
		t.Error("expected nil releases for 404")
	}
}

func TestGitHubSourceListReleasesServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	_, err := src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))
	if err == nil {
		t.Error("expected error for 500")
	}
}

func TestGitHubSourceListReleasesInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	_, err := src.ListReleases(context.Background(), NewRepositorySlug("test", "repo"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGitHubSourceListReleasesInvalidSlug(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{})
	_, err := src.ListReleases(context.Background(), NewRepositorySlug("", ""))
	if err == nil {
		t.Error("expected error for invalid slug")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	data := make([]byte, 100)
	n, _ := rc.Read(data)
	if string(data[:n]) != "binary data" {
		t.Errorf("unexpected data: %q", data[:n])
	}
}

func TestGitHubSourceDownloadNilRelease(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{})
	_, err := src.DownloadReleaseAsset(context.Background(), nil, 1)
	if err != ErrInvalidRelease {
		t.Errorf("expected ErrInvalidRelease, got %v", err)
	}
}

func TestGitHubSourceDownloadInvalidSlug(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{})
	rel := &Release{repository: NewRepositorySlug("", "")}
	_, err := src.DownloadReleaseAsset(context.Background(), rel, 1)
	if err == nil {
		t.Error("expected error for invalid slug")
	}
}

func TestGitHubSourceDownloadServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: srv.URL})
	rel := &Release{repository: NewRepositorySlug("test", "repo")}
	_, err := src.DownloadReleaseAsset(context.Background(), rel, 1)
	if err == nil {
		t.Error("expected error for 500")
	}
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

	if gotAuth != "Bearer test-token" {
		t.Errorf("expected Bearer auth, got %q", gotAuth)
	}
}

func TestGitHubSourceDefaultBaseURL(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{})
	if src.baseURL != "https://api.github.com" {
		t.Errorf("expected default base URL, got %s", src.baseURL)
	}
}

func TestGitHubSourceEnterpriseURL(t *testing.T) {
	src, _ := NewGitHubSource(GitHubConfig{EnterpriseBaseURL: "https://github.corp.com/api/v3/"})
	if src.baseURL != "https://github.corp.com/api/v3" {
		t.Errorf("expected trimmed URL, got %s", src.baseURL)
	}
}

func TestGitHubReleaseInterface(t *testing.T) {
	r := &githubRelease{
		ID: 1, TagName: "v1.0.0", Name: "Release 1",
		Draft: false, Prerelease: true, Body: "notes", HTMLURL: "https://example.com",
		Assets: []githubAsset{{ID: 10, Name: "asset", Size: 100, BrowserDownloadURL: "https://example.com/asset"}},
	}

	if r.GetID() != 1 {
		t.Error("ID mismatch")
	}
	if r.GetTagName() != "v1.0.0" {
		t.Error("TagName mismatch")
	}
	if r.GetDraft() {
		t.Error("Draft should be false")
	}
	if !r.GetPrerelease() {
		t.Error("Prerelease should be true")
	}
	if r.GetReleaseNotes() != "notes" {
		t.Error("ReleaseNotes mismatch")
	}
	if r.GetURL() != "https://example.com" {
		t.Error("URL mismatch")
	}
	if r.GetName() != "Release 1" {
		t.Error("Name mismatch")
	}

	assets := r.GetAssets()
	if len(assets) != 1 {
		t.Fatal("expected 1 asset")
	}
	a := assets[0]
	if a.GetID() != 10 || a.GetName() != "asset" || a.GetSize() != 100 || a.GetBrowserDownloadURL() != "https://example.com/asset" {
		t.Error("asset fields mismatch")
	}
}
