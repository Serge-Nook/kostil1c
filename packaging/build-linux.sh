#!/usr/bin/env bash
# Builds the Linux binary, kostil1c_<version>_amd64.deb (Debian/Ubuntu/Astra Linux)
# and Kostil1C-<version>-x86_64.AppImage (SteamOS and others) into dist/.
#
# Run it on an old distribution (Debian 10 "buster", glibc 2.28 — the base of
# Astra Linux 1.7) so the binary also starts on Astra 1.7/1.8 and SteamOS.
# packaging/release.sh does that inside a debian:buster Docker container.
#
# Requirements: go, gcc, pkg-config, libgl1-mesa-dev, xorg-dev, dpkg-deb, curl.
# GLFW is built X11-only (-tags x11): its Wayland backend needs libwayland >= 1.20,
# newer than Debian 10 ships; Astra (Fly) and SteamOS run X11/XWayland.
set -euo pipefail

VERSION="${VERSION:-$(sed -n 's/.*appVersion *= *"\(.*\)".*/\1/p' "$(dirname "$0")/../main.go")}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="${ROOT}/dist"
TOOLS="${DIST}/tools"
mkdir -p "${DIST}" "${TOOLS}"

cd "${ROOT}"
CGO_ENABLED=1 go build -tags x11 -trimpath -ldflags '-s -w' -o "${DIST}/kostil1c" .

install_tree() {
  local dest="$1"
  install -Dm755 "${DIST}/kostil1c" "${dest}/usr/bin/kostil1c"
  install -Dm644 packaging/kostil1c.desktop "${dest}/usr/share/applications/kostil1c.desktop"
  for s in 16 24 32 48 64 128 256 512; do
    install -Dm644 "build/icons/${s}x${s}.png" "${dest}/usr/share/icons/hicolor/${s}x${s}/apps/kostil1c.png"
  done
  install -Dm644 build/icon.svg "${dest}/usr/share/icons/hicolor/scalable/apps/kostil1c.svg"
}

# --- .deb ---
PKG="${DIST}/deb/kostil1c_${VERSION}_amd64"
rm -rf "${PKG}"
mkdir -p "${PKG}/DEBIAN"
install_tree "${PKG}"
install -Dm644 README.md "${PKG}/usr/share/doc/kostil1c/README.md"
install -Dm644 LICENSE "${PKG}/usr/share/doc/kostil1c/copyright"
SIZE="$(du -sk "${PKG}/usr" | cut -f1)"
cat > "${PKG}/DEBIAN/control" <<CTRL
Package: kostil1c
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Installed-Size: ${SIZE}
Maintainer: Горшков Сергей Владимирович <nookbat@gmail.com>
Homepage: https://nookbat.ru/
Depends: libc6 (>= 2.28), libgl1, libx11-6, libxcursor1, libxrandr2, libxinerama1, libxi6, libxxf86vm1, libxkbcommon0
Description: Split a 1C CSV export into per-department tables
 «Костыль 1С» loads a CSV table exported from 1C (separator «;», UTF-8),
 lists the departments (3rd column) with the number of employees and
 exports every selected department into a separate CSV, XLSX, PDF or
 HTML file that keeps the original header.
CTRL
dpkg-deb --root-owner-group -Zxz --build "${PKG}" "${DIST}/kostil1c_${VERSION}_amd64.deb" >/dev/null
echo "Built ${DIST}/kostil1c_${VERSION}_amd64.deb"

# --- AppImage ---
if [ ! -x "${TOOLS}/appimagetool/AppRun" ]; then
  curl -fsSL -o "${TOOLS}/appimagetool.AppImage" \
    https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
  chmod +x "${TOOLS}/appimagetool.AppImage"
  (cd "${TOOLS}" && ./appimagetool.AppImage --appimage-extract >/dev/null && rm -rf appimagetool && mv squashfs-root appimagetool && rm appimagetool.AppImage)
fi
APPDIR="${DIST}/AppDir"
rm -rf "${APPDIR}"
install_tree "${APPDIR}"
cp packaging/kostil1c.desktop "${APPDIR}/kostil1c.desktop"
cp build/icon.png "${APPDIR}/kostil1c.png"
cat > "${APPDIR}/AppRun" <<'RUN'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
exec "${HERE}/usr/bin/kostil1c" "$@"
RUN
chmod +x "${APPDIR}/AppRun"
ARCH=x86_64 "${TOOLS}/appimagetool/AppRun" --no-appstream "${APPDIR}" "${DIST}/Kostil1C-${VERSION}-x86_64.AppImage" >/dev/null
echo "Built ${DIST}/Kostil1C-${VERSION}-x86_64.AppImage"
