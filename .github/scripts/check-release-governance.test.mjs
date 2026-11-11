import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { execFileSync } from "node:child_process";
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
  // build; a path-filtered build would never trigger workflow_run publication.
  const native = read(".github/workflows/ghostftp-build.yml");
  const pushBlock = native.split("  push:")[1].split("  pull_request:")[0];
  assert.ok(pushBlock.includes("branches: [main]"));
  assert.ok(!pushBlock.includes("paths:"));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-preview-metadata.yml"'),
    "publication must fail closed when source-matched historical release metadata audit fails");
  assert.ok(workflow.includes('wait_for_gate "ghostftp-quality.yml"'));
  assert.ok(workflow.includes('wait_for_gate "ghostftp-protocol-e2e.yml"'));
  assert.ok(workflow.includes('wait_for_gate "validate-win-hardening.yml"'));
  assert.ok(!workflow.includes('ghostftp-server.yml'), "retired standalone server must not be a release gate");
  assert.ok(workflow.includes("RECOVERY_STATE"));
  assert.ok(workflow.includes("false:true:21"));
  assert.ok(workflow.includes("Refusing v0.95.52 while v0.95.51"));

});

test("merged changes advance version, except exact unpublished release repairs with immutable lineage", () => {
  const quality = read(".github/workflows/ghostftp-quality.yml");
  assert.ok(quality.includes('BASE_VERSION'));
  assert.ok(quality.includes('PREVIOUS_SOURCE_VERSION'));
  assert.ok(quality.includes('PREVIOUS_PUBLISHED_VERSION'));
  assert.ok(quality.includes('The previous main version'));
  assert.ok(quality.includes('Refusing another PR for an already tagged/published version'));
  assert.ok(quality.includes('Unpublished version repair cannot skip a canonical published predecessor'));
  assert.ok(quality.includes('PREVIOUS_SOURCE_VERSION" = "$PREVIOUS_PUBLISHED_VERSION"'));
  assert.ok(quality.includes('repos/$GITHUB_REPOSITORY/git/ref/tags/v$CURRENT_VERSION'));
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
  const safeFileGate = '[[ -f "$file" && ! -L "$file" ]] || continue';
  assert.equal(script.split(safeFileGate).length - 1, 2,
    "both upload and digest loops must skip staging directories and symlinks");
  assert.ok(script.includes("find dist -maxdepth 1 -type f"),
    "public inventory must include only regular files");
  assert.ok(script.includes("Reusing verified draft asset"), "reruns must preserve matching draft uploads");
  assert.ok(script.includes('Remote SHA256 mismatch'), "all remote asset digests remain mandatory");
  assert.ok(!script.includes('gh api --method DELETE "repos/$GITHUB_REPOSITORY/releases/assets/$ASSET_ID"'), "draft assets must not be deleted wholesale on retry");
  assert.ok(read(".github/workflows/ghostftp-release.yml").includes('select(.event == "push")'), "release gates must ignore skipped metadata follow-up events");
});

