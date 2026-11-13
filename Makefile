APP    ?= hello
IMAGE  ?= go-pocketbook-base-sdk
DIST   := dist

.PHONY: all image app dist preview test clean
all: dist

image:            ## build the SDK + Go toolchain docker image
	docker build -t $(IMAGE) docker

app: image        ## cross-compile the .app (linux/arm, cgo + libinkview)
	docker run --rm -v "$(CURDIR)":/app -v pbgo-cache:/gopath $(IMAGE) \
		build -o $(DIST)/applications/$(APP).app ./cmd/$(APP)

dist: app         ## single file to copy to the device: dist/applications/$(APP).app

test:             ## unit tests (desktop)
	go test ./...

preview:          ## render UI to preview.png (no device needed)
	go run ./cmd/preview

clean:
	rm -rf $(DIST) preview.png
