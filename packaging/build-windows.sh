#!/usr/bin/env bash
# Cross-builds Windows 10/11 x64 binaries from Linux into dist/:
#   Kostil1C-<version>-portable.exe  — single executable, no installation
#   Kostil1C-<version>-Setup.exe     — NSIS installer (Start menu + desktop shortcut)
#
# Requirements: go, gcc-mingw-w64-x86-64, nsis (makensis).
# The binary is linked statically so it needs only system DLLs.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-$(sed -n 's/.*appVersion *= *"\(.*\)".*/\1/p' "${ROOT}/main.go")}"
DIST="${ROOT}/dist"
mkdir -p "${DIST}/windows"
cd "${ROOT}"

GOWINRES="$(go env GOPATH)/bin/go-winres"
[ -x "${GOWINRES}" ] || go install github.com/tc-hib/go-winres@v0.3.3
"${GOWINRES}" simply \
  --arch amd64 \
  --icon build/icon.png \
  --manifest gui \
  --product-name "Костыль 1С" \
  --file-description "Костыль 1С — разделение CSV из 1С по подразделениям" \
  --product-version "${VERSION}.0" \
  --file-version "${VERSION}.0" \
  --copyright "Горшков Сергей Владимирович, https://nookbat.ru/" \
  --original-filename kostil1c.exe \
  --out rsrc
trap 'rm -f "${ROOT}"/rsrc_windows_*.syso' EXIT

CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
  go build -trimpath -ldflags '-s -w -H windowsgui -extldflags=-static' -o "${DIST}/windows/kostil1c.exe" .
cp "${DIST}/windows/kostil1c.exe" "${DIST}/Kostil1C-${VERSION}-portable.exe"
echo "Built ${DIST}/Kostil1C-${VERSION}-portable.exe"

cp build/icon.ico "${DIST}/windows/kostil1c.ico"
makensis -V2 -DVERSION="${VERSION}" -DSRCDIR="${DIST}/windows" -DOUTFILE="${DIST}/Kostil1C-${VERSION}-Setup.exe" \
  packaging/windows/installer.nsi
echo "Built ${DIST}/Kostil1C-${VERSION}-Setup.exe"
