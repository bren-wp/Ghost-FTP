#!/usr/bin/env bash
set -euo pipefail

SOURCE_SHA="${1:?source SHA is required}"
node .github/scripts/check-release-channel.mjs
VERSION="$(node -p 'require("./version.json").version')"
TAG="v${VERSION}"
RELEASE_NAME="Ghost FTP ${VERSION} Preview"
NOTES_FILE="docs/releases/${VERSION}.md"
CHECKSUM_FILE="GhostFTP-v${VERSION}-SHA256SUMS.txt"

test -f "$NOTES_FILE"
test -d dist
test -s "dist/$CHECKSUM_FILE"
(cd dist && sha256sum -c "$CHECKSUM_FILE")

REQUIRED_CORE_FILES=(
  "GhostFTP-Windows-x64-Portable-v${VERSION}.exe"
  "GhostFTP-Windows-x64-Setup-v${VERSION}.exe"
  "GhostFTP-Windows-x64-Setup-v${VERSION}.msi"
  "GhostFTP-Windows-x64-v${VERSION}.zip"
  "GhostFTP-Windows-x64-v${VERSION}-Native-QA.zip"
  "GhostFTP-Linux-x86_64-v${VERSION}"
  "GhostFTP-Linux-x86_64-v${VERSION}.AppImage"
  "GhostFTP-Linux-amd64-v${VERSION}.deb"
  "GhostFTP-Linux-x86_64-v${VERSION}.rpm"
  "GhostFTP-Linux-x86_64-v${VERSION}.tar.gz"
  "GhostFTP-Android-v${VERSION}.apk.unsigned"
  "GhostFTP-Android-v${VERSION}.aab.unsigned"
  "GhostFTP-Android-v${VERSION}-Installable-Preview.apk"
  "GhostFTP-macOS-v${VERSION}-Preview.zip"
  "GhostFTP-v${VERSION}-Source.zip"
  "GhostFTP-v${VERSION}-Desktop-Source.zip"
  "GhostFTP-v${VERSION}-Android-Source.zip"
  "GhostFTP-v${VERSION}-macOS-Source.zip"
  "GhostFTP-v${VERSION}-Updates.zip"
  "GhostFTP-v${VERSION}-Documentation.zip"
  "$CHECKSUM_FILE"
)
for required in "${REQUIRED_CORE_FILES[@]}"; do
  if [ ! -s "dist/$required" ]; then
    echo "Required release artifact is missing or empty: dist/$required" >&2
    exit 1
  fi
done

UPDATER_FILES=(
  "GhostFTP-Windows-x64-Setup-v${VERSION}.exe.sig"
  "GhostFTP-Linux-x86_64-v${VERSION}.AppImage.sig"
  "GhostFTP-v${VERSION}-latest.json"
  "GhostFTP-v${VERSION}-Update-Service.zip"
)
UPDATER_PRESENT=0
for updater in "${UPDATER_FILES[@]}"; do
  if [ -s "dist/$updater" ]; then
    UPDATER_PRESENT=$((UPDATER_PRESENT + 1))
  fi
done

if [ "$UPDATER_PRESENT" -ne 0 ] && [ "$UPDATER_PRESENT" -ne "${#UPDATER_FILES[@]}" ]; then
  echo "Signed updater publication is incomplete; updater proof must be all present or all absent." >&2
  exit 1
fi

if [ "$UPDATER_PRESENT" -eq "${#UPDATER_FILES[@]}" ]; then
  node updates/scripts/verify-manifest.mjs     "dist/GhostFTP-v${VERSION}-latest.json"     --expected-version="$VERSION"
  unzip -p "dist/GhostFTP-v${VERSION}-Update-Service.zip" updates/latest.json     > "$RUNNER_TEMP/ghostftp-latest-from-package.json"
  cmp "$RUNNER_TEMP/ghostftp-latest-from-package.json"     "dist/GhostFTP-v${VERSION}-latest.json"
else
  echo "Publishing verified packages without a signed in-app updater bundle."
fi

git fetch --tags --force
if git show-ref --verify --quiet "refs/tags/$TAG"; then
  TAG_SHA="$(git rev-list -n1 "$TAG")"
  if [ "$TAG_SHA" != "$SOURCE_SHA" ]; then
    echo "$TAG already points to $TAG_SHA, expected $SOURCE_SHA. Refusing to rewrite a published version tag." >&2
    exit 1
  fi
