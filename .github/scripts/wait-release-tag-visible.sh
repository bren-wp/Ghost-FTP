#!/usr/bin/env bash
# Fail-closed bounded eventual-consistency check after a published tag push.
# No tag, release or asset mutation is performed here.
set -euo pipefail
TAG="${1:?release tag required}"
EXPECTED_SHA="${2:?source SHA required}"
[[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "::error::Invalid release tag" >&2; exit 1; }
[[ "$EXPECTED_SHA" =~ ^[a-f0-9]{40}$ ]] || { echo "::error::Invalid source SHA" >&2; exit 1; }
[[ "${GITHUB_REPOSITORY:-}" = "bren-wp/Ghost-FTP" ]] || { echo "::error::Unexpected repository" >&2; exit 1; }

for attempt in 1 2 3 4 5 6 7 8; do
  if OBSERVED="$(gh api "repos/$GITHUB_REPOSITORY/git/ref/tags/$TAG" --jq '.object.sha' 2>&1)"; then
    if [ "$OBSERVED" = "$EXPECTED_SHA" ]; then
      echo "Verified immutable tag $TAG at $EXPECTED_SHA (attempt $attempt/8)."
      exit 0
    fi
    echo "::error::Visible $TAG points to '$OBSERVED', not '$EXPECTED_SHA'; refusing rewrite." >&2
    exit 1
  fi
  if [[ "$OBSERVED" != *"HTTP 404"* && "$OBSERVED" != *"Not Found"* ]]; then
    echo "::error::Non-retryable Git tag lookup failure for $TAG: $OBSERVED" >&2
    exit 1
  fi
  if [ "$attempt" -lt 8 ]; then
    echo "::notice::$TAG not visible in Git-ref API ($attempt/8); retrying."
    sleep "$attempt"
  fi
done
echo "::error::$TAG not visible after 8 attempts; preserving immutable tag." >&2
exit 1
