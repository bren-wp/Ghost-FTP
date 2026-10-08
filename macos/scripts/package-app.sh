#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

VERSION="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' version.json | head -n1)"
test -n "$VERSION"

swift build --package-path macos -c release

APP_ROOT="$ROOT/dist/macos/Ghost FTP.app"
CONTENTS="$APP_ROOT/Contents"
MACOS_DIR="$CONTENTS/MacOS"
RESOURCES_DIR="$CONTENTS/Resources"

rm -rf "$ROOT/dist/macos"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

BINARY="$ROOT/macos/.build/release/GhostFTPMacApp"
test -x "$BINARY"
cp "$BINARY" "$MACOS_DIR/GhostFTP"
chmod +x "$MACOS_DIR/GhostFTP"


# macOS Finder/Dock artwork uses the exact supplied SVG, not a generic system glyph.
# Rasterization runs on the packaging host, keeping source assets auditable.
ICON_SOURCE="$ROOT/assets/branding/ghostftp-app-icon.svg"
test -s "$ICON_SOURCE"
command -v rsvg-convert >/dev/null 2>&1
ICONSET="$ROOT/dist/macos/GhostFTP.iconset"
MASTER="$ROOT/dist/macos/GhostFTP-master.png"
mkdir -p "$ICONSET"
rsvg-convert --width=1024 --height=1024 "$ICON_SOURCE" --output="$MASTER"
for size in 16 32 128 256 512; do
  sips --resampleHeightWidth "$size" "$size" "$MASTER" \
    --out "$ICONSET/icon_${size}x${size}.png" >/dev/null
  twice=$((size * 2))
  sips --resampleHeightWidth "$twice" "$twice" "$MASTER" \
    --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil --convert icns "$ICONSET" --output "$RESOURCES_DIR/GhostFTP.icns"
test -s "$RESOURCES_DIR/GhostFTP.icns"
rm -rf "$ICONSET" "$MASTER"

cat > "$CONTENTS/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleDisplayName</key>
  <string>Ghost FTP</string>
  <key>CFBundleExecutable</key>
  <string>GhostFTP</string>
  <key>CFBundleIconFile</key>
  <string>GhostFTP.icns</string>
  <key>CFBundleIdentifier</key>
  <string>com.brendigo.ghostftp.macos</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>Ghost FTP</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>$VERSION</string>
  <key>CFBundleVersion</key>
  <string>$VERSION</string>
  <key>LSApplicationCategoryType</key>
  <string>public.app-category.utilities</string>
  <key>LSMinimumSystemVersion</key>
  <string>13.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

plutil -lint "$CONTENTS/Info.plist"
codesign --force --deep --sign - "$APP_ROOT"
codesign --verify --deep --strict --verbose=2 "$APP_ROOT"

ZIP="$ROOT/dist/macos/GhostFTP-macOS-v$VERSION-Preview.zip"
ditto -c -k --sequesterRsrc --keepParent "$APP_ROOT" "$ZIP"
shasum -a 256 "$ZIP" > "$ROOT/dist/macos/GhostFTP-macOS-SHA256SUMS.txt"

test -s "$ZIP"
test -s "$ROOT/dist/macos/GhostFTP-macOS-SHA256SUMS.txt"

echo "Ghost FTP macOS preview bundle prepared."
