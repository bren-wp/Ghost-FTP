#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION="$(tr -d '\r\n' < "$SCRIPT_DIR/../VERSION")"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid root VERSION: $VERSION" >&2
  exit 1
fi

SOURCE="$SCRIPT_DIR/Sources/GhostFTPApp/main.swift"
ICON_SOURCE="$REPO_ROOT/build/icon.png"
OUT="$SCRIPT_DIR/out"
DIST="$SCRIPT_DIR/dist"
APP="$OUT/Ghost FTP.app"
CONTENTS="$APP/Contents"
MACOS="$CONTENTS/MacOS"
RESOURCES="$CONTENTS/Resources"
ZIP="$DIST/Ghost-FTP-${VERSION}-macOS.app.zip"
SDK="$(xcrun --sdk macosx --show-sdk-path)"
DEPLOYMENT_TARGET="13.0"
BUNDLE_ID="app.ghostftp.client"

for required in "$SOURCE" "$ICON_SOURCE"; do
  if [[ ! -s "$required" ]]; then
    echo "Missing required macOS build input: $required" >&2
    exit 1
  fi
done

rm -rf "$OUT" "$DIST"
mkdir -p "$MACOS" "$RESOURCES" "$DIST"

build_arch() {
  local arch="$1"
  local target="$arch-apple-macosx${DEPLOYMENT_TARGET}"
  xcrun --sdk macosx swiftc \
    -sdk "$SDK" \
    -target "$target" \
    -O \
    -framework AppKit \
    "$SOURCE" \
    -o "$OUT/GhostFTP-$arch"
}

build_arch arm64
build_arch x86_64
xcrun lipo -create "$OUT/GhostFTP-arm64" "$OUT/GhostFTP-x86_64" -output "$MACOS/GhostFTP"
chmod 0755 "$MACOS/GhostFTP"

ARCHS="$(xcrun lipo -archs "$MACOS/GhostFTP")"
[[ " $ARCHS " == *" arm64 "* ]] || { echo "Universal app is missing arm64" >&2; exit 1; }
[[ " $ARCHS " == *" x86_64 "* ]] || { echo "Universal app is missing x86_64" >&2; exit 1; }

cat > "$CONTENTS/Info.plist" <<EOF
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
  <string>GhostFTP</string>
  <key>CFBundleIdentifier</key>
  <string>${BUNDLE_ID}</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>Ghost FTP</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>LSMinimumSystemVersion</key>
  <string>${DEPLOYMENT_TARGET}</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
EOF

ICONSET="$OUT/GhostFTP.iconset"
mkdir -p "$ICONSET"
make_icon() {
  local pixels="$1"
  local name="$2"
  sips -s format png -z "$pixels" "$pixels" "$ICON_SOURCE" --out "$ICONSET/$name" >/dev/null
}
make_icon 16 icon_16x16.png
make_icon 32 icon_16x16@2x.png
make_icon 32 icon_32x32.png
make_icon 64 icon_32x32@2x.png
make_icon 128 icon_128x128.png
make_icon 256 icon_128x128@2x.png
make_icon 256 icon_256x256.png
make_icon 512 icon_256x256@2x.png
make_icon 512 icon_512x512.png
make_icon 1024 icon_512x512@2x.png
iconutil -c icns "$ICONSET" -o "$RESOURCES/GhostFTP.icns"

/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$CONTENTS/Info.plist" | grep -Fx "$BUNDLE_ID" >/dev/null
/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$CONTENTS/Info.plist" | grep -Fx "$VERSION" >/dev/null
/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "$CONTENTS/Info.plist" | grep -Fx "$VERSION" >/dev/null

# Development CI uses ad-hoc signing only. A future public Mac release must use
# an explicitly configured Apple Developer identity plus hardened runtime and
# notarization; this build never fabricates a production identity.
codesign --force --deep --sign - "$APP"
codesign --verify --deep --strict "$APP"

ditto -c -k --sequesterRsrc --keepParent "$APP" "$ZIP"
test -s "$ZIP"

printf 'MACOS_APP=%s\n' "$APP"
printf 'MACOS_DEVELOPMENT_ARTIFACT=%s\n' "$ZIP"
printf 'MACOS_VERSION=%s\n' "$VERSION"
printf 'MACOS_ARCHS=%s\n' "$ARCHS"
printf 'MACOS_SIGNING=adhoc-development\n'
