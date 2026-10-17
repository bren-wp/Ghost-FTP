#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const pkg = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf8"));
const lock = JSON.parse(fs.readFileSync(path.join(root, "package-lock.json"), "utf8"));
const cargo = fs.readFileSync(path.join(root, "src-tauri", "Cargo.toml"), "utf8");
const tauriConfig = JSON.parse(fs.readFileSync(path.join(root, "src-tauri", "tauri.conf.json"), "utf8"));

const rustMatch = cargo.match(/^tauri\s*=\s*\{\s*version\s*=\s*"([^"]+)"/m);
if (!rustMatch) throw new Error("src-tauri/Cargo.toml: tauri version requirement not found");

const minor = (value) => {
  const m = String(value).match(/(\d+)\.(\d+)/);
  if (!m) throw new Error(`could not parse Tauri major/minor from ${value}`);
  return `${m[1]}.${m[2]}`;
};

const rustReq = rustMatch[1];
const apiSpec = pkg.dependencies?.["@tauri-apps/api"];
const cliSpec = pkg.devDependencies?.["@tauri-apps/cli"];
const apiResolved = lock.packages?.["node_modules/@tauri-apps/api"]?.version;
const cliResolved = lock.packages?.["node_modules/@tauri-apps/cli"]?.version;

for (const [label, value] of Object.entries({ rustReq, apiSpec, cliSpec, apiResolved, cliResolved })) {
  if (!value) throw new Error(`missing ${label} Tauri version metadata`);
}

const expected = minor(rustReq);
if (minor(apiResolved) !== expected) {
  throw new Error(`Tauri version mismatch: Rust ${rustReq} expects frontend API ${expected}.x but resolved to ${apiResolved}`);
}
if (!String(apiSpec).startsWith(`~${expected}.`)) {
  throw new Error(`@tauri-apps/api must stay on the ~${expected}.x train; found ${apiSpec}`);
}

// Signed updater version binding requires the security-patched Tauri CLI 2.12+.
// The CLI is a packaging tool, not the runtime JS API: keep its own explicit
// patch train and requireSignedVersion guard instead of forcing its minor to
// equal Rust tauri's last supported 2.11.x runtime train.
const cliMinor = minor(cliResolved);
if (cliMinor !== "2.12" || !String(cliSpec).startsWith("~2.12.")) {
  throw new Error(`Signed-version updater requires a locked @tauri-apps/cli ~2.12.x train; found ${cliSpec} -> ${cliResolved}`);
}
const cliPatch = Number(cliResolved.split(".")[2]);
if (!Number.isInteger(cliPatch) || cliPatch < 2) {
  throw new Error(`Signed-version Tauri CLI must include the verified 2.12.2+ patch; found ${cliResolved}`);
}
if (tauriConfig.plugins?.updater?.requireSignedVersion !== true ||
    tauriConfig.plugins?.updater?.allowDowngrades !== false) {
  throw new Error("Signed updater version binding and downgrade prevention must remain enabled");
}
if (rustReq !== "=2.11.6") {
  throw new Error(`Rust tauri must stay pinned to the security-patched tested core (=2.11.6); found ${rustReq}`);
}

if (!/\[profile\.release\][\s\S]*?opt-level\s*=\s*"s"/m.test(cargo)) {
  throw new Error('release profile must keep opt-level = "s" for size-optimized native binaries');
}
if (tauriConfig.bundle?.linux?.appimage?.bundleMediaFramework !== false) {
  throw new Error("AppImage media framework must remain disabled unless Ghost FTP gains an actual audio/video requirement");
}

const bundle = tauriConfig.bundle ?? {};
const nsis = bundle.windows?.nsis ?? {};
const requiredBundleValues = {
  publisher: [bundle.publisher, "Brendigo"],
  homepage: [bundle.homepage, "https://ghostftp.com"],
  licenseFile: [bundle.licenseFile, "../../EULA.txt"],
  installerIcon: [nsis.installerIcon, "icons/icon.ico"],
  uninstallerIcon: [nsis.uninstallerIcon, "icons/icon.ico"],
  headerImage: [nsis.headerImage, "nsis/ghostftp-header.bmp"],
  sidebarImage: [nsis.sidebarImage, "nsis/ghostftp-sidebar.bmp"],
  uninstallerHeaderImage: [nsis.uninstallerHeaderImage, "nsis/ghostftp-header.bmp"],
  installMode: [nsis.installMode, "currentUser"],
  startMenuFolder: [nsis.startMenuFolder, "Ghost FTP"],
};
for (const [label, [actual, expectedValue]] of Object.entries(requiredBundleValues)) {
  if (actual !== expectedValue) {
    throw new Error(`Tauri installer metadata mismatch for ${label}: expected ${expectedValue}, found ${actual}`);
  }
}
if (!Array.isArray(nsis.languages) || !nsis.languages.includes("English") || !nsis.languages.includes("Croatian")) {
  throw new Error("NSIS installer must ship English and Croatian language support");
}
if (nsis.displayLanguageSelector !== false) {
  throw new Error("NSIS language selector must stay automatic so setup follows the Windows locale");
}
for (const rel of [
  ["src-tauri/nsis/ghostftp-header.bmp", 150, 57],
  ["src-tauri/nsis/ghostftp-sidebar.bmp", 164, 314],
]) {
  const file = path.join(root, rel[0]);
  if (!fs.existsSync(file) || fs.statSync(file).size <= 54) {
    throw new Error(`missing or empty branded NSIS bitmap: ${rel[0]}`);
  }
  const bmp = fs.readFileSync(file);
  if (bmp.toString("ascii", 0, 2) !== "BM") {
    throw new Error(`invalid BMP signature: ${rel[0]}`);
  }
  if (bmp.readInt32LE(18) !== rel[1] || Math.abs(bmp.readInt32LE(22)) !== rel[2]) {
    throw new Error(`unexpected NSIS bitmap dimensions for ${rel[0]}; expected ${rel[1]}x${rel[2]}`);
  }
}
const eulaPath = path.resolve(root, "..", "EULA.txt");
if (!fs.existsSync(eulaPath)) throw new Error("root EULA.txt is required by the NSIS license page");
const eula = fs.readFileSync(eulaPath, "utf8");
if (!/IMPORTANT.+READ CAREFULLY/i.test(eula) || !/1\. ACCEPTANCE/.test(eula)) {
  throw new Error("EULA.txt must contain the installer acceptance terms");
}

console.log(`Tauri alignment OK: Rust ${rustReq}, API ${apiResolved}, CLI ${cliResolved}; branded NSIS/EULA contract OK`);
