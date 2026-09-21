#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
VERSION=${VERSION:-2026.9.21}
ARCH=${ARCH:-x86_64}
OUT="$ROOT/dist"
BUNDLE="$ROOT/build/linux/x64/release/bundle"

rm -rf "$OUT"
mkdir -p "$OUT"

flutter build linux --release

tar -C "$BUNDLE" -czf "$OUT/supernova-desktop-linux-${ARCH}.tar.gz" .

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

mkdir -p   "$STAGE/opt/supernova-desktop"   "$STAGE/usr/bin"   "$STAGE/usr/share/applications"   "$STAGE/usr/share/icons/hicolor/512x512/apps"

cp -a "$BUNDLE/." "$STAGE/opt/supernova-desktop/"
ln -s /opt/supernova-desktop/supernova-desktop "$STAGE/usr/bin/supernova-desktop"
cp "$ROOT/packaging/com.soltros.Supernova.desktop" "$STAGE/usr/share/applications/"
cp "$ROOT/assets/icon.png"   "$STAGE/usr/share/icons/hicolor/512x512/apps/com.soltros.Supernova.png"

mkdir -p "$STAGE/DEBIAN"
cat > "$STAGE/DEBIAN/control" <<EOF
Package: supernova-desktop
Version: $VERSION
Section: sound
Priority: optional
Architecture: amd64
Depends: libgtk-3-0, libsecret-1-0
Maintainer: Supernova <https://github.com/soltros/Supernova>
Description: Native Flutter desktop client for Supernova
 A desktop music player for self-hosted Supernova servers.
EOF

dpkg-deb --build --root-owner-group   "$STAGE" "$OUT/supernova-desktop_${VERSION}_amd64.deb"

if command -v fpm >/dev/null 2>&1; then
  rm -rf "$STAGE/DEBIAN"
  fpm     -s dir     -t rpm     -n supernova-desktop     -v "$VERSION"     -a x86_64     --description "Native Flutter desktop client for Supernova"     --license GPL-3.0-only     --depends gtk3     --depends libsecret     -C "$STAGE"     -p "$OUT/supernova-desktop-${VERSION}-1.x86_64.rpm"     opt usr
fi
