package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// target is one operating system and processor the installer is built for.
type target struct{ OS, Arch string }

// targets are the systems the installer runs on.
var targets = []target{
	{"windows", "amd64"},
	{"linux", "amd64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
}

// exe is the file name of the installer on a system.
func (t target) exe() string {
	if t.OS == "windows" {
		return "pbgo-installer.exe"
	}
	return "pbgo-installer"
}

// archive is the file name of the download of a system: a zip for Windows, a
// tar.gz (which keeps the executable bit) for the others.
func (t target) archive(version string) string {
	ext := ".tar.gz"
	if t.OS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("pbgo-installer_%s_%s_%s%s", version, t.OS, t.Arch, ext)
}

// stage fills the installer's payload folder with the built apps, their
// manifests and the version. It returns the ids it staged.
func stage(distApps, appsDir, payload, version string) ([]string, error) {
	built, err := filepath.Glob(filepath.Join(distApps, "*.app"))
	if err != nil || len(built) == 0 {
		return nil, fmt.Errorf("no built apps in %s (run .\\build.ps1 -All first)", distApps)
	}
	sort.Strings(built)
	var ids []string
	for _, file := range built {
		id := strings.TrimSuffix(filepath.Base(file), ".app")
		manifest, err := os.ReadFile(filepath.Join(appsDir, id, "app.json"))
		if err != nil {
			return nil, fmt.Errorf("%s has no manifest: %w", id, err)
		}
		if err := copyFile(file, filepath.Join(payload, id+".app")); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(payload, id+".json"), manifest, 0o644); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, os.WriteFile(filepath.Join(payload, "VERSION"), []byte(version+"\n"), 0o644)
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o755)
}

// writeArchive puts the executable at exePath, named name, in the download of t.
func writeArchive(path string, t target, name, exePath string) error {
	data, err := os.ReadFile(exePath)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if t.OS == "windows" {
		zw := zip.NewWriter(f)
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		return zw.Close()
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}); err != nil {
		return err
	}
	if _, err := io.Copy(tw, bytes.NewReader(data)); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// checksums writes SHA256SUMS for the files, in the format `sha256sum -c` reads.
func checksums(dir string, files []string) error {
	sort.Strings(files)
	var out strings.Builder
	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		fmt.Fprintf(&out, "%x  %s\n", sha256.Sum256(data), name)
	}
	return os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(out.String()), 0o644)
}
