#!/bin/sh
# Stage 1 (golang:1.17, the SDK's Go): compile the tests of every pure-Go package for ARMv7.
# Run by scripts/arm-test.ps1; the repository is mounted at /src.
cd /src || exit 1
go version
mkdir -p dist/armtests
rm -f dist/armtests/*
: > dist/armtests/list.txt
for pkg in $(CGO_ENABLED=0 go list ./internal/... 2>/dev/null); do
  rel=${pkg#go-pocketbook-base/}
  case "$rel" in internal/device/inkview) continue ;; esac # needs libinkview: it only runs on the reader
  out=dist/armtests/$(echo "$rel" | tr '/' '_').test
  if CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go test -c -o "$out" "./$rel" >/tmp/build.log 2>&1; then
    [ -f "$out" ] && echo "$rel $out" >>dist/armtests/list.txt
  elif ! grep -q "no test files" /tmp/build.log; then
    echo "BUILD FAIL $rel"
    head -5 /tmp/build.log
  fi
done
echo "compiled: $(wc -l <dist/armtests/list.txt) test binaries"
