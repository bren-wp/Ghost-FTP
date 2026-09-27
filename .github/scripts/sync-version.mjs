#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const checkOnly = process.argv.includes("--check");
const nextIndex = process.argv.indexOf("--check-next");
const baseVersion = nextIndex >= 0 ? process.argv[nextIndex + 1] : null;
const config = JSON.parse(fs.readFileSync(path.join(root, "version.json"), "utf8"));
const { version, previousVersion, build, releaseDate, androidVersionCode, legacyVersion, legacyDisplay } = config;
const semverPattern = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

if (!semverPattern.test(version)) throw new Error(`invalid SemVer: ${version}`);
if (previousVersion && !semverPattern.test(previousVersion)) {
  throw new Error(`invalid previous SemVer: ${previousVersion}`);
}
if (previousVersion === version) {
  throw new Error("previousVersion must not equal the active version");
}
if (!Number.isInteger(androidVersionCode) || androidVersionCode <= 0) throw new Error("androidVersionCode must be positive");

const drift = [];
const read = (rel) => fs.readFileSync(path.join(root, rel), "utf8");
const apply = (rel, next) => {
  const file = path.join(root, rel);
  const current = fs.readFileSync(file, "utf8");
  const normalize = (value) => value.replace(/\r\n/g, "\n");
  if (normalize(current) === normalize(next)) return;
  if (checkOnly) drift.push(rel);
  else { fs.writeFileSync(file, next); console.log(`updated ${rel}`); }
};
const updateJson = (rel, mutate) => {
  const json = JSON.parse(read(rel));
  mutate(json);
  apply(rel, JSON.stringify(json, null, 2) + "\n");
};
const replaceRequired = (text, pattern, replacement, rel) => {
  if (!pattern.test(text)) throw new Error(`${rel}: expected pattern missing`);
  pattern.lastIndex = 0;
  return text.replace(pattern, replacement);
};

updateJson("ghostftp-desktop/package.json", (j) => { j.version = version; });
updateJson("ghostftp-desktop/package-lock.json", (j) => {
  j.version = version;
  if (j.packages?.[""]) j.packages[""].version = version;
});
for (const rel of [
  "ghostftp-desktop/src-tauri/Cargo.toml",
  "ghostftp-desktop/src-tauri/ghostftp-cli/Cargo.toml",
  "ghostftp-desktop/src-tauri/ghostftp-agent-proto/Cargo.toml",
  "ghostftp-desktop/src-tauri/ghostftp-agentd/Cargo.toml",
]) {
  apply(rel, replaceRequired(read(rel), /^version = "[^"]+"$/m, `version = "${version}"`, rel));
}
updateJson("ghostftp-desktop/src-tauri/tauri.conf.json", (j) => { j.version = version; });

{
  const rel = "ghostftp-desktop/src/lib/release.ts";
  let text = read(rel);
  for (const [pattern, value] of [
    [/PRODUCT_VERSION = "[^"]+"/, `PRODUCT_VERSION = "${version}"`],
    [/PRODUCT_VERSION_DISPLAY = "[^"]+"/, `PRODUCT_VERSION_DISPLAY = "${version}"`],
    [/PRODUCT_VERSION_BADGE = "[^"]+"/, `PRODUCT_VERSION_BADGE = "${version}"`],
    [/PRODUCT_BUILD = "[^"]+"/, `PRODUCT_BUILD = "${build}"`],
    [/PRODUCT_RELEASE_DATE = "[^"]+"/, `PRODUCT_RELEASE_DATE = "${releaseDate}"`],
  ]) text = replaceRequired(text, pattern, value, rel);
  apply(rel, text);
}
{
  const rel = "android/app/build.gradle.kts";
  let text = read(rel);
  text = replaceRequired(text, /versionCode = \d+/, `versionCode = ${androidVersionCode}`, rel);
  text = replaceRequired(text, /versionName = "[^"]+"/, `versionName = "${version}"`, rel);
  apply(rel, text);
}
{
  const rel = "android/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt";
  let text = read(rel);
  for (const [pattern, value] of [
    [/const val VERSION = "[^"]+"/, `const val VERSION = "${version}"`],
    [/const val VERSION_DISPLAY = "[^"]+"/, `const val VERSION_DISPLAY = "${version}"`],
    [/const val VERSION_BADGE = "[^"]+"/, `const val VERSION_BADGE = "${version}"`],
    [/const val BUILD = "[^"]+"/, `const val BUILD = "${build}"`],
    [/const val RELEASE_DATE = "[^"]+"/, `const val RELEASE_DATE = "${releaseDate}"`],
  ]) text = replaceRequired(text, pattern, value, rel);
  apply(rel, text);
}
for (const rel of ["tools/ghostftp-runtime/main.go", "tools/ghostftp-installer/main.go"]) {
  apply(rel, replaceRequired(read(rel), /const version = "[^"]+"/, `const version = "${version}"`, rel));
}
updateJson("updates/latest.template.json", (j) => {
  j.version = version;
  j.notes = `Ghost FTP ${version}`;
  j.platforms["windows-x86_64"].url =
    `https://github.com/bren-wp/Ghost-FTP/releases/download/v${version}/GhostFTP-Windows-x64-Setup-v${version}.exe`;
  j.platforms["linux-x86_64"].url =
    `https://github.com/bren-wp/Ghost-FTP/releases/download/v${version}/GhostFTP-Linux-x86_64-v${version}.AppImage`;
});

// Documentation screenshots intentionally stay pinned to the newest published
// release. Development version bumps must not relabel release-proven imagery
// before that version is actually published.

if (baseVersion) {
  const parse = (v) => {
    const m = semverPattern.exec(v);
    if (!m) throw new Error(`invalid base version: ${v}`);
    return m.slice(1).map(Number);
  };
  const [bm,bn,bp] = parse(baseVersion);
  const [m,n,p] = parse(version);
  const sameVersion = m === bm && n === bn && p === bp;
  const nextMinor = m === bm && n === bn + 1 && p === 0;
  const nextPatch = m === bm && n === bn && p === bp + 1;
  const firstStable = bm === 0 && m === 1 && n === 0 && p === 0;
  if (!sameVersion && !nextMinor && !nextPatch && !firstStable) {
    throw new Error(
      `invalid version step ${baseVersion} -> ${version}; keep the active development version, use next minor for a new development release, or next patch for a hotfix`,
    );
  }
}
const notes = path.join(root, "docs", "releases", `${version}.md`);
if (!fs.existsSync(notes)) drift.push(path.relative(root, notes));
if (checkOnly && drift.length) {
  console.error("Version metadata drift detected:");
  for (const rel of drift) console.error(` - ${rel}`);
  process.exit(1);
}
