#!/usr/bin/env bash
# Repair only release metadata of fully verified source-pinned Preview editions.
# NEVER DELETE OR REWRITE releases, their artifacts, or their immutable tags.
set -euo pipefail
test "$GITHUB_REPOSITORY" = "bren-wp/Ghost-FTP"
test -n "$GH_TOKEN"
test "$(node -p 'require("./version.json").channel')" = "preview"
CURRENT="$(node -p 'require("./version.json").version')"
MODE="$GHOSTFTP_METADATA_EVENT"
case "$MODE" in push|schedule|successful-release|release-edited) ;; *) exit 1 ;; esac

enforce_preview() {
  local version="$1" sha="$2" tag="v$1" release id name after
  [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
  [[ "$sha" =~ ^[a-f0-9]{40}$ ]]
  test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$tag" --jq '.object.sha')" = "$sha"
  release="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$tag")"
  test "$(jq -r '.draft' <<< "$release")" = false
  test "$(jq -r '.tag_name' <<< "$release")" = "$tag"
  test "$(jq -r '.target_commitish' <<< "$release")" = "$sha"
  test "$(jq -r '.assets | length' <<< "$release")" -ge 21
  for name in "GhostFTP-v$version-SHA256SUMS.txt" "GhostFTP-macOS-v$version-Preview.zip"; do
    test "$(jq -r --arg name "$name" '[.assets[].name | select(. == $name)] | length' <<< "$release")" -eq 1
  done
  test "$(jq '[.assets[] | select((.size > 0) and (.digest | type == "string") and (.digest | test("^sha256:[a-f0-9]{64}$")))] | length' <<< "$release")" -eq "$(jq '.assets | length' <<< "$release")"
  id="$(jq -r '.id' <<< "$release")"
  test "$id" -gt 0
  # Keep the public title plain while preserving the honest GitHub prerelease
  # classification. Historical release titles are immutable for audit.
  if [ "$version" = "$CURRENT" ]; then
    name="Ghost FTP $version"
  else
    name="Ghost FTP $version Preview"
  fi
  if [ "$(jq -r '.prerelease' <<< "$release")" != true ] || [ "$(jq -r '.name' <<< "$release")" != "$name" ]; then
    echo "::notice::Correcting verified $tag Preview metadata; preserving all assets and immutable tags."
    jq -n --arg name "$name" '{prerelease:true,make_latest:"false",name:$name}' |
      gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/$id" --input - >/dev/null
  fi
  for attempt in $(seq 1 12); do
    after="$(gh api "repos/$GITHUB_REPOSITORY/releases/$id")"
    if [ "$(jq -r '.prerelease' <<< "$after")" = true ] &&
       [ "$(jq -r '.name' <<< "$after")" = "$name" ] &&
       [ "$(jq -r '.draft' <<< "$after")" = false ]; then
      test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$tag" --jq '.object.sha')" = "$sha"
      # Detect any unexpected artifact replacement, reordering or digest mutation.
      # Sorting by immutable asset id permits harmless API list-order changes only.
      test "$(jq -c '[.assets[] | {id,name,size,digest}] | sort_by(.id)' <<< "$after")" = \
           "$(jq -c '[.assets[] | {id,name,size,digest}] | sort_by(.id)' <<< "$release")"
      test "$(jq -r '.target_commitish' <<< "$after")" = "$sha"
      echo "Confirmed Preview classification and unchanged SHA-256 asset inventory for immutable $tag."
      return 0
    fi
    sleep 5
  done
  echo "::error::GitHub did not preserve Preview metadata for $tag." >&2
  exit 1
}