test("draft asset file iteration cannot hash a staging directory or follow symlinks", () => {
  const temp = fs.mkdtempSync(path.join(process.cwd(), ".ghostftp-release-glob-"));
  try {
    const dist = path.join(temp, "dist");
    fs.mkdirSync(dist);
    fs.writeFileSync(path.join(dist, "GhostFTP-test.zip"), "real release bytes");
    fs.mkdirSync(path.join(dist, "native"));
    fs.writeFileSync(path.join(dist, "native", "secret.txt"), "not a release asset");
    fs.symlinkSync(path.join(dist, "native", "secret.txt"), path.join(dist, "GhostFTP-leak.zip"));
    const output = execFileSync("bash", [
      "-c",
      'for file in "$1"/*; do [[ -f "$file" && ! -L "$file" ]] || continue; basename "$file"; done',
      "release-glob-test",
      dist,
    ], { encoding: "utf8" }).trim().split("\n");
    assert.deepEqual(output, ["GhostFTP-test.zip"]);
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});

test("released editions and tags must never be automatically pruned", () => {
  assert.equal(fs.existsSync(path.join(root, ".github/workflows/ghostftp-release-retention.yml")), false);
  assert.equal(fs.existsSync(path.join(root, ".github/scripts/prune-release-history-keep10.sh")), false);
  for (const file of fs.readdirSync(path.join(root, ".github/workflows"))) {
    if (!/\.ya?ml$/.test(file)) continue;
    const body = read(".github/workflows/" + file);
    assert.ok(!body.includes("gh release delete"), file + " must preserve releases");
    assert.ok(!body.includes("--cleanup-tag"), file + " must preserve tags");
  }
});

test("version synchronization updates only local Cargo workspace package records", () => {
  const script = read(".github/scripts/sync-version.mjs");
  const verifier = read(".github/scripts/refresh-cargo-lock.sh");
  for (const name of ["ghostftp", "ghostftp-agent-proto", "ghostftp-agentd"]) {
    assert.ok(script.includes(`"${name}"`));
  }
  assert.ok(script.includes('const rel = "ghostftp-desktop/src-tauri/Cargo.lock"'));
  assert.ok(script.includes("lockedVersion !== version && lockedVersion !== previousVersion"));
  assert.ok(script.includes("found !== workspacePackages.size"));
  assert.ok(script.includes("apply(rel, lock)"));
  assert.ok(verifier.includes("cargo metadata --locked"));
  assert.ok(!verifier.includes("cargo update"));
});

test("Vite stays patched without a time-limited version-reuse exception", () => {
  const quality = read(".github/workflows/ghostftp-quality.yml");
  const pkg = JSON.parse(read("ghostftp-desktop/package.json"));
  const lock = JSON.parse(read("ghostftp-desktop/package-lock.json"));
  assert.equal(pkg.devDependencies.vite, "8.3.4");
  assert.equal(lock.packages["node_modules/vite"].version, "8.3.4");
  assert.ok(!quality.includes("Recovering previously unpublished 0.95.53"));
  assert.ok(quality.includes("Refusing another PR for an already tagged/published version"));
  assert.ok(quality.includes("Unpublished version repair cannot skip a canonical published predecessor"));
});

test("all Preview releases remain immutable and cannot be classified stable", () => {
  const publishing=read(".github/scripts/publish-release.sh");
  const repair=read(".github/scripts/enforce-preview-release-metadata.sh");
  const workflow=read(".github/workflows/ghostftp-preview-metadata.yml");
  assert.ok(publishing.includes('make_latest:"false"'));
  assert.ok(publishing.includes("draft:false,prerelease:true"));
  assert.ok(repair.includes('make_latest:"false"'));
  assert.ok(repair.includes('node .github/scripts/validate-release-assets.mjs "$version" --list-required'), "must select version-specific immutable asset manifest");
  assert.ok(repair.includes('test "$(jq -r \'.assets | length\' <<< "$release")" -ge "${#required_assets[@]}"'), "legacy and modern releases must preserve their exact required asset count");
  assert.ok(repair.includes("sha256:[a-f0-9]{64}"));
  assert.ok(repair.includes("0.99.0"));
  assert.ok(repair.includes("0.95.53"));
  assert.ok(repair.includes('enforce_preview "0.99.1" "eb2e1899f0bb953b731e7b41e0109ba4decb324a"'));
  assert.ok(repair.includes("GHOSTFTP_PUBLISHED_SOURCE_SHA"));
  assert.ok(workflow.includes('workflows: ["Ghost FTP release"]'));
  assert.ok(workflow.includes("github.event.workflow_run.conclusion == 'success'"));
  assert.ok(workflow.includes("branches: [main]"));
  assert.ok(workflow.includes("cron: '*/15 * * * *'"),"public release classification must be reverified after push events");
  assert.ok(workflow.includes("github.event_name == 'schedule'"));
  assert.ok(repair.includes("sort_by(.id)"),"metadata-only correction must verify unchanged public asset digests");
  assert.ok(repair.includes("0.99.20") && repair.includes("0.99.21"));
  assert.ok(repair.includes("audit_incomplete_v09924"), "missing 0.99.24 integrity quarantine");
  assert.ok(repair.includes('sort_by(.id)'), "quarantined assets must use immutable IDs");
  assert.ok(repair.includes("v0.99.24-observed-assets.json"), "quarantine must use pinned incident evidence");
  assert.ok(repair.includes("asset inventory changed"), "quarantine must fail on unexpected historical mutation");
  assert.ok(!repair.includes('enforce_preview "0.99.24"'), "never reclassify a release with missing assets");
  assert.ok(repair.includes('enforce_preview "0.99.25" "2e00a640c65dcd4d73c0401071703c9312eb7881"'),
    "complete v0.99.25 Preview must be verified and classified correctly");
  for (const [filename, source] of [["publisher", publishing], ["metadata corrector",repair],["metadata workflow",workflow]]) {
    assert.ok(!source.includes("--cleanup-tag"), filename);
    assert.ok(!source.includes("gh release delete"), filename);
    assert.ok(!source.includes("git push origin :refs/tags/"), filename);
    if (filename !== "publisher") assert.ok(!source.includes("releases/assets/$ASSET_ID"), filename);
  }
});

test("canonical latest release is determined by numeric immutable version, never PATCHable publish date", () => {
  for (const name of ["ghostftp-quality.yml", "ghostftp-release.yml"]) {
    const yaml = read(".github/workflows/" + name);
    assert.ok(yaml.includes('sort_by(.tag_name | ltrimstr("v") | split(".") | map(tonumber))'),name);
    assert.ok(!yaml.includes('sort_by(.published_at)'),name);
  }
});

test("incomplete historical v0.99.24 snapshot is explicit, pinned and fail-closed", () => {
  const pinned = JSON.parse(read(".github/scripts/v0.99.24-observed-assets.json"));
  const allIDs = pinned.map(x => x.id);
  const allNames = pinned.map(x => x.name);
  assert.equal(pinned.length, 13);
  assert.equal(new Set(allIDs).size, pinned.length);
  assert.equal(new Set(allNames).size, pinned.length);
  assert.deepEqual([...allIDs].sort((a,b)=>a-b), allIDs);
  assert.ok(pinned.every(x => Number.isSafeInteger(x.id) && x.id>0 &&
    Number.isSafeInteger(x.size) && x.size>0 &&
    /^sha256:[0-9a-f]{64}$/.test(x.digest) &&
    x.name.includes("0.99.24")));
  const script=read(".github/scripts/enforce-preview-release-metadata.sh");
  assert.ok(script.includes('if [ "$actual" != "$expected" ]; then'));
  assert.ok(script.includes("manual provenance review required"));
  assert.ok(script.includes("Metadata was not modified"));
});
