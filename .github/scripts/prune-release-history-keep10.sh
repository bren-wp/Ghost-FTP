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
# On the first post-recovery release fix the legacy Preview classification
# before deleting any history. Future releases must not depend on a tag that
# will itself legitimately age out of the newest ten after enough versions.
if [ "$VERSION" = "0.95.53" ]; then
  REPAIRED="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/v0.95.51")"
  test "$(jq -r '.draft' <<< "$REPAIRED")" = false
  test "$(jq -r '.assets | length' <<< "$REPAIRED")" -eq 21
  test "$(jq -r '.tag_name' <<< "$REPAIRED")" = "v0.95.51"
  test "$(jq -r '.target_commitish' <<< "$REPAIRED")" = "1d6d5683c0fc4dee8e4b9c6de1750599db33fc4f"
  test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/v0.95.51" --jq '.object.sha')" = "1d6d5683c0fc4dee8e4b9c6de1750599db33fc4f"
  test "$(jq -r '[.assets[] | select(.name == "GhostFTP-v0.95.51-SHA256SUMS.txt" and (.digest|startswith("sha256:")))] | length' <<< "$REPAIRED")" -eq 1
  if [ "$(jq -r '.prerelease' <<< "$REPAIRED")" != true ]; then
    echo "::warning::Correcting verified v0.95.51 Preview classification; no asset bytes are changed."
    jq -n '{prerelease:true}' |
      gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/418806377" --input - >/dev/null
  fi
  test "$(gh api "repos/$GITHUB_REPOSITORY/releases/418806377" --jq '.prerelease')" = true
fi
# A protected screenshot manifest must cite actual native Windows QA, not a
# historical tag whose removal would break README or source build checks.
CAPTURE_VERSION="$(node -p 'require("./docs/assets/screenshots/README-NATIVE-CAPTURE.json").version')"
CAPTURE_SHA="$(node -p 'require("./docs/assets/screenshots/README-NATIVE-CAPTURE.json").capture_source_sha')"
CAPTURE_RUN_ID="$(node -p 'require("./docs/assets/screenshots/README-NATIVE-CAPTURE.json").github_actions_run_id')"
[[ "$CAPTURE_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
[[ "$CAPTURE_SHA" =~ ^[a-f0-9]{40}$ ]]
[[ "$CAPTURE_RUN_ID" =~ ^[0-9]+$ ]]
if [ "$CAPTURE_VERSION" = "0.95.51" ]; then
  test "$CAPTURE_SHA" = "1d6d5683c0fc4dee8e4b9c6de1750599db33fc4f"
  test "$CAPTURE_RUN_ID" = "39484945427"
fi
# Confirm the claimed native QA run's source identity, not an invented caption.
test "$(gh api "repos/$GITHUB_REPOSITORY/actions/runs/$CAPTURE_RUN_ID" --jq '.head_sha')" = "$CAPTURE_SHA"
test "$(gh api "repos/$GITHUB_REPOSITORY/actions/runs/$CAPTURE_RUN_ID" --jq '.conclusion')" = success
GHOSTFTP_LATEST_RELEASE_TAG="$LATEST_TAG" node .github/scripts/check-release-images.mjs

# Pagination must cover ALL historical releases, not the first 100.
# Sort by canonical numeric SemVer, never mutable publication timestamps:
# correcting old Preview metadata can change its update/publication ordering.
ALL="$(gh api "repos/$GITHUB_REPOSITORY/releases?per_page=100" --paginate --slurp |
  jq -c '[.[][] | select(.draft == false)] | sort_by(.tag_name | ltrimstr("v") | split(".") | map(tonumber)) | reverse')"
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
  # All retained packages contain unsigned Android payloads and unsigned or
  # ad-hoc-signed desktop previews. Never present them as production stable.
  if [ "$(jq -r '.prerelease' <<< "$item")" != true ]; then
    id="$(jq -r '.id' <<< "$item")"
    test "$id" -gt 0
    echo "Correcting Preview metadata for legacy retained $tag."
    jq -n '{prerelease:true}' |
      gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/$id" --input - >/dev/null
    test "$(gh api "repos/$GITHUB_REPOSITORY/releases/$id" --jq '.prerelease')" = true
  fi
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
