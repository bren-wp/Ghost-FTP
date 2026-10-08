#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const contract = JSON.parse(fs.readFileSync(path.join(root, "docs/qa/premium-75-screen-contract.json"), "utf8"));
const expected = { windows: 30, linux: 29, android: 16 };
const errors = [];
const seen = new Set();

for (const [platform, count] of Object.entries(expected)) {
  const entries = contract.platforms?.[platform];
  if (!Array.isArray(entries) || entries.length !== count) {
    errors.push(`${platform}: expected ${count} references, found ${entries?.length ?? "none"}`);
    continue;
  }
  for (const entry of entries) {
    const key = `${platform}/${entry.id}`;
    if (seen.has(key)) errors.push(`duplicate screen: ${key}`);
    seen.add(key);
    if (entry.source !== `screens/${platform}/${entry.id}.png`) {
      errors.push(`incorrect ZIP reference for ${key}`);
    }
    if (typeof entry.surface !== "string" ||
        !/^(android\/app\/|ghostftp-desktop\/)/.test(entry.surface) ||
        !fs.existsSync(path.join(root, entry.surface))) {
      errors.push(`missing live product surface for ${key}: ${entry.surface}`);
    }
    // Mapping a source file is *not* a successful screenshot pixel-diff or
    // functional test. Keep this fact explicit until actual captures exist.
    if (entry.visual_acceptance !== "pending") {
      errors.push(`unproven pixel parity claim for ${key}`);
    }
  }
}
if (seen.size !== 75) errors.push(`expected 75 unique references, got ${seen.size}`);
if (contract.macos?.referenceCount !== 0) {
  errors.push("macOS must not invent a reference-screen set absent from the ZIP");
}
if (errors.length) {
  console.error("Ghost FTP premium reference traceability failed:");
  for (const failure of errors) console.error(" - " + failure);
  process.exit(1);
}
console.log("Ghost FTP premium reference traceability OK: 30 Windows + 29 Linux + 16 Android mapped; pixel acceptance remains pending.");
