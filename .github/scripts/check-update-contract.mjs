#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (rel) => fs.readFileSync(path.join(root, rel), "utf8");
const exists = (rel) => fs.existsSync(path.join(root, rel));
const meta = JSON.parse(read("version.json"));
const failures = [];

const template = JSON.parse(read("updates/latest.template.json"));
if (template.version !== meta.version) failures.push("updates/latest.template.json version drift");
const allowedTop = ["notes", "platforms", "pub_date", "version"];
for (const key of Object.keys(template)) {
  if (!allowedTop.includes(key)) failures.push(`unsupported public updater field in template: ${key}`);
}
for (const platform of ["windows-x86_64", "linux-x86_64"]) {
  const pkg = template.platforms?.[platform];
  if (!pkg) failures.push(`missing update platform ${platform}`);
  else {
    const keys = Object.keys(pkg).sort().join(",");
    if (keys !== "signature,url") failures.push(`${platform} must contain only signature,url`);
    if (!pkg.url?.startsWith("https://")) failures.push(`${platform} URL must use HTTPS`);
  }
}

for (const obsolete of [
  "updates/channels/preview.template.json",
  "updates/channels/stable.template.json",
]) {
  if (exists(obsolete)) failures.push(`obsolete incompatible update template must be removed: ${obsolete}`);
}

const tauri = JSON.parse(read("ghostftp-desktop/src-tauri/tauri.conf.json"));
if (tauri.plugins?.updater?.endpoints?.[0] !== "https://ghostftp.com/updates/latest.json") {
  failures.push("desktop updater must use the official Ghost FTP update service");
}
const releaseConfig = JSON.parse(read("ghostftp-desktop/src-tauri/updater-release.conf.json"));
if (releaseConfig.bundle?.createUpdaterArtifacts !== true) {
  failures.push("release updater config must enable createUpdaterArtifacts");
}

for (const rel of [
  "updates/README.md",
  "updates/DEPLOYMENT.md",
  "updates/RELEASE_RUNBOOK.md",
  "updates/SECURITY.md",
  "updates/schema/latest.schema.json",
  "updates/scripts/build-manifest.mjs",
  "updates/scripts/verify-manifest.mjs",
  "updates/hosting/.htaccess.example",
  "updates/hosting/nginx.conf.example",
]) {
  if (!exists(rel)) failures.push(`missing update-system file: ${rel}`);
}

const userFacing = [
  "ghostftp-desktop/src/components/AboutDialog.tsx",
];
for (const rel of userFacing) {
  const source = read(rel);
  for (const [pattern, label] of [
    [/latest\.json/i, "service filename"],
    [/update endpoint/i, "endpoint terminology"],
    [/update manifest/i, "manifest terminology"],
    [/\bJSON\b/, "JSON terminology"],
  ]) {
    if (pattern.test(source)) failures.push(`${rel}: user-facing update copy exposes ${label}`);
  }
}

for (const [rel, needle] of [
  [".github/workflows/ghostftp-build.yml", "TAURI_SIGNING_PRIVATE_KEY"],
  [".github/workflows/ghostftp-build.yml", "Enforce stable update signing preflight"],
  [".github/workflows/ghostftp-build.yml", "Stable main build requires TAURI_SIGNING_PRIVATE_KEY before native packaging."],
  [".github/workflows/ghostftp-build.yml", "updater-release.conf.json"],
  [".github/workflows/ghostftp-release.yml", "GhostFTP-v$VERSION-Update-Service.zip"],
  [".github/workflows/ghostftp-release.yml", "updates/scripts/build-manifest.mjs"],
]) {
  if (!read(rel).includes(needle)) failures.push(`${rel}: missing update release contract: ${needle}`);
}

for (const [rel, needle] of [
  [".github/workflows/ghostftp-release.yml", "Enforce stable updater publication contract"],
  [".github/workflows/ghostftp-release.yml", "RELEASE_CHANNEL=$RELEASE_CHANNEL"],
  [".github/workflows/ghostftp-release.yml", "Stable publication requires both Windows and Linux Tauri updater signatures."],
  [".github/scripts/publish-release.sh", 'if [ "$CHANNEL" = "stable" ]; then'],
  [".github/scripts/publish-release.sh", "GhostFTP-Windows-x64-Setup-v${VERSION}.exe.sig"],
  [".github/scripts/publish-release.sh", "GhostFTP-Linux-x86_64-v${VERSION}.AppImage.sig"],
  [".github/scripts/publish-release.sh", "GhostFTP-v${VERSION}-latest.json"],
  [".github/scripts/publish-release.sh", "GhostFTP-v${VERSION}-Update-Service.zip"],
  [".github/scripts/publish-release.sh", "updates/scripts/verify-manifest.mjs"],
]) {
  if (!read(rel).includes(needle)) {
    failures.push(`${rel}: stable release must fail closed without updater proof: ${needle}`);
  }
}

if (failures.length) {
  console.error("Desktop update contract failed:");
  for (const failure of failures) console.error(` - ${failure}`);
  process.exit(1);
}
console.log(`Desktop update contract OK for Ghost FTP ${meta.version}.`);
