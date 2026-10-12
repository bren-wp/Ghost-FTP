#!/usr/bin/env node
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (relative) => fs.readFileSync(path.join(root, relative), "utf8");
const cargo = read("ghostftp-desktop/src-tauri/Cargo.toml");
const lock = read("ghostftp-desktop/src-tauri/Cargo.lock");
const pkg = JSON.parse(read("ghostftp-desktop/package.json"));
const pkgLock = JSON.parse(read("ghostftp-desktop/package-lock.json"));
const tauriConfig = JSON.parse(read("ghostftp-desktop/src-tauri/tauri.conf.json"));

function version(value) {
  assert.match(value, /^\\d+\\.\\d+\\.\\d+$/, "Invalid package SemVer");
  return value.split(".").map(Number);
}
function atLeast(actual, minimum) {
  const a = version(actual);
  const b = version(minimum);
  for (let i = 0; i < 3; i++) if (a[i] !== b[i]) return a[i] > b[i];
  return true;
}
function rustLocked(name) {
  const escaped = name.replace(/[.*+?^$\{\}()|[\]\\]/g, "\\$&");
  const match = lock.match(new RegExp('\\[\\[package\\]\\]\\nname = "' + escaped + '"\\nversion = "([^"]+)"'));
  assert.ok(match, `Missing Rust lockfile package ${name}`);
  return match[1];
}

const corePin = cargo.match(/^tauri = \\{ version = "=(\\d+\\.\\d+\\.\\d+)"/m)?.[1];
assert.ok(corePin && atLeast(corePin, "2.11.6"), "Tauri Rust core must be >= 2.11.6 (GHSA-w28w-mhc8-qvjv)");
assert.ok(atLeast(rustLocked("tauri"), "2.11.6"), "Cargo.lock still includes vulnerable Tauri IPC core");
assert.equal(rustLocked("tauri"), corePin, "Rust core pin differs from Cargo.lock");

const updaterFloor = cargo.match(/^tauri-plugin-updater = "~(\\d+\\.\\d+\\.\\d+)"$/m)?.[1];
assert.ok(updaterFloor && atLeast(updaterFloor, "2.12.0"), "Updater must not allow webview-selected downgrades");
assert.ok(atLeast(rustLocked("tauri-plugin-updater"), "2.12.0"), "Rust updater lock is vulnerable to downgrade attacks");
assert.equal(tauriConfig.plugins?.updater?.allowDowngrades, false, "Updater policy must disallow downgrades");

const jsRange = pkg.dependencies?.["@tauri-apps/plugin-updater"];
const jsMin = jsRange?.match(/^\\^(\\d+\\.\\d+\\.\\d+)$/)?.[1];
const jsLocked = pkgLock.packages?.["node_modules/@tauri-apps/plugin-updater"]?.version;
assert.ok(jsMin && atLeast(jsMin, "2.12.0"), "JS updater minimum version is vulnerable");
assert.ok(jsLocked && atLeast(jsLocked, "2.12.0"), "JS updater lock is vulnerable");
assert.equal(pkgLock.packages?.[""]?.dependencies?.["@tauri-apps/plugin-updater"], jsRange);

console.log(`Security dependency floor passed: Tauri ${corePin}, Rust updater ${rustLocked("tauri-plugin-updater")}, JS updater ${jsLocked}; allowDowngrades=false.`);
