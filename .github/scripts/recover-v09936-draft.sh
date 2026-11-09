#!/usr/bin/env bash
set -euo pipefail
# One-time recovery of a *complete* unpublished release. Never rewrite its tag,
# edit its files, overwrite an existing public release or downgrade its channel.
REPO="bren-wp/Ghost-FTP"
TAG="v0.99.36"
PINNED_SOURCE_SHA="53c096ffcf961f5e33c7c7656e5f403d7146c71f"
test "${GITHUB_REPOSITORY:-}" = "$REPO"
test "$(gh api "repos/$REPO/git/ref/tags/$TAG" --jq '.object.sha')" = "$PINNED_SOURCE_SHA"
test "$(gh api "repos/$REPO/git/ref/heads/main" --jq '.object.sha')" = "$PINNED_SOURCE_SHA"
test "$(gh api "repos/$REPO/releases/tags/v0.99.35" --jq '(.draft|tostring)+":" +(.prerelease|tostring)')" = "false:false" || {
  # The historical v0.99.35 release may already have Preview metadata.
  test "$(gh api "repos/$REPO/releases/tags/v0.99.35" --jq '.draft')" = false
}
RELEASES="$(gh api "repos/$REPO/releases?per_page=100")"
MATCHES="$(jq -c --arg tag "$TAG" '[.[] | select(.tag_name == $tag)]' <<< "$RELEASES")"
test "$(jq 'length' <<< "$MATCHES")" = 1
RELEASE="$(jq '.[0]' <<< "$MATCHES")"
RELEASE_ID="$(jq -r '.id' <<< "$RELEASE")"
test "$(jq -r '.target_commitish' <<< "$RELEASE")" = "$PINNED_SOURCE_SHA"
test "$(jq -r '.prerelease' <<< "$RELEASE")" = true
if [ "$(jq -r '.draft' <<< "$RELEASE")" = false ]; then
  echo "$TAG has already been published. Keeping it unchanged."
  exit 0
fi
test "$(jq -r '.draft' <<< "$RELEASE")" = true
ASSETS="$(gh api "repos/$REPO/releases/$RELEASE_ID/assets?per_page=100")"
test "$(jq 'length' <<< "$ASSETS")" = 21
CHECKSUM_ID="$(jq -r --arg name "GhostFTP-v0.99.36-SHA256SUMS.txt" '[.[] | select(.name == $name) | .id] | if length == 1 then .[0] else empty end' <<< "$ASSETS")"
test -n "$CHECKSUM_ID"
TEMP="$(mktemp -d)"
trap 'rm -rf "$TEMP"' EXIT
printf '%s' "$ASSETS" > "$TEMP/assets.json"
# Download by asset ID because GitHub's by-tag endpoint omits drafts.
curl --fail --silent --show-error --location --retry 3 --connect-timeout 30 --max-time 90 \
  -H "Accept: application/octet-stream" \
  -H "Authorization: Bearer $GH_TOKEN" \
  "https://api.github.com/repos/$REPO/releases/assets/$CHECKSUM_ID" \
  -o "$TEMP/checksums.txt"
python3 - "$TEMP/assets.json" "$TEMP/checksums.txt" <<'PY'
import hashlib,json,re,sys
with open(sys.argv[1],encoding="utf8") as f: assets=json.load(f)
with open(sys.argv[2],"rb") as f: checksum_bytes=f.read()
by_name={a["name"]:a for a in assets}
assert len(assets)==21 and len(by_name)==21,"incomplete/duplicate release asset inventory"
check_name="GhostFTP-v0.99.36-SHA256SUMS.txt"
assert by_name[check_name].get("digest")==("sha256:"+hashlib.sha256(checksum_bytes).hexdigest()),"checksum manifest remote digest mismatch"
expected_lines=checksum_bytes.decode("utf8").splitlines()
assert len(expected_lines)==20,"expected 20 checksummed desktop assets"
found=set()
for line in expected_lines:
    m=re.fullmatch(r"([a-f0-9]{64})  ([A-Za-z0-9._-]+)",line)
    assert m,"invalid checksum record"
    digest,name=m.groups()
    assert name.startswith("GhostFTP-") and name not in found and name!=check_name,"unsafe/duplicate asset name"
    found.add(name)
    asset=by_name.get(name)
    assert asset and asset.get("state")=="uploaded" and asset.get("size",0)>0,"incomplete desktop asset"
    assert asset.get("digest")=="sha256:"+digest, "remote asset checksum mismatch for "+name
assert found|{check_name}==set(by_name),"draft inventory differs from checksum manifest"
assert all("Android" not in name and not name.endswith((".apk",".aab")) for name in by_name),"retired mobile package found"
print("Verified complete 21-asset draft inventory, checksum manifest and every remote SHA-256.")
PY
jq -n '{draft:false,prerelease:true,make_latest:"false"}' |
  gh api --method PATCH "repos/$REPO/releases/$RELEASE_ID" --input - >/dev/null
test "$(gh api "repos/$REPO/releases/$RELEASE_ID" --jq '.draft')" = false
test "$(gh api "repos/$REPO/releases/$RELEASE_ID" --jq '.prerelease')" = true
echo "Published verified Ghost FTP $TAG Preview; no public release/tag/assets rewritten."
