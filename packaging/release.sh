#!/usr/bin/env bash
# Builds every release artifact into dist/:
#   Linux (.deb, .AppImage) inside a debian:buster container (glibc 2.28, Astra 1.7 base),
#   Windows (.exe) on the host with MinGW + NSIS. Writes dist/SHA256SUMS.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOROOT_HOST="$(go env GOROOT)"
MODCACHE="$(go env GOMODCACHE)"
cd "${ROOT}"
rm -rf dist && mkdir -p dist

docker run --rm \
  -v "${ROOT}:/src" -v "${GOROOT_HOST}:/usr/local/go:ro" -v "${MODCACHE}:/gomod" \
  -e GOMODCACHE=/gomod -e GOFLAGS=-buildvcs=false -e HOME=/tmp -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
  -w /src debian:buster bash -euc '
    printf "%s\n" \
      "deb http://archive.debian.org/debian buster main" \
      "deb http://archive.debian.org/debian buster-updates main" \
      "deb http://archive.debian.org/debian-security buster/updates main" > /etc/apt/sources.list
    apt-get -qq -o Acquire::Check-Valid-Until=false update
    DEBIAN_FRONTEND=noninteractive apt-get -q install -y --no-install-recommends \
      gcc libc6-dev pkg-config libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libxext-dev libwayland-dev libxkbcommon-dev libegl1-mesa-dev libgles2-mesa-dev curl ca-certificates file | tail -3
    export PATH=/usr/local/go/bin:$PATH
    ./packaging/build-linux.sh
    chown -R "$HOST_UID:$HOST_GID" dist
  '

./packaging/build-windows.sh

cd dist
sha256sum Kostil1C-*.AppImage kostil1c_*.deb Kostil1C-*.exe > SHA256SUMS
cat SHA256SUMS
