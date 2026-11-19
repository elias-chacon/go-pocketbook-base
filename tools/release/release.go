package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go-pocketbook-base/internal/launcher/infrastructure/github"
)

// manifestFile is the name of an app's manifest, in apps/<id>/ and inside its zip.
const manifestFile = "app.json"

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
// tar.gz (which keeps the executable bit) for the others. It has no version, so
// .../releases/latest/download/<archive> is always the newest installer.
func (t target) archive() string {
	ext := ".tar.gz"
	if t.OS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("pbgo-installer_%s_%s%s", t.OS, t.Arch, ext)
}

// notes is the text of the release: how to install, for the three systems, with
// the commands to paste. base is the download address of the release's files.
func notes(base string) string {
	archive := func(os, arch string) string { return base + "/" + target{os, arch}.archive() }
	return fmt.Sprintf(`## Instalação / Installation

Conecte o PocketBook ao computador por USB (modo *Link com PC* / *PC Link*) e execute **no seu sistema**:
Connect the PocketBook by USB (*PC Link* mode) and run **on your system**:

**Windows (PowerShell)**

`+"```powershell"+`
Invoke-WebRequest %[1]s -OutFile pbgo-installer.zip
Expand-Archive pbgo-installer.zip -DestinationPath pbgo-installer -Force
.\pbgo-installer\pbgo-installer.exe
`+"```"+`

**Linux**

`+"```sh"+`
curl -fsSL %[2]s | tar xz && ./pbgo-installer
`+"```"+`

**macOS (Intel / Apple silicon)**

`+"```sh"+`
arch=$(uname -m); [ "$arch" = x86_64 ] && arch=amd64
curl -fsSL %[3]s | tar xz && ./pbgo-installer
`+"```"+`

Ou baixe o arquivo do seu sistema abaixo e dê dois cliques. / Or download the file for your system below and double-click it.

- Windows: se aparecer *"O Windows protegeu o computador"*, clique em **Mais informações → Executar assim mesmo**.
- macOS: se o sistema bloquear, clique com o botão direito no arquivo → **Abrir**.
- Conferência / checksums: `+"`SHA256SUMS`"+`.

Os arquivos `+"`<id>.zip`"+` são os aplicativos para o leitor (o `+"`<id>.app`"+` e o `+"`app.json`"+` de cada um): o Launcher os lista pelo `+"`catalog.json`"+` e baixa daqui. Para instalar à mão, baixe o zip e copie o `+"`<id>.app`"+` para a pasta `+"`applications`"+` do leitor; o app se registra na primeira execução.
`,
		archive("windows", "amd64"), archive("linux", "amd64"),
		strings.Replace(archive("darwin", "amd64"), "darwin_amd64", "darwin_${arch}", 1))
}

// packageApp writes <outDir>/<id>.zip with the app (<id>.app, executable) and its
// app.json, and returns the catalog entry that describes it. Zip entries have a
// fixed time so the same build always gives the same bytes.
func packageApp(distApps, appsDir, outDir, id, version string) (github.IndexEntry, error) {
	binary, err := os.ReadFile(filepath.Join(distApps, id+".app"))
	if err != nil {
		return github.IndexEntry{}, err
	}
	manifest, err := os.ReadFile(filepath.Join(appsDir, id, manifestFile))
	if err != nil {
		return github.IndexEntry{}, fmt.Errorf("%s has no manifest: %w", id, err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range []struct {
		name string
		mode os.FileMode
		data []byte
	}{{id + ".app", 0o755, binary}, {manifestFile, 0o644, manifest}} {
		h := &zip.FileHeader{Name: f.name, Method: zip.Deflate}
		h.SetMode(f.mode)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return github.IndexEntry{}, err
		}
		if _, err := w.Write(f.data); err != nil {
			return github.IndexEntry{}, err
		}
	}
	if err := zw.Close(); err != nil {
		return github.IndexEntry{}, err
	}

	asset := github.AppAsset(id)
	if err := os.WriteFile(filepath.Join(outDir, asset), buf.Bytes(), 0o644); err != nil {
		return github.IndexEntry{}, err
	}
	sum := sha256.Sum256(buf.Bytes())
	return github.IndexEntry{
		ID: id, Version: version, Asset: asset, Size: int64(buf.Len()),
		SHA256: hex.EncodeToString(sum[:]), Manifest: json.RawMessage(manifest),
	}, nil
}

// writeCatalog writes catalog.json, the list of the apps of the release.
func writeCatalog(outDir string, entries []github.IndexEntry) error {
	data, err := json.MarshalIndent(github.Index{Apps: entries}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, github.CatalogAsset), append(data, '\n'), 0o644)
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
		manifest, err := os.ReadFile(filepath.Join(appsDir, id, manifestFile))
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
