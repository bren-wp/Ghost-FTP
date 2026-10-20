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
  assert.ok(workflow.includes('wait_for_gate "ghostftp-quality.yml"'));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-protocol-e2e.yml"'));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-android.yml"'));
  assert.ok(workflow.includes('wait_for_gate "validate-win-hardening.yml"'));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-macos.yml"'));
});

test("every merge must advance the canonical version before auto-publication", () => {
  const quality = read(".github/workflows/ghostftp-quality.yml");
  assert.ok(quality.includes('BASE_VERSION'));
  assert.ok(quality.includes('Every merged change requires a new version'));
  assert.ok(quality.includes('--check-next "$BASE_VERSION"'));
  assert.ok(read("docs/release/PROCESS.md").includes("new version"));
});
test("public release occurs only after complete uploaded asset verification", () => {
  const script = read(".github/scripts/publish-release.sh");
  const draft = script.indexOf("gh release create");
  const upload = script.indexOf('gh release upload "$TAG"');
  const verify = script.indexOf('test "$REMOTE_COUNT" = "$LOCAL_COUNT"');
  const publish = script.indexOf('gh release edit "$TAG" --draft=false --prerelease');
  assert.ok(draft >= 0 && draft < upload && upload < verify && verify < publish);
  assert.ok(script.includes("--prerelease --draft"));
  assert.ok(script.includes("Published $TAG already exists; refusing to overwrite"));
  assert.ok(script.includes("'.prerelease'"));
});
