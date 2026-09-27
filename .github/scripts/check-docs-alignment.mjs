#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (rel) => fs.readFileSync(path.join(root, rel), "utf8");
const exists = (rel) => fs.existsSync(path.join(root, rel));

const meta = JSON.parse(read("version.json"));
const version = meta.version;
const previousVersion = meta.previousVersion;

const failures = [];
const requireFile = (rel) => {
  if (!exists(rel)) failures.push(`missing required documentation/release file: ${rel}`);
};
const requireIncludes = (rel, needle, label) => {
  if (!read(rel).includes(needle)) failures.push(`${rel}: missing ${label}: ${needle}`);
};

const operationalDocs = [
  "README.md",
  "docs/README.md",
  "docs/ROADMAP.md",
  "docs/architecture/NAMING.md",
  "docs/architecture/STRUCTURE.md",
  "docs/build/REFERENCE_RUNTIME.md",
  "docs/build/STATUS.md",
  "docs/development/BUILDING.md",
  "docs/guides/INSTALLATION.md",
  "docs/guides/SUPPORT.md",
  "docs/guides/UNINSTALL.md",
  "docs/guides/UPDATES.md",
  "docs/mobile/ANDROID_PARITY.md",
  "docs/product/FEATURES.md",
  "docs/product/ROADMAP.md",
  "docs/product/STATUS_AND_NEXT.md",
  "docs/product/UI_UX.md",
  "docs/qa/README.md",
  "docs/qa/CLICK.md",
  "docs/qa/INSTALLER.md",
  "docs/qa/PIXEL_PARITY.md",
  "docs/qa/PROTOCOL_E2E.md",
  "docs/qa/RESPONSIVE.md",
  "docs/qa/TITLEBAR.md",
  "docs/qa/TRANSFERS.md",
  "docs/release/PROCESS.md",
];

const forbidden = [
  [/2\.1\.1-rc\.\d+/i, "legacy 2.1.1-rc version"],
  [/2\.1\.1 RC\d+/i, "legacy 2.1.1 RC display version"],
  [/\bRC\d+\b/, "legacy RC badge/version"],
  [/desktop-tauri/i, "obsolete desktop-tauri path"],
  [/BUILD_STATUS\.md/, "obsolete BUILD_STATUS.md reference"],
  [/ghostftp-native-rc\d*/i, "obsolete RC native workflow"],
  [/ghostftp-rc\d*-release/i, "obsolete RC release workflow"],
  [/Ghost-FTP-Premium/, "obsolete Premium repository reference"],
];

for (const rel of operationalDocs) {
  requireFile(rel);
  if (!exists(rel)) continue;
  const source = read(rel);
  for (const [pattern, label] of forbidden) {
    if (pattern.test(source)) failures.push(`${rel}: contains ${label}`);
  }
}

for (const rel of [
  "README.md",
  "docs/README.md",
  "docs/build/STATUS.md",
  "docs/mobile/ANDROID_PARITY.md",
  "docs/product/STATUS_AND_NEXT.md",
  "docs/releases/VERSIONING.md",
  `docs/releases/${version}.md`,
]) {
  requireFile(rel);
  if (exists(rel) && !read(rel).includes(version)) {
    failures.push(`${rel}: must mention active version ${version}`);
  }
}

for (const rel of [
  "README.md",
  "docs/build/STATUS.md",
  "docs/release/PROCESS.md",
  "docs/releases/VERSIONING.md",
]) {
  if (previousVersion) requireIncludes(rel, previousVersion, "previous/latest published canonical version");
}

requireFile("docs/releases/README.md");
requireFile("docs/releases/version-map.json");
requireFile(".github/workflows/ghostftp-preview-release.yml");

for (const obsolete of [
  ".github/workflows/ghostftp-release.yml",
  ".github/workflows/backfill-0.15.0.yml",
  ".github/workflows/ghostftp-android-release.yml",
  ".github/workflows/ghostftp-native-preview.yml",
]) {
  if (exists(obsolete)) failures.push(`obsolete workflow must be removed: ${obsolete}`);
}

requireIncludes(
  ".github/workflows/ghostftp-preview-release.yml",
  'workflows: ["Ghost FTP native build"]',
  "canonical native-build release trigger",
);
requireIncludes(
  ".github/workflows/ghostftp-preview-release.yml",
  "gh run list --workflow ghostftp-build.yml",
  "newest exact-SHA native-build selector",
);

if (failures.length) {
  console.error("Documentation/release alignment failed:");
  for (const failure of failures) console.error(` - ${failure}`);
  process.exit(1);
}

console.log(`Documentation/release alignment OK for Ghost FTP ${version} (previous ${previousVersion || "none"}).`);
