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
assert.ok(about.includes("url={UPDATE_CENTER_URL}"), "Manual installer link missing");
assert.ok(release.includes('UPDATE_CENTER_URL = "https://update.ghostftp.com/"'), "Manual updates must open the official update center");
assert.ok(rust.includes('host == "update.ghostftp.com" && path == "/"'), "Rust IPC allowlist must allow only the update center root");
assert.ok(channel.includes('New releases must use the stable channel'), "Stable distribution is mandatory");
const config = JSON.parse(fs.readFileSync(new URL("../src-tauri/tauri.conf.json", import.meta.url), "utf8"));
assert.equal(config.plugins.updater.allowDowngrades, false);
assert.equal(config.plugins.updater.requireSignedVersion, true);
const compiled = ts.transpileModule(updater, { compilerOptions: {
  target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS,
}});
let state;
let offered = null;
let manualOffered = null;
let manualChecks = 0;
let checks = 0;
let transfers = [];
let waiting = [];
let snapshotFailure = false;
let snapshotGate = null;
let installs = 0;
let restarts = 0;
let maintenanceFailure = false;
let maintenanceCalls = 0;
let recoveryNotice = null;
let targetVersion = null;
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
  "@tauri-apps/plugin-process": { relaunch: async () => { restarts++; } },
  "@/lib/ipc": { ipc: { settingsSet: async () => {}, settingsGetAll: async () => ({}), checkManualUpdate: async () => { manualChecks++; return manualOffered; }, listTransfers: async () => { if (snapshotGate) await snapshotGate; if (snapshotFailure) throw Error("snapshot unavailable"); return transfers; }, transferQueueState: async () => ({ waiting }), beginUpdateMaintenance: async (target) => { targetVersion = target; maintenanceCalls++; if (maintenanceFailure) throw Error("busy"); }, updateRecoveryNotice: async () => recoveryNotice } },
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
assert.equal(manualChecks, 1);
manualOffered = "0.99.91";
offered = { available: true, version: "9.9.9", currentVersion: "0.99.33", body: "signed update", downloadAndInstall: async () => { installs++; } };
await store.getState().check(false);
assert.equal(store.getState().status, "available");
assert.equal(store.getState().version, "9.9.9");
assert.equal(store.getState().manualVersion, "0.99.91");
store.setState({ status: "ready" });
await store.getState().check(false);
assert.equal(store.getState().status, "ready", "An installed update awaiting restart must not be lost");
assert.equal(checks, 2, "Do not start a new updater check while ready to restart");
store.setState({ status: "idle" });
offered = null;
await store.getState().check(false);
assert.equal(store.getState().status, "checked");
assert.equal(store.getState().manualVersion, "0.99.91");
assert.equal(manualChecks, 3);
await store.getState().check(true);
assert.equal(manualChecks, 3, "Quiet checks do not query manual availability");
assert.equal(store.getState().manualVersion, null);
// Exercise the real store's preflight with native IPC and OS operations mocked.
await store.getState().restart();
assert.equal(restarts, 0, "Restart requires an installed update");
offered = { available: true, version: "9.9.9", currentVersion: "0.99.91", downloadAndInstall: async () => { installs++; } };
await store.getState().check(false);
for (const status of ["queued", "transferring", "paused", "unknown"]) {
  transfers = [{ status }];
  await store.getState().downloadAndInstall();
  assert.equal(installs, 0, status);
  assert.equal(store.getState().status, "available");
}
transfers = [];
waiting = ["pending"];
await store.getState().downloadAndInstall();
assert.equal(installs, 0, "Native queue may contain work absent from transfer snapshot");
waiting = [];
snapshotFailure = true;
await store.getState().downloadAndInstall();
assert.equal(installs, 0, "Unknown queue state must fail closed");
snapshotFailure = false;
transfers = ["done", "skipped", "error", "canceled"].map(status => ({ status }));
maintenanceFailure = true;
await store.getState().downloadAndInstall();
assert.equal(installs, 0, "Exclusive native admission refusal must block installation");
assert.equal(store.getState().status, "available");
maintenanceFailure = false;
let unblock;
snapshotGate = new Promise(resolve => { unblock = resolve; });
const first = store.getState().downloadAndInstall();
await store.getState().downloadAndInstall();
unblock();
await first;
snapshotGate = null;
assert.equal(installs, 1, "Parallel requests must not install twice");
assert.equal(maintenanceCalls, 2, "Only admitted installer attempts acquire maintenance");
assert.equal(targetVersion, "9.9.9", "Native journal records the offered target");
assert.equal(store.getState().status, "ready");
transfers = [{ status: "transferring" }];
await store.getState().restart();
assert.equal(restarts, 0, "New transfer must prevent restart");
snapshotFailure = true;
transfers = [];
await store.getState().restart();
assert.equal(restarts, 0);
snapshotFailure = false;
snapshotGate = new Promise(resolve => { unblock = resolve; });
const restart = store.getState().restart();
await store.getState().restart();
unblock();
await restart;
snapshotGate = null;
assert.equal(restarts, 1, "Parallel requests must not restart twice");
recoveryNotice = "Previous update did not reach target";
await store.getState().init();
assert.equal(store.getState().recoveryNotice, recoveryNotice, "Startup surfaces a persisted recovery notice");
console.log("Windows/Linux unsigned manual-update fallback, signed-update preflight and restart safety: passed.");
