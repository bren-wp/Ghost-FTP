#!/usr/bin/env bash
# Ghost FTP by Brendigo
# Owner-only GitHub branch protection setup: never publish credentials.
set -euo pipefail

REPO="bren-wp/Ghost-FTP"
BRANCH="main"
ACTION="${1:---check}"
if [[ "$ACTION" != "--check" && "$ACTION" != "--apply" ]]; then
  echo "Usage: $0 [--check|--apply]" >&2
  exit 2
fi
for cmd in gh jq; do
  command -v "$cmd" >/dev/null || { echo "$cmd is required" >&2; exit 2; }
done

# Exact GitHub Actions check-run names from the verified v0.99.71 main SHA.
required_checks=(
  "Rust workspace"
  "Product + compatibility tooling"
  "build (windows-latest, --bundles nsis,msi, GhostFTP-Windows-x64-Preview)"
  "build (ubuntu-22.04, --bundles deb,rpm,appimage, GhostFTP-Linux-x86_64-Preview)"
  "Real FTP / FTPS / SFTP roundtrip"
  "Real Windows FTP upload/download roundtrip"
  "Windows filesystem security regressions"
  "Desktop security gate"
  "windows"
)

verify() {
  local protected check
  protected="$(gh api "repos/$REPO/branches/$BRANCH" --jq '.protected')"
  if [[ "$protected" != "true" ]]; then
    echo "UNPROTECTED: $REPO/$BRANCH. Issue #302 remains open." >&2
    return 1
  fi
  local payload
  payload="$(gh api "repos/$REPO/branches/$BRANCH/protection")"
  jq -e '
    .required_status_checks.strict == true and
    .enforce_admins.enabled == true and
    .required_conversation_resolution.enabled == true and
    .allow_force_pushes.enabled == false and
    .allow_deletions.enabled == false and
    .required_pull_request_reviews != null
  ' <<<"$payload" >/dev/null || {
    echo "Protection settings are incomplete. Issue #302 remains open." >&2
    return 1
  }
  local actual
  actual="$(jq -r '.required_status_checks.contexts[]' <<<"$payload")"
  for check in "${required_checks[@]}"; do
    if ! grep -Fxq -- "$check" <<<"$actual"; then
      echo "Missing mandatory context: $check" >&2
      return 1
    fi
  done
  echo "GitHub reports main protected and all required check contexts configured."
}

if [[ "$ACTION" == "--check" ]]; then
  verify
  exit $?
fi

echo "Applying required GitHub branch protection to $REPO/$BRANCH as the authenticated repository administrator."
echo "This requires a token with repository Administration:write permission. No tokens are stored in this repository."
checks_json="$(printf '%s\n' "${required_checks[@]}" | jq -R . | jq -s .)"
payload="$(mktemp)"
trap 'rm -f -- "$payload"' EXIT
chmod 600 "$payload"
jq -n --argjson checks "$checks_json" '{
  required_status_checks: {strict: true, contexts: $checks},
  enforce_admins: true,
  required_pull_request_reviews: {
    dismiss_stale_reviews: true,
    require_code_owner_reviews: false,
    required_approving_review_count: 0
  },
  restrictions: null,
  required_conversation_resolution: true,
  allow_force_pushes: false,
  allow_deletions: false,
  required_linear_history: false,
  lock_branch: false
}' >"$payload"

gh api --method PUT "repos/$REPO/branches/$BRANCH/protection" \
  -H "Accept: application/vnd.github+json" --input "$payload" >/dev/null
verify
echo "Next: verify a deliberately failing PR cannot merge, while an exact-head green PR can merge. Close issue #302 only after both are evidenced."
