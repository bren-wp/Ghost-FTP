#!/usr/bin/env node
/**
 * Shared, fail-closed release artifact contract. Used before publishing and
 * after publication, never hardcodes an obsolete release asset count.
 * A valid GH API digest is matched to the downloaded SHA256SUMS content.
 */
import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const contract = JSON.parse(fs.readFileSync(path.join(root, ".github/scripts/release-artifacts.json"), "utf8"));
const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");
const fail = (message) => { throw new Error("Release asset verification: " + message); };
const ensure = (condition, message) => { if (!condition) fail(message); };

export function expectedReleaseAssets(version) {
  ensure(/^\d+\.\d+\.\d+$/.test(version), "invalid semantic version");
  ensure(contract.schema === 1, "unsupported release artifact contract schema");
  const expand = (template) => template.replaceAll("{version}", version);
  const required = contract.required.map(expand);
  const optional = contract.optional_updater.map(expand);
  ensure(required.length > 0, "empty required asset inventory");
  ensure(new Set([...required, ...optional]).size === required.length + optional.length,
    "duplicate names in release asset contract");
  ensure(required.every((name) => /^[A-Za-z0-9._-]+$/.test(name)) &&
    optional.every((name) => /^[A-Za-z0-9._-]+$/.test(name)),
    "invalid asset name in release contract");
  return { required, optional, checksum: `GhostFTP-v${version}-SHA256SUMS.txt` };
}

function expectedSet(version, observedNames) {
  const { required, optional, checksum } = expectedReleaseAssets(version);
  const names = new Set(observedNames);
  ensure(names.size === observedNames.length, "duplicate uploaded asset name");
  const optionalPresent = optional.filter((name) => names.has(name));
  ensure(optionalPresent.length === 0 || optionalPresent.length === optional.length,
    `partial signed updater group: ${optionalPresent.length}/${optional.length}`);
  const expected = [...required, ...(optionalPresent.length ? optional : [])];
  const missing = expected.filter((name) => !names.has(name));
  const extra = observedNames.filter((name) => !expected.includes(name));
  ensure(missing.length === 0 && extra.length === 0,
    `asset inventory mismatch: missing=[${missing.join(', ')}], unexpected=[${extra.join(', ')}]`);
  ensure(expected.includes(checksum), "SHA256SUMS file missing from contract");
  return { expected, checksum };
}

function checksumEntries(text) {
  ensure(text.endsWith("\n"), "SHA256SUMS must end in a newline");
  const entries = new Map();
  for (const line of text.trimEnd().split("\n")) {
    const match = /^([0-9a-f]{64})  ([A-Za-z0-9._-]+)$/.exec(line);
    ensure(match !== null, `invalid SHA256SUMS line: ${line.slice(0, 150)}`);
    ensure(!entries.has(match[2]), `duplicate checksum entry: ${match[2]}`);
    entries.set(match[2], match[1]);
  }
  return entries;
}

export function validateReleaseAssets(version, records, checksumBytes) {
  ensure(Array.isArray(records), "release asset list is not an array");
  const { expected, checksum } = expectedSet(version, records.map((a) => a.name));
  const byName = new Map(records.map((asset) => [asset.name, asset]));
  for (const asset of records) {
    ensure(Number.isSafeInteger(asset.size) && asset.size > 0,
      `empty or invalid asset size: ${asset.name}`);
    ensure(/^sha256:[a-f0-9]{64}$/.test(asset.digest ?? ""),
      `missing/invalid GitHub SHA-256 digest: ${asset.name}`);
    if (asset.state !== undefined) {
      ensure(asset.state === "uploaded", `incomplete upload state: ${asset.name}`);
    }
  }
  ensure(Buffer.isBuffer(checksumBytes), "SHA256SUMS bytes not provided");
  ensure(checksumBytes.length === byName.get(checksum).size,
    "SHA256SUMS published size differs from downloaded bytes");
  ensure(sha256(checksumBytes) === byName.get(checksum).digest.slice(7),
    "SHA256SUMS downloaded content does not match GitHub SHA-256 digest");
  const entries = checksumEntries(checksumBytes.toString("utf8"));
  const expectedChecksummed = expected.filter((name) => name !== checksum);
  const missing = expectedChecksummed.filter((name) => !entries.has(name));
  const extra = [...entries.keys()].filter((name) => !expectedChecksummed.includes(name));
  ensure(missing.length === 0 && extra.length === 0,
    `SHA256SUMS inventory mismatch: missing=[${missing.join(', ')}], unexpected=[${extra.join(', ')}]`);
  for (const [name, digest] of entries) {
    ensure(byName.get(name).digest === "sha256:" + digest,
      `SHA-256 mismatch between GitHub and SHA256SUMS for ${name}`);
  }
  return { assetCount: expected.length, optionalUpdater: expected.length > contract.required.length };
}

export function validateLocalReleaseDirectory(version, directory) {
  const records = [];
  for (const name of fs.readdirSync(directory)) {
    const file = path.join(directory, name);
    const stat = fs.lstatSync(file);
    if (stat.isDirectory()) continue; // staging dirs are not public artifacts
    ensure(stat.isFile() && !stat.isSymbolicLink(),
      `non-regular or symlinked asset in dist/: ${name}`);
    records.push({ name, size: stat.size, digest: "sha256:" + sha256(fs.readFileSync(file)) });
  }
  const checksum = expectedReleaseAssets(version).checksum;
  const checksumFile = path.join(directory, checksum);
  ensure(fs.existsSync(checksumFile), "missing SHA256SUMS file");
  return validateReleaseAssets(version, records, fs.readFileSync(checksumFile));
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const [version, mode, value, arg4, arg5] = process.argv.slice(2);
    if (mode === "--list-required" || mode === "--list-optional") {
      const names = expectedReleaseAssets(version)[mode === "--list-required" ? "required" : "optional"];
      process.stdout.write(names.join("\n") + "\n");
    } else if (mode === "--dist" && value) {
      const result = validateLocalReleaseDirectory(version, value);
      console.log(`Verified local release manifest: ${result.assetCount} exact assets (signed updater: ${result.optionalUpdater}).`);
    } else if (mode === "--release-json" && value && arg4 === "--checksums" && arg5) {
      const release = JSON.parse(fs.readFileSync(value, "utf8"));
      ensure(release.draft === false, "release remains Draft");
      ensure(release.prerelease === true, "Preview must be classified as prerelease");
      ensure(release.tag_name === "v" + version, "release tag mismatch");
      const result = validateReleaseAssets(version, release.assets,
        fs.readFileSync(arg5));
      console.log(`Verified public Preview: ${result.assetCount} exact assets; all names, sizes and remote SHA-256 digests match SHA256SUMS.`);
    } else {
      fail("usage: VERSION (--list-required|--list-optional|--dist DIRECTORY|--release-json FILE --checksums FILE)");
    }
  } catch (error) {
    console.error("::error::" + error.message);
    process.exitCode = 1;
  }
}
