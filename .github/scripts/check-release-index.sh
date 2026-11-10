#!/usr/bin/env bash
# Post-release proof: newest immutable Preview is publicly visible and every
# artifact matches the SAME release-artifacts.json contract as the publisher.
# GitHub /releases/latest intentionally excludes previews.
set -euo pipefail
VERSION="$1"
SOURCE_SHA="$2"
REPO="$GITHUB_REPOSITORY"
die() { echo "::error::$*" >&2; exit 1; }
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "Invalid version: $VERSION"
[[ "$SOURCE_SHA" =~ ^[a-f0-9]{40}$ ]] || die "Invalid source SHA"
[[ -n "$GH_TOKEN" ]] || die "GH_TOKEN is required"
[[ "$REPO" = "bren-wp/Ghost-FTP" ]] || die "Wrong repository: $REPO"
TAG="v$VERSION"
CHECKSUM_FILE="GhostFTP-v$VERSION-SHA256SUMS.txt"
TAG_SHA="$(gh api "repos/$REPO/git/ref/tags/$TAG" --jq '.object.sha')"
[[ "$TAG_SHA" = "$SOURCE_SHA" ]] || die "Immutable $TAG tag points to $TAG_SHA, expected $SOURCE_SHA"

INVENTORY="$(gh api "repos/$REPO/releases?per_page=100" --paginate --slurp)"
MATCHES="$(jq --arg tag "$TAG" '[.[][] | select(.draft == false and .tag_name == $tag)] | length' <<< "$INVENTORY")"
[[ "$MATCHES" = 1 ]] || die "Expected one public $TAG release, found $MATCHES"
LATEST_TAG="$(jq -r '
  [.[][] | select(.draft == false) | .tag_name |
    select(test("^v[0-9]+[.][0-9]+[.][0-9]+$"))] |
  sort_by(ltrimstr("v") | split(".") | map(tonumber)) | last // empty
' <<< "$INVENTORY")"
test "$LATEST_TAG" = "$TAG" || die "Newest published GitHub Release is $LATEST_TAG, expected $TAG"
FIRST_PAGE="$(gh api "repos/$REPO/releases?per_page=30")"
[[ "$(jq --arg tag "$TAG" '[.[] | select(.draft == false and .tag_name == $tag)] | length' <<< "$FIRST_PAGE")" = 1 ]] ||
  die "$TAG is not visible on first GitHub Releases API page"

RELEASE="$(gh api "repos/$REPO/releases/tags/$TAG")"
[[ "$(jq -r '.draft' <<< "$RELEASE")" = false ]] || die "$TAG remains Draft"
[[ "$(jq -r '.prerelease' <<< "$RELEASE")" = true ]] || die "$TAG is not marked Preview/prerelease"
[[ "$(jq -r '.target_commitish' <<< "$RELEASE")" = "$SOURCE_SHA" ]] || die "$TAG release target SHA mismatch"
[[ "$(jq -r '.html_url' <<< "$RELEASE")" = "https://github.com/$REPO/releases/tag/$TAG" ]] ||
  die "$TAG URL mismatch"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
printf '%s\n' "$RELEASE" > "$TMP/release.json"
gh release download "$TAG" --pattern "$CHECKSUM_FILE" --dir "$TMP" >/dev/null ||
  die "Cannot download published SHA256SUMS for $TAG"
[[ -s "$TMP/$CHECKSUM_FILE" ]] || die "Published SHA256SUMS missing/empty for $TAG"
node .github/scripts/validate-release-assets.mjs "$VERSION" \
  --release-json "$TMP/release.json" --checksums "$TMP/$CHECKSUM_FILE" ||
  die "Published release asset contract failed for $TAG"
echo "::notice::Verified public $TAG Preview: exact release manifest, sizes and SHA-256 cross-checks from $SOURCE_SHA."
