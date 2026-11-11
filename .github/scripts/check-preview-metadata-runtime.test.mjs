import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const OBSERVED = JSON.parse(fs.readFileSync(path.join(ROOT, ".github/scripts/v0.99.24-observed-assets.json"), "utf8"));
const CONTRACT = JSON.parse(fs.readFileSync(path.join(ROOT, ".github/scripts/release-artifacts.json"), "utf8"));
const PINNED = {
  "0.95.53": "27a3f8413f168e4cb8cc31cf6d7b7c85c353f5af",
  "0.99.0": "900d76e6e3273058270a48affe2664f968910a1b",
  "0.99.1": "eb2e1899f0bb953b731e7b41e0109ba4decb324a",
  "0.99.20": "567c324213e20f1b87a4b6444a65b8150f747966",
  "0.99.21": "8b68b7bb7ae2a1ac35779ac4f7f965597fcc114d",
  "0.99.24": "f36026188b3defedb9464ad04e4c234afb8a399b",
  "0.99.25": "2e00a640c65dcd4d73c0401071703c9312eb7881",
};
const VERSION_METADATA = JSON.parse(fs.readFileSync(path.join(ROOT, "version.json"), "utf8"));
const PREVIOUS_PUBLISHED = VERSION_METADATA.previousPublishedVersion;
const RECENT = {
  "0.99.37": "12656d9c78189b916ca3bf04fbb281a84e066013",
  "0.99.38": "6a192fbb41cc6af89e0ad75432156f97c869253d",
  "0.99.39": "fb213d07a1c9da8a037ad20e8e7a32f6040f66d2",
  "0.99.40": "e9e3f21bf6daa9e109ed4e26cee82931815e9857",
  "0.99.41": "0e420f35683ca8f85285e50bbbfb0286562ff5cd",
};
// Unit-test-only fixtures: as the release train advances, model the newest
// canonical predecessor without falsely asserting an actual GitHub release
// exists. The production script verifies the REAL immutable remote tag SHA.
if (!(PREVIOUS_PUBLISHED in RECENT)) {
  RECENT[PREVIOUS_PUBLISHED] = createHash("sha1")
    .update("unit-fixture:" + PREVIOUS_PUBLISHED).digest("hex");
}
const sha = (bytes) => createHash("sha256").update(bytes).digest("hex");

// This fake emulates *both* legacy metadata API checks and the modern
// checksum download and source-tag check. No network or real GH token needed.
const mockGh = String.raw`#!/usr/bin/env node
const fs = require("node:fs");
const path = require("node:path");
const args = process.argv.slice(2);
const state = process.env.MOCK_GH_DIRECTORY;
if (!state) process.exit(2);
const fileFor = v => path.join(state, v + ".json");
const read = v => {
  if (!fs.existsSync(fileFor(v))) process.exit(3);
  return JSON.parse(fs.readFileSync(fileFor(v), "utf8"));
};
if (args[0] === "release" && args[1] === "download") {
  const version = args[2].slice(1);
  const output = args[args.indexOf("--dir") + 1];
  const name = args[args.indexOf("--pattern") + 1];
  if (!name || !output || !fs.existsSync(fileFor(version))) process.exit(4);
  fs.copyFileSync(path.join(state, version + ".sum"), path.join(output, name));
  process.exit(0);
}
if (args[0] !== "api") process.exit(2);
const resource = args.find(x => x.startsWith("repos/"));
if (!resource) process.exit(2);
if (resource.includes("/git/ref/tags/")) {
  process.stdout.write(read(resource.split("/").pop().slice(1)).target_commitish + "\n");
} else if (resource.includes("/contents/version.json?ref=")) {
  const version = resource.split("?ref=v").pop();
  read(version);
  process.stdout.write(Buffer.from(JSON.stringify({version})).toString("base64") + "\n");
} else {
  let version;
  if (resource.includes("/releases/tags/")) {
    version = resource.split("/").pop().slice(1);
  } else {
    const id = Number(resource.split("/").pop());
    const versions = fs.readdirSync(state).filter(x => x.endsWith(".json")).map(x => x.slice(0,-5));
    version = versions.find(x => read(x).id === id);
  }
  if (!version) process.exit(3);
  const data = read(version);
  if (args.includes("PATCH")) {
    const patch = JSON.parse(fs.readFileSync(0, "utf8"));
    if (patch.make_latest !== "false") process.exit(4);
    data.name = patch.name;
    data.prerelease = patch.prerelease;
    fs.writeFileSync(fileFor(version), JSON.stringify(data));
  }
  process.stdout.write(JSON.stringify(data) + "\n");
}
`;

