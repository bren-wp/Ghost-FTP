#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '\r\n' < VERSION)"
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid VERSION: $VERSION" >&2
  exit 1
fi
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"
for part in "$MAJOR" "$MINOR" "$PATCH"; do
  (( part >= 0 && part <= 999 )) || { echo 'VERSION components must be between 0 and 999.' >&2; exit 1; }
done

# GhostFTP uses a new bundle identity. CFBundleVersion is derived directly from
# the canonical semantic version and contains no migration epoch or legacy floor.
BUILD_NUMBER=$((MAJOR * 1000000 + MINOR * 1000 + PATCH))
(( BUILD_NUMBER > 0 )) || { echo 'Ghost FTP iOS build number must be positive.' >&2; exit 1; }

PROJECT="$ROOT/ios/GhostFTP.xcodeproj"
SCHEME="GhostFTP"
BUILD_ROOT="$ROOT/dist/ios-build"
DERIVED_DATA="$BUILD_ROOT/DerivedData"
APP="$DERIVED_DATA/Build/Products/Release-iphoneos/GhostFTP.app"
ICON_SOURCE="$ROOT/build/icon.png"
ICON_DIR="$ROOT/ios/GhostFTP/Assets.xcassets/AppIcon.appiconset"

for required in "$PROJECT/project.pbxproj" "$ICON_SOURCE" "$ROOT/ios/GhostFTP/Info.plist"; do
  [[ -f "$required" ]] || { echo "Missing required iOS build input: $required" >&2; exit 1; }
done
for tool in xcodebuild xcrun sips python3 lipo; do
  command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }
done

rm -rf "$BUILD_ROOT"
mkdir -p "$BUILD_ROOT" "$ICON_DIR"

GENERATED_ICONS=(
  Icon-20@2x.png Icon-20@3x.png Icon-29@2x.png Icon-29@3x.png
  Icon-40@2x.png Icon-40@3x.png Icon-60@2x.png Icon-60@3x.png
  Icon-20@2x-ipad.png Icon-29@2x-ipad.png Icon-40@2x-ipad.png
  Icon-76@2x.png Icon-83.5@2x.png Icon-1024.png
)
cleanup_icons() {
  for name in "${GENERATED_ICONS[@]}"; do rm -f "$ICON_DIR/$name"; done
}
trap cleanup_icons EXIT

resize_icon() {
  local size="$1" name="$2"
  sips -z "$size" "$size" "$ICON_SOURCE" --out "$ICON_DIR/$name" >/dev/null
}
resize_icon 40 Icon-20@2x.png
resize_icon 60 Icon-20@3x.png
resize_icon 58 Icon-29@2x.png
resize_icon 87 Icon-29@3x.png
resize_icon 80 Icon-40@2x.png
resize_icon 120 Icon-40@3x.png
resize_icon 120 Icon-60@2x.png
resize_icon 180 Icon-60@3x.png
resize_icon 40 Icon-20@2x-ipad.png
resize_icon 58 Icon-29@2x-ipad.png
resize_icon 80 Icon-40@2x-ipad.png
resize_icon 152 Icon-76@2x.png
resize_icon 167 Icon-83.5@2x.png
resize_icon 1024 Icon-1024.png

printf '[1/4] iOS model and path regressions\n'
xcrun swiftc \
  "$ROOT/ios/GhostFTP/ConnectionConfig.swift" \
  "$ROOT/ios/GhostFTP/RemoteModels.swift" \
  "$ROOT/ios/Tests/ModelTests.swift" \
  -o "$BUILD_ROOT/model-tests"
"$BUILD_ROOT/model-tests"

printf '[2/4] Xcode project and scheme validation\n'
xcodebuild -list -project "$PROJECT" >/dev/null

printf '[3/4] Unsigned arm64 iPhoneOS Release build\n'
xcodebuild \
  -project "$PROJECT" \
  -scheme "$SCHEME" \
  -configuration Release \
  -sdk iphoneos \
  -destination 'generic/platform=iOS' \
  -derivedDataPath "$DERIVED_DATA" \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGN_IDENTITY='' \
  DEVELOPMENT_TEAM='' \
  PRODUCT_BUNDLE_IDENTIFIER='com.ghostftp.client' \
  MARKETING_VERSION="$VERSION" \
  CURRENT_PROJECT_VERSION="$BUILD_NUMBER" \
  ONLY_ACTIVE_ARCH=NO \
  ARCHS=arm64 \
  build

[[ -d "$APP" ]] || { echo "GhostFTP.app was not generated." >&2; exit 1; }
[[ -x "$APP/GhostFTP" ]] || { echo "GhostFTP executable is missing." >&2; exit 1; }
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$APP/Info.plist")" == 'com.ghostftp.client' ]]
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$APP/Info.plist")" == "$VERSION" ]]
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "$APP/Info.plist")" == "$BUILD_NUMBER" ]]
lipo -archs "$APP/GhostFTP" | tr ' ' '\n' | grep -qx 'arm64'

printf '[4/4] IPA and app bundle packaging\n'
python3 scripts/package_ios.py --app "$APP" --output-dir dist

echo "IOS_BUILD=PASS ($VERSION)"
echo "IOS_BUILD_NUMBER=$BUILD_NUMBER"
echo "IOS_DEVICE_ARCH=arm64"
echo "IOS_CODE_SIGNING=UNSIGNED_EXTERNAL_APPLE_IDENTITY_REQUIRED"
