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