else
  git tag "$TAG" "$SOURCE_SHA"
  git push origin "refs/tags/$TAG"
fi
test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA"

# GitHub drafts may have temporary 'untagged-*' URLs and return HTTP 404
# for GET /releases/tags/TAG. Pin the ID from the REST creation response.
EXISTING_RELEASE="$(gh api "repos/$GITHUB_REPOSITORY/releases?per_page=100" |
  jq -c --arg tag "$TAG" '[.[] | select(.tag_name == $tag)] | if length > 1 then error("duplicate release tag") else .[0] // empty end')"
if [ -n "$EXISTING_RELEASE" ]; then
  RELEASE_ID="$(jq -r '.id' <<< "$EXISTING_RELEASE")"
  DRAFT_STATE="$(jq -r '.draft' <<< "$EXISTING_RELEASE")"
  if [ "$DRAFT_STATE" != "true" ]; then
    echo "Published $TAG already exists; refusing to overwrite public assets or metadata." >&2
    exit 1
  fi
else
  RELEASE_JSON="$(jq -n --arg tag "$TAG" --arg sha "$SOURCE_SHA" \
    --arg name "$RELEASE_NAME" --rawfile notes "$NOTES_FILE" \
    '{tag_name:$tag,target_commitish:$sha,name:$name,body:$notes,draft:true,prerelease:true}')"
  RELEASE_ID="$(printf '%s' "$RELEASE_JSON" |
    gh api --method POST "repos/$GITHUB_REPOSITORY/releases" --input - --jq '.id')"
fi

test "$RELEASE_ID" -gt 0
RELEASE_INFO="$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID")"
test "$(jq -r '.draft' <<< "$RELEASE_INFO")" = "true"
test "$(jq -r '.prerelease' <<< "$RELEASE_INFO")" = "true"
test "$(jq -r '.tag_name' <<< "$RELEASE_INFO")" = "$TAG"

# A rerun may resume a failed draft. Reset ONLY an unpublished inventory.
for ASSET_ID in $(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?per_page=100" --jq '.[].id'); do
  gh api --method DELETE "repos/$GITHUB_REPOSITORY/releases/assets/$ASSET_ID"
done

for file in dist/*; do
  test -f "$file" && test -s "$file"
  NAME="$(basename "$file")"
  [[ "$NAME" =~ ^[A-Za-z0-9._-]+$ ]] || {
    echo "Unexpected release asset name: $NAME" >&2
    exit 1
  }
  # Upload raw octets by immutable release ID, NOT through a draft tag lookup.
  gh api --method POST -H "Content-Type: application/octet-stream" --input "$file" \
    "https://uploads.github.com/repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?name=$NAME" >/dev/null
done

EXPECTED_FILE_LIST="$(mktemp)"
REMOTE_ASSET_LIST="$(mktemp)"
trap 'rm -f "$EXPECTED_FILE_LIST" "$REMOTE_ASSET_LIST"' EXIT
find dist -maxdepth 1 -type f -printf '%f\n' | sort > "$EXPECTED_FILE_LIST"
gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?per_page=100" |
  jq -r '.[].name' | sort > "$REMOTE_ASSET_LIST"
diff -u "$EXPECTED_FILE_LIST" "$REMOTE_ASSET_LIST"
LOCAL_COUNT="$(wc -l < "$EXPECTED_FILE_LIST" | tr -d ' ')"
REMOTE_COUNT="$(wc -l < "$REMOTE_ASSET_LIST" | tr -d ' ')"
test "$REMOTE_COUNT" = "$LOCAL_COUNT"
for file in dist/*; do
  NAME="$(basename "$file")"
  LOCAL_SHA="$(sha256sum "$file" | awk '{print $1}')"
  REMOTE_DIGEST="$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?per_page=100" |
    jq -r --arg name "$NAME" '[.[] | select(.name == $name) | .digest] | if length != 1 then error("asset count mismatch") else .[0] end')"
  test "$REMOTE_DIGEST" = "sha256:$LOCAL_SHA" || {
    echo "Remote SHA256 mismatch for $NAME; leaving $TAG unpublished." >&2
    exit 1
  }
done

test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA"
test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --jq '.draft')" = "true"
jq -n '{draft:false,prerelease:true}' |
  gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --input - >/dev/null
test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --jq '.draft')" = "false"
test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --jq '.prerelease')" = "true"
echo "Verified $TAG Preview publication: $REMOTE_COUNT SHA-256-checked assets from $SOURCE_SHA."
