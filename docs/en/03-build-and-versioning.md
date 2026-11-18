# 3. Build and versioning

[← Installation](02-installation.md) · [Index](README.md) · [Next: Concepts →](04-concepts.md)

## Building the app

The reader runs 32-bit ARM Linux and the UI library talks to the PocketBook SDK (`libinkview`) through
cgo, so the app is built **inside a Docker image** that brings the SDK's cross compiler and a patched
Go 1.17. You never install the SDK by hand.

### One command

```powershell
.\build.ps1                 # Windows: builds ./apps/hello
.\build.ps1 -App myapp      # builds ./apps/myapp
.\build.ps1 -All            # builds every app of ./apps (one version bump)
```
```sh
make dist                   # Linux / macOS / WSL
make dist APP=myapp
```

The result is **one file**: `dist/applications/<app>.app`. The script also:

- picks Docker Desktop, or Docker inside **WSL** when `docker` is not an executable on Windows;
- builds the image (`docker/Dockerfile`) the first time (later builds reuse the cache);
- **increments the build number** and embeds the version in the binary;
- fails loudly if the `.app` was not produced (it never reports a stale file as success).

### What the Docker image contains

| Piece | Why |
|---|---|
| PocketBook SDK 6.3.0 (branch `6.5`, `SDK-B288`) | cross compiler `arm-obreey-linux-gnueabi-clang`, headers and `libinkview` |
| Go 1.17.13 with `cmd/go` patched | the SDK linker rejects `-Wl,--build-id=none`, which `cmd/go` adds |
| `GOARCH=arm GOARM=7 CGO_ENABLED=1` | the target of the reader |

Build flags used internally (for reference):

```sh
go build -ldflags=-X=go-pocketbook-base/internal/kernel/version.Value=<version> -o dist/applications/hello.app ./apps/hello
```

### Make targets

| Target | What it does |
|---|---|
| `make dist` | builds the image and the `.app` (increments BUILD) |
| `make test` | `go test ./...` |
| `make preview` | renders the demo pages to PNG |
| `make task-done` | **MINOR + 1** (see below) |
| `make major` | **MAJOR + 1** |
| `make sonar` | static analysis (see below) |
| `make clean` | removes `dist/` and preview images |

## Versioning

The version lives in the file [`.version`](../../.version), in the format **`MAJOR.MINOR.BUILD`**, and
starts at `1.0.0`.

| When | What goes up | How |
|---|---|---|
| every **build** | `BUILD` | automatic in `build.ps1` / `make dist` |
| a **task is finished** | `MINOR` (and `BUILD` goes back to 0) | `make task-done` |
| a **big change** | `MAJOR` (`MINOR` and `BUILD` go back to 0) | `make major` |

```sh
go run ./tools/bumpversion build    # 1.2.3 -> 1.2.4
go run ./tools/bumpversion minor    # 1.2.4 -> 1.3.0
go run ./tools/bumpversion major    # 1.3.0 -> 2.0.0
```

The tool prints the new version. The build embeds it in the binary (`internal/kernel/version.Value`) and the
demo shows it in the footer, so you always know which build is on the reader.

In your own code:

```go
import "go-pocketbook-base/internal/kernel/version"

fmt.Println("running v" + version.Value) // "dev" when not built with the scripts
```

## Tests

```sh
go test ./...                        # all packages
go test -cover ./...                 # with coverage
go test ./internal/toolkit/widget -run Layout -v
```

The suite covers the layout engine, every component, touch accuracy (the center of every interactive
element must receive its own touch), the versioning rules, the installer and all the documentation
samples.

## Documentation tools

```sh
go run ./tools/docgen       # redraws docs/components/*.png from the real widgets
go run ./tools/docsync      # fills the code blocks of docs/ from examples/
go run ./tools/docsync -check   # only checks (the test suite runs this too)
```

To put a tested sample in a page, write a fenced `go` block whose **first line** is a comment naming
the source, leave the rest empty and run `docsync`:

- `// examples/recipes/login.go` embeds the whole file;
- `// examples/components/button.go#ButtonExample` embeds one function (with its doc comment).

The block is filled (and later refreshed) with the real code. Pages with a stale block make
`go test ./tools/docsync` fail, so documentation and code cannot drift apart.

## Static analysis (SonarQube)

With a SonarQube server on `http://localhost:9000` and a token in the `SONAR_GLOBAL_TOKEN` environment
variable:

```powershell
.\scripts\sonar.ps1        # Windows: tests with coverage + scanner (Docker in WSL)
```
```sh
scripts/sonar.sh           # Linux / WSL
```

The rules live in `sonar-project.properties`. The goal is **0 code smells, bugs and vulnerabilities** and
a passing quality gate.

## Continuous integration (suggestion)

```sh
go vet ./...
go test ./...
go run ./tools/docsync -check
make dist
```

Next: [Concepts →](04-concepts.md)
