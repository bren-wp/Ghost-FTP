#!/usr/bin/env bash
set -euo pipefail

# Import only CI-captured Windows pixels from the already published 0.30.19
# native QA artifact. Never replace README with illustration/mockup images.
: "${GITHUB_REPOSITORY:?}"
: "${GH_TOKEN:?}"
artifact_id=11593868391
run_id=37879036243
source_sha=9412fbac440d4231a3ebdb7869a1f171240e9681

actual_sha="$(gh api "repos/${GITHUB_REPOSITORY}/actions/runs/${run_id}" --jq .head_sha)"
test "$actual_sha" = "$source_sha"
actual_artifact_run="$(gh api "repos/${GITHUB_REPOSITORY}/actions/artifacts/${artifact_id}" --jq .workflow_run.id)"
test "$actual_artifact_run" = "$run_id"
published_sha="$(gh api "repos/${GITHUB_REPOSITORY}/git/ref/tags/v0.30.19" --jq .object.sha)"
# Lightweight tags point to the merge commit; verify them before importing.
test "$published_sha" = "$source_sha"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
curl --fail --location --retry 3 --silent --show-error \
  -H "Authorization: Bearer ${GH_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/${GITHUB_REPOSITORY}/actions/artifacts/${artifact_id}/zip" \
  --output "$work/screenshots.zip"
unzip -q "$work/screenshots.zip" -d "$work/screens"

# Current README uses exactly seven <img> paths. Keep its HTML, table,
# gallery position and all other content completely unchanged.
python3 - "$work/screens" <<'PY'
import re, struct, sys
from pathlib import Path

src = Path(sys.argv[1])
dest = Path("docs/assets/screenshots")
mapping = {
    "ghostftp-native-files.png": "GhostFTP-Windows-Preview-Native-Window.png",
    "ghostftp-native-new-connection.png": "GhostFTP-Windows-Preview-New-Connection.png",
    "ghostftp-native-sites.png": "GhostFTP-Windows-Preview-Site-Manager.png",
    "ghostftp-native-transfers.png": "GhostFTP-Windows-Preview-Transfer-Center.png",
    "ghostftp-native-settings.png": "GhostFTP-Windows-Preview-Preferences.png",
    "ghostftp-native-file-properties.png": "GhostFTP-Windows-Preview-File-Properties.png",
    "ghostftp-native-about.png": "GhostFTP-Windows-Preview-About.png",
}
readme = Path("README.md").read_text(encoding="utf-8")
assert len(mapping) == 7
for target, source in mapping.items():
    assert readme.count("docs/assets/screenshots/" + target) == 2, target
    original = dest / target
    photo = src / source
    evidence = photo.with_suffix(".txt")
    assert original.is_file() and photo.is_file() and evidence.is_file(), source
    raw = photo.read_bytes()
    assert raw[:8] == b"\x89PNG\r\n\x1a\n", source
    width, height = struct.unpack(">II", raw[16:24])
    assert (width, height) == (1290, 852), (source, width, height)
    proof = evidence.read_text(encoding="utf-8")
    assert "WindowTitle=Ghost FTP" in proof
    assert "WindowBounds=1290x852" in proof
    assert "CaptureMethod=" in proof and "DecodedColors=" in proof
    original.write_bytes(raw)
assert readme == Path("README.md").read_text(encoding="utf-8")
print("Replaced precisely seven existing README Windows image files with release-native 0.30.19 QA captures.")
PY

git diff --exit-code -- README.md
git status --short
