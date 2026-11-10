#!/usr/bin/env bash
set -euo pipefail

SOURCE_SHA="${1:?source SHA is required}"
node .github/scripts/check-release-channel.mjs
VERSION="$(node -p 'require("./version.json").version')"
TAG="v${VERSION}"
RELEASE_NAME="Ghost FTP ${VERSION}"
NOTES_FILE="docs/releases/${VERSION}.md"
LINUX_PACKAGE_MATRIX="docs/distribution/LINUX_PACKAGE_MATRIX.md"
CHECKSUM_FILE="GhostFTP-v${VERSION}-SHA256SUMS.txt"

test -f "$NOTES_FILE"
test -s "$LINUX_PACKAGE_MATRIX"
test -d dist
test -s "dist/$CHECKSUM_FILE"
(cd dist && sha256sum -c "$CHECKSUM_FILE")
# Check the exact inventory and every SHA-256 before uploading.
node .github/scripts/validate-release-assets.mjs "$VERSION" --dist dist

# The canonical manifest is used by publication and post-publication QA.
mapfile -t REQUIRED_CORE_FILES < <(node .github/scripts/validate-release-assets.mjs "$VERSION" --list-required)
for required in "${REQUIRED_CORE_FILES[@]}"; do
  if [ ! -s "dist/$required" ]; then
    echo "Required release artifact is missing or empty: dist/$required" >&2
    exit 1
  fi
done

mapfile -t UPDATER_FILES < <(node .github/scripts/validate-release-assets.mjs "$VERSION" --list-optional)
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
  # Every NEW release explicitly labels DEB/RPM/AppImage/native Linux packages
  # by distro family. Never mutate the body/assets of an already-public release.
  # The footer is centrally maintained so future release notes cannot omit it.
  RELEASE_NOTES_WITH_LINUX="$(mktemp)"
  {
    cat "$NOTES_FILE"
    printf '\n\n---\n\n'
    cat "$LINUX_PACKAGE_MATRIX"
  } > "$RELEASE_NOTES_WITH_LINUX"
  RELEASE_JSON="$(jq -n --arg tag "$TAG" --arg sha "$SOURCE_SHA" \
    --arg name "$RELEASE_NAME" --rawfile notes "$RELEASE_NOTES_WITH_LINUX" \
    '{tag_name:$tag,target_commitish:$sha,name:$name,body:$notes,draft:true,prerelease:true}')"
  rm -f "$RELEASE_NOTES_WITH_LINUX"
  RELEASE_ID="$(printf '%s' "$RELEASE_JSON" |
    gh api --method POST "repos/$GITHUB_REPOSITORY/releases" --input - --jq '.id')"
fi

test "$RELEASE_ID" -gt 0
RELEASE_INFO="$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID")"
test "$(jq -r '.draft' <<< "$RELEASE_INFO")" = "true"
test "$(jq -r '.prerelease' <<< "$RELEASE_INFO")" = "true"
test "$(jq -r '.tag_name' <<< "$RELEASE_INFO")" = "$TAG"

# Draft uploads are resumable. Existing correct bytes are immutable: never
# delete a complete asset because a previous CI attempt lost its HTTP response.
# GitHub may return 422 after accepting a request; recheck the remote digest
# before deciding whether an upload actually failed.
for file in dist/*; do
  # Native, macOS and other staging directories are not release assets.
  # Symlinks are also excluded to prevent uploading unintended files.
  [[ -f "$file" && ! -L "$file" ]] || continue
  test -s "$file"
  NAME="$(basename "$file")"
  [[ "$NAME" =~ ^[A-Za-z0-9._-]+$ ]] || {
    echo "Unexpected release asset name: $NAME" >&2
    exit 1
  }
  LOCAL_SHA="$(sha256sum "$file" | awk '{print $1}')"
  LOCAL_SIZE="$(stat -c '%s' "$file")"
  UPLOADED=false
  for attempt in 1 2 3 4; do
    CURRENT="$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?per_page=100" |
      jq -c --arg name "$NAME" '[.[] | select(.name == $name)]')"
    COUNT="$(jq 'length' <<< "$CURRENT")"
    if [ "$COUNT" = 1 ]; then
      DIGEST="$(jq -r '.[0].digest // ""' <<< "$CURRENT")"
      SIZE="$(jq -r '.[0].size' <<< "$CURRENT")"
      STATE="$(jq -r '.[0].state' <<< "$CURRENT")"
      if [ "$DIGEST" != "sha256:$LOCAL_SHA" ] || [ "$SIZE" != "$LOCAL_SIZE" ] || [ "$STATE" != "uploaded" ]; then
        echo "Conflicting draft asset $NAME; refusing to overwrite its provenance." >&2
        exit 1
      fi
      echo "Reusing verified draft asset: $NAME"
      UPLOADED=true
      break
    fi
    test "$COUNT" = 0 || { echo "Duplicate remote draft assets for $NAME" >&2; exit 1; }
    # HTTP 422 can mean an accepted upload whose response was lost. A repeated
    # request must not delete existing assets or create parallel duplicates.
    HTTP_CODE="$(curl --silent --show-error --location --connect-timeout 30 --max-time 900 \
      --request POST \
      --header "Accept: application/vnd.github+json" \
      --header "Authorization: Bearer $GH_TOKEN" \
      --header "Content-Type: application/octet-stream" \
      --data-binary "@$file" \
      --output /dev/null --write-out '%{http_code}' \
      "https://uploads.github.com/repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?name=$NAME" || true)"
    if [ "$HTTP_CODE" = 201 ]; then
      UPLOADED=true
      break
    fi
    echo "Asset upload response $HTTP_CODE for $NAME (attempt $attempt); checking remote digest." >&2
    sleep "$attempt"
  done
  if [ "$UPLOADED" != true ]; then
    # A delayed 201/422 can still have committed correct remote bytes.
    REMOTE="$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets?per_page=100" |
      jq -c --arg name "$NAME" '[.[] | select(.name == $name)]')"
    test "$(jq 'length' <<< "$REMOTE")" = 1 &&
      test "$(jq -r '.[0].digest' <<< "$REMOTE")" = "sha256:$LOCAL_SHA" &&
      test "$(jq -r '.[0].size' <<< "$REMOTE")" = "$LOCAL_SIZE" || {
        echo "Unable to verify release asset $NAME; leaving $TAG unpublished." >&2
        exit 1
      }
  fi
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
  [[ -f "$file" && ! -L "$file" ]] || continue
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
# make_latest=false prevents GitHub's latest-release promotion from
# misclassifying unsigned/ad-hoc signed developer packages as stable.
jq -n --arg name "$RELEASE_NAME" '{draft:false,prerelease:true,make_latest:"false",name:$name}' |
  gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --input - >/dev/null
test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --jq '.draft')" = "false"
test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID" --jq '.prerelease')" = "true"
echo "Verified $TAG Preview publication: $REMOTE_COUNT SHA-256-checked assets from $SOURCE_SHA."
