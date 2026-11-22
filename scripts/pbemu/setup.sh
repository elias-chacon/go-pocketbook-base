#!/bin/sh
# Sets up the pbemu emulator (https://codeberg.org/datyoma/pbemu) for this repository, inside WSL or
# on Linux: it runs the real firmware of the PB741 (the libinkview, the home screen, the monitor) so an
# app built for the reader can be started and looked at without the device. Run by scripts/pbemu.ps1.
#
#   sh scripts/pbemu/setup.sh            # clone, download the firmware, build the image and the probes
#
# Everything goes to $PBEMU_WORK (default ~/pbemu-work), outside the repository: the firmware is
# PocketBook's and is never committed. Needs docker (the scripts translate pbemu's podman calls).
set -eu

WORK="${PBEMU_WORK:-$HOME/pbemu-work}"
HERE="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# The commit of pbemu this was made and tested with.
PBEMU_COMMIT="${PBEMU_COMMIT:-e68b8c76522e10d68fee1cfa536000767bd9d204}"
# The firmware of the PB741 (InkPad Color): the official download of PocketBook.
FW_NAME="${FW_NAME:-sw_20240906_U741_6.8.4079_user.zip}"
FW_URL="${FW_URL:-https://download.pocketbook-int.com/fw/Yitoa/741/user/20240906/$FW_NAME}"
FW_DIR="${FW_DIR:-U741_6.8.4079}"

mkdir -p "$WORK/bin"
tr -d '\r' < "$HERE/podman-shim.sh" > "$WORK/bin/podman"
chmod +x "$WORK/bin/podman"
tr -d '\r' < "$HERE/pbx.py" > "$WORK/pbx.py"
chmod +x "$WORK/pbx.py"
export PODMAN="$WORK/bin/podman"

if [ ! -d "$WORK/pbemu/.git" ]; then
    git clone -q https://codeberg.org/datyoma/pbemu "$WORK/pbemu"
fi
git -C "$WORK/pbemu" fetch -q --depth 1 origin "$PBEMU_COMMIT" 2>/dev/null || true
git -C "$WORK/pbemu" checkout -q "$PBEMU_COMMIT"

if [ ! -f "$WORK/$FW_NAME" ]; then
    echo "downloading the firmware: $FW_URL"
    curl -fL -o "$WORK/$FW_NAME.part" "$FW_URL"
    mv "$WORK/$FW_NAME.part" "$WORK/$FW_NAME"
fi

cd "$WORK/pbemu"
./pbemu image
[ -d "$FW_DIR/rootfs" ] || ./pbemu install "../$FW_NAME" < /dev/null
tr -d '\r' < "$HERE/fb_pgm.c" > src/probes/host/src/fb_pgm.c
./pbemu build "$FW_DIR" < /dev/null
docker run --rm -v "$PWD/src:/workspace/src" pbdev sh -c \
    "clang -O1 -I/workspace/src/common -I/workspace/src/pbcore/include /workspace/src/probes/host/src/fb_pgm.c /workspace/src/pbcore/src/frame_source.c -o /workspace/src/viewer/build-pc/fb_pgm"
echo "pbemu ready in $WORK ($FW_DIR)"
