#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT/ghostftp-desktop/src-tauri"
before="$(mktemp)"
cp Cargo.lock "$before"
trap 'rm -f "$before"' EXIT

# This one-way repair removes dependency features associated with a retired
# platform without silently upgrading or adding third-party package versions.
cargo metadata --format-version 1 >/dev/null

node - "$before" Cargo.lock <<'NODE'
const fs=require("node:fs");
const assert=require("node:assert/strict");
function packages(p) {
  const input=fs.readFileSync(p,"utf8").split(/(?=^\[\[package\]\]$)/m);
  const entries=new Map();
  for(const block of input) {
    const name=block.match(/^name = "([^"]+)"/m)?.[1];
    const version=block.match(/^version = "([^"]+)"/m)?.[1];
    const source=block.match(/^source = "([^"]+)"/m)?.[1]??"local";
    if(name&&version)entries.set(name+"@"+version+"#"+source,true);
  }
  return entries;
}
const old=packages(process.argv[2]), updated=packages(process.argv[3]);
for(const pkg of updated.keys()){
  assert.ok(old.has(pkg),"Refuse unexpected Cargo third-party package upgrade/addition: "+pkg);
}
assert.ok(updated.size>0,"Refuse empty Cargo lock graph");
console.log("Cargo lockfile retired-feature reconciliation: kept "+updated.size+" existing package versions; removed "+(old.size-updated.size)+" orphaned packages");
NODE
