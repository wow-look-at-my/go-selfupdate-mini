package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func fakeNpmServer(t *testing.T, name, version string) *httptest.Server {
	t.Helper()
	npmOS := goToNpmOS[runtime.GOOS]
	npmArch := goToNpmArch[runtime.GOARCH]
	platPkg := name + "-" + npmOS + "-" + npmArch

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/@test-scope/" + platPkg:
			tarballURL := srv.URL + "/tarball/" + version + ".tgz"
			fmt.Fprintf(w, `{
				"dist-tags": {"latest": %q},
				"versions": {%q: {"dist": {"tarball": %q}}},
				"time": {%q: "2026-01-01T00:00:00Z"}
			}`, version, version, tarballURL, version)

		case "/tarball/" + version + ".tgz":
			w.Header().Set("Content-Type", "application/octet-stream")
			writeFakeBinaryTarball(t, w, name)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeFakeBinaryTarball(t *testing.T, w io.Writer, name string) {
	t.Helper()
	binName := name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	content := []byte("#!/bin/sh\necho hello\n")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.Nil(t, tw.WriteHeader(&tar.Header{
		Name: "package/bin/" + binName,
		Mode: 0755,
		Size: int64(len(content)),
	}))
	_, err := tw.Write(content)
	require.Nil(t, err)
	require.Nil(t, tw.Close())
	require.Nil(t, gz.Close())
	_, err = w.Write(buf.Bytes())
	require.Nil(t, err)
}

func TestNpmSource_ListReleases(t *testing.T) {
	srv := fakeNpmServer(t, "myapp", "1.2.3")

	source, err := NewNpmSource(NpmConfig{Registry: srv.URL})
	require.Nil(t, err)

	repo := NpmRepository{Scope: "@test-scope", Name: "myapp"}
	releases, err := source.ListReleases(context.Background(), repo)
	require.Nil(t, err)
	require.Len(t, releases, 1)

	rel := releases[0]
	assert.Equal(t, "v1.2.3", rel.GetTagName())
	assert.False(t, rel.GetDraft())
	assert.False(t, rel.GetPrerelease())

	assets := rel.GetAssets()
	require.Len(t, assets, 1)
	expectedName := fmt.Sprintf("myapp_%s_%s.tgz", runtime.GOOS, runtime.GOARCH)
	assert.Equal(t, expectedName, assets[0].GetName())
	assert.Contains(t, assets[0].GetBrowserDownloadURL(), "/tarball/1.2.3.tgz")
}

func TestNpmSource_ListReleases_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	source, err := NewNpmSource(NpmConfig{Registry: srv.URL})
	require.Nil(t, err)

	repo := NpmRepository{Scope: "@test-scope", Name: "myapp"}
	releases, err := source.ListReleases(context.Background(), repo)
	require.Nil(t, err)
	assert.Nil(t, releases)
}

func TestNpmSource_DownloadReleaseAsset(t *testing.T) {
	srv := fakeNpmServer(t, "myapp", "1.2.3")

	source, err := NewNpmSource(NpmConfig{Registry: srv.URL})
	require.Nil(t, err)

	repo := NpmRepository{Scope: "@test-scope", Name: "myapp"}
	releases, err := source.ListReleases(context.Background(), repo)
	require.Nil(t, err)
	require.Len(t, releases, 1)

	assetID := releases[0].GetAssets()[0].GetID()
	rc, err := source.DownloadReleaseAsset(context.Background(), nil, assetID)
	require.Nil(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.Nil(t, err)
	assert.True(t, len(data) > 0, "downloaded tarball should not be empty")

	// Verify it's a valid gzip stream.
	gz, err := gzip.NewReader(bytes.NewReader(data))
	require.Nil(t, err)
	gz.Close()
}

func TestNpmSource_DownloadReleaseAsset_UnknownID(t *testing.T) {
	source, err := NewNpmSource(NpmConfig{Registry: "http://localhost"})
	require.Nil(t, err)

	_, err = source.DownloadReleaseAsset(context.Background(), nil, 99999)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "unknown asset ID")
}

