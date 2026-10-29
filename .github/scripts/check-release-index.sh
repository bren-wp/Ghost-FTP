#!/usr/bin/env bash
# Post-release proof: newest immutable edition appears on public Releases.
# GitHub /releases/latest intentionally omits prerelease/Preview versions.
set -euo pipefail
VERSION="$1"
SOURCE_SHA="$2"
REPO="$GITHUB_REPOSITORY"
test -n "$VERSION"; test -n "$SOURCE_SHA"; test -n "$GH_TOKEN"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
[[ "$SOURCE_SHA" =~ ^[a-f0-9]{40}$ ]]
test "$REPO" = "bren-wp/Ghost-FTP"
TAG="v$VERSION"
test "$(gh api "repos/$REPO/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA"

# Pagination and numeric SemVer sorting avoid misleading published_at dates.
INVENTORY="$(gh api "repos/$REPO/releases?per_page=100" --paginate --slurp)"
test "$(jq --arg tag "$TAG" '[.[][] | select(.draft == false and .tag_name == $tag)] | length' <<< "$INVENTORY")" -eq 1
LATEST_TAG="$(jq -r '
  [.[][] | select(.draft == false) | .tag_name |
    select(test("^v[0-9]+[.][0-9]+[.][0-9]+$"))] |
  sort_by(ltrimstr("v") | split(".") | map(tonumber)) | last // empty
' <<< "$INVENTORY")"
test "$LATEST_TAG" = "$TAG" || {
  echo "::error::Newest published GitHub Release is $LATEST_TAG, expected $TAG" >&2
  exit 1
}

# The actual Releases page includes this version on its first API page.
FIRST_PAGE="$(gh api "repos/$REPO/releases?per_page=30")"
test "$(jq --arg tag "$TAG" '[.[] | select(.draft == false and .tag_name == $tag)] | length' <<< "$FIRST_PAGE")" -eq 1
RELEASE="$(gh api "repos/$REPO/releases/tags/$TAG")"
test "$(jq -r '.draft' <<< "$RELEASE")" = false
test "$(jq -r '.prerelease' <<< "$RELEASE")" = true
test "$(jq -r '.target_commitish' <<< "$RELEASE")" = "$SOURCE_SHA"
test "$(jq -r '.html_url' <<< "$RELEASE")" = "https://github.com/$REPO/releases/tag/$TAG"
test "$(jq -r '.assets | length' <<< "$RELEASE")" -ge 43
test "$(jq '[.assets[] | select(.size > 0 and (.digest | type == "string") and (.digest | test("^sha256:[a-f0-9]{64}$")))] | length' <<< "$RELEASE")" -eq "$(jq '.assets | length' <<< "$RELEASE")"
for name in \
  "GhostFTP-Demo-Windows-x64-Portable-v$VERSION.exe" \
  "GhostFTP-Demo-Windows-x64-Setup-Unsigned-Preview-v$VERSION.exe" \
  "GhostFTP-Demo-Linux-x86_64-v$VERSION" \
  "GhostFTP-Demo-Linux-x86_64-v$VERSION.tar.gz" \
  "GhostFTP-Demo-Linux-amd64-v$VERSION.deb" \
  "GhostFTP-Demo-Linux-x86_64-v$VERSION.rpm" \
  "GhostCLI-Windows-x64-Setup-Unsigned-Preview-v$VERSION.exe" \
  "GhostCLI-Linux-x86_64-v$VERSION.tar.gz" \
  "GhostCLI-Linux-amd64-v$VERSION.deb" \
  "GhostCLI-Linux-x86_64-v$VERSION.rpm" \
  "GhostCLI-Windows-x64-v$VERSION.exe" \
  "GhostCLI-Linux-x86_64-v$VERSION" \
  "GhostFTP-Server-Windows-x64-Portable-v$VERSION.exe" \
  "GhostFTP-Server-Windows-x64-Setup-Unsigned-Preview-v$VERSION.exe" \
  "GhostFTP-Server-Linux-x86_64-v$VERSION" \
  "GhostFTP-Server-Linux-x86_64-v$VERSION.tar.gz" \
  "GhostFTP-Server-Linux-amd64-v$VERSION.deb" \
  "GhostFTP-Server-Linux-x86_64-v$VERSION.rpm" \
  "GhostFTP-Windows-x64-Portable-v$VERSION.exe" \
  "GhostFTP-Windows-x64-Setup-v$VERSION.exe" \
  "GhostFTP-Windows-x64-Setup-v$VERSION.msi" \
  "GhostFTP-Linux-x86_64-v$VERSION.AppImage" \
  "GhostFTP-Linux-amd64-v$VERSION.deb" \
  "GhostFTP-Linux-x86_64-v$VERSION.rpm" \
  "GhostFTP-Android-v$VERSION.apk.unsigned" \
  "GhostFTP-Android-v$VERSION.aab.unsigned" \
  "GhostFTP-Android-v$VERSION-Installable-Preview.apk" \
  "GhostFTP-macOS-v$VERSION-Preview.zip" \
  "GhostFTP-macOS-Shared-arm64-v$VERSION-Preview.zip" \
  "GhostFTP-macOS-Shared-x86_64-v$VERSION-Preview.zip" \
  "GhostFTP-macOS-Shared-arm64-v$VERSION-Installable-Preview.dmg" \
  "GhostFTP-macOS-Shared-x86_64-v$VERSION-Installable-Preview.dmg" \
  "GhostFTP-v$VERSION-SHA256SUMS.txt"; do
  test "$(jq -r --arg name "$name" '[.assets[] | select(.name == $name)] | length' <<< "$RELEASE")" -eq 1 || {
    echo "::error::Missing required artifact from public Releases: $name" >&2
    exit 1
  }
done
echo "::notice::Verified $TAG on https://github.com/$REPO/releases with $(jq '.assets|length' <<< "$RELEASE") digest-checked Preview assets from $SOURCE_SHA."
