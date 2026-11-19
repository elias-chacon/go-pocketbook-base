#!/bin/sh
# Stage 2 (alpine amd64): run the ARM test binaries under qemu-arm (user mode), each from its
# package folder, in a container that has no zoneinfo and no system libraries beyond musl,
# like the reader. Run by scripts/arm-test.ps1; the repository is mounted at /src.
apk add --no-cache qemu-arm >/dev/null 2>&1
qemu-arm --version | head -1
pass=0
fail=0
while read -r rel out; do
  if (cd "/src/$rel" && qemu-arm "/src/$out" >/tmp/run.log 2>&1); then
    pass=$((pass + 1))
    echo "ok   $rel"
  else
    fail=$((fail + 1))
    echo "FAIL $rel"
    tail -15 /tmp/run.log
  fi
done </src/dist/armtests/list.txt
echo "ARM: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
