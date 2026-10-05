#!/usr/bin/env bash
set -euo pipefail

SOURCE_SHA="${1:?source SHA is required}"
VERSION="$(node -p 'require("./version.json").version')"
TAG="v${VERSION}"
RELEASE_NAME="Ghost FTP ${VERSION}"
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

ARGS=(--target "$SOURCE_SHA" --title "$RELEASE_NAME" --notes-file "$NOTES_FILE" --latest)

if gh release view "$TAG" >/dev/null 2>&1; then
  gh release edit "$TAG" "${ARGS[@]}"
else
  gh release create "$TAG" "${ARGS[@]}"
fi

gh release upload "$TAG" dist/* --clobber

RELEASE_ID="$(gh api "repos/${GITHUB_REPOSITORY}/releases/tags/$TAG" --jq '.id')"
for file in dist/*; do
  NAME="$(basename "$file")"
  LOCAL_SHA="$(sha256sum "$file" | awk '{print $1}')"
  REMOTE_DIGEST="$(
    gh api "repos/${GITHUB_REPOSITORY}/releases/$RELEASE_ID/assets" --paginate       --jq ".[] | select(.name == \"$NAME\") | .digest" | tail -n1
  )"
  test "$REMOTE_DIGEST" = "sha256:$LOCAL_SHA"
done

find dist -maxdepth 1 -type f -printf '%f\n' | sort > "$RUNNER_TEMP/expected-assets.txt"
for asset in $(gh release view "$TAG" --json assets --jq '.assets[].name'); do
  if ! grep -Fxq "$asset" "$RUNNER_TEMP/expected-assets.txt"; then
    gh release delete-asset "$TAG" "$asset" -y
  fi
done

LOCAL_COUNT="$(find dist -maxdepth 1 -type f | wc -l | tr -d ' ')"
REMOTE_COUNT="$(gh api "repos/${GITHUB_REPOSITORY}/releases/$RELEASE_ID/assets" --paginate --jq 'length')"
test "$REMOTE_COUNT" = "$LOCAL_COUNT"
test "$(gh api "repos/${GITHUB_REPOSITORY}/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA"
