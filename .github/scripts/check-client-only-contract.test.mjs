import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import {fileURLToPath} from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const has = (p) => fs.existsSync(path.join(root, p));
const read = (p) => fs.readFileSync(path.join(root, p), "utf8");

test("standalone Server, Demo and CLI source and service installers are absent", () => {
  for (const p of [
    "ghost-server", "ghost-demo", "ghost-cli",
    "ghostftp-desktop/src-tauri/ghostftp-cli",
    ".github/workflows/ghostftp-server.yml",
    "packaging/linux/build-portable-packages.sh",
    "scripts/ci/resolve-nsis.ps1",
    "ghostftp-desktop/src-tauri/src/path_integration.rs",
    "ghostftp-desktop/src/lib/syncScripts.ts",
  ]) assert.equal(has(p), false, `Retired component still present: ${p}`);
});

test("all release entry points contain only Ghost FTP client packages", () => {
  const files = [
    ".github/workflows/ghostftp-build.yml",
    ".github/workflows/ghostftp-release.yml",
    ".github/scripts/publish-release.sh",
    ".github/scripts/check-release-index.sh",
  ];
  for (const file of files) {
    const source = read(file);
    for (const forbidden of [
      "GhostCLI-", "GhostFTP-Server-", "GhostFTP-Demo-",
      "ghostftp-cli", "ghostftp-server.yml", "ghost-demo/", "ghost-cli/",
    ]) assert.ok(!source.includes(forbidden), `${file}: retired ${forbidden}`);
  }
  assert.match(read(".github/scripts/check-release-index.sh"), /-ge 25/);
  assert.ok(read(".github/scripts/publish-release.sh").includes("sha256sum -c"));
});

test("GUI has no inoperative CLI settings or external sync script buttons", () => {
  const cargo = read("ghostftp-desktop/src-tauri/Cargo.toml");
  assert.ok(!cargo.includes('"ghostftp-cli"'));
  const lib = read("ghostftp-desktop/src-tauri/src/lib.rs");
  assert.ok(!lib.includes("path_integration::"));
  const shell = read("ghostftp-desktop/src/components/Settings.tsx");
  assert.ok(!shell.includes("Shell integration"));
  assert.ok(!read("ghostftp-desktop/src/components/SyncSettings.tsx").includes("copyScript"));
  assert.ok(!read("ghostftp-desktop/src/components/AgentBridge.tsx").includes("ghostftp-cli"));
  assert.ok(!read("ghostftp-desktop/src/lib/commands.tsx").includes('id: "agent-bridge"'));
});

test("0.99.24 remains a Preview until mandatory production signers and platform tests exist", () => {
  const config = JSON.parse(read("version.json"));
  assert.equal(config.version, "0.99.24");
  assert.equal(config.previousPublishedVersion, "0.99.23");
  assert.equal(config.channel, "preview");
  assert.ok(read(".github/scripts/check-release-channel.mjs").includes('channel === "stable"'));
});
