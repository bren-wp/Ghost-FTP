#!/usr/bin/env bash
# Historical canonical release recovery. Never rebuild, re-tag or relabel a
# different source commit. Uses already-green exact-main artifacts only.
set -euo pipefail

VERSION="${1:?version}"
SOURCE_SHA="${2:?exact merged main SHA}"
NATIVE_RUN="${3:?native build run}"
ANDROID_RUN="${4:?Android run}"
MACOS_RUN="${5:?macOS run}"
TAG="v${VERSION}"

case "${VERSION}:${SOURCE_SHA}" in
  0.30.11:3ddfee0edb0039f090c3b7b7c475fe536ba0d040|0.30.12:8c768856bee6226f0dab00191a5d0f74123008a5) ;;
  *) echo "Unapproved historical release version/source" >&2; exit 1 ;;
esac

test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/v0.30.13" --jq '.object.sha')" = "ddebaa5d3a0657caaa5d5b1ffe8be257d2556ea3"
gh release view v0.30.13 >/dev/null
git merge-base --is-ancestor "$SOURCE_SHA" HEAD
test "$(git show "$SOURCE_SHA:version.json" | node -e 'let s="";process.stdin.on("data",c=>s+=c).on("end",()=>{if(JSON.parse(s).version!==process.argv[1])process.exit(1)})' "$VERSION")" = ""

for workflow in ghostftp-quality.yml ghostftp-protocol-e2e.yml ghostftp-build.yml ghostftp-android.yml validate-win-hardening.yml ghostftp-macos.yml; do
  state="$(gh run list --workflow "$workflow" --commit "$SOURCE_SHA" --limit 40 \
    --json headSha,conclusion --jq '[.[] | select(.headSha == "'$SOURCE_SHA'" and .conclusion == "success")] | length')"
  test "$state" -ge 1 || { echo "Exact-source gate missing: $workflow" >&2; exit 1; }
done

for run_id in "$NATIVE_RUN" "$ANDROID_RUN" "$MACOS_RUN"; do
  state="$(gh api "repos/$GITHUB_REPOSITORY/actions/runs/$run_id" --jq '.head_sha + ":" + .conclusion')"
  test "$state" = "$SOURCE_SHA:success" || { echo "Invalid artifact run: $run_id ($state)" >&2; exit 1; }
done

if gh release view "$TAG" >/dev/null 2>&1; then
  test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA" || {
    echo "Refusing to alter a historical release tag with a different SHA" >&2; exit 1;
  }
  echo "$TAG already exists on exact source; validating/uploading verified assets."
fi

WORK="$(mktemp -d)"
trap 'git worktree remove --force "$WORK/src" >/dev/null 2>&1 || true; rm -rf "$WORK"' EXIT
git worktree add --detach "$WORK/src" "$SOURCE_SHA" >/dev/null
SRC="$WORK/src"
DIST="$WORK/dist"
mkdir -p "$DIST/native" "$DIST/android" "$DIST/macos"
test -s "$SRC/docs/releases/$VERSION.md"
test "$(node -p "JSON.parse(require('fs').readFileSync('$SRC/version.json','utf8')).version")" = "$VERSION"
gh run download "$NATIVE_RUN" -D "$DIST/native"
gh run download "$ANDROID_RUN" --name GhostFTP-Android-APK -D "$DIST/android"
gh run download "$MACOS_RUN" --name GhostFTP-macOS-Preview -D "$DIST/macos"

( cd "$SRC" && git archive --format=zip --output="$DIST/GhostFTP-$TAG-Source.zip" HEAD )
( cd "$SRC" && zip -qr "$DIST/GhostFTP-$TAG-Desktop-Source.zip" ghostftp-desktop )
( cd "$SRC" && zip -qr "$DIST/GhostFTP-$TAG-Android-Source.zip" android )
( cd "$SRC" && zip -qr "$DIST/GhostFTP-$TAG-macOS-Source.zip" macos )
( cd "$SRC" && zip -qr "$DIST/GhostFTP-$TAG-Updates.zip" updates )
( cd "$SRC" && zip -qr "$DIST/GhostFTP-$TAG-Documentation.zip" docs README.md CHANGELOG.md SECURITY.md LICENSE.txt EULA.txt )

find_first() { find "$1" -type f -iname "$2" | head -n1; }
copy_required() {
  local from="$1" to="$2"
  test -n "$from" && test -s "$from" || { echo "Missing release input for $to" >&2; exit 1; }
  cp "$from" "$DIST/$to"
}

