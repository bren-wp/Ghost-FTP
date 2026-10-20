import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (file) => fs.readFileSync(path.join(root, file), "utf8");

test("release publishing is opt-in, main-only and never auto-triggered", () => {
  const workflow = read(".github/workflows/ghostftp-release.yml");
  assert.ok(workflow.includes("  workflow_dispatch:"));
  assert.ok(!workflow.includes("  workflow_run:"));
  assert.ok(workflow.includes("inputs.publish_preview == true"));
  assert.ok(workflow.includes("github.ref == 'refs/heads/main'"));
  assert.ok(workflow.includes("CURRENT_MAIN_SHA"));
  assert.ok(workflow.includes("needs.select-trigger.outputs.native_run_id"));
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
