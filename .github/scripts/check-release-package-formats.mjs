#!/usr/bin/env node
/**
 * Refuse mislabeled native release binaries before checksumming and upload.
 * Byte-format checks are deliberately not code-signature verification.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { expectedReleaseAssets } from "./validate-release-assets.mjs";

const self = fileURLToPath(import.meta.url);
const VERSION = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;
const begins = (data, header) => data.subarray(0, header.length).equals(header);
const magic = (hex) => Buffer.from(hex.replaceAll(" ", ""), "hex");

export function formatForReleaseAsset(name, version) {
  if (!VERSION.test(version)) throw new Error("Invalid Ghost FTP release version");
  const exact = (template) => template.replaceAll("{version}", version) === name;
  if (exact("GhostFTP-Windows-x64-Portable-v{version}.exe") ||
      exact("GhostFTP-Windows-x64-Setup-v{version}.exe")) return "windows-pe64";
  if (exact("GhostFTP-Windows-x64-Setup-v{version}.msi")) return "msi";
  if (exact("GhostFTP-Linux-x86_64-v{version}") ||
      exact("GhostFTP-Linux-x86_64-v{version}.AppImage")) return "linux-elf64";
  if (exact("GhostFTP-Linux-amd64-v{version}.deb")) return "deb";
  if (exact("GhostFTP-Linux-x86_64-v{version}.rpm")) return "rpm";
  if (name.endsWith(".zip")) return "zip";
  if (exact("GhostFTP-Linux-x86_64-v{version}.tar.gz")) return "gzip";
  if (exact("GhostFTP-v{version}-SHA256SUMS.txt")) return "checksums";
  throw new Error("No file-format rule for release asset: " + name);
}

export function validatePackageHeader(bytes, format, filename) {
  const fail = (message) => { throw new Error(filename + ": " + message); };
  if (!Buffer.isBuffer(bytes) || bytes.length < 16) fail("missing or truncated binary header");
  switch (format) {
    case "windows-pe64": {
      if (!begins(bytes, magic("4d 5a")) || bytes.length < 0x40) fail("not a Windows PE executable");
      const offset = bytes.readUInt32LE(0x3c);
      if (offset < 0x40 || offset + 6 > bytes.length) fail("invalid PE header offset");
      if (!bytes.subarray(offset, offset + 4).equals(magic("50 45 00 00"))) fail("missing PE signature");
      if (bytes.readUInt16LE(offset + 4) !== 0x8664) fail("not a Windows x64 executable");
      return;
    }
    case "msi":
      if (!begins(bytes, magic("d0 cf 11 e0 a1 b1 1a e1"))) fail("not an MSI compound-file installer");
      return;
    case "linux-elf64":
      if (!begins(bytes, magic("7f 45 4c 46")) || bytes[4] !== 2 ||
          bytes.readUInt16LE(18) !== 0x3e) fail("not an x86_64 ELF executable");
      return;
    case "deb":
      if (!begins(bytes, Buffer.from("!<arch>\n")) ||
          !bytes.subarray(8, Math.min(bytes.length, 120)).includes(Buffer.from("debian-binary"))) {
        fail("not a Debian ar package with a debian-binary entry");
      }
      return;
    case "rpm":
      if (!begins(bytes, magic("ed ab ee db"))) fail("not an RPM package");
      return;
    case "zip":
      if (!begins(bytes, magic("50 4b 03 04"))) fail("not a nonempty ZIP archive");
      return;
    case "gzip":
      if (!begins(bytes, magic("1f 8b 08"))) fail("not a gzip-compressed archive");
      return;
    default:
      fail("unsupported binary format " + format);
  }
}

export function verifyReleasePackageFormats(version, directory) {
  if (!VERSION.test(version)) throw new Error("Invalid Ghost FTP release version");
  const { required, checksum } = expectedReleaseAssets(version);
  let checked = 0;
  for (const name of required) {
    if (name === checksum) continue; // Generated after this exact-format preflight.
    const filepath = path.join(directory, name);
    const stat = fs.lstatSync(filepath);
    if (!stat.isFile() || stat.isSymbolicLink() || stat.size < 16) {
      throw new Error(name + ": release artifact is empty, symlinked or not a regular file");
    }
    const format = formatForReleaseAsset(name, version);
    const handle = fs.openSync(filepath, "r");
    let bytes;
    try {
      bytes = Buffer.alloc(Math.min(65536, stat.size));
      const read = fs.readSync(handle, bytes, 0, bytes.length, 0);
      bytes = bytes.subarray(0, read);
    } finally {
      fs.closeSync(handle);
    }
    validatePackageHeader(bytes, format, name);
    checked++;
  }
  if (checked !== required.length - 1) throw new Error("Unexpected release package inventory");
  return checked;
}

if (process.argv[1] && path.resolve(process.argv[1]) === self) {
  try {
    const [, , version, directory] = process.argv;
    if (!version || !directory) throw new Error("Usage: check-release-package-formats.mjs VERSION DIST");
    const count = verifyReleasePackageFormats(version, directory);
    console.log("Validated " + count + " native/archived release file headers (not signing identity).");
  } catch (error) {
    console.error("::error::" + error.message);
    process.exitCode = 1;
  }
}
