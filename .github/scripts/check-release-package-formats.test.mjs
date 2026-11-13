import assert from "node:assert/strict";
import test from "node:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { formatForReleaseAsset, validatePackageHeader, validateZipFooter, verifyReleasePackageFormats } from "./check-release-package-formats.mjs";
import { expectedReleaseAssets } from "./validate-release-assets.mjs";

const version = "0.99.58";
const pe = Buffer.alloc(512);
pe.write("MZ");
pe.writeUInt32LE(128, 0x3c);
pe.write("PE\\0\\0".replaceAll("\\0", "\0"), 128, "binary");
pe.writeUInt16LE(0x8664, 132);
pe.writeUInt16LE(1, 134);
pe.writeUInt16LE(0xf0, 148);
pe.writeUInt16LE(0x20b, 152);
const elf = Buffer.alloc(128);
Buffer.from("7f454c46", "hex").copy(elf);
elf[4] = 2;
elf[5] = 1;
elf[6] = 1;
elf.writeUInt16LE(3, 16);
elf.writeUInt16LE(0x3e, 18);
elf.writeUInt32LE(1, 20);
elf.writeBigUInt64LE(64n, 32);
elf.writeUInt16LE(64, 52);
elf.writeUInt16LE(56, 54);
elf.writeUInt16LE(1, 56);
const msi = Buffer.concat([Buffer.from("d0cf11e0a1b11ae1", "hex"), Buffer.alloc(64)]);
const deb = Buffer.concat([Buffer.from("!<arch>\n"), Buffer.from("debian-binary/"), Buffer.alloc(128)]);
const rpm = Buffer.concat([Buffer.from("edabeedb", "hex"), Buffer.alloc(64)]);
const zip = Buffer.concat([Buffer.from("504b0304", "hex"), Buffer.alloc(64)]);
const gz = Buffer.concat([Buffer.from("1f8b08", "hex"), Buffer.alloc(64)]);

test("release filename contract maps to real package formats", () => {
  for (const [name, fmt] of [
    ["GhostFTP-Windows-x64-Portable-v0.99.58.exe", "windows-pe64"],
    ["GhostFTP-Windows-x64-Setup-v0.99.58.exe", "windows-pe"],
    ["GhostFTP-Windows-x64-Setup-v0.99.58.msi", "msi"],
    ["GhostFTP-Linux-x86_64-v0.99.58", "linux-elf64"],
    ["GhostFTP-Linux-x86_64-v0.99.58.AppImage", "linux-elf64"],
    ["GhostFTP-Linux-amd64-v0.99.58.deb", "deb"],
    ["GhostFTP-Linux-x86_64-v0.99.58.rpm", "rpm"],
    ["GhostFTP-Windows-x64-v0.99.58.zip", "zip"],
    ["GhostFTP-v0.99.58-Source.zip", "zip"],
    ["GhostFTP-Linux-x86_64-v0.99.58.tar.gz", "gzip"]
  ]) assert.equal(formatForReleaseAsset(name, version), fmt);
  assert.throws(() => formatForReleaseAsset("GhostFTP-Windows-x64-Setup-v0.99.56.msi", version));
});

test("native format headers are validated and cross-format substitutions rejected", () => {
  for (const [bytes, fmt] of [[pe, "windows-pe64"], [pe, "windows-pe"], [elf, "linux-elf64"],
    [msi, "msi"], [deb, "deb"], [rpm, "rpm"], [zip, "zip"], [gz, "gzip"]]) {
    assert.doesNotThrow(() => validatePackageHeader(bytes, fmt, "valid"));
    assert.throws(() => validatePackageHeader(Buffer.alloc(128), fmt, "wrong"));
  }
  assert.throws(() => validatePackageHeader(zip, "msi", "fake.msi"));
  assert.throws(() => validatePackageHeader(pe, "linux-elf64", "fake.AppImage"));
  const x86 = Buffer.from(pe);
  x86.writeUInt16LE(0x014c, 132);
  x86.writeUInt16LE(0x10b, 152);
  assert.throws(() => validatePackageHeader(x86, "windows-pe64", "x86-portable.exe"));
  assert.doesNotThrow(() => validatePackageHeader(x86, "windows-pe", "x86-nsis-setup.exe"));
  const arm = Buffer.from(elf);
  arm.writeUInt16LE(0xb7, 18);
  assert.throws(() => validatePackageHeader(arm, "linux-elf64", "arm.AppImage"));
});

