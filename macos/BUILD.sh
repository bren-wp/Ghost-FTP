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
SITE_MANAGER_SOURCE="$SCRIPT_DIR/Sources/GhostFTPApp/SiteManager.swift"
APPLICATION_WINDOWS_SOURCE="$SCRIPT_DIR/Sources/GhostFTPApp/ApplicationWindows.swift"
PREPARE_SITE_MANAGER_SOURCES="$SCRIPT_DIR/prepare_site_manager_sources.py"
BRIDGE_SOURCE="$SCRIPT_DIR/Bridge/main.go"
APPLICATION_BRIDGE_SOURCE="$SCRIPT_DIR/Bridge/application.go"
ASKPASS_SOURCE="$SCRIPT_DIR/AskPass/main.go"
ICON_SOURCE="$REPO_ROOT/build/icon.png"
OUT="$SCRIPT_DIR/out"
DIST="$SCRIPT_DIR/dist"
APP="$OUT/Ghost FTP.app"
CONTENTS="$APP/Contents"
MACOS="$CONTENTS/MacOS"
FRAMEWORKS="$CONTENTS/Frameworks"
RESOURCES="$CONTENTS/Resources"
MODULE_DIR="$OUT/GhostFTPEngineModule"
GENERATED_SOURCE_DIR="$OUT/generated-swift"
GENERATED_SOURCE="$GENERATED_SOURCE_DIR/main.swift"
GENERATED_SITE_MANAGER_SOURCE="$GENERATED_SOURCE_DIR/SiteManager.swift"
ZIP="$DIST/Ghost-FTP-${VERSION}-macOS.app.zip"
SDK="$(xcrun --sdk macosx --show-sdk-path)"
DEPLOYMENT_TARGET="13.0"
BUNDLE_ID="app.ghostftp.client"

for required in "$SOURCE" "$SITE_MANAGER_SOURCE" "$APPLICATION_WINDOWS_SOURCE" "$PREPARE_SITE_MANAGER_SOURCES" "$BRIDGE_SOURCE" "$APPLICATION_BRIDGE_SOURCE" "$ASKPASS_SOURCE" "$ICON_SOURCE"; do
  if [[ ! -s "$required" ]]; then
    echo "Missing required macOS build input: $required" >&2
    exit 1
  fi
done

rm -rf "$OUT" "$DIST"
mkdir -p "$MACOS" "$FRAMEWORKS" "$RESOURCES" "$DIST" "$MODULE_DIR" "$GENERATED_SOURCE_DIR"

python3 "$PREPARE_SITE_MANAGER_SOURCES" \
  "$SOURCE" \
  "$SITE_MANAGER_SOURCE" \
  "$GENERATED_SOURCE" \
  "$GENERATED_SITE_MANAGER_SOURCE"
test -s "$GENERATED_SOURCE"
test -s "$GENERATED_SITE_MANAGER_SOURCE"
grep -F 'NSButton(title: "Site Manager"' "$GENERATED_SOURCE" >/dev/null
grep -F 'NSButton(title: "Bookmarks"' "$GENERATED_SOURCE" >/dev/null
grep -F 'NSButton(title: "Settings"' "$GENERATED_SOURCE" >/dev/null
grep -F 'NSButton(title: "About"' "$GENERATED_SOURCE" >/dev/null
grep -F 'NSButton(title: "Diagnostics"' "$GENERATED_SOURCE" >/dev/null
grep -F 'controller.onConnected' "$GENERATED_SOURCE" >/dev/null
grep -F 'bookmarkNavigationApplied' "$GENERATED_SOURCE" >/dev/null
grep -F 'settingsAppearanceChanged' "$GENERATED_SOURCE" >/dev/null
grep -F 'GhostFTPSettingsConfirmDelete() == 0' "$GENERATED_SOURCE" >/dev/null
grep -F 'var onConnected: ((String, String, String) -> Void)?' "$GENERATED_SITE_MANAGER_SOURCE" >/dev/null

build_go_arch() {
  local arch="$1"
  local clang_arch="$2"
  mkdir -p "$OUT/go-$arch"
  (
    cd "$REPO_ROOT"
    CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" \
      CGO_CFLAGS="-mmacosx-version-min=${DEPLOYMENT_TARGET} -arch ${clang_arch}" \
      CGO_LDFLAGS="-mmacosx-version-min=${DEPLOYMENT_TARGET} -arch ${clang_arch}" \
      go build -trimpath -buildmode=c-shared \
        -ldflags "-X main.productVersion=${VERSION}" \
        -o "$OUT/go-$arch/libGhostFTPEngine.dylib" ./macos/Bridge
    CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" \
      CGO_CFLAGS="-mmacosx-version-min=${DEPLOYMENT_TARGET} -arch ${clang_arch}" \
      CGO_LDFLAGS="-mmacosx-version-min=${DEPLOYMENT_TARGET} -arch ${clang_arch}" \
      go build -trimpath -o "$OUT/go-$arch/GhostFTPAskPass" ./macos/AskPass
  )
}

