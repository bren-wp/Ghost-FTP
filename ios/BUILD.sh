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
BUILD_NUMBER=$((MAJOR * 1000000 + MINOR * 1000 + PATCH))
if (( BUILD_NUMBER < 1 )); then BUILD_NUMBER=1; fi

PROJECT="$ROOT/ios/ByFTP.xcodeproj"
SCHEME="ByFTP"
BUILD_ROOT="$ROOT/dist/ios-build"
DERIVED_DATA="$BUILD_ROOT/DerivedData"
APP="$DERIVED_DATA/Build/Products/Release-iphoneos/ByFTP.app"
ICON_SOURCE="$ROOT/build/icon.png"
ICON_DIR="$ROOT/ios/ByFTP/Assets.xcassets/AppIcon.appiconset"

for required in "$PROJECT/project.pbxproj" "$ICON_SOURCE" "$ROOT/ios/ByFTP/Info.plist"; do
  [[ -f "$required" ]] || { echo "Missing required iOS build input: $required" >&2; exit 1; }
done
command -v xcodebuild >/dev/null
command -v xcrun >/dev/null
command -v sips >/dev/null
command -v python3 >/dev/null

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
  "$ROOT/ios/ByFTP/ConnectionConfig.swift" \
  "$ROOT/ios/ByFTP/RemoteModels.swift" \
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
  MARKETING_VERSION="$VERSION" \
  CURRENT_PROJECT_VERSION="$BUILD_NUMBER" \
  ONLY_ACTIVE_ARCH=NO \
  ARCHS=arm64 \
  build

[[ -d "$APP" ]] || { echo "ByFTP.app was not generated." >&2; exit 1; }
[[ -x "$APP/ByFTP" ]] || { echo "ByFTP executable is missing." >&2; exit 1; }
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$APP/Info.plist")" == 'com.byftp.client' ]]
[[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$APP/Info.plist")" == "$VERSION" ]]
lipo -archs "$APP/ByFTP" | tr ' ' '\n' | grep -qx 'arm64'

printf '[4/4] IPA and app bundle packaging\n'
python3 scripts/package_ios.py --app "$APP" --output-dir dist

echo "IOS_BUILD=PASS ($VERSION)"
echo "IOS_DEVICE_ARCH=arm64"
echo "IOS_CODE_SIGNING=UNSIGNED_EXTERNAL_APPLE_IDENTITY_REQUIRED"
