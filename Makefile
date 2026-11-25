APP    ?= hello
IMAGE  ?= go-pocketbook-base-sdk
DIST   := dist
PKG    := go-pocketbook-base/internal/kernel/version

.PHONY: all image app dist dist-all preview test sonar task-done major clean
all: dist

image:            ## build the SDK + Go toolchain docker image
	docker build -t $(IMAGE) docker

# Every build increments BUILD in .version (the release); the app has its own apps/<id>/.version,
# bumped only when its sources changed, and embedded in the binary.
app: image        ## cross-compile the .app (linux/arm, cgo + libinkview)
	go run ./tools/bumpversion build >/dev/null && v=$$(go run ./tools/bumpversion -app $(APP) auto) && echo "$(APP) $$v" && \
	docker run --rm -v "$(CURDIR)":/app -v pbgo-cache:/gopath $(IMAGE) \
		build -trimpath '-ldflags=-s -w -buildid= -X=$(PKG).Value='$$v -o $(DIST)/applications/$(APP).app ./apps/$(APP)

dist: app         ## single file to copy to the device: dist/applications/$(APP).app

dist-all: image   ## every app of apps/ (one release bump, an own version per app): dist/applications/<id>.app
	go run ./tools/bumpversion build >/dev/null; \
	for a in $$(ls apps); do [ -f apps/$$a/main.go ] && v=$$(go run ./tools/bumpversion -app $$a auto) && echo "$$a $$v" && \
		docker run --rm -v "$(CURDIR)":/app -v pbgo-cache:/gopath $(IMAGE) \
		build -trimpath '-ldflags=-s -w -buildid= -X=$(PKG).Value='$$v -o $(DIST)/applications/$$a.app ./apps/$$a; done

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
