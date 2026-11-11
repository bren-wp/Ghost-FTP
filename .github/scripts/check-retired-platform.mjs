#!/usr/bin/env node
/**
 * Fail-closed product retirement gate. Only previously published historical
 * release attestations may mention discontinued platform packages.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

export function auditRetiredPlatform(root = ROOT) {
  const forbiddenPaths = [
    "android",
    "macos",
    "ghostftp-desktop/src-tauri/tauri.macos.conf.json",
    ".github/workflows/ghostftp-macos.yml",
    ".github/workflows/ghostftp-macos-shared-desktop.yml",
    ".github/workflows/ghostftp-macos-developer-id.yml",
    "ghostftp-desktop/scripts/check-macos-shortcut-modifiers.mjs",
    "docs/guides/MACOS_INSTALLATION.md",
    "docs/handbook/MACOS.md",
    "docs/mobile",
    ".github/workflows/ghostftp-android.yml",
    ".github/workflows/ghostftp-android-play-upload.yml",
    ".github/workflows/ghostftp-release-history.yml",
    ".github/scripts/backfill-releases.sh",
    "assets/branding/ghostftp-adaptive-foreground.svg",
  ];
  const findings = forbiddenPaths.filter(rel => fs.existsSync(path.join(root, rel)))
    .map(path => ({path, problem: "discontinued platform or obsolete delivery code restored"}));

  const activeFiles = [
    "version.json",
    "README.md",
    "SECURITY.md",
    "EULA.txt",
    "LICENSE.txt",
    ".github/workflows/ghostftp-quality.yml",
    ".github/workflows/ghostftp-release.yml",
    ".github/workflows/version-sync.yml",
    ".github/scripts/sync-version.mjs",
    ".github/scripts/publish-release.sh",
    ".github/scripts/check-release-index.sh",
    ".github/scripts/check-release-channel.mjs",
    "docs/architecture/STRUCTURE.md",
    "docs/architecture/NAMING.md",
    "docs/guides/INSTALLATION.md",
    "docs/guides/UPDATES.md",
    "docs/qa/PLATFORM_PARITY.md",
    "docs/qa/README.md",
    "docs/status/PRODUCTION_READINESS.md",
    "ghostftp-desktop/src-tauri/Cargo.toml",
    "ghostftp-desktop/src-tauri/src/lib.rs",
    "ghostftp-desktop/src-tauri/src/remotefs/local.rs",
    "ghostftp-desktop/src/lib/shortcuts.ts",
    "ghostftp-desktop/src/components/TitleBar.tsx",
    "ghostftp-desktop/src/styles.css",
    "ghostftp-desktop/scripts/check-ui-contract.mjs",
    "ghostftp-desktop/scripts/check-interaction-hardening.mjs",
    "ghostftp-desktop/scripts/check-premium-files-reference.mjs",
  ];
  const oldPlatform = /(?:\bandroid\b|ghostftp-android|\b(?:apk|aab)\b|androidversioncode|\bmacos\b|swiftui|\.dmg\b|apple-native)/i;
  for (const rel of activeFiles) {
    const absolute = path.join(root, rel);
    if (!fs.existsSync(absolute)) {
      findings.push({path: rel, problem: "required desktop/release contract is missing"});
      continue;
    }
    const source = fs.readFileSync(absolute, "utf8");
    if (oldPlatform.test(source)) {
      findings.push({path: rel, problem: "active desktop source still advertises or depends on discontinued platform"});
    }
  }

  const versionFile = path.join(root, "version.json");
  if (fs.existsSync(versionFile)) {
    try {
      const version = JSON.parse(fs.readFileSync(versionFile, "utf8"));
      if (Object.keys(version).some(key => /android|apk|aab/i.test(key))) {
        findings.push({path:"version.json", problem:"discontinued platform release version remains"});
      }
    } catch {
      findings.push({path:"version.json", problem:"invalid JSON"});
    }
  }
  return {passed: findings.length===0, findings, activeFilesChecked: activeFiles.length,
    activePlatforms:["windows","linux"]};
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const result=auditRetiredPlatform();
  console.log(JSON.stringify(result,null,2));
  if (!result.passed) process.exitCode=1;
}