copy_required "$(find_first "$DIST/native" 'ghostftp.exe')" "GhostFTP-Windows-x64-Portable-$TAG.exe"
copy_required "$(find "$DIST/native" -type f -iname '*setup*.exe' ! -iname '*.sig' | head -n1)" "GhostFTP-Windows-x64-Setup-$TAG.exe"
copy_required "$(find_first "$DIST/native" '*.msi')" "GhostFTP-Windows-x64-Setup-$TAG.msi"
copy_required "$(find "$DIST/native" -type f -name ghostftp | head -n1)" "GhostFTP-Linux-x86_64-$TAG"
chmod +x "$DIST/GhostFTP-Linux-x86_64-$TAG"
copy_required "$(find "$DIST/native" -type f -iname '*.AppImage' ! -iname '*.sig' | head -n1)" "GhostFTP-Linux-x86_64-$TAG.AppImage"
copy_required "$(find_first "$DIST/native" '*.deb')" "GhostFTP-Linux-amd64-$TAG.deb"
copy_required "$(find_first "$DIST/native" '*.rpm')" "GhostFTP-Linux-x86_64-$TAG.rpm"
copy_required "$(find_first "$DIST/android" '*Release-Unsigned.apk.unsigned')" "GhostFTP-Android-$TAG.apk.unsigned"
copy_required "$(find_first "$DIST/android" '*Installable-Preview.apk')" "GhostFTP-Android-$TAG-Installable-Preview.apk"
copy_required "$(find_first "$DIST/macos" "GhostFTP-macOS-$TAG-Preview.zip")" "GhostFTP-macOS-$TAG-Preview.zip"

QA_DIR="$(find "$DIST/native" -type d -name GhostFTP-Windows-x64-Preview-QA | head -n1)"
test -n "$QA_DIR" && test -d "$QA_DIR"
test "$(find "$QA_DIR" -type f -iname '*.png' | wc -l)" -ge 7
test "$(find "$QA_DIR" -type f -iname '*.txt' | wc -l)" -ge 7
( cd "$QA_DIR" && zip -q "$DIST/GhostFTP-Windows-x64-$TAG-Native-QA.zip" ./* )
( cd "$DIST" && zip -q "GhostFTP-Windows-x64-$TAG.zip" \
    "GhostFTP-Windows-x64-Portable-$TAG.exe" "GhostFTP-Windows-x64-Setup-$TAG.exe" "GhostFTP-Windows-x64-Setup-$TAG.msi" )
( cd "$DIST" && tar -czf "GhostFTP-Linux-x86_64-$TAG.tar.gz" \
    "GhostFTP-Linux-x86_64-$TAG" "GhostFTP-Linux-x86_64-$TAG.AppImage" \
    "GhostFTP-Linux-amd64-$TAG.deb" "GhostFTP-Linux-x86_64-$TAG.rpm" )
rm -rf "$DIST/native" "$DIST/android" "$DIST/macos"

printf 'Version: %s\nSource SHA: %s\nNative run: %s\nAndroid run: %s\nmacOS run: %s\nAll six source gates: success\n' \
  "$TAG" "$SOURCE_SHA" "$NATIVE_RUN" "$ANDROID_RUN" "$MACOS_RUN" > "$DIST/GhostFTP-$TAG-BUILD-PROVENANCE.txt"
( cd "$DIST" && find . -maxdepth 1 -type f ! -name "GhostFTP-$TAG-SHA256SUMS.txt" -printf '%f\0' \
   | sort -z | xargs -0 sha256sum ) > "$DIST/GhostFTP-$TAG-SHA256SUMS.txt"
( cd "$DIST" && sha256sum -c "GhostFTP-$TAG-SHA256SUMS.txt" )

if ! gh release view "$TAG" >/dev/null 2>&1; then
  gh release create "$TAG" --target "$SOURCE_SHA" --title "Ghost FTP $VERSION" \
    --notes-file "$SRC/docs/releases/$VERSION.md" --latest=false
fi
gh release upload "$TAG" "$DIST"/* --clobber

RELEASE_ID="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$TAG" --jq '.id')"
for file in "$DIST"/*; do
  name="$(basename "$file")"
  local_digest="$(sha256sum "$file" | cut -d' ' -f1)"
  remote_digest="$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets" --paginate \
    --jq ".[] | select(.name == \"$name\") | .digest" | tail -n1)"
  test "$remote_digest" = "sha256:$local_digest" || { echo "Remote digest mismatch: $name" >&2; exit 1; }
done
test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA"
gh release edit "$TAG" --latest=false
latest="$(gh api "repos/$GITHUB_REPOSITORY/releases/latest" --jq '.tag_name')"
case "$latest" in
  v0.30.13|v0.30.14) ;;
  *) echo "Historical recovery unexpectedly changed canonical latest release: $latest" >&2; exit 1 ;;
esac
echo "Verified historical release $TAG from exact merged source $SOURCE_SHA"
