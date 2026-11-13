import assert from "node:assert/strict";
import test from "node:test";
import { formatForReleaseAsset, validatePackageHeader } from "./check-release-package-formats.mjs";

const version = "0.99.57";
const pe = Buffer.alloc(512);
pe.write("MZ");
pe.writeUInt32LE(128, 0x3c);
pe.write("PE\\0\\0".replaceAll("\\0", "\0"), 128, "binary");
pe.writeUInt16LE(0x8664, 132);
const elf = Buffer.alloc(64);
Buffer.from("7f454c46", "hex").copy(elf);
elf[4] = 2;
elf.writeUInt16LE(0x3e, 18);
const msi = Buffer.concat([Buffer.from("d0cf11e0a1b11ae1", "hex"), Buffer.alloc(64)]);
const deb = Buffer.concat([Buffer.from("!<arch>\n"), Buffer.from("debian-binary/"), Buffer.alloc(128)]);
const rpm = Buffer.concat([Buffer.from("edabeedb", "hex"), Buffer.alloc(64)]);
const zip = Buffer.concat([Buffer.from("504b0304", "hex"), Buffer.alloc(64)]);
const gz = Buffer.concat([Buffer.from("1f8b08", "hex"), Buffer.alloc(64)]);

test("release filename contract maps to real package formats", () => {
  for (const [name, fmt] of [
    ["GhostFTP-Windows-x64-Portable-v0.99.57.exe", "windows-pe64"],
    ["GhostFTP-Windows-x64-Setup-v0.99.57.exe", "windows-pe64"],
    ["GhostFTP-Windows-x64-Setup-v0.99.57.msi", "msi"],
    ["GhostFTP-Linux-x86_64-v0.99.57", "linux-elf64"],
    ["GhostFTP-Linux-x86_64-v0.99.57.AppImage", "linux-elf64"],
    ["GhostFTP-Linux-amd64-v0.99.57.deb", "deb"],
    ["GhostFTP-Linux-x86_64-v0.99.57.rpm", "rpm"],
    ["GhostFTP-Windows-x64-v0.99.57.zip", "zip"],
    ["GhostFTP-v0.99.57-Source.zip", "zip"],
    ["GhostFTP-Linux-x86_64-v0.99.57.tar.gz", "gzip"]
  ]) assert.equal(formatForReleaseAsset(name, version), fmt);
  assert.throws(() => formatForReleaseAsset("GhostFTP-Windows-x64-Setup-v0.99.56.msi", version));
});

test("native format headers are validated and cross-format substitutions rejected", () => {
  for (const [bytes, fmt] of [[pe, "windows-pe64"], [elf, "linux-elf64"],
    [msi, "msi"], [deb, "deb"], [rpm, "rpm"], [zip, "zip"], [gz, "gzip"]]) {
    assert.doesNotThrow(() => validatePackageHeader(bytes, fmt, "valid"));
    assert.throws(() => validatePackageHeader(Buffer.alloc(128), fmt, "wrong"));
  }
  assert.throws(() => validatePackageHeader(zip, "msi", "fake.msi"));
  assert.throws(() => validatePackageHeader(pe, "linux-elf64", "fake.AppImage"));
  const x86 = Buffer.from(pe);
  x86.writeUInt16LE(0x014c, 132);
  assert.throws(() => validatePackageHeader(x86, "windows-pe64", "x86.exe"));
  const arm = Buffer.from(elf);
  arm.writeUInt16LE(0xb7, 18);
  assert.throws(() => validatePackageHeader(arm, "linux-elf64", "arm.AppImage"));
});
