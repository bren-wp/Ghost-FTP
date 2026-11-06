#!/usr/bin/env node
// Exercise the real Zustand updater store with mocked native side effects.
// No network, application state, or downloadable binary is simulated as success.
import assert from "node:assert/strict";
import fs from "node:fs";
import { runInNewContext } from "node:vm";
import ts from "typescript";

const updater = fs.readFileSync(new URL("../src/stores/updaterStore.ts", import.meta.url), "utf8");
const about = fs.readFileSync(new URL("../src/components/AboutDialog.tsx", import.meta.url), "utf8");
const release = fs.readFileSync(new URL("../src/lib/release.ts", import.meta.url), "utf8");
const rust = fs.readFileSync(new URL("../src-tauri/src/lib.rs", import.meta.url), "utf8");
const channel = fs.readFileSync(new URL("../../.github/scripts/check-release-channel.mjs", import.meta.url), "utf8");
assert.ok(!about.includes("Stable channel ·"), "Unsigned desktop UI must not falsely label itself stable");
assert.ok(about.includes("url={OFFICIAL_RELEASES_URL}"), "Manual installer link missing");
assert.ok(release.includes('OFFICIAL_RELEASES_URL = "https://github.com/bren-wp/Ghost-FTP/releases"'), "Only the verified Releases index is allowed");
assert.ok(rust.includes('"/bren-wp/Ghost-FTP/releases"'), "Rust IPC allowlist blocks manual downloads");
assert.ok(channel.includes('if (channel === "stable")'), "Global Android/macOS signing gates must not be weakened");
const compiled = ts.transpileModule(updater, { compilerOptions: {
  target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS,
}});
let state;
let offered = null;
let checks = 0;
const notifications = [];
const store = { getState: () => state, setState: (patch) => { state = { ...state, ...patch }; } };
const create = (definition) => {
  const set = (change) => {
    state = { ...state, ...(typeof change === "function" ? change(state) : change) };
  };
  state = definition(set, () => state);
  return store;
};
const deps = {
  zustand: { create },
  "@tauri-apps/plugin-updater": { check: async () => { checks++; return offered; } },
  "@tauri-apps/plugin-process": { relaunch: async () => {} },
  "@/lib/ipc": { ipc: { settingsSet: async () => {}, settingsGetAll: async () => ({}) } },
  "./toastStore": { toast: {
    info: (...args) => notifications.push(["info", ...args]),
    warning: (...args) => notifications.push(["warning", ...args]),
    success: (...args) => notifications.push(["success", ...args]),
    error: (...args) => notifications.push(["error", ...args]),
  }},
  "@/lib/errors": { messageOf: (v) => String(v) },
  "@/lib/redact": { redactSensitiveText: () => "[redacted]" },
};
const exported = {};
runInNewContext(compiled.outputText, {
  exports: exported,
  require: (name) => { if (!deps[name]) throw Error("Unexpected updater dependency " + name); return deps[name]; },
  console: { warn: () => {} }, Date, JSON, Promise,
}, { timeout: 2000 });
assert.ok(exported.useUpdater, "Shipped updater store did not initialize");

await store.getState().check(false);
assert.equal(store.getState().status, "checked", "Unsigned/no-offer response is not proof of current version");
assert.ok(!notifications.some(row => row[0] === "success" || row.join(" ").toLowerCase().includes("up to date")),
  "Signed-offer absence must never produce a false up-to-date toast");
assert.equal(checks, 1);
offered = { available: true, version: "9.9.9", currentVersion: "0.99.33", body: "signed update", downloadAndInstall: async () => {} };
await store.getState().check(false);
assert.equal(store.getState().status, "available");
assert.equal(store.getState().version, "9.9.9");
store.setState({ status: "ready" });
await store.getState().check(false);
assert.equal(store.getState().status, "ready", "An installed update awaiting restart must not be lost");
assert.equal(checks, 2, "Do not start a new updater check while ready to restart");
console.log("Windows/Linux unsigned manual-update fallback and signed-update safety: passed.");
