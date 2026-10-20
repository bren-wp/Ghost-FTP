#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const workflowDir = path.join(root, ".github", "workflows");
const failures = [];

const files = fs
  .readdirSync(workflowDir)
  .filter((name) => /\.ya?ml$/i.test(name))
  .sort();

const obsoleteActionRefs = [
  [/actions\/checkout@v[1-6]\b/g, "actions/checkout must use v7+"],
  [/actions\/setup-node@v[1-6]\b/g, "actions/setup-node must use v7+"],
  [/actions\/setup-go@v[1-6]\b/g, "actions/setup-go must use v7+"],
  [/actions\/setup-java@v[1-5]\b/g, "actions/setup-java must use v6+"],
  [/android-actions\/setup-android@v[1-3]\b/g, "android-actions/setup-android must use v4+"],
  [/gradle\/actions\/setup-gradle@v[1-5]\b/g, "gradle/actions/setup-gradle must use v6+"],
  [/actions\/upload-artifact@v[1-6]\b/g, "actions/upload-artifact must use v7+"],
];

for (const name of files) {
  const rel = path.join(".github", "workflows", name).replaceAll("\\", "/");
  const source = fs.readFileSync(path.join(workflowDir, name), "utf8");
  for (const [pattern, label] of obsoleteActionRefs) {
    pattern.lastIndex = 0;
    if (pattern.test(source)) failures.push(`${rel}: ${label}`);
  }
}

for (const rel of [
  ".github/workflows/ghostftp-quality.yml",
  ".github/workflows/validate-win-hardening.yml",
]) {
  const source = fs.readFileSync(path.join(root, rel), "utf8");
  if (!source.includes("actions/setup-go@v7")) {
    failures.push(`${rel}: expected actions/setup-go@v7`);
  }
  if (!source.includes("cache: false")) {
    failures.push(`${rel}: setup-go cache must remain disabled while helper modules have no go.sum`);
  }
}

// Publishing is automatic ONLY from a successful main push native build.
const releaseWorkflow = fs.readFileSync(
  path.join(workflowDir, "ghostftp-release.yml"), "utf8"
);
if (!releaseWorkflow.includes("  workflow_run:") ||
    !releaseWorkflow.includes('workflows: ["Ghost FTP native build"]') ||
    releaseWorkflow.includes("  workflow_dispatch:") ||
    !releaseWorkflow.includes("github.event.workflow_run.event == 'push'")) {
  failures.push("Automatic release trigger must require completed native builds on main pushes");
}
if (!releaseWorkflow.includes("  group: ghostftp-release-${{ github.event.workflow_run.head_branch }}") ||
    !/^  cancel-in-progress: false$/m.test(releaseWorkflow)) {
  failures.push("PR events must not cancel active main release promotions");
}
if (!releaseWorkflow.includes("github.event.workflow_run.head_branch == 'main'") ||
    !releaseWorkflow.includes("github.event.workflow_run.head_repository.full_name == github.repository") ||
    !releaseWorkflow.includes("CURRENT_MAIN_SHA") ||
    !releaseWorkflow.includes("TRIGGER_RUN_ID") ||
    !releaseWorkflow.includes("ghostftp-build.yml")) {
  failures.push("Auto-promotion must authenticate origin, current main SHA and successful exact-SHA native build");
}

const viteConfig = fs.readFileSync(
  path.join(root, "ghostftp-desktop", "vite.config.ts"),
  "utf8"
);
if (viteConfig.includes("__dirname")) {
  failures.push("ghostftp-desktop/vite.config.ts: __dirname is incompatible with Vite native config loading");
}
if (!viteConfig.includes("fileURLToPath(import.meta.url)")) {
  failures.push("ghostftp-desktop/vite.config.ts: missing ESM-safe config directory resolution");
}

if (failures.length) {
  console.error("CI/runtime hardening contract failed:");
  for (const failure of failures) console.error(` - ${failure}`);
  process.exit(1);
}

console.log("CI/runtime hardening contract OK");