build_go_arch arm64 arm64
build_go_arch amd64 x86_64

xcrun lipo -create \
  "$OUT/go-arm64/libGhostFTPEngine.dylib" \
  "$OUT/go-amd64/libGhostFTPEngine.dylib" \
  -output "$FRAMEWORKS/libGhostFTPEngine.dylib"
xcrun install_name_tool -id '@rpath/libGhostFTPEngine.dylib' "$FRAMEWORKS/libGhostFTPEngine.dylib"

xcrun lipo -create \
  "$OUT/go-arm64/GhostFTPAskPass" \
  "$OUT/go-amd64/GhostFTPAskPass" \
  -output "$MACOS/GhostFTPAskPass"
chmod 0755 "$MACOS/GhostFTPAskPass"

cp "$OUT/go-arm64/libGhostFTPEngine.h" "$MODULE_DIR/GhostFTPEngine.h"
cat > "$MODULE_DIR/module.modulemap" <<'MODULEMAP'
module GhostFTPEngine [system] {
  header "GhostFTPEngine.h"
  export *
}
MODULEMAP

build_swift_arch() {
  local arch="$1"
  local target="$arch-apple-macosx${DEPLOYMENT_TARGET}"
  xcrun --sdk macosx swiftc \
    -sdk "$SDK" \
    -target "$target" \
    -O \
    -I "$MODULE_DIR" \
    -L "$FRAMEWORKS" \
    -lGhostFTPEngine \
    -Xlinker -rpath \
    -Xlinker '@executable_path/../Frameworks' \
    -framework AppKit \
    "$GENERATED_SOURCE" "$GENERATED_SITE_MANAGER_SOURCE" "$APPLICATION_WINDOWS_SOURCE" \
    -o "$OUT/GhostFTP-$arch"
}

build_swift_arch arm64
build_swift_arch x86_64
xcrun lipo -create "$OUT/GhostFTP-arm64" "$OUT/GhostFTP-x86_64" -output "$MACOS/GhostFTP"
chmod 0755 "$MACOS/GhostFTP"

for binary in "$MACOS/GhostFTP" "$MACOS/GhostFTPAskPass" "$FRAMEWORKS/libGhostFTPEngine.dylib"; do
  archs="$(xcrun lipo -archs "$binary")"
  [[ " $archs " == *" arm64 "* ]] || { echo "$binary is missing arm64" >&2; exit 1; }
  [[ " $archs " == *" x86_64 "* ]] || { echo "$binary is missing x86_64" >&2; exit 1; }
done

cat > "$CONTENTS/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key><string>en</string>
  <key>CFBundleDisplayName</key><string>Ghost FTP</string>
  <key>CFBundleExecutable</key><string>GhostFTP</string>
  <key>CFBundleIconFile</key><string>GhostFTP</string>
  <key>CFBundleIdentifier</key><string>${BUNDLE_ID}</string>
  <key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
  <key>CFBundleName</key><string>Ghost FTP</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>LSMinimumSystemVersion</key><string>${DEPLOYMENT_TARGET}</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF

ICONSET="$OUT/GhostFTP.iconset"
mkdir -p "$ICONSET"
make_icon() {
  local pixels="$1" name="$2"
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

# Sign nested executable code before the outer development bundle.
codesign --force --sign - "$FRAMEWORKS/libGhostFTPEngine.dylib"
codesign --force --sign - "$MACOS/GhostFTPAskPass"
codesign --force --sign - "$MACOS/GhostFTP"
codesign --force --sign - "$APP"
codesign --verify --deep --strict "$APP"

ditto -c -k --sequesterRsrc --keepParent "$APP" "$ZIP"
test -s "$ZIP"

printf 'MACOS_APP=%s\n' "$APP"
printf 'MACOS_DEVELOPMENT_ARTIFACT=%s\n' "$ZIP"
printf 'MACOS_VERSION=%s\n' "$VERSION"
printf 'MACOS_ARCHS=%s\n' "$(xcrun lipo -archs "$MACOS/GhostFTP")"
printf 'MACOS_ENGINE_ARCHS=%s\n' "$(xcrun lipo -archs "$FRAMEWORKS/libGhostFTPEngine.dylib")"
printf 'MACOS_ASKPASS_ARCHS=%s\n' "$(xcrun lipo -archs "$MACOS/GhostFTPAskPass")"
printf 'MACOS_SIGNING=adhoc-development\n'
