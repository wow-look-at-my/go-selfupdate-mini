package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// semverPattern for matching version strings inside archive filenames.
var semverPattern = `(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`

// builtinDecompressors returns the default decompressor registry (stdlib only).
func builtinDecompressors(os, arch string) map[string]Decompressor {
	return map[string]Decompressor{
		".zip":    DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) { return unzip(src, cmd, os, arch) }),
		".tar.gz": DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) { return untar(src, cmd, os, arch) }),
		".tgz":    DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) { return untar(src, cmd, os, arch) }),
		".gzip":   DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) { return gunzip(src, cmd, os, arch) }),
		".gz":     DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) { return gunzip(src, cmd, os, arch) }),
		".bz2":    DecompressorFunc(func(src io.Reader, cmd string) (io.Reader, error) { return unbz2(src, cmd) }),
	}
}

// decompressCommand decompresses the given source using the decompressor
// registry. Format is detected from the asset filename extension. If no
// matching decompressor is found, src is returned as-is.
func decompressCommand(src io.Reader, assetName, cmd string, decompressors map[string]Decompressor) (io.Reader, error) {
	// Check extensions longest-first to match .tar.gz before .gz
	for _, ext := range sortedExtensions(decompressors) {
		if strings.HasSuffix(assetName, ext) {
			return decompressors[ext].Decompress(src, cmd)
		}
	}
	log.Print("File is not compressed")
	return src, nil
}

// sortedExtensions returns extensions sorted longest first so .tar.gz matches before .gz.
func sortedExtensions(m map[string]Decompressor) []string {
	exts := make([]string, 0, len(m))
	for ext := range m {
		exts = append(exts, ext)
	}
	// simple insertion sort, longest first
	for i := 1; i < len(exts); i++ {
		for j := i; j > 0 && len(exts[j]) > len(exts[j-1]); j-- {
			exts[j], exts[j-1] = exts[j-1], exts[j]
		}
	}
	return exts
}

func unzip(src io.Reader, cmd, os, arch string) (io.Reader, error) {
	log.Print("Decompressing zip file")
	buf, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("%w zip file: %v", ErrCannotDecompressFile, err)
	}

	r := bytes.NewReader(buf)
	z, err := zip.NewReader(r, r.Size())
	if err != nil {
		return nil, fmt.Errorf("%w zip file: %s", ErrCannotDecompressFile, err)
	}

	for _, file := range z.File {
		_, name := filepath.Split(file.Name)
		if !file.FileInfo().IsDir() && matchExecutableName(cmd, os, arch, name) {
			log.Printf("Executable file %q was found in zip archive", file.Name)
			return file.Open()
		}
	}
	return nil, fmt.Errorf("%w in zip file: %q", ErrExecutableNotFoundInArchive, cmd)
}

func untar(src io.Reader, cmd, os, arch string) (io.Reader, error) {
	log.Print("Decompressing tar.gz file")
	gz, err := gzip.NewReader(src)
	if err != nil {
		return nil, fmt.Errorf("%w tar.gz file: %s", ErrCannotDecompressFile, err)
	}
	return unarchiveTar(gz, cmd, os, arch)
}

func gunzip(src io.Reader, cmd, os, arch string) (io.Reader, error) {
	log.Print("Decompressing gzip file")
	r, err := gzip.NewReader(src)
	if err != nil {
		return nil, fmt.Errorf("%w gzip file: %s", ErrCannotDecompressFile, err)
	}
	name := r.Header.Name
	if !matchExecutableName(cmd, os, arch, name) {
		return nil, fmt.Errorf("%w: expected %q but found %q", ErrExecutableNotFoundInArchive, cmd, name)
	}
	log.Printf("Executable file %q was found in gzip file", name)
	return r, nil
}

func unbz2(src io.Reader, cmd string) (io.Reader, error) {
	log.Print("Decompressing bzip2 file")
	log.Printf("Decompressed file from bzip2 is assumed to be an executable: %s", cmd)
	return bzip2.NewReader(src), nil
}

func matchExecutableName(cmd, os, arch, target string) bool {
	cmd = strings.TrimSuffix(cmd, ".exe")
	pattern := regexp.MustCompile(
		fmt.Sprintf(
			`^%s([_-]v?%s)?([_-]%s[_-]%s)?(\.exe)?$`,
			regexp.QuoteMeta(cmd),
			semverPattern,
			regexp.QuoteMeta(os),
			regexp.QuoteMeta(arch),
		),
	)
	return pattern.MatchString(target)
}

func unarchiveTar(src io.Reader, cmd, os, arch string) (io.Reader, error) {
	t := tar.NewReader(src)
	for {
		h, err := t.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w tar file: %s", ErrCannotDecompressFile, err)
		}
		_, name := filepath.Split(h.Name)
		if matchExecutableName(cmd, os, arch, name) {
			log.Printf("Executable file %q was found in tar archive", h.Name)
			return t, nil
		}
	}
	return nil, fmt.Errorf("%w in tar: %q", ErrExecutableNotFoundInArchive, cmd)
}
