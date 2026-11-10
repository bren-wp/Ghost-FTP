import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { expectedReleaseAssets, validateLocalReleaseDirectory, validateReleaseAssets } from "./validate-release-assets.mjs";

const hash = (bytes) => createHash("sha256").update(bytes).digest("hex");
const version = "0.99.39";
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
test("manifest covers the exact current 21-file desktop baseline and optional complete signer group", () => {
  assert.equal(expectedReleaseAssets(version).required.length, 21);
  assert.equal(expectedReleaseAssets(version).optional.length, 4);
  assert.equal(validateReleaseAssets(version, ...Object.values(fixture(false)).slice(0,2)).assetCount, 21);
  assert.equal(validateReleaseAssets(version, ...Object.values(fixture(true)).slice(0,2)).assetCount, 25);
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
    assert.equal(validateLocalReleaseDirectory(version, tmp).assetCount, 21);
    fs.symlinkSync(path.join(tmp, "native", "do-not-release"), path.join(tmp, "leak.txt"));
    assert.throws(() => validateLocalReleaseDirectory(version, tmp), /symlinked/);
    fs.unlinkSync(path.join(tmp, "leak.txt"));
    fs.writeFileSync(path.join(tmp, records[0].name), "tampered");
    assert.throws(() => validateLocalReleaseDirectory(version, tmp), /SHA-256 mismatch/);
  } finally { fs.rmSync(tmp, {recursive:true, force:true}); }
});