function legacyFixture(version, index) {
  const required = [
    `GhostFTP-v${version}-SHA256SUMS.txt`,
    `GhostFTP-Android-v${version}.apk.unsigned`,
    `GhostFTP-macOS-v${version}-Preview.zip`,
  ];
  const assets = version === "0.99.24" ? OBSERVED : Array.from({length: 21}, (_, i) => ({
    id: index * 100 + i + 1,
    name: required[i] || `GhostFTP-v${version}-synthetic-${i}.zip`,
    size: 100 + i,
    digest: "sha256:" + "a".repeat(64),
  }));
  return {
    id: 500 + index,
    tag_name: "v" + version,
    target_commitish: PINNED[version],
    draft: false,
    prerelease: version !== "0.99.24" && version !== "0.99.25",
    name: "Ghost FTP " + version + (version === "0.99.24" || version === "0.99.25" ? "" : " Preview"),
    assets,
  };
}

function recentFixture(version, index) {
  const required = CONTRACT.required.map(n => n.replaceAll("{version}", version));
  const checksum = `GhostFTP-v${version}-SHA256SUMS.txt`;
  const assets = required.filter(n => n !== checksum).map((name, i) => {
    const bytes = Buffer.from("package bytes for " + name);
    return { id: 10000 + index * 100 + i, name, size: bytes.length,
      digest: "sha256:" + sha(bytes), state: "uploaded" };
  });
  const sums = Buffer.from(assets.map(a => a.digest.slice(7) + "  " + a.name).join("\n") + "\n");
  assets.push({ id: 10000 + index * 100 + 99, name: checksum, size: sums.length,
    digest: "sha256:" + sha(sums), state: "uploaded" });
  return {
    release: {id: 1000+index, tag_name: "v"+version, target_commitish: RECENT[version],
      draft: false, prerelease: false, name: "Ghost FTP "+version, assets},
    sums,
  };
}

function runScenario(change, mode = "push") {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-release-metadata-test-"));
  const bin = path.join(directory, "bin");
  const data = path.join(directory, "state");
  fs.mkdirSync(bin);
  fs.mkdirSync(data);
  fs.writeFileSync(path.join(bin, "gh"), mockGh, {mode:0o755});
  for (const [i, version] of Object.keys(PINNED).entries())
    fs.writeFileSync(path.join(data, version + ".json"), JSON.stringify(legacyFixture(version, i)));
  for (const [i, version] of Object.keys(RECENT).entries()) {
    const sample = recentFixture(version, i);
    fs.writeFileSync(path.join(data, version + ".json"), JSON.stringify(sample.release));
    fs.writeFileSync(path.join(data, version + ".sum"), sample.sums);
  }
  if (change) change(data);
  try {
    const result = spawnSync("bash", [".github/scripts/enforce-preview-release-metadata.sh"], {
      cwd: ROOT, encoding:"utf8", timeout:60000,
      env:{...process.env, PATH:bin+path.delimiter+process.env.PATH,
        GH_TOKEN:"fake-for-local-unit-test", GITHUB_REPOSITORY:"bren-wp/Ghost-FTP",
        GHOSTFTP_METADATA_EVENT:mode, MOCK_GH_DIRECTORY:data},
    });
    const get = version => JSON.parse(fs.readFileSync(path.join(data, version + ".json"), "utf8"));
    return {result,historical:get("0.99.24"),newer:get("0.99.25"),
      recent:Object.fromEntries(Object.keys(RECENT).map(v=>[v,get(v)]))};
  } finally {
    fs.rmSync(directory, {recursive:true, force:true});
  }
}

