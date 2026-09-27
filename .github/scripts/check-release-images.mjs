#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const releaseTag = process.env.GHOSTFTP_LATEST_RELEASE_TAG?.trim();

if (!releaseTag) {
  throw new Error("GHOSTFTP_LATEST_RELEASE_TAG is required");
}

const ignoredDirs = new Set([".git", "node_modules", "target", "dist", "build"]);
const imageExt = new Set([".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".avif"]);
const markdownFiles = [];
const candidates = new Map();

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory() && ignoredDirs.has(entry.name)) continue;
    const absolute = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(absolute);
    } else if (entry.isFile() && entry.name.toLowerCase().endsWith(".md")) {
      markdownFiles.push(absolute);
    }
  }
}

function normalizeLocalReference(docFile, raw) {
  let ref = raw.trim();
  if (!ref || /^(?:https?:|data:|mailto:|#)/i.test(ref)) return null;
  ref = ref.split("#", 1)[0].split("?", 1)[0];
  try {
    ref = decodeURIComponent(ref);
  } catch {
    // Keep the literal path and let the existence check explain the failure.
  }
  const absolute = ref.startsWith("/")
    ? path.join(root, ref.replace(/^\/+/, ""))
    : path.resolve(path.dirname(docFile), ref);
  const relative = path.relative(root, absolute).split(path.sep).join("/");
  if (!relative || relative.startsWith("../") || path.isAbsolute(relative)) {
    throw new Error(`image reference escapes repository: ${raw} in ${path.relative(root, docFile)}`);
  }
  if (!imageExt.has(path.extname(relative).toLowerCase())) return null;
  return relative;
}

walk(root);

const markdownImage = /!\[[^\]]*\]\(([^)\s]+)(?:\s+["'][^"']*["'])?\)/g;
const htmlImage = /<img\b[^>]*\bsrc\s*=\s*["']([^"']+)["'][^>]*>/gi;

for (const docFile of markdownFiles) {
  const source = fs.readFileSync(docFile, "utf8");
  for (const regex of [markdownImage, htmlImage]) {
    regex.lastIndex = 0;
    let match;
    while ((match = regex.exec(source)) !== null) {
      const relative = normalizeLocalReference(docFile, match[1]);
      if (relative) {
        const refs = candidates.get(relative) ?? [];
        refs.push(path.relative(root, docFile).split(path.sep).join("/"));
        candidates.set(relative, refs);
      }
    }
  }
}

const screenshotDir = path.join(root, "docs", "assets", "screenshots");
if (fs.existsSync(screenshotDir)) {
  for (const entry of fs.readdirSync(screenshotDir, { withFileTypes: true })) {
    if (!entry.isFile()) continue;
    const relative = path.posix.join("docs/assets/screenshots", entry.name);
    if (imageExt.has(path.extname(relative).toLowerCase()) && !candidates.has(relative)) {
      candidates.set(relative, ["docs/assets/screenshots/* provenance set"]);
    }
  }
}

const failures = [];
for (const [relative, refs] of [...candidates.entries()].sort(([a], [b]) => a.localeCompare(b))) {
  const absolute = path.join(root, relative);
  if (!fs.existsSync(absolute) || !fs.statSync(absolute).isFile()) {
    failures.push(`${relative}: referenced by ${refs.join(", ")} but missing from current tree`);
    continue;
  }

  let releaseBlob;
  try {
    releaseBlob = execFileSync("git", ["rev-parse", `${releaseTag}:${relative}`], {
      cwd: root,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    }).trim();
  } catch {
    failures.push(`${relative}: not present in latest published release ${releaseTag}`);
    continue;
  }

  const currentBlob = execFileSync("git", ["hash-object", "--", relative], {
    cwd: root,
    encoding: "utf8",
  }).trim();

  if (currentBlob !== releaseBlob) {
    failures.push(
      `${relative}: current blob ${currentBlob} differs from ${releaseTag} blob ${releaseBlob}; documentation images must come from the latest published release`,
    );
  }
}

if (!candidates.size) failures.push("no local documentation images were discovered");

if (failures.length) {
  console.error(`Latest-release image provenance failed against ${releaseTag}:`);
  for (const failure of failures) console.error(` - ${failure}`);
  process.exit(1);
}

console.log(
  `Latest-release image provenance OK: ${candidates.size} local images match ${releaseTag}.`,
);
