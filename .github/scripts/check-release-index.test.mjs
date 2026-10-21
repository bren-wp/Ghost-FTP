import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import {fileURLToPath} from "node:url";
import test from "node:test";

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),"../..");
const read=(p)=>fs.readFileSync(path.join(root,p),"utf8");

test("post-publication verifies newest Preview is visible on public Releases index",()=>{
  const verify=read(".github/scripts/check-release-index.sh");
  const workflow=read(".github/workflows/ghostftp-preview-metadata.yml");
  assert.ok(verify.includes("https://github.com/$REPO/releases"));
  assert.ok(verify.includes('repos/$REPO/releases?per_page=100'));
  assert.ok(verify.includes("--paginate --slurp"));
  assert.ok(verify.includes('repos/$REPO/releases?per_page=30'));
  assert.ok(verify.includes('test "$LATEST_TAG" = "$TAG"'));
  assert.ok(verify.includes('sort_by(ltrimstr("v") | split(".") | map(tonumber))'));
  assert.ok(verify.includes("sha256:[a-f0-9]{64}"));
  assert.ok(verify.includes("GhostFTP-Android-v$VERSION.aab.unsigned"));
  assert.ok(verify.includes("GhostFTP-macOS-v$VERSION-Preview.zip"));
  assert.ok(verify.includes("GhostFTP-macOS-Shared-arm64-v$VERSION-Preview.zip"));
  assert.ok(verify.includes("GhostFTP-macOS-Shared-x86_64-v$VERSION-Preview.zip"));
  assert.ok(workflow.includes("bash .github/scripts/check-release-index.sh"));
  assert.ok(workflow.includes("GHOSTFTP_PUBLISHED_SOURCE_SHA"));
  for(const text of ["gh release delete","--cleanup-tag","git push origin :refs/tags/"]){
    assert.ok(!verify.includes(text));
  }
});

test("README links to all public Previews, never stable-only /releases/latest",()=>{
  const readme=read("README.md");
  assert.ok(readme.includes("[**Releases**](https://github.com/bren-wp/Ghost-FTP/releases)"));
  assert.ok(!readme.includes("releases/latest"));
  const version = JSON.parse(read("version.json"));
  assert.ok(readme.includes(`Ghost FTP ${version.version} (development Preview)`));
  assert.ok(readme.includes(`Previous verified edition: v${version.previousPublishedVersion}.`));
});