test("PE section tables and ELF executable metadata cannot be spoofed by a short magic prefix", () => {
  const shortPe = Buffer.from(pe);
  shortPe.writeUInt16LE(0, 134);
  assert.throws(() => validatePackageHeader(shortPe, "windows-pe64", "no-sections.exe"));
  const wrongPe = Buffer.from(pe);
  wrongPe.writeUInt16LE(0x10b, 152);
  assert.throws(() => validatePackageHeader(wrongPe, "windows-pe64", "arch-mismatch.exe"));
  const wrongElf = Buffer.from(elf);
  wrongElf[5] = 2;
  assert.throws(() => validatePackageHeader(wrongElf, "linux-elf64", "big-endian"));
  const truncatedElf = Buffer.from(elf);
  truncatedElf.writeBigUInt64LE(65536n, 32);
  assert.throws(() => validatePackageHeader(truncatedElf, "linux-elf64", "invalid-program-table"));
});

function sampleZip() {
  const local = Buffer.alloc(34);
  Buffer.from("504b0304", "hex").copy(local);
  local.writeUInt16LE(1, 26);
  local[30] = 0x61;
  const central = Buffer.alloc(47);
  Buffer.from("504b0102", "hex").copy(central);
  central.writeUInt16LE(1, 28);
  central[46] = 0x61;
  const footer = Buffer.alloc(22);
  Buffer.from("504b0506", "hex").copy(footer);
  footer.writeUInt16LE(1, 8);
  footer.writeUInt16LE(1, 10);
  footer.writeUInt32LE(central.length, 12);
  footer.writeUInt32LE(local.length, 16);
  return Buffer.concat([local, central, footer]);
}

test("ZIP central-directory trailer rejects truncation, wrong offsets and trailing junk", () => {
  const valid = sampleZip();
  assert.deepEqual(validateZipFooter(valid, valid.length, "valid.zip"), {
    centralOffset: 34,
  });
  assert.throws(() => validateZipFooter(valid.subarray(0, -2), valid.length - 2, "truncated.zip"));
  assert.throws(() => validateZipFooter(Buffer.concat([valid, Buffer.from("junk")]), valid.length + 4, "trailing.zip"));
  const misplaced = Buffer.from(valid);
  misplaced.writeUInt32LE(999999, valid.length - 22 + 16);
  assert.throws(() => validateZipFooter(misplaced, misplaced.length, "bad-directory.zip"));
});

test("on-disk preflight verifies the full required package inventory and refuses symlinks", () => {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-release-formats-"));
  const { required, checksum } = expectedReleaseAssets(version);
  const fixtures = {
    "windows-pe64": pe,
    "windows-pe": pe,
    msi,
    "linux-elf64": elf,
    deb,
    rpm,
    zip: sampleZip(),
    gzip: gz,
  };
  try {
    for (const name of required) {
      if (name === checksum) continue;
      const fmt = formatForReleaseAsset(name, version);
      assert.ok(fixtures[fmt], "missing sample for " + fmt);
      fs.writeFileSync(path.join(tmp, name), fixtures[fmt]);
    }
    assert.equal(verifyReleasePackageFormats(version, tmp), required.length - 1);
    const zipName = required.find(name => name.endsWith(".zip"));
    assert.ok(zipName);
    const zipPath = path.join(tmp, zipName);
    const validZip = fs.readFileSync(zipPath);
    fs.writeFileSync(zipPath, validZip.subarray(0, -3));
    assert.throws(() => verifyReleasePackageFormats(version, tmp), /ZIP/);
    fs.writeFileSync(zipPath, validZip);
    if (process.platform !== "win32") {
      fs.rmSync(zipPath);
      fs.symlinkSync(path.join(tmp, required.find(name => name.endsWith(".rpm"))), zipPath);
      assert.throws(() => verifyReleasePackageFormats(version, tmp), /symlinked/);
    }
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
});
