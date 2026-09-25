#!/usr/bin/env bash
set -euo pipefail

TAG="v2.1.1-rc.23"
RELEASE_NAME="Ghost FTP 2.1.1 RC23"
NOTES_FILE="docs/releases/2.1.1-rc.23.md"
CHECKSUM_FILE="GhostFTP-v2.1.1-RC23-SHA256SUMS.txt"

repair_metadata() {
  local release_id tag_sha
  release_id="$(gh api "repos/${GITHUB_REPOSITORY}/releases/tags/$TAG" --jq '.id')"
  tag_sha="$(gh api "repos/${GITHUB_REPOSITORY}/git/ref/tags/$TAG" --jq '.object.sha')"

  gh api --method PATCH "repos/${GITHUB_REPOSITORY}/releases/$release_id" \
    -F prerelease=true \
    -f target_commitish="$tag_sha" \
    -f name="$RELEASE_NAME" >/dev/null

  test "$(gh api "repos/${GITHUB_REPOSITORY}/releases/$release_id" --jq '.prerelease')" = "true"
  test "$(gh api "repos/${GITHUB_REPOSITORY}/releases/$release_id" --jq '.tag_name')" = "$TAG"
}

if [ "${1:-}" = "--repair-metadata" ]; then
  repair_metadata
  exit 0
fi

SOURCE_SHA="${1:?source SHA is required}"
test -d dist
test -s "dist/$CHECKSUM_FILE"

# The manifest must verify before any GitHub release state is mutated.
(cd dist && sha256sum -c "$CHECKSUM_FILE")

git fetch --tags --force
if git show-ref --verify --quiet "refs/tags/$TAG"; then
  CURRENT_TAG_SHA="$(git rev-list -n1 "$TAG")"
  if [ "$CURRENT_TAG_SHA" != "$SOURCE_SHA" ]; then
    git tag -f "$TAG" "$SOURCE_SHA"
    git push origin "refs/tags/$TAG" --force
  fi
else
  git tag "$TAG" "$SOURCE_SHA"
  git push origin "refs/tags/$TAG"
fi

git fetch --tags --force
test "$(git rev-list -n1 "$TAG")" = "$SOURCE_SHA"

if gh release view "$TAG" >/dev/null 2>&1; then
  gh release edit "$TAG" \
    --target "$SOURCE_SHA" \
    --title "$RELEASE_NAME" \
    --notes-file "$NOTES_FILE" \
    --prerelease
else
  gh release create "$TAG" \
    --target "$SOURCE_SHA" \
    --title "$RELEASE_NAME" \
    --notes-file "$NOTES_FILE" \
    --prerelease
fi

# Replace same-name assets first. Never empty the existing release before the
# complete new asset set has uploaded successfully.
gh release upload "$TAG" dist/* --clobber

RELEASE_ID="$(gh api "repos/${GITHUB_REPOSITORY}/releases/tags/$TAG" --jq '.id')"
gh api --method PATCH "repos/${GITHUB_REPOSITORY}/releases/$RELEASE_ID" -F prerelease=true >/dev/null

# Verify GitHub's release-asset SHA-256 digest for every generated file.
for file in dist/*; do
  NAME="$(basename "$file")"
  LOCAL_SHA="$(sha256sum "$file" | awk '{print $1}')"
  REMOTE_DIGEST="$(
    gh api "repos/${GITHUB_REPOSITORY}/releases/$RELEASE_ID/assets" --paginate \
      --jq ".[] | select(.name == \"$NAME\") | .digest" | tail -n1
  )"
  test "$REMOTE_DIGEST" = "sha256:$LOCAL_SHA"
done

# Remove obsolete names only after all new RC23 assets are present and verified.
find dist -maxdepth 1 -type f -printf '%f\n' | sort > "$RUNNER_TEMP/expected-assets.txt"
for asset in $(gh release view "$TAG" --json assets --jq '.assets[].name'); do
  if ! grep -Fxq "$asset" "$RUNNER_TEMP/expected-assets.txt"; then
    gh release delete-asset "$TAG" "$asset" -y
  fi
done

LOCAL_COUNT="$(find dist -maxdepth 1 -type f | wc -l | tr -d ' ')"
REMOTE_COUNT="$(gh api "repos/${GITHUB_REPOSITORY}/releases/$RELEASE_ID/assets" --paginate --jq 'length')"
test "$REMOTE_COUNT" = "$LOCAL_COUNT"
test "$(gh api "repos/${GITHUB_REPOSITORY}/releases/$RELEASE_ID" --jq '.prerelease')" = "true"
test "$(gh api "repos/${GITHUB_REPOSITORY}/git/ref/tags/$TAG" --jq '.object.sha')" = "$SOURCE_SHA"
