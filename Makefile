APP    ?= hello
IMAGE  ?= go-pocketbook-base-sdk
DIST   := dist
PKG    := go-pocketbook-base/internal/version

.PHONY: all image app dist preview test sonar task-done major clean
all: dist

image:            ## build the SDK + Go toolchain docker image
	docker build -t $(IMAGE) docker

# Every build increments BUILD in .version and embeds the version in the binary.
app: image        ## cross-compile the .app (linux/arm, cgo + libinkview)
	v=$$(go run ./tools/bumpversion build) && echo "Version: $$v" && \
	docker run --rm -v "$(CURDIR)":/app -v pbgo-cache:/gopath $(IMAGE) \
		build -ldflags=-X=$(PKG).Value=$$v -o $(DIST)/applications/$(APP).app ./cmd/$(APP)

dist: app         ## single file to copy to the device: dist/applications/$(APP).app

task-done:        ## a task was finished: MINOR+1 (BUILD restarts at 0)
	go run ./tools/bumpversion minor

major:            ## a big change: MAJOR+1 (MINOR and BUILD restart at 0)
	go run ./tools/bumpversion major

test:             ## unit tests (desktop)
	go test ./...

preview:          ## render every demo page to preview-<n>.png (no device needed)
	go run ./cmd/preview

sonar:            ## static analysis against the local SonarQube (needs SONAR_GLOBAL_TOKEN)
	./scripts/sonar.sh

clean:
	rm -rf $(DIST) preview*.png