# Two immutable released developer previews were incorrectly marked stable.
enforce_preview "0.95.53" "27a3f8413f168e4cb8cc31cf6d7b7c85c353f5af"
enforce_preview "0.99.0" "900d76e6e3273058270a48affe2664f968910a1b"
# v0.99.1 was published with 21 verified artifacts but incorrectly marked stable.
# The exact immutable tag and target commit must match before PATCHing metadata.
enforce_preview "0.99.1" "eb2e1899f0bb953b731e7b41e0109ba4decb324a"
# The latest unsigned Client Preview releases were also misclassified stable.
# Repair metadata only; preserve tag SHA and every original asset digest.
enforce_preview "0.99.20" "567c324213e20f1b87a4b6444a65b8150f747966"
enforce_preview "0.99.21" "8b68b7bb7ae2a1ac35779ac4f7f965597fcc114d"
# v0.99.24 has a live inventory of only 13 release assets; the original
# manifest required >=21. Those 13 assets are OBSERVED after the incident,
# not an attestation that the missing assets were never published. Fail if
# even this observed subset changes. Never patch incomplete release metadata,
# invent digests, silently upload replacements, or remove the immutable tag.
audit_incomplete_v09924() {
  local tag="v0.99.24" sha="f36026188b3defedb9464ad04e4c234afb8a399b"
  local release expected actual observed
  test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$tag" --jq '.object.sha')" = "$sha"
  release="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$tag")"
  test "$(jq -r '.draft' <<< "$release")" = false
  test "$(jq -r '.tag_name' <<< "$release")" = "$tag"
  test "$(jq -r '.target_commitish' <<< "$release")" = "$sha"
  expected="$(jq -c 'sort_by(.id)' .github/scripts/v0.99.24-observed-assets.json)"
  actual="$(jq -c '[.assets[] | {id,name,size,digest}] | sort_by(.id)' <<< "$release")"
  test "$(jq 'length' .github/scripts/v0.99.24-observed-assets.json)" -eq 13
  if [ "$actual" != "$expected" ]; then
    echo "::error::Quarantined $tag asset inventory changed; manual provenance review required." >&2
    exit 1
  fi
  echo "::warning::Historical $tag remains INCOMPLETE (13 observed assets, at least 21 required). Existing IDs, sizes and SHA-256 digests match the incident snapshot. Metadata was not modified; restore only from authenticated original evidence."
}

audit_incomplete_v09924
# v0.99.25 contains all 25 source-pinned assets but was observed with
# stable GitHub metadata. Reclassify only after the normal immutable gates.
enforce_preview "0.99.25" "2e00a640c65dcd4d73c0401071703c9312eb7881"

# Recent desktop releases have one exact package contract, unlike legacy tags.
# Re-check their immutable source and the FULL downloadable SHA256SUMS before
# repairing GitHub's editable prerelease/name fields. Never touch body or assets.
PREVIOUS="$(node -p 'require("./version.json").previousPublishedVersion')"

repair_recent() {
  local version="$1" pinned_sha="${2:-}" tag="v$1"
  local release sha source_version tmp checksum
  # A push runs before the release workflow. No public current release exists
  # yet: that is expected, not evidence of a successful publication.
  if [ "$version" = "$CURRENT" ] && [ "$MODE" != "successful-release" ]; then
    if ! gh api "repos/$GITHUB_REPOSITORY/releases/tags/$tag" >/dev/null 2>&1; then
      echo "::notice::$tag not yet publicly visible; release workflow is responsible for publishing it."
      return 0
    fi
  fi
  release="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$tag")"
  sha="$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$tag" --jq '.object.sha')"
  [[ "$sha" =~ ^[a-f0-9]{40}$ ]] || { echo "::error::Non-commit tag for $tag" >&2; exit 1; }
  if [ -n "$pinned_sha" ] && [ "$sha" != "$pinned_sha" ]; then
    echo "::error::Source SHA for $tag drifted from expected $pinned_sha." >&2
    exit 1
  fi
  test "$(jq -r '.target_commitish' <<< "$release")" = "$sha" || {
    echo "::error::Release $tag target SHA disagrees with the immutable tag." >&2
    exit 1
  }
  source_version="$(gh api "repos/$GITHUB_REPOSITORY/contents/version.json?ref=$tag" --jq '.content' |
    base64 --decode | node -p 'JSON.parse(require("fs").readFileSync(0,"utf8")).version')"
  test "$source_version" = "$version" || {
    echo "::error::Tag $tag does not contain matching version.json." >&2
    exit 1
  }
  checksum="GhostFTP-v${version}-SHA256SUMS.txt"
  tmp="$(mktemp -d)"
  printf '%s\n' "$release" > "$tmp/release.json"
  gh release download "$tag" --pattern "$checksum" --dir "$tmp" >/dev/null
  test -s "$tmp/$checksum" || { echo "::error::Missing released checksums for $tag" >&2; exit 1; }
  # Metadata repair may read prerelease:false to FIX it. This is NOT the
  # normal post-publication gate, which still rejects prerelease:false.
  node .github/scripts/validate-release-assets.mjs "$version" \
    --metadata-repair-json "$tmp/release.json" --checksums "$tmp/$checksum"
  rm -rf "$tmp"
  enforce_preview "$version" "$sha"
}

if [ "$MODE" = successful-release ]; then
  SOURCE_SHA="$GHOSTFTP_PUBLISHED_SOURCE_SHA"
  test "$(git rev-parse HEAD)" = "$SOURCE_SHA"
fi

