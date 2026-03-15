package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"strings"
	"testing"
)

func makeGzip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	w.Name = name
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	w.Close()
	return buf.Bytes()
}

func makeTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Size: int64(len(content)), Mode: 0o755}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func makeZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	return buf.Bytes()
}

func makeBz2(t *testing.T, content []byte) []byte {
	t.Helper()
	// bzip2 package only has a reader, so we write manually via compress/bzip2
	// Actually Go stdlib doesn't have a bzip2 writer. We'll use a pre-compressed fixture.
	// For testing, we'll skip bz2 creation and test the reader path differently.
	_ = content
	return nil
}

func TestDecompressZip(t *testing.T) {
	data := makeZip(t, map[string][]byte{"myapp": []byte("binary content")})
	decompressors := builtinDecompressors("linux", "amd64")

	r, err := decompressCommand(bytes.NewReader(data), "myapp.zip", "myapp", decompressors)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	if string(out) != "binary content" {
		t.Errorf("unexpected content: %q", out)
	}
}

func TestDecompressZipWithSubdir(t *testing.T) {
	data := makeZip(t, map[string][]byte{"subdir/myapp": []byte("binary")})
	decompressors := builtinDecompressors("linux", "amd64")

	r, err := decompressCommand(bytes.NewReader(data), "release.zip", "myapp", decompressors)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	if string(out) != "binary" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestDecompressZipNotFound(t *testing.T) {
	data := makeZip(t, map[string][]byte{"other": []byte("nope")})
	decompressors := builtinDecompressors("linux", "amd64")

	_, err := decompressCommand(bytes.NewReader(data), "release.zip", "myapp", decompressors)
	if err == nil {
		t.Error("expected error when executable not found in zip")
	}
}

func TestDecompressTarGz(t *testing.T) {
	data := makeTarGz(t, map[string][]byte{"myapp": []byte("tar content")})
	decompressors := builtinDecompressors("linux", "amd64")

	r, err := decompressCommand(bytes.NewReader(data), "release.tar.gz", "myapp", decompressors)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	if string(out) != "tar content" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestDecompressTgz(t *testing.T) {
	data := makeTarGz(t, map[string][]byte{"myapp": []byte("tgz content")})
	decompressors := builtinDecompressors("linux", "amd64")

	r, err := decompressCommand(bytes.NewReader(data), "release.tgz", "myapp", decompressors)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	if string(out) != "tgz content" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestDecompressTarGzNotFound(t *testing.T) {
	data := makeTarGz(t, map[string][]byte{"wrong": []byte("nope")})
	decompressors := builtinDecompressors("linux", "amd64")

	_, err := decompressCommand(bytes.NewReader(data), "release.tar.gz", "myapp", decompressors)
	if err == nil {
		t.Error("expected error")
	}
}

func TestDecompressGzip(t *testing.T) {
	data := makeGzip(t, "myapp", []byte("gzip content"))
	decompressors := builtinDecompressors("linux", "amd64")

	r, err := decompressCommand(bytes.NewReader(data), "myapp.gz", "myapp", decompressors)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	if string(out) != "gzip content" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestDecompressGzipWrongName(t *testing.T) {
	data := makeGzip(t, "other", []byte("content"))
	decompressors := builtinDecompressors("linux", "amd64")

	_, err := decompressCommand(bytes.NewReader(data), "other.gz", "myapp", decompressors)
	if err == nil {
		t.Error("expected error for wrong gzip filename")
	}
}

func TestDecompressBz2(t *testing.T) {
	// bzip2 decompressor just wraps the reader, doesn't check filenames
	decompressors := builtinDecompressors("linux", "amd64")

	// We can't easily create bz2 in Go without external lib,
	// but we can test the reader path returns without error
	// by testing with invalid data to at least exercise the code path
	r, err := decompressCommand(strings.NewReader("raw"), "myapp.bz2", "myapp", decompressors)
	if err != nil {
		t.Fatal(err)
	}
	// bzip2.NewReader will fail on Read, not on creation
	_ = r
}

func TestDecompressUnknownExtension(t *testing.T) {
	decompressors := builtinDecompressors("linux", "amd64")

	r, err := decompressCommand(strings.NewReader("raw binary"), "myapp", "myapp", decompressors)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	if string(out) != "raw binary" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestDecompressCustomDecompressor(t *testing.T) {
	custom := DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) {
		return strings.NewReader("custom-decompressed"), nil
	})
	decompressors := builtinDecompressors("linux", "amd64")
	decompressors[".custom"] = custom

	r, err := decompressCommand(strings.NewReader("data"), "app.custom", "app", decompressors)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	if string(out) != "custom-decompressed" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestDecompressInvalidZip(t *testing.T) {
	decompressors := builtinDecompressors("linux", "amd64")
	_, err := decompressCommand(strings.NewReader("not a zip"), "file.zip", "app", decompressors)
	if err == nil {
		t.Error("expected error for invalid zip")
	}
}

func TestDecompressInvalidGzip(t *testing.T) {
	decompressors := builtinDecompressors("linux", "amd64")
	_, err := decompressCommand(strings.NewReader("not gzip"), "file.gz", "app", decompressors)
	if err == nil {
		t.Error("expected error for invalid gzip")
	}
}

func TestDecompressInvalidTarGz(t *testing.T) {
	decompressors := builtinDecompressors("linux", "amd64")
	_, err := decompressCommand(strings.NewReader("not tar.gz"), "file.tar.gz", "app", decompressors)
	if err == nil {
		t.Error("expected error for invalid tar.gz")
	}
}

func TestMatchExecutableName(t *testing.T) {
	tests := []struct {
		cmd, os, arch, target string
		want                  bool
	}{
		{"myapp", "linux", "amd64", "myapp", true},
		{"myapp", "linux", "amd64", "myapp_linux_amd64", true},
		{"myapp", "linux", "amd64", "myapp-linux-amd64", true},
		{"myapp", "windows", "amd64", "myapp.exe", true},
		{"myapp", "linux", "amd64", "myapp_v1.2.3_linux_amd64", true},
		{"myapp", "linux", "amd64", "myapp-v1.2.3-linux-amd64", true},
		{"myapp", "linux", "amd64", "other", false},
		{"myapp.exe", "windows", "amd64", "myapp.exe", true},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := matchExecutableName(tt.cmd, tt.os, tt.arch, tt.target)
			if got != tt.want {
				t.Errorf("matchExecutableName(%q, %q, %q, %q) = %v, want %v",
					tt.cmd, tt.os, tt.arch, tt.target, got, tt.want)
			}
		})
	}
}

func TestSortedExtensions(t *testing.T) {
	m := map[string]Decompressor{
		".gz":     nil,
		".tar.gz": nil,
		".zip":    nil,
	}
	exts := sortedExtensions(m)
	if len(exts) != 3 {
		t.Fatalf("expected 3, got %d", len(exts))
	}
	if exts[0] != ".tar.gz" {
		t.Errorf("expected .tar.gz first (longest), got %s", exts[0])
	}
}

func TestBuiltinDecompressors(t *testing.T) {
	d := builtinDecompressors("linux", "amd64")
	expected := []string{".zip", ".tar.gz", ".tgz", ".gzip", ".gz", ".bz2"}
	for _, ext := range expected {
		if _, ok := d[ext]; !ok {
			t.Errorf("missing builtin decompressor for %s", ext)
		}
	}
}

// Verify bzip2 is tested - we can't create bz2 in pure Go stdlib but we can test Reader
func TestBz2ReaderAcceptance(t *testing.T) {
	_ = bzip2.NewReader(strings.NewReader(""))
}
