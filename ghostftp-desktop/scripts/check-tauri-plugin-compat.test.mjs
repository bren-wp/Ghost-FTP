import assert from "node:assert/strict";
import test from "node:test";
import { PLUGIN_PAIRS, assertTauriPluginCompatibility } from "./tauri-plugin-compat.mjs";

const spec = Object.fromEntries(PLUGIN_PAIRS.map(([js], i) => [js, ["~2.7.1", "~2.3.3", "~2.3.1", "~2.12.0"][i]]));
const versions = ["2.7.1", "2.3.3", "2.3.1", "2.12.0"];
const rustVersions = ["2.7.3", "2.3.3", "2.3.1", "2.12.0"];
const fixture = () => ({
  pkg: { dependencies: { ...spec } },
  npm: { packages: {
    "": { dependencies: { ...spec } },
    ...Object.fromEntries(PLUGIN_PAIRS.map(([js], i) => [`node_modules/${js}`, { version: versions[i] }])),
  }},
  cargo: PLUGIN_PAIRS.map(([,rust], i) => `[[package]]\nname = "${rust}"\nversion = "${rustVersions[i]}"\n`).join("\n"),
});
test("matches all four locked Rust/JS Tauri plugins", () => {
  const x = fixture();
  assert.equal(assertTauriPluginCompatibility(x.pkg, x.npm, x.cargo).length, 4);
});
test("rejects a mismatched JavaScript plugin minor", () => {
  const x = fixture();
  x.npm.packages["node_modules/@tauri-apps/plugin-dialog"].version = "2.8.4";
  assert.throws(() => assertTauriPluginCompatibility(x.pkg, x.npm, x.cargo), /violates patch-only range|Tauri plugin mismatch/);
});
test("rejects too-broad ranges before npm updates them", () => {
  const x = fixture();
  x.pkg.dependencies["@tauri-apps/plugin-notification"] = "^2.3.3";
  assert.throws(() => assertTauriPluginCompatibility(x.pkg, x.npm, x.cargo), /patch-only/);
});
test("rejects missing or modified Rust lock entries", () => {
  const x = fixture();
  x.cargo = x.cargo.replace('name = "tauri-plugin-process"', 'name = "removed-plugin"');
  assert.throws(() => assertTauriPluginCompatibility(x.pkg, x.npm, x.cargo), /Missing locked Rust crate/);
});
test("rejects desynchronized package lock declarations", () => {
  const x = fixture();
  x.npm.packages[""].dependencies["@tauri-apps/plugin-updater"] = "^2.12.0";
  assert.throws(() => assertTauriPluginCompatibility(x.pkg, x.npm, x.cargo), /package-lock root differs/);
});
