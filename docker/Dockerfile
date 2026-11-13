# Go toolchain + PocketBook SDK (clang cross compiler + libinkview) in one image.
# Base image packages the official SDK (pocketbook/SDK_6.3.0, branch 6.5, SDK-B288).
FROM 5keeve/pocketbook-sdk:6.3.0-b288-v1

# Debian stretch is EOL: use the archive mirrors.
RUN printf 'deb http://archive.debian.org/debian stretch main\n' > /etc/apt/sources.list \
 && rm -f /etc/apt/sources.list.d/* \
 && apt-get -o Acquire::Check-Valid-Until=false update \
 && apt-get install -y --allow-unauthenticated curl ca-certificates git \
 && apt-get clean

# Go 1.17 is the last release known to work with this old SDK linker.
# cmd/go adds -Wl,--build-id=none, which the SDK's linker rejects: patch it out
# and rebuild cmd/go (same approach as github.com/dennwc/inkview).
ARG GO_VERSION=1.17.13
RUN curl -fsSL https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar xz -C / \
 && cd /go/src/cmd/go \
 && sed -i 's|^\(\s*\)ldflags = append(ldflags, "-Wl,--build-id=none")|\1// &|' internal/work/exec.go \
 && grep -n 'build-id=none' internal/work/exec.go \
 && /go/bin/go build -o /go/bin/go . 

ENV GOROOT=/go GOPATH=/gopath \
    PATH="/go/bin:${SDK_BASE}/usr/bin:${PATH}" \
    CC=${SDK_BASE}/usr/bin/arm-obreey-linux-gnueabi-clang \
    GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1

WORKDIR /app
ENTRYPOINT ["/go/bin/go"]
CMD ["build"]
