#!/bin/sh
# podman -> docker: the few podman-only forms that pbemu uses.
if [ "$1" = "image" ] && [ "$2" = "exists" ]; then
    exec docker image inspect "$3" >/dev/null 2>&1
fi
if [ "$1" = "container" ] && [ "$2" = "exists" ]; then
    exec docker container inspect "$3" >/dev/null 2>&1
fi
for a in "$@"; do
    shift
    case "$a" in
    --userns=keep-id) set -- "$@" --user "$(id -u):$(id -g)" ;;
    --init) set -- "$@" --init --tmpfs /sys:rw,mode=755 ;;
    *) set -- "$@" "$a" ;;
    esac
done
exec docker "$@"
