import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (file) => fs.readFileSync(path.join(root, file), "utf8");

test("every passing main native build can automatically trigger a release", () => {
  const workflow = read(".github/workflows/ghostftp-release.yml");
  assert.ok(workflow.includes("  workflow_run:"));
  assert.ok(workflow.includes('workflows: ["Ghost FTP native build"]'));
  assert.ok(!workflow.includes("  workflow_dispatch:"));
  assert.ok(workflow.includes("github.event.workflow_run.conclusion == 'success'"));
  assert.ok(workflow.includes("github.event.workflow_run.head_branch == 'main'"));
  assert.ok(workflow.includes("github.event.workflow_run.event == 'push'"));
  assert.ok(workflow.includes("github.event.workflow_run.head_repository.full_name == github.repository"));
  assert.ok(workflow.includes("CURRENT_MAIN_SHA"));
  assert.ok(workflow.includes("LATEST_ID"));
  assert.ok(workflow.includes("TRIGGER_RUN_ID"));
  assert.ok(workflow.includes("needs.select-trigger.outputs.native_run_id"));
  // Android-only, macOS-only and documentation changes must also create a native
  // build; a path-filtered build would never trigger workflow_run publication.
  const native = read(".github/workflows/ghostftp-build.yml");
  const pushBlock = native.split("  push:")[1].split("  pull_request:")[0];
  assert.ok(pushBlock.includes("branches: [main]"));
  assert.ok(!pushBlock.includes("paths:"));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-quality.yml"'));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-protocol-e2e.yml"'));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-android.yml"'));
  assert.ok(workflow.includes('wait_for_gate "validate-win-hardening.yml"'));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-macos.yml"'));
});

test("every merge must advance the canonical version before auto-publication", () => {
  const quality = read(".github/workflows/ghostftp-quality.yml");
  assert.ok(quality.includes('BASE_VERSION'));
  assert.ok(quality.includes('PREVIOUS_SOURCE_VERSION'));
  assert.ok(quality.includes('PREVIOUS_PUBLISHED_VERSION'));
  assert.ok(quality.includes('The previous main version'));
  assert.ok(quality.includes('Every merged change requires a new version'));
  assert.ok(quality.includes('--check-next "$BASE_VERSION"'));
  assert.ok(read("docs/release/PROCESS.md").includes("new version"));
});
test("public release follows draft-ID upload, inventory and remote SHA256 verification", () => {
  const script = read(".github/scripts/publish-release.sh");
  const create = script.indexOf('gh api --method POST "repos/$GITHUB_REPOSITORY/releases"');
  const upload = script.indexOf("https://uploads.github.com/repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID/assets");
  const verify = script.indexOf('test "$REMOTE_COUNT" = "$LOCAL_COUNT"');
  const publish = script.indexOf('gh api --method PATCH "repos/$GITHUB_REPOSITORY/releases/$RELEASE_ID"');
  assert.ok(create >= 0 && create < upload && upload < verify && verify < publish);
  assert.ok(script.includes("draft:true,prerelease:true"));
  assert.ok(script.includes("draft:false,prerelease:true"));
  assert.ok(script.includes("Published $TAG already exists; refusing to overwrite"));
  assert.ok(script.includes("Remote SHA256 mismatch"));
  assert.ok(!script.includes('gh release upload "$TAG"'));
  assert.ok(!script.includes('releases/tags/$TAG'));
});
