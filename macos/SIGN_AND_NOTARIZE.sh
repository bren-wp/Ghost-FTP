#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION="$(tr -d '\r\n' < "$REPO_ROOT/VERSION")"
APP="$SCRIPT_DIR/out/Ghost FTP.app"
DIST="$SCRIPT_DIR/dist"
SUBMISSION_ZIP="$DIST/Ghost-FTP-${VERSION}-macOS-notary-submission.zip"
FINAL_ZIP="$DIST/Ghost-FTP-${VERSION}-macOS-notarized.app.zip"
IDENTITY="${MACOS_DEVELOPER_IDENTITY:-}"
NOTARY_PROFILE="${MACOS_NOTARY_KEYCHAIN_PROFILE:-}"

fail() {
  printf 'MACOS_PRODUCTION_SIGNING_FAILED: %s\n' "$*" >&2
  exit 1
}

[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "invalid root VERSION: $VERSION"
[[ -n "$IDENTITY" ]] || fail "MACOS_DEVELOPER_IDENTITY is required"
[[ "$IDENTITY" == Developer\ ID\ Application:* ]] || fail "MACOS_DEVELOPER_IDENTITY must be a Developer ID Application identity"
[[ -n "$NOTARY_PROFILE" ]] || fail "MACOS_NOTARY_KEYCHAIN_PROFILE is required"

for tool in codesign security ditto xcrun spctl; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool is unavailable: $tool"
done

# The notary profile and signing identity must already be provisioned in the
# current macOS user's Keychain. This script never accepts raw certificate,
# private-key, Apple ID password, app-specific password or API-key material.
security find-identity -v -p codesigning | grep -F -- "$IDENTITY" >/dev/null \
  || fail "Developer ID Application identity is not available in the current Keychain"

# Build the already-tested universal development bundle first, then replace
# every ad-hoc signature explicitly from the inside out. Do not use --deep for
# signing because it can hide an incomplete nested-code inventory.
bash "$SCRIPT_DIR/BUILD.sh"
[[ -d "$APP" ]] || fail "development app bundle was not produced"

ENGINE="$APP/Contents/Frameworks/libGhostFTPEngine.dylib"
ASKPASS="$APP/Contents/MacOS/GhostFTPAskPass"
EXECUTABLE="$APP/Contents/MacOS/GhostFTP"
for file in "$ENGINE" "$ASKPASS" "$EXECUTABLE"; do
  [[ -s "$file" ]] || fail "missing nested code: $file"
done

sign_one() {
  local target="$1"
  codesign \
    --force \
    --sign "$IDENTITY" \
    --options runtime \
    --timestamp \
    "$target"
}

sign_one "$ENGINE"
sign_one "$ASKPASS"
sign_one "$EXECUTABLE"
sign_one "$APP"

codesign --verify --deep --strict --verbose=2 "$APP"
for target in "$ENGINE" "$ASKPASS" "$EXECUTABLE" "$APP"; do
  details="$(codesign --display --verbose=4 "$target" 2>&1)"
  grep -F 'runtime' <<<"$details" >/dev/null || fail "Hardened Runtime missing from $target"
  grep -F 'Timestamp=' <<<"$details" >/dev/null || fail "secure timestamp missing from $target"
done
if codesign --display --entitlements :- "$APP" 2>/dev/null | grep -F 'com.apple.security.get-task-allow' >/dev/null; then
  fail "distribution app must not contain com.apple.security.get-task-allow"
fi

rm -f "$SUBMISSION_ZIP" "$FINAL_ZIP"
ditto -c -k --sequesterRsrc --keepParent "$APP" "$SUBMISSION_ZIP"
[[ -s "$SUBMISSION_ZIP" ]] || fail "notary submission archive was not produced"

# The credentials referenced by this profile are intentionally outside the
# repository and process arguments. The Apple notary service result must be
# accepted before any final distribution artifact is emitted.
xcrun notarytool submit "$SUBMISSION_ZIP" \
  --keychain-profile "$NOTARY_PROFILE" \
  --wait

xcrun stapler staple "$APP"
xcrun stapler validate "$APP"
codesign --verify --deep --strict --verbose=2 "$APP"
spctl --assess --type execute --verbose=4 "$APP"

ditto -c -k --sequesterRsrc --keepParent "$APP" "$FINAL_ZIP"
[[ -s "$FINAL_ZIP" ]] || fail "final notarized archive was not produced"
rm -f "$SUBMISSION_ZIP"

printf 'MACOS_PRODUCTION_APP=%s\n' "$APP"
printf 'MACOS_PRODUCTION_ARTIFACT=%s\n' "$FINAL_ZIP"
printf 'MACOS_PRODUCTION_VERSION=%s\n' "$VERSION"
printf 'MACOS_PRODUCTION_SIGNING=developer-id-hardened-runtime\n'
printf 'MACOS_PRODUCTION_NOTARIZATION=accepted-stapled\n'
