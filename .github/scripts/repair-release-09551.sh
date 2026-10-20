#!/usr/bin/env bash
# Source-locked, one-time recovery of the broken v0.95.51 public entry.
set -euo pipefail
VERSION="0.95.51"
TAG="v0.95.51"
SOURCE_SHA="1d6d5683c0fc4dee8e4b9c6de1750599db33fc4f"
RELEASE_ID=418806377
ROOT="$GITHUB_WORKSPACE/legacy"
test "$(git -C "$ROOT" rev-parse HEAD)" = "$SOURCE_SHA"
test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA"
for RUN_ID in 39484945427 39484945441 39484945430 39484945422 39484945419 39484945479; do
  test "$(gh api "repos/$GITHUB_REPOSITORY/actions/runs/$RUN_ID" --jq '.head_sha')" = "$SOURCE_SHA"
  test "$(gh api "repos/$GITHUB_REPOSITORY/actions/runs/$RUN_ID" --jq '.conclusion')" = success
done
RELEASE="$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID")"
test "$(jq -r '.tag_name' <<< "$RELEASE")" = "$TAG"
test "$(jq -r '.target_commitish' <<< "$RELEASE")" = "$SOURCE_SHA"
test "$(jq -r '.draft' <<< "$RELEASE")" = false
test "$(jq -r '.prerelease' <<< "$RELEASE")" = false
test "$(jq '.assets | length' <<< "$RELEASE")" -eq 0
cd "$ROOT"
mkdir -p dist/native dist/android dist/macos
git archive --format=zip --output="dist/GhostFTP-v$VERSION-Source.zip" HEAD
zip -qr "dist/GhostFTP-v$VERSION-Desktop-Source.zip" ghostftp-desktop
zip -qr "dist/GhostFTP-v$VERSION-Android-Source.zip" android
zip -qr "dist/GhostFTP-v$VERSION-macOS-Source.zip" macos
zip -qr "dist/GhostFTP-v$VERSION-Updates.zip" updates
zip -qr "dist/GhostFTP-v$VERSION-Documentation.zip" docs README.md CHANGELOG.md SECURITY.md LICENSE.txt EULA.txt
gh run download 39484945427 -D dist/native
gh run download 39484945441 --name GhostFTP-Android-APK -D dist/android
gh run download 39484945430 --name GhostFTP-macOS-Preview -D dist/macos
one_file() {
  local found
  found="$(find "$2" -type f -iname "$1" | head -n1 || true)"
  test -n "$found" && test -s "$found" || {
    echo "::error::Missing exact-source artifact $1 under $2" >&2
    exit 1
  }
  printf '%s' "$found"
}
cp "$(one_file ghostftp.exe dist/native)" "dist/GhostFTP-Windows-x64-Portable-v$VERSION.exe"
cp "$(one_file '*setup*.exe' dist/native)" "dist/GhostFTP-Windows-x64-Setup-v$VERSION.exe"
cp "$(one_file '*.msi' dist/native)" "dist/GhostFTP-Windows-x64-Setup-v$VERSION.msi"
cp "$(one_file ghostftp dist/native)" "dist/GhostFTP-Linux-x86_64-v$VERSION"
cp "$(one_file '*.AppImage' dist/native)" "dist/GhostFTP-Linux-x86_64-v$VERSION.AppImage"
cp "$(one_file '*.deb' dist/native)" "dist/GhostFTP-Linux-amd64-v$VERSION.deb"
cp "$(one_file '*.rpm' dist/native)" "dist/GhostFTP-Linux-x86_64-v$VERSION.rpm"
cp "$(one_file '*Release-Unsigned.apk.unsigned' dist/android)" "dist/GhostFTP-Android-v$VERSION.apk.unsigned"
cp "$(one_file '*Release-Unsigned.aab.unsigned' dist/android)" "dist/GhostFTP-Android-v$VERSION.aab.unsigned"
cp "$(one_file '*Installable-Preview.apk' dist/android)" "dist/GhostFTP-Android-v$VERSION-Installable-Preview.apk"
cp "$(one_file "GhostFTP-macOS-v$VERSION-Preview.zip" dist/macos)" "dist/GhostFTP-macOS-v$VERSION-Preview.zip"
QA_DIR="$(find dist/native -type d -name GhostFTP-Windows-x64-Preview-QA | head -n1)"
test -d "$QA_DIR"
test "$(find "$QA_DIR" -type f -name '*.png' | wc -l)" -ge 7
(cd "$QA_DIR" && zip -q "$ROOT/dist/GhostFTP-Windows-x64-v$VERSION-Native-QA.zip" ./*)
(cd dist && zip -q "GhostFTP-Windows-x64-v$VERSION.zip" "GhostFTP-Windows-x64-Portable-v$VERSION.exe" "GhostFTP-Windows-x64-Setup-v$VERSION.exe" "GhostFTP-Windows-x64-Setup-v$VERSION.msi")
(cd dist && tar -czf "GhostFTP-Linux-x86_64-v$VERSION.tar.gz" "GhostFTP-Linux-x86_64-v$VERSION" "GhostFTP-Linux-x86_64-v$VERSION.AppImage" "GhostFTP-Linux-amd64-v$VERSION.deb" "GhostFTP-Linux-x86_64-v$VERSION.rpm")
rm -rf dist/native dist/android dist/macos
(cd dist && find . -maxdepth 1 -type f ! -name "GhostFTP-v$VERSION-SHA256SUMS.txt" -printf '%f\0' | sort -z | xargs -0 sha256sum) > "dist/GhostFTP-v$VERSION-SHA256SUMS.txt"
(cd dist && sha256sum -c "GhostFTP-v$VERSION-SHA256SUMS.txt")
test "$(find dist -maxdepth 1 -type f | wc -l)" -eq 21
# Withdraw the empty public entry only after all packages are locally ready.
jq -n --arg name "Ghost FTP 0.95.51 Preview" --rawfile notes "docs/releases/$VERSION.md" \
  '{draft:true,prerelease:true,name:$name,body:$notes}' |
  gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --input - >/dev/null
test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --jq '.draft')" = true
for file in dist/*; do
  NAME="$(basename "$file")"
  curl --fail --silent --show-error --location --connect-timeout 30 --max-time 900 \
    --request POST \
    --header "Accept: application/vnd.github+json" \
    --header "Authorization: Bearer $GH_TOKEN" \
    --header "Content-Type: application/octet-stream" \
    --data-binary "@$file" \
    "https://uploads.github.com/repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?name=$NAME" >/dev/null
done
REMOTE="$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?per_page=100")"
test "$(jq 'length' <<< "$REMOTE")" -eq 21
for file in dist/*; do
  NAME="$(basename "$file")"
  LOCAL_SHA="$(sha256sum "$file" | awk '{print $1}')"
  test "$(jq -r --arg name "$NAME" '[.[] | select(.name == $name) | .digest] | if length == 1 then .[0] else empty end' <<< "$REMOTE")" = "sha256:$LOCAL_SHA"
done
test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA"
jq -n '{draft:false,prerelease:true}' |
  gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --input - >/dev/null
test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --jq '.draft')" = false
test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --jq '.prerelease')" = true
echo "Recovered 21 SHA-256 verified v0.95.51 Preview assets from immutable source $SOURCE_SHA."