# Restore the ORIGINAL published filename of one known v0.99.41 asset.
# On 10 Nov 2026 its public GitHub release metadata was edited after successful
# publication: asset ID 729952759 was renamed from the checksum-manifest name
# to a non-Preview name, and the release was marked stable. This is a
# single-incident, fail-closed metadata-only correction. Never rewrite bytes.
repair_v09941_asset_name() {
  local tag="v0.99.41"
  local source_sha="0e420f35683ca8f85285e50bbbfb0286562ff5cd"
  local asset_id="729952759"
  local old_name="GhostFTP-macOS-v0.99.41.zip"
  local expected_name="GhostFTP-macOS-v0.99.41-Preview.zip"
  local expected_digest="sha256:201600cd6de99ac805f848a3ccdb951edcd84a4a3e2299fb2542e6d8d9fa254c"
  local checksum="GhostFTP-v0.99.41-SHA256SUMS.txt"
  local checksum_digest="0bc07fa0acb79f427dab8bb075ad19d5561046416c5ab1ea787cd7bc5c264cda"
  local release normalized tmp patched current
  # Older source unit fixtures may predate the publication of this release.
  if ! release="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$tag")"; then
    return 0
  fi
  if jq -e --arg name "$expected_name" '[.assets[] | select(.name == $name)] | length == 1' <<< "$release" >/dev/null; then
    return 0
  fi
  test "$(jq -r '.target_commitish' <<< "$release")" = "$source_sha" || {
    echo "::error::v0.99.41 source changed; asset recovery forbidden." >&2; exit 1;
  }
  test "$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$tag" --jq '.object.sha')" = "$source_sha" || {
    echo "::error::v0.99.41 immutable tag changed; asset recovery forbidden." >&2; exit 1;
  }
  test "$(jq -r '.draft' <<< "$release")" = false
  test "$(jq -r '.assets | length' <<< "$release")" = 21
  test "$(jq -r --argjson id "$asset_id" --arg name "$old_name" --arg digest "$expected_digest" \
    '[.assets[] | select(.id == $id and .name == $name and .digest == $digest and .size == 1250624 and .state == "uploaded")] | length' <<< "$release")" = 1 || {
    echo "::error::v0.99.41 asset identity, size or hash changed; manual provenance review required." >&2; exit 1;
  }
  tmp="$(mktemp -d)"
  gh release download "$tag" --pattern "$checksum" --dir "$tmp" >/dev/null
  test "$(sha256sum "$tmp/$checksum" | awk '{print $1}')" = "$checksum_digest" || {
    echo "::error::v0.99.41 original published SHA256SUMS changed; manual review required." >&2; exit 1;
  }
  # Verify the ENTIRE original checksum manifest by temporarily normalizing
  # ONLY the erroneous API asset label, leaving all IDs, sizes and hashes intact.
  jq --argjson id "$asset_id" --arg name "$expected_name" \
    '.assets |= map(if .id == $id then .name = $name else . end)' \
    <<< "$release" > "$tmp/normalized.json"
  node .github/scripts/validate-release-assets.mjs 0.99.41 \
    --metadata-repair-json "$tmp/normalized.json" --checksums "$tmp/$checksum"
  patched="$(gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/assets/$asset_id" -f name="$expected_name")"
  test "$(jq -r '.id' <<< "$patched")" = "$asset_id"
  test "$(jq -r '.name' <<< "$patched")" = "$expected_name"
  test "$(jq -r '.digest' <<< "$patched")" = "$expected_digest"
  test "$(jq -r '.size' <<< "$patched")" = 1250624
  current="$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$tag")"
  test "$(jq -c '[.assets[] | {id,name,size,digest}] | sort_by(.id)' <<< "$current")" = \
       "$(jq -c '[.assets[] | {id,name,size,digest}] | sort_by(.id)' "$tmp/normalized.json")" || {
    echo "::error::v0.99.41 asset inventory changed during filename restoration." >&2; exit 1;
  }
  rm -rf "$tmp"
  echo "::notice::Restored verified original v0.99.41 asset filename without changing its bytes, ID, size or SHA-256."
}

repair_v09941_asset_name

# Repair past misclassified 21-asset desktop releases as well as the moving
# predecessor/current pair. On main push the current version may not yet
# exist; on successful-release it MUST exist and match the source SHA.
declare -A SEEN_RECENT
for recent in "0.99.37" "0.99.38" "$PREVIOUS" "$CURRENT"; do
  if [ -n "${SEEN_RECENT[$recent]:-}" ]; then continue; fi
  SEEN_RECENT[$recent]=1
  if [ "$MODE" = successful-release ] && [ "$recent" = "$CURRENT" ]; then
    repair_recent "$recent" "$SOURCE_SHA"
  else
    repair_recent "$recent"
  fi
done
