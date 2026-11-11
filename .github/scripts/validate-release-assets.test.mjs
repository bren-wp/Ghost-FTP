import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { expectedReleaseAssets, validateLocalReleaseDirectory, validateReleaseAssets } from "./validate-release-assets.mjs";

const hash = (bytes) => createHash("sha256").update(bytes).digest("hex");
const version = "0.99.47";
function fixture(signed = false) {
  const { required, optional, checksum } = expectedReleaseAssets(version);
  const names = [...required.filter(n => n !== checksum), ...(signed ? optional : [])];
  const records = names.map((name) => {
    const bytes = Buffer.from("genuine bytes for " + name);
    return { name, size: bytes.length, digest: "sha256:" + hash(bytes), state: "uploaded" };
  });
  const sums = Buffer.from(records.map(a => a.digest.slice(7) + "  " + a.name).join("\n") + "\n");
  records.push({ name: checksum, size: sums.length, digest: "sha256:" + hash(sums), state: "uploaded" });
  return { records, sums, checksum };
}
test("manifest covers the exact current 15-file Windows/Linux desktop baseline and optional complete signer group", () => {
  assert.equal(expectedReleaseAssets(version).required.length, 15);
  assert.equal(expectedReleaseAssets(version).optional.length, 4);
  assert.equal(validateReleaseAssets(version, ...Object.values(fixture(false)).slice(0,2)).assetCount, 15);
  assert.equal(validateReleaseAssets(version, ...Object.values(fixture(true)).slice(0,2)).assetCount, 19);
});
test("prior release manifests remain immutable with the original 21-asset inventory", () => {
  const previousVersion = "0.99.46";
  const { required, optional, checksum } = expectedReleaseAssets(previousVersion);
  assert.equal(required.length, 21);
  assert.equal(optional.length, 4);
  assert.ok(required.some(name => name.includes("macOS")), "historic published platform must remain verifiable");
  const assets = required.filter(name => name !== checksum).map(name => {
    const bytes = Buffer.from("historical bytes: " + name);
    return { name, size: bytes.length, digest: "sha256:" + hash(bytes), state: "uploaded" };
  });
  const sums = Buffer.from(assets.map(asset => asset.digest.slice(7) + "  " + asset.name).join("\n") + "\n");
  assets.push({ name: checksum, size: sums.length, digest: "sha256:" + hash(sums), state: "uploaded" });
  assert.equal(validateReleaseAssets(previousVersion, assets, sums).assetCount, 21);
  assert.throws(() => validateReleaseAssets(version, assets, sums), /asset inventory mismatch/);
  assert.throws(() => validateReleaseAssets(previousVersion, assets.slice(1), sums), /asset inventory mismatch/);
  assert.throws(() => validateReleaseAssets(previousVersion, assets.map((asset,i) => i === 0 ? {...asset, digest:"sha256:"+"0".repeat(64)} : asset), sums), /SHA-256 mismatch/);
});

test("missing, extra and partial signed updater artifacts are rejected", () => {
  const { records, sums } = fixture();
  assert.throws(() => validateReleaseAssets(version, records.slice(1), sums), /missing/);
  assert.throws(() => validateReleaseAssets(version, [...records, {...records[0], name:"android.apk"}], sums), /unexpected/);
  assert.throws(() => validateReleaseAssets(version, [...records, {name:expectedReleaseAssets(version).optional[0],size:1,digest:"sha256:"+ "a".repeat(64)}], sums), /partial signed updater/);
  assert.throws(() => validateReleaseAssets(version, [...records, records[0]], sums), /duplicate/);
});
test("detects invalid GitHub digest, empty bytes, and corrupt checksum records", () => {
  const { records, sums } = fixture();
  assert.throws(() => validateReleaseAssets(version, records.map((a,i) => i === 0 ? {...a,size:0} : a), sums), /size/);
  assert.throws(() => validateReleaseAssets(version, records.map((a,i) => i === 0 ? {...a,digest:null} : a), sums), /digest/);
  assert.throws(() => validateReleaseAssets(version, records, Buffer.from(sums.toString().replace(/[0-9a-f]/, "x"))), /size|SHA256SUMS/);
  const badDigest = records.map((a,i) => i === 0 ? {...a,digest:"sha256:"+ "f".repeat(64)} : a);
  assert.throws(() => validateReleaseAssets(version, badDigest, sums), /SHA-256 mismatch/);
});
test("local dist validates real content, rejects symlink and ignores staging directories", () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-release-contract-"));
  try {
    const { records, sums, checksum } = fixture();
    for (const asset of records) {
      fs.writeFileSync(path.join(tmp, asset.name), asset.name === checksum ? sums : "genuine bytes for " + asset.name);
    }
    fs.mkdirSync(path.join(tmp, "native"));
    fs.writeFileSync(path.join(tmp, "native", "do-not-release"), "secret");
    assert.equal(validateLocalReleaseDirectory(version, tmp).assetCount, 15);
    fs.symlinkSync(path.join(tmp, "native", "do-not-release"), path.join(tmp, "leak.txt"));
    assert.throws(() => validateLocalReleaseDirectory(version, tmp), /symlinked/);
    fs.unlinkSync(path.join(tmp, "leak.txt"));
    fs.writeFileSync(path.join(tmp, records[0].name), "tampered");
    assert.throws(() => validateLocalReleaseDirectory(version, tmp), /SHA-256 mismatch/);
  } finally { fs.rmSync(tmp, {recursive:true, force:true}); }
});

test("metadata repair accepts false Preview ONLY with verified complete immutable assets, never as public release", () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-preview-repair-"));
  try {
    const { records, sums } = fixture();
    const releasePath = path.join(tmp, "release.json");
    const checksumPath = path.join(tmp, "sums.txt");
    const script = fileURLToPath(new URL("./validate-release-assets.mjs", import.meta.url));
    fs.writeFileSync(checksumPath, sums);
    const verify = (mode) => spawnSync(process.execPath,
      [script, version, mode, releasePath, "--checksums", checksumPath], { encoding: "utf8" });
    let release = { draft: false, prerelease: false, tag_name: "v" + version, assets: records };
    fs.writeFileSync(releasePath, JSON.stringify(release));
    assert.equal(verify("--metadata-repair-json").status, 0, "misclassified complete release may be repaired");
    assert.notEqual(verify("--release-json").status, 0, "strict publication gate must reject false Preview");
    release = {...release, draft: true};
    fs.writeFileSync(releasePath, JSON.stringify(release));
    assert.notEqual(verify("--metadata-repair-json").status, 0, "Draft cannot be repaired as published");
    release = {...release, draft: false, assets: records.slice(1)};
    fs.writeFileSync(releasePath, JSON.stringify(release));
    assert.notEqual(verify("--metadata-repair-json").status, 0, "incomplete published inventory cannot be repaired");
    release = {...release, assets: records, prerelease: true};
    fs.writeFileSync(releasePath, JSON.stringify(release));
    assert.equal(verify("--release-json").status, 0, "strict publication gate accepts complete Preview");
    assert.equal(verify("--metadata-repair-json").status, 0, "idempotent repair accepts true Preview");
    fs.writeFileSync(checksumPath, Buffer.from("corrupt data\n"));
    assert.notEqual(verify("--metadata-repair-json").status, 0, "tampered checksums block repair");
  } finally {
    fs.rmSync(tmp, {recursive:true, force:true});
  }
});
