import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const OBSERVED = JSON.parse(fs.readFileSync(path.join(ROOT, ".github/scripts/v0.99.24-observed-assets.json"), "utf8"));
const PINNED = {
  "0.95.53": "27a3f8413f168e4cb8cc31cf6d7b7c85c353f5af",
  "0.99.0": "900d76e6e3273058270a48affe2664f968910a1b",
  "0.99.1": "eb2e1899f0bb953b731e7b41e0109ba4decb324a",
  "0.99.20": "567c324213e20f1b87a4b6444a65b8150f747966",
  "0.99.21": "8b68b7bb7ae2a1ac35779ac4f7f965597fcc114d",
  "0.99.24": "f36026188b3defedb9464ad04e4c234afb8a399b",
  "0.99.25": "2e00a640c65dcd4d73c0401071703c9312eb7881",
};

const mockGh = "#!/usr/bin/env node\nconst fs = require(\"node:fs\");\nconst path = require(\"node:path\");\nconst args = process.argv.slice(2);\nconst resource = args.find(x => x.startsWith(\"repos/\"));\nconst state = process.env.MOCK_GH_DIRECTORY;\nif (args[0] !== \"api\" || !resource || !state) process.exit(2);\nconst fileFor = version => path.join(state, version + \".json\");\nconst read = version => JSON.parse(fs.readFileSync(fileFor(version), \"utf8\"));\nconst versions = fs.readdirSync(state).filter(x=>x.endsWith(\".json\")).map(x=>x.slice(0,-5));\nif (resource.includes(\"/git/ref/tags/\")) {\n  const version = resource.split(\"/\").pop().slice(1);\n  process.stdout.write(read(version).target_commitish + \"\\n\");\n} else {\n  let version;\n  if (resource.includes(\"/releases/tags/\")) {\n    version = resource.split(\"/\").pop().slice(1);\n  } else {\n    const id = Number(resource.split(\"/\").pop());\n    version = versions.find(x => read(x).id === id);\n  }\n  if (!version) process.exit(3);\n  const data = read(version);\n  if (args.includes(\"PATCH\")) {\n    const patch = JSON.parse(fs.readFileSync(0, \"utf8\"));\n    if (patch.make_latest !== \"false\") process.exit(4);\n    data.name = patch.name;\n    data.prerelease = patch.prerelease;\n    fs.writeFileSync(fileFor(version), JSON.stringify(data));\n  }\n  process.stdout.write(JSON.stringify(data) + \"\\n\");\n}\n";
function fixture(version, index) {
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

function runScenario(change) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-release-metadata-test-"));
  const bin = path.join(directory, "bin");
  const data = path.join(directory, "state");
  fs.mkdirSync(bin);
  fs.mkdirSync(data);
  const gh = path.join(bin, "gh");
  fs.writeFileSync(gh, mockGh, {mode: 0o755});
  for (const [i, version] of Object.keys(PINNED).entries())
    fs.writeFileSync(path.join(data, version + ".json"), JSON.stringify(fixture(version, i)));
  if (change) change(data);
  try {
    const result = spawnSync("bash", [".github/scripts/enforce-preview-release-metadata.sh"], {
      cwd: ROOT, encoding: "utf8", timeout: 60_000,
      env: {...process.env, PATH: bin + path.delimiter + process.env.PATH,
        GH_TOKEN: "fake-for-local-unit-test", GITHUB_REPOSITORY: "bren-wp/Ghost-FTP",
        GHOSTFTP_METADATA_EVENT: "push", MOCK_GH_DIRECTORY: data},
    });
    return {result, historical: JSON.parse(fs.readFileSync(path.join(data, "0.99.24.json"), "utf8")),
      newer: JSON.parse(fs.readFileSync(path.join(data, "0.99.25.json"), "utf8"))};
  } finally {
    fs.rmSync(directory, {recursive: true, force: true});
  }
}

test("release corrector quarantines incomplete v0.99.24 while repairing verified v0.99.25", () => {
  const {result, historical, newer} = runScenario();
  assert.equal(result.status, 0, [result.stdout, result.stderr, result.error].join("\n"));
  assert.match(result.stdout, /Historical v0\.99\.24 remains INCOMPLETE/);
  assert.match(result.stdout, /Confirmed Preview classification.*v0\.99\.25/);
  assert.deepEqual(historical.assets, OBSERVED);
  assert.equal(historical.prerelease, false, "cannot certify the incomplete historical release");
  assert.equal(newer.prerelease, true);
  assert.equal(newer.name, "Ghost FTP 0.99.25 Preview");
});

test("release corrector blocks further tampering with observed v0.99.24 assets", () => {
  const {result, newer} = runScenario(data => {
    const file = path.join(data, "0.99.24.json");
    const historical = JSON.parse(fs.readFileSync(file, "utf8"));
    historical.assets[0].size++;
    fs.writeFileSync(file, JSON.stringify(historical));
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /asset inventory changed; manual provenance review required/);
  assert.equal(newer.prerelease, false, "must fail before reclassifying other releases");
});