test("release corrector quarantines incomplete v0.99.24 and repairs recent 21-asset Preview metadata", () => {
  const {result, historical, newer, recent} = runScenario();
  assert.equal(result.status, 0, [result.stdout,result.stderr,result.error].join("\n"));
  assert.match(result.stdout, /Historical v0\.99\.24 remains INCOMPLETE/);
  assert.match(result.stdout, /Confirmed Preview classification.*v0\.99\.25/);
  assert.deepEqual(historical.assets, OBSERVED);
  assert.equal(historical.prerelease, false, "cannot certify the incomplete historical release");
  assert.equal(newer.prerelease, true);
  assert.equal(newer.name, "Ghost FTP 0.99.25 Preview");
  for (const [version, release] of Object.entries(recent)) {
    // The corrector checks recent historical editions and the rolling
    // predecessor/current pair, not every intervening archived version.
    const shouldRepair = ["0.99.37", "0.99.38", PREVIOUS_PUBLISHED].includes(version);
    assert.equal(release.prerelease, shouldRepair, version+" unexpected classification");
    if (shouldRepair) assert.equal(release.name,"Ghost FTP "+version+" Preview");
    assert.equal(release.assets.length,21);
    assert.equal(release.target_commitish,RECENT[version]);
  }
  assert.match(result.stdout, /not yet publicly visible/, "the next unpublished version may be absent on push");
});

test("manual Preview metadata recovery repairs a published predecessor after external classification drift", () => {
  const {result, recent, newer} = runScenario(null, "manual");
  assert.equal(result.status, 0, [result.stdout, result.stderr, result.error].join("\n"));
  assert.equal(recent[PREVIOUS_PUBLISHED].prerelease, true);
  assert.equal(recent[PREVIOUS_PUBLISHED].name, "Ghost FTP " + PREVIOUS_PUBLISHED + " Preview");
  assert.equal(recent[PREVIOUS_PUBLISHED].assets.length, 21);
  assert.equal(newer.prerelease, true, "legacy release still receives verified correction");
});

test("release corrector blocks historical asset tampering before any recent mutation", () => {
  const {result,newer,recent} = runScenario(data => {
    const file = path.join(data,"0.99.24.json");
    const historical = JSON.parse(fs.readFileSync(file,"utf8"));
    historical.assets[0].size++;
    fs.writeFileSync(file,JSON.stringify(historical));
  });
  assert.notEqual(result.status,0);
  assert.match(result.stderr,/asset inventory changed; manual provenance review required/);
  assert.equal(newer.prerelease,false,"must fail before editing other releases");
  assert.equal(recent[PREVIOUS_PUBLISHED].prerelease,false);
});

test("release corrector refuses tampered released SHA256SUMS without changing the affected release", () => {
  const {result,recent} = runScenario(data =>
    fs.writeFileSync(path.join(data,PREVIOUS_PUBLISHED + ".sum"),"corrupt\n"));
  assert.notEqual(result.status,0);
  assert.match(result.stderr,/SHA256SUMS|SHA-256/);
  assert.equal(recent[PREVIOUS_PUBLISHED].prerelease,false);
});


test("v0.99.41 unexpected historical asset rename must fail closed before any release metadata mutation", () => {
  const {result,recent} = runScenario(data => {
    const file = path.join(data, "0.99.41.json");
    const release = JSON.parse(fs.readFileSync(file, "utf8"));
    const asset = release.assets.find(a => a.name === "GhostFTP-macOS-v0.99.41-Preview.zip");
    assert.ok(asset);
    asset.name = "GhostFTP-macOS-v0.99.41.zip";
    fs.writeFileSync(file, JSON.stringify(release));
  });
  assert.notEqual(result.status, 0, "unproven asset identity must never be auto-repaired");
  assert.match(result.stderr, /asset identity|manual provenance|asset recovery forbidden/);
  assert.equal(recent["0.99.41"].prerelease, false);
});