func TestNpmSource_WithToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		npmOS := goToNpmOS[runtime.GOOS]
		npmArch := goToNpmArch[runtime.GOARCH]
		platPkg := "myapp-" + npmOS + "-" + npmArch
		if r.URL.Path == "/@test-scope/"+platPkg {
			fmt.Fprint(w, `{"dist-tags":{"latest":"1.0.0"},"versions":{"1.0.0":{"dist":{"tarball":"http://example.com/t.tgz"}}}}`)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	source, err := NewNpmSource(NpmConfig{Registry: srv.URL, Token: "secret-token"})
	require.Nil(t, err)

	repo := NpmRepository{Scope: "@test-scope", Name: "myapp"}
	_, err = source.ListReleases(context.Background(), repo)
	require.Nil(t, err)
	assert.Equal(t, "Bearer secret-token", gotAuth)
}

func TestNpmRepository_GetSlug(t *testing.T) {
	repo := NpmRepository{Scope: "@wow-look-at-my", Name: "go-toolchain"}
	scope, name, err := repo.GetSlug()
	require.Nil(t, err)
	assert.Equal(t, "@wow-look-at-my", scope)
	assert.Equal(t, "go-toolchain", name)
}

func TestNpmSource_MultipleVersions(t *testing.T) {
	npmOS := goToNpmOS[runtime.GOOS]
	npmArch := goToNpmArch[runtime.GOARCH]
	platPkg := "myapp-" + npmOS + "-" + npmArch

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/@test-scope/"+platPkg {
			fmt.Fprintf(w, `{
				"dist-tags": {"latest": "2.0.0"},
				"versions": {
					"1.0.0": {"dist": {"tarball": "%s/t/1.0.0.tgz"}},
					"2.0.0": {"dist": {"tarball": "%s/t/2.0.0.tgz"}}
				}
			}`, srv.URL, srv.URL)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	source, err := NewNpmSource(NpmConfig{Registry: srv.URL})
	require.Nil(t, err)

	repo := NpmRepository{Scope: "@test-scope", Name: "myapp"}
	releases, err := source.ListReleases(context.Background(), repo)
	require.Nil(t, err)
	assert.Len(t, releases, 2)

	tags := map[string]bool{}
	for _, r := range releases {
		tags[r.GetTagName()] = true
	}
	assert.True(t, tags["v1.0.0"])
	assert.True(t, tags["v2.0.0"])
}

func TestNpmSource_EndToEnd_WithUpdater(t *testing.T) {
	srv := fakeNpmServer(t, "myapp", "1.0.0")

	source, err := NewNpmSource(NpmConfig{Registry: srv.URL})
	require.Nil(t, err)

	updater, err := NewUpdater(Config{Source: source})
	require.Nil(t, err)

	repo := NpmRepository{Scope: "@test-scope", Name: "myapp"}
	rel, found, err := updater.DetectLatest(context.Background(), repo)
	require.Nil(t, err)
	require.True(t, found)
	assert.Equal(t, "v1.0.0", rel.Version.Original)
	assert.Equal(t, "1.0.0", rel.Version.Version)
}

func TestNpmRepositoryFromBuildInfo(t *testing.T) {
	repo, err := NpmRepositoryFromBuildInfo()
	// Build info is always available in a properly built test binary.
	require.Nil(t, err)

	// The scope must be "@<something>".
	assert.True(t, len(repo.Scope) > 1 && repo.Scope[0] == '@', "scope %q should start with @", repo.Scope)
	// The name must be non-empty.
	assert.True(t, len(repo.Name) > 0, "name should be non-empty")

	// When running tests for this module the module path is
	// "github.com/wow-look-at-my/go-selfupdate-mini", so verify the derivation
	// is consistent (don't hard-code the exact value to stay fork-friendly).
	assert.Equal(t, "@"+strings.Split(repo.Scope, "@")[1], repo.Scope)
	assert.NotContains(t, repo.Name, "/", "name should not contain slashes")
}
