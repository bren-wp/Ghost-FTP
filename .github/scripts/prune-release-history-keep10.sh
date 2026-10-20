#!/usr/bin/env bash
# Owner-authorized archive policy: keep exactly ten latest complete public
# releases and remove older releases AND matching refs/tags. Never touch drafts.
set -euo pipefail
test "$GHOSTFTP_RELEASE_RETENTION_CONFIRM" = "KEEP_10"
test "$GITHUB_REPOSITORY" = "bren-wp/Ghost-FTP"
CURRENT_MAIN_SHA="$(gh api "repos/$GITHUB_REPOSITORY/git/ref/heads/main" --jq '.object.sha')"
test "$(git rev-parse HEAD)" = "$CURRENT_MAIN_SHA"
VERSION="$(node -p 'require("./version.json").version')"
LATEST_TAG="v$VERSION"
LATEST="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$LATEST_TAG")"
test "$(jq -r '.draft' <<< "$LATEST")" = false
test "$(jq -r '.prerelease' <<< "$LATEST")" = true
test "$(jq -r '.assets | length' <<< "$LATEST")" -ge 21
test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$LATEST_TAG" --jq '.object.sha')" = "$CURRENT_MAIN_SHA"
# Refuse deletion until the previously broken release is repaired and Preview.
REPAIRED="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/v0.95.51")"
test "$(jq -r '.draft' <<< "$REPAIRED")" = false
test "$(jq -r '.prerelease' <<< "$REPAIRED")" = true
test "$(jq -r '.assets | length' <<< "$REPAIRED")" -eq 21
# A protected screenshot manifest must cite actual native Windows QA, not a
# historical tag whose removal would break README or source build checks.
test "$(node -p 'require("./docs/assets/screenshots/README-NATIVE-CAPTURE.json").version')" = "0.95.51"
test "$(node -p 'require("./docs/assets/screenshots/README-NATIVE-CAPTURE.json").capture_source_sha')" = "1d6d5683c0fc4dee8e4b9c6de1750599db33fc4f"
test "$(node -p 'require("./docs/assets/screenshots/README-NATIVE-CAPTURE.json").github_actions_run_id')" = "39484945427"
GHOSTFTP_LATEST_RELEASE_TAG="$LATEST_TAG" node .github/scripts/check-release-images.mjs

# Pagination must cover ALL historical releases, not the first 100.
ALL="$(gh api "repos/$GITHUB_REPOSITORY/releases?per_page=100" --paginate --slurp |
  jq -c '[.[][] | select(.draft == false)] | sort_by(.published_at) | reverse')"
TOTAL="$(jq 'length' <<< "$ALL")"
test "$TOTAL" -ge 10
test "$TOTAL" -le 1000
KEEP="$(jq -c '.[0:10]' <<< "$ALL")"
OLDEST="$(jq -c '.[10:] | reverse' <<< "$ALL")"
test "$(jq -r '.[0].tag_name' <<< "$KEEP")" = "$LATEST_TAG"
# Fail closed for missing/empty/incorrectly named assets in any keeper.
while IFS= read -r item; do
  tag="$(jq -r '.tag_name' <<< "$item")"
  [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || exit 1
  asset_count="$(jq -r '.assets | length' <<< "$item")"
  test "$asset_count" -ge 20
  version="$(printf '%s' "$tag" | sed 's/^v//')"
  test "$(jq -r --arg expected "GhostFTP-v$version-SHA256SUMS.txt" \
     '[.assets[].name | select(. == $expected)] | length' <<< "$item")" -eq 1
  test -n "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$tag" --jq '.object.sha')"
done < <(jq -c '.[]' <<< "$KEEP")

echo "Verified complete public releases. Retaining: $(jq -r '[.[].tag_name] | join(", ")' <<< "$KEEP")"
echo "Owner-authorized old releases to remove: $(jq 'length' <<< "$OLDEST")"
# Never delete a keeper, noncanonical tag, draft, or a tag whose release moved
# since inventory. Delete oldest first, rechecking the public ID before each.
while IFS= read -r item; do
  tag="$(jq -r '.tag_name' <<< "$item")"
  id="$(jq -r '.id' <<< "$item")"
  [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
    echo "Refusing unexpected historical tag $tag" >&2
    exit 1
  }
  if jq -e --arg tag "$tag" '.[] | select(.tag_name == $tag)' <<< "$KEEP" >/dev/null; then
    echo "::error::Attempted to remove retained release $tag"
    exit 1
  fi
  before="$(gh api "repos/$GITHUB_REPOSITORY/releases/$id")"
  test "$(jq -r '.tag_name' <<< "$before")" = "$tag"
  test "$(jq -r '.draft' <<< "$before")" = false
  test "$(jq -r '.id' <<< "$before")" = "$id"
  echo "Pruning historical $tag and corresponding tag reference ($id)"
  gh release delete "$tag" --yes --cleanup-tag
done < <(jq -c '.[]' <<< "$OLDEST")
echo "Retention completed; kept latest ten published releases and their refs/tags."
