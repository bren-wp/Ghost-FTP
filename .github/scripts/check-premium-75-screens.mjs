#!/usr/bin/env node
import fs from "node:fs";
import { createHash } from "node:crypto";
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
// The registry alone is insufficient: enforce that the real UI uses the
// shared brand mark and that platform-specific launch/window geometries survive.
const read = (relative) => fs.readFileSync(path.join(root, relative), "utf8");
const desktopCSS = read("ghostftp-desktop/src/styles.css");
const androidActivity = read("android/app/src/main/java/com/ghostftp/android/MainActivity.kt");
const androidSplash31 = read("android/app/src/main/res/values-v31/styles.xml");
const androidSplash35 = read("android/app/src/main/res/values-v35/styles.xml");
const macShell = read("macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift");
for (const [label, condition] of [
  ["214px reference sidebar", desktopCSS.includes("width: 214px !important")],
  ["43px desktop navigation rows", desktopCSS.includes("min-height: 43px")],
  ["approved Android live icon", androidActivity.includes("setImageResource(R.drawable.ic_ghost_ftp)")],
  ["no divergent Canvas-drawn ghost", !androidActivity.includes("bodyPath.cubicTo")],
  ["Android 12-14 navy splash", androidSplash31.includes("android:windowSplashScreenAnimatedIcon")],
  ["Android 15 navy splash", androidSplash35.includes("android:windowSplashScreenAnimatedIcon")],
  ["responsive macOS window", macShell.includes(".frame(minWidth: 840, minHeight: 560)")],
  ["macOS premium brand tagline", macShell.includes('Text("TOTAL CONTROL")')],
]) {
  if (!condition) errors.push(`missing real platform brand contract: ${label}`);
}

// These three vector resources were copied unchanged from the user-provided
// design ZIP. Git-blob SHA-1 comparisons catch even tiny Bézier/gradient drift.
for (const [asset, originalBlob] of Object.entries({
  "assets/branding/ghostftp-adaptive-foreground.svg": "88466c215763451a4a2830a69a11c5097cb4fd5a",
  "assets/branding/ghostftp-logo-light.svg": "6beb59d914f407bbe3e40667c9f70923df9707f0",
  "assets/branding/ghostftp-symbol-monochrome.svg": "f7e90f05aa15bfcee747ff668e19a32eeff7ac53",
})) {
  if (!fs.existsSync(path.join(root, asset))) {
    errors.push(`missing approved vector: ${asset}`);
    continue;
  }
  const content = fs.readFileSync(path.join(root, asset));
  const digest = createHash("sha1").update(`blob ${content.length}\\0`.replace("\\0", "\0")).update(content).digest("hex");
  if (digest !== originalBlob) errors.push(`approved vector bytes changed: ${asset}`);
}

if (errors.length) {
  console.error("Ghost FTP premium reference traceability failed:");
  for (const failure of errors) console.error(" - " + failure);
  process.exit(1);
}
console.log("Ghost FTP premium reference traceability OK: 30 Windows + 29 Linux + 16 Android mapped; pixel acceptance remains pending.");
