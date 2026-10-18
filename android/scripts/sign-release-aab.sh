#!/usr/bin/env bash
# Manually gated Android upload-key signing. Never upload an unsigned AAB to Play.
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

fail() { printf 'Android upload signing refused: %s\n' "$1" >&2; exit 1; }
for name in EXPECTED_SOURCE_SHA ANDROID_UPLOAD_KEYSTORE_B64 ANDROID_UPLOAD_STORE_PASSWORD ANDROID_UPLOAD_KEY_PASSWORD ANDROID_UPLOAD_KEY_ALIAS ANDROID_UPLOAD_CERT_SHA256; do
  [[ -n "${!name:-}" ]] || fail "missing required signing input: $name"
done

[[ "$EXPECTED_SOURCE_SHA" =~ ^[a-f0-9]{40}$ ]] || fail "invalid pinned source SHA"
SOURCE_SHA="$(git rev-parse HEAD)"
[[ "$SOURCE_SHA" == "$EXPECTED_SOURCE_SHA" ]] || fail "checkout no longer matches selected source SHA"
[[ "$(git status --porcelain)" == "" ]] || fail "dirty source tree"

for command in node keytool jarsigner unzip base64 sha256sum aapt; do
  command -v "$command" >/dev/null 2>&1 || fail "missing prerequisite: $command"
done

normalize_fingerprint() {
  printf '%s' "$1" | tr -d ':[:space:]' | tr '[:lower:]' '[:upper:]'
}
EXPECTED_CERT="$(normalize_fingerprint "$ANDROID_UPLOAD_CERT_SHA256")"
[[ "$EXPECTED_CERT" =~ ^[A-F0-9]{64}$ ]] || fail "invalid pinned certificate SHA-256"

VERSION="$(node -p 'require("./version.json").version')"
VERSION_CODE="$(node -p 'require("./version.json").androidVersionCode')"
AAB="android/app/build/outputs/bundle/release/app-release.aab"
APK="android/app/build/outputs/apk/release/app-release-unsigned.apk"
[[ -s "$AAB" && -s "$APK" ]] || fail "unsigned release APK/AAB build artifacts are missing"
unzip -tqq "$AAB" || fail "invalid unsigned AAB archive"
unzip -Z -1 "$AAB" | grep -Fxq 'base/manifest/AndroidManifest.xml' || fail "missing AAB manifest"

BADGING="$(aapt dump badging "$APK")" || fail "cannot inspect release package metadata"
grep -Fq "package: name='com.ghostftp.android' versionCode='$VERSION_CODE' versionName='$VERSION'" <<< "$BADGING" ||
  fail "release application ID or version does not match the pinned product"
if grep -Fq "application-debuggable" <<< "$BADGING"; then
  fail "release package is debuggable"
fi

PRIVATE_DIR="$(mktemp -d)" || fail "cannot allocate private keystore directory"
trap 'rm -rf -- "$PRIVATE_DIR"' EXIT
KEYSTORE="$PRIVATE_DIR/upload-key.jks"
printf '%s' "$ANDROID_UPLOAD_KEYSTORE_B64" | base64 --decode > "$KEYSTORE" ||
  fail "invalid upload keystore secret"
chmod 600 "$KEYSTORE"
[[ -s "$KEYSTORE" ]] || fail "empty upload keystore"

# -storepass:env / -keypass:env keep passwords out of argv and build logs.
KEY_INFO="$(keytool -list -v -keystore "$KEYSTORE" \
  -storepass:env ANDROID_UPLOAD_STORE_PASSWORD \
  -alias "$ANDROID_UPLOAD_KEY_ALIAS")" || fail "upload key/certificate could not be opened"
KEY_CERT="$(printf '%s\n' "$KEY_INFO" | sed -n 's/^[[:space:]]*SHA256: //p' | head -n 1)"
[[ "$(normalize_fingerprint "$KEY_CERT")" == "$EXPECTED_CERT" ]] ||
  fail "keystore certificate does not match the pinned persistent upload identity"

mkdir -p dist/android/signed
SIGNED="dist/android/signed/GhostFTP-Android-v${VERSION}-Release-Signed.aab"
jarsigner -keystore "$KEYSTORE" \
  -storepass:env ANDROID_UPLOAD_STORE_PASSWORD \
  -keypass:env ANDROID_UPLOAD_KEY_PASSWORD \
  -digestalg SHA-256 \
  -signedjar "$SIGNED" "$AAB" "$ANDROID_UPLOAD_KEY_ALIAS" >/dev/null ||
  fail "AAB signing failed"

VERIFY_OUTPUT="$(jarsigner -verify "$SIGNED")" || fail "AAB signature verification failed"
grep -Fq 'jar verified.' <<< "$VERIFY_OUTPUT" || fail "AAB signature not verified"
SIGNED_INFO="$(keytool -printcert -jarfile "$SIGNED")" ||
  fail "cannot inspect signed AAB certificate"
SIGNED_CERT="$(printf '%s\n' "$SIGNED_INFO" | sed -n 's/^[[:space:]]*SHA256: //p' | head -n 1)"
[[ "$(normalize_fingerprint "$SIGNED_CERT")" == "$EXPECTED_CERT" ]] ||
  fail "signed AAB certificate differs from pinned upload identity"

(
  cd dist/android/signed
  sha256sum "$(basename "$SIGNED")" > SHA256SUMS.txt
)
{
  printf 'status=signed-and-locally-verified; NOT Play Console accepted\n'
  printf 'source_sha=%s\n' "$SOURCE_SHA"
  printf 'application_id=com.ghostftp.android\n'
  printf 'version_name=%s\n' "$VERSION"
  printf 'version_code=%s\n' "$VERSION_CODE"
  printf 'upload_certificate_sha256=%s\n' "$EXPECTED_CERT"
  printf 'aab_sha256=%s\n' "$(sha256sum "$SIGNED" | cut -d ' ' -f 1)"
} > dist/android/signed/SIGNING-EVIDENCE.txt
echo "AAB locally signed and certificate-pinned for $SOURCE_SHA; Play upload and upgrade remain unverified."
