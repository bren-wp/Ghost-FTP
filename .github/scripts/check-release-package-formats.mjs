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
  if (exact("GhostFTP-Windows-x64-Portable-v{version}.exe")) return "windows-pe64";
  // NSIS Setup can legitimately be an x86 bootstrapper for an x64 app.
  if (exact("GhostFTP-Windows-x64-Setup-v{version}.exe")) return "windows-pe";
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
    case "windows-pe":
    case "windows-pe64": {
      if (!begins(bytes, magic("4d 5a")) || bytes.length < 0x40) fail("not a Windows PE executable");
      const offset = bytes.readUInt32LE(0x3c);
      if (offset < 0x40 || offset + 6 > bytes.length) fail("invalid PE header offset");
      if (!bytes.subarray(offset, offset + 4).equals(magic("50 45 00 00"))) fail("missing PE signature");
      const machine = bytes.readUInt16LE(offset + 4);
      if (machine !== 0x014c && machine !== 0x8664) fail("unsupported Windows PE machine type");
      if (format === "windows-pe64" && machine !== 0x8664) fail("not a Windows x64 executable");
      const sections = bytes.readUInt16LE(offset + 6);
      const optionalSize = bytes.readUInt16LE(offset + 20);
      const optionalOffset = offset + 24;
      if (sections < 1 || sections > 96 || optionalSize < 96 ||
          optionalOffset + optionalSize + sections * 40 > bytes.length) {
        fail("incomplete PE optional header or section table");
      }
      const optionalMagic = bytes.readUInt16LE(optionalOffset);
      if (optionalMagic !== (machine === 0x8664 ? 0x20b : 0x10b)) {
        fail("Windows PE architecture and optional header disagree");
      }
      return;
    }
    case "msi":
      if (!begins(bytes, magic("d0 cf 11 e0 a1 b1 1a e1"))) fail("not an MSI compound-file installer");
      return;
    case "linux-elf64":
      if (!begins(bytes, magic("7f 45 4c 46")) || bytes.length < 64 ||
          bytes[4] !== 2 || bytes[5] !== 1 || bytes[6] !== 1 ||
          ![2, 3].includes(bytes.readUInt16LE(16)) ||
          bytes.readUInt16LE(18) !== 0x3e || bytes.readUInt32LE(20) !== 1) {
        fail("not a supported little-endian x86_64 ELF executable");
      }
      const programOffset = Number(bytes.readBigUInt64LE(32));
      const programEntrySize = bytes.readUInt16LE(54);
      const programCount = bytes.readUInt16LE(56);
      if (bytes.readUInt16LE(52) < 64 || programEntrySize < 56 ||
          programCount < 1 || programCount > 512 ||
          !Number.isSafeInteger(programOffset) ||
          programOffset + programCount * programEntrySize > bytes.length) {
        fail("incomplete ELF program header table");
      }
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

// ZIPs need a complete central directory, not just an easily forged "PK" prefix.
// A ZIP comment may be up to 65535 bytes, so callers read at most 65557 bytes.
export function validateZipFooter(tail, fileSize, filename) {
  const fail = (message) => { throw new Error(filename + ": " + message); };
  if (!Buffer.isBuffer(tail) || tail.length < 22 || tail.length > fileSize) {
    fail("missing ZIP central-directory footer");
  }
  const base = fileSize - tail.length;
  for (let at = tail.length - 22; at >= 0; at--) {
    if (tail.readUInt32LE(at) !== 0x06054b50 ||
        at + 22 + tail.readUInt16LE(at + 20) !== tail.length) continue;
    const disk = tail.readUInt16LE(at + 4);
    const centralDisk = tail.readUInt16LE(at + 6);
    const diskEntries = tail.readUInt16LE(at + 8);
    const totalEntries = tail.readUInt16LE(at + 10);
    const centralSize = tail.readUInt32LE(at + 12);
    const centralOffset = tail.readUInt32LE(at + 16);
    if (disk !== 0 || centralDisk !== 0 || totalEntries === 0 ||
        diskEntries !== totalEntries || centralSize < 46 ||
        centralSize === 0xffffffff || centralOffset === 0xffffffff ||
        centralOffset + centralSize > base + at) {
      fail("invalid, empty or unsupported ZIP central directory");
    }
    return { centralOffset };
  }
  fail("missing complete ZIP end-of-central-directory record");
}

export function verifyReleasePackageFormats(version, directory) {
  if (!VERSION.test(version)) throw new Error("Invalid Ghost FTP release version");
  const { required, checksum } = expectedReleaseAssets(version);
  let checked = 0;
  for (const name of required) {
    if (name === checksum) continue; // Generated after this exact-format preflight.
    const filepath = path.join(directory, name);
    const expected = fs.lstatSync(filepath);
    if (!expected.isFile() || expected.isSymbolicLink() || expected.size < 16) {
      throw new Error(name + ": release artifact is empty, symlinked or not a regular file");
    }
    const format = formatForReleaseAsset(name, version);
    // Opening by name after lstat alone follows a symlink swapped in between.
    // O_NOFOLLOW rejects that substitution on supported POSIX build runners.
    const flags = fs.constants.O_RDONLY |
      (process.platform === "win32" ? 0 : (fs.constants.O_NOFOLLOW ?? 0));
    const handle = fs.openSync(filepath, flags);
    try {
      const stat = fs.fstatSync(handle);
      if (!stat.isFile() || stat.dev !== expected.dev ||
          stat.ino !== expected.ino || stat.size !== expected.size) {
        throw new Error(name + ": release artifact changed while opening");
      }
      const bytes = Buffer.alloc(Math.min(65536, stat.size));
      const read = fs.readSync(handle, bytes, 0, bytes.length, 0);
      if (read !== bytes.length) throw new Error(name + ": truncated release header");
      validatePackageHeader(bytes, format, name);
      if (format === "zip") {
        const tailSize = Math.min(stat.size, 65557);
        const tail = Buffer.alloc(tailSize);
        const tailRead = fs.readSync(handle, tail, 0, tailSize, stat.size - tailSize);
        if (tailRead !== tailSize) throw new Error(name + ": truncated ZIP footer");
        const { centralOffset } = validateZipFooter(tail, stat.size, name);
        const centralMagic = Buffer.alloc(4);
        if (fs.readSync(handle, centralMagic, 0, 4, centralOffset) !== 4 ||
            !centralMagic.equals(magic("50 4b 01 02"))) {
          throw new Error(name + ": missing ZIP central directory entries");
        }
      }
      const finalStat = fs.fstatSync(handle);
      if (finalStat.size !== stat.size || finalStat.mtimeMs !== stat.mtimeMs) {
        throw new Error(name + ": release artifact mutated during preflight");
      }
    } finally {
      fs.closeSync(handle);
    }
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
    console.log("Validated " + count + " release binary structures and ZIP directory footers (not signing identity).");
  } catch (error) {
    console.error("::error::" + error.message);
    process.exitCode = 1;
  }
}
