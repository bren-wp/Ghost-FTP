#!/usr/bin/env node
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const mapping = JSON.parse(fs.readFileSync(path.join(root, "docs/releases/version-map.json"), "utf8")).versions;
const repo = process.env.GITHUB_REPOSITORY;
if (!repo) throw new Error("GITHUB_REPOSITORY is required");

const apply = process.argv.includes("--apply");
const verifyOnly = process.argv.includes("--verify");
if (apply && verifyOnly) throw new Error("use either --apply or --verify");

function run(command, args, { allowFailure = false, encoding = "utf8" } = {}) {
  const result = spawnSync(command, args, {
    cwd: root,
    env: process.env,
    encoding: encoding === null ? undefined : encoding,
    maxBuffer: 64 * 1024 * 1024,
  });
  if (!allowFailure && result.status !== 0) {
    throw new Error(
      `${command} ${args.join(" ")} failed (${result.status}):\n${String(result.stderr || result.stdout || "")}`,
    );
  }
  return result;
}

function ghJson(endpoint, { method = "GET", fields = [], allow404 = false } = {}) {
  const args = ["api", endpoint];
  if (method !== "GET") args.push("--method", method);
  for (const [kind, key, value] of fields) args.push(kind, `${key}=${value}`);
  const result = run("gh", args, { allowFailure: allow404 });
  if (result.status !== 0) return null;
  const text = String(result.stdout || "").trim();
  return text ? JSON.parse(text) : null;
}

function remoteTagSha(tag) {
  const out = run("git", ["ls-remote", "--tags", "origin", `refs/tags/${tag}`], { allowFailure: true });
  const line = String(out.stdout || "").trim();
  return line ? line.split(/\s+/)[0] : null;
}

function sourceTagSha(tag) {
  const result = run("git", ["rev-list", "-n1", tag], { allowFailure: true });
  return result.status === 0 ? String(result.stdout).trim() : null;
}

function refObject(tag) {
  return ghJson(`repos/${repo}/git/ref/tags/${tag}`, { allow404: true })?.object ?? null;
}

function resolvedRefCommitSha(tag) {
  let object = refObject(tag);
  for (let depth = 0; object && depth < 4; depth += 1) {
    if (object.type === "commit") return object.sha;
    if (object.type !== "tag") return null;
    object = ghJson(`repos/${repo}/git/tags/${object.sha}`)?.object ?? null;
  }
  return null;
}

function createCanonicalTag(tag, sourceSha, legacyTag) {
  const existing = resolvedRefCommitSha(tag);
  if (existing) {
    if (existing !== sourceSha) {
      throw new Error(`${tag} resolves to ${existing}, expected ${sourceSha}`);
    }
    return;
  }

  let apiError = null;
  try {
    // Prefer an annotated tag object. This preserves the exact historical
    // source commit while avoiding a direct historical-commit ref write when
    // GitHub App workflow protections are stricter.
    const tagObject = ghJson(`repos/${repo}/git/tags`, {
      method: "POST",
      fields: [
        ["-f", "tag", tag],
        ["-f", "message", `Canonical Ghost FTP release tag for ${legacyTag}`],
        ["-f", "object", sourceSha],
        ["-f", "type", "commit"],
      ],
    });
    if (!tagObject?.sha) throw new Error(`${tag}: GitHub did not return a tag object SHA`);

    ghJson(`repos/${repo}/git/refs`, {
      method: "POST",
      fields: [
        ["-f", "ref", `refs/tags/${tag}`],
        ["-f", "sha", tagObject.sha],
      ],
    });
  } catch (error) {
    apiError = error;
  }

  if (resolvedRefCommitSha(tag) !== sourceSha) {
    // Fallback to authenticated git transport. checkout persists GITHUB_TOKEN
    // credentials, and a normal tag push is accepted on repositories where
    // the Git Data ref endpoint rejects historical workflow-bearing commits.
    run("git", ["config", "user.name", "github-actions[bot]"]);
    run("git", ["config", "user.email", "41898282+github-actions[bot]@users.noreply.github.com"]);
    run("git", ["tag", "-f", "-a", tag, sourceSha, "-m", `Canonical Ghost FTP release tag for ${legacyTag}`]);
    const pushed = run("git", ["push", "origin", `refs/tags/${tag}`], { allowFailure: true });
    if (pushed.status !== 0) {
      throw new Error(
        `${tag}: API tag creation failed (${apiError ?? "unknown error"}) and git push failed: ${String(pushed.stderr || pushed.stdout || "")}`,
      );
    }
  }

  const resolved = resolvedRefCommitSha(tag);
  if (resolved !== sourceSha) {
    throw new Error(`${tag}: resolves to ${resolved}, expected ${sourceSha}`);
  }
}

function canonicalizeText(text, item) {
  const rc = item.legacy.match(/-rc\.(\d+)$/i)?.[1];
  if (!rc) throw new Error(`invalid legacy version ${item.legacy}`);
  const base = item.legacy.replace(/-rc\.\d+$/i, "");
  const basePattern = base.replace(/\./g, "\\.");
  let out = String(text ?? "");

  // Cover every historical spelling used by old release assets:
  // 2.1.1-rc.8, 2.1.1-RC8, 2.1.1-rc8, 2.1.1.rc8, and v-prefixed forms.
  out = out.replace(
    new RegExp(`v${basePattern}(?:[-_. ]?rc[.-]?${rc})`, "gi"),
    `v${item.canonical}`,
  );
  out = out.replace(
    new RegExp(`${basePattern}(?:[-_. ]?rc[.-]?${rc})`, "gi"),
    item.canonical,
  );
  out = out.replace(
    new RegExp(`\\bRC[.-]?${rc}\\b`, "gi"),
    item.canonical,
  );
  return out;
}

function isChecksumAsset(name) {
  return /SHA256SUMS\.txt$/i.test(name) || /\.sha256$/i.test(name);
}

function getReleaseByTag(tag) {
  return ghJson(`repos/${repo}/releases/tags/${tag}`, { allow404: true });
}

function getAssets(releaseId) {
  return ghJson(`repos/${repo}/releases/${releaseId}/assets?per_page=100`) || [];
}

function downloadAsset(assetId) {
  const result = run(
    "gh",
    ["api", "-H", "Accept: application/octet-stream", `repos/${repo}/releases/assets/${assetId}`],
    { encoding: null },
  );
  return result.stdout;
}

function uploadAsset(tag, file) {
  run("gh", ["release", "upload", tag, file, "--clobber"]);
}

function assertRelease(item, release, assetCountBefore) {
  const expectedTag = `v${item.canonical}`;
  if (release.tag_name !== expectedTag) throw new Error(`${expectedTag}: release tag mismatch`);
  if (release.name !== `Ghost FTP ${item.canonical}`) throw new Error(`${expectedTag}: release name mismatch`);
  if (release.prerelease) throw new Error(`${expectedTag}: still marked prerelease`);
  const assets = getAssets(release.id);
  if (assets.length !== assetCountBefore) {
    throw new Error(`${expectedTag}: asset count changed ${assetCountBefore} -> ${assets.length}`);
  }
  const legacyRc = item.legacy.match(/-rc\.(\d+)$/i)?.[1];
  for (const asset of assets) {
    if (/2\.1\.1|RC\d+/i.test(asset.name)) {
      throw new Error(`${expectedTag}: legacy asset name remains: ${asset.name}`);
    }
  }
  if (legacyRc && new RegExp(`(?:2\\.1\\.1|RC${legacyRc}\\b)`, "i").test(release.body || "")) {
    throw new Error(`${expectedTag}: legacy version remains in release body`);
  }
}

run("git", ["fetch", "--tags", "--force"]);

for (const item of mapping) {
  const oldTag = `v${item.legacy}`;
  const newTag = `v${item.canonical}`;
  let release = getReleaseByTag(newTag) || getReleaseByTag(oldTag);
  if (!release) throw new Error(`release not found for ${oldTag} / ${newTag}`);

  const oldRemoteSha = remoteTagSha(oldTag);
  const newRemoteSha = refObject(newTag)?.sha ?? null;
  const oldResolvedSha = oldRemoteSha ? (sourceTagSha(oldTag) || oldRemoteSha) : null;
  const newResolvedSha = newRemoteSha ? resolvedRefCommitSha(newTag) : null;

  if (oldResolvedSha && oldResolvedSha !== item.sourceSha) {
    throw new Error(`${oldTag} points to ${oldResolvedSha}, expected ${item.sourceSha}`);
  }
  if (newResolvedSha && newResolvedSha !== item.sourceSha) {
    throw new Error(`${newTag} points to ${newResolvedSha}, expected ${item.sourceSha}`);
  }

  const assetsBefore = getAssets(release.id);
  const binaryDigests = new Map(
    assetsBefore.filter(a => !isChecksumAsset(a.name)).map(a => [canonicalizeText(a.name, item), a.digest]),
  );

  if (verifyOnly) {
    assertRelease(item, release, assetsBefore.length);
    if (oldRemoteSha) throw new Error(`${oldTag}: legacy Git tag still exists`);
    if (!newRemoteSha || newResolvedSha !== item.sourceSha) {
      throw new Error(`${newTag}: canonical Git tag missing or points at the wrong source`);
    }
    console.log(`verified ${newTag}`);
    continue;
  }

  console.log(`${apply ? "migrating" : "would migrate"} ${oldTag} -> ${newTag} at ${item.sourceSha}`);
  if (!apply) continue;

  // Download checksum assets before changing the release so their contents can
  // be rewritten to the canonical asset names without touching binary assets.
  const checksumBackups = [];
  for (const asset of assetsBefore.filter(a => isChecksumAsset(a.name))) {
    checksumBackups.push({
      asset,
      name: canonicalizeText(asset.name, item),
      bytes: downloadAsset(asset.id),
    });
  }

  createCanonicalTag(newTag, item.sourceSha, oldTag);

  const body = canonicalizeText(release.body || "", item);
  release = ghJson(`repos/${repo}/releases/${release.id}`, {
    method: "PATCH",
    fields: [
      ["-f", "tag_name", newTag],
      ["-f", "target_commitish", item.sourceSha],
      ["-f", "name", `Ghost FTP ${item.canonical}`],
      ["-f", "body", body],
      ["-F", "prerelease", "false"],
      ["-F", "draft", "false"],
    ],
  });

  // Rename binary assets in place so their bytes and GitHub digest remain
  // unchanged. Checksums are re-uploaded because their filename references
  // must change alongside the renamed assets.
  for (const asset of assetsBefore.filter(a => !isChecksumAsset(a.name))) {
    const newName = canonicalizeText(asset.name, item);
    if (newName !== asset.name) {
      ghJson(`repos/${repo}/releases/assets/${asset.id}`, {
        method: "PATCH",
        fields: [["-f", "name", newName]],
      });
    }
  }

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-release-migrate-"));
  try {
    for (const backup of checksumBackups) {
      ghJson(`repos/${repo}/releases/assets/${backup.asset.id}`, { method: "DELETE" });
      const updated = canonicalizeText(Buffer.from(backup.bytes).toString("utf8"), item);
      const target = path.join(tmp, backup.name);
      fs.writeFileSync(target, updated);
      uploadAsset(newTag, target);
    }
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }

  const migrated = getReleaseByTag(newTag);
  assertRelease(item, migrated, assetsBefore.length);

  for (const asset of getAssets(migrated.id).filter(a => !isChecksumAsset(a.name))) {
    const expectedDigest = binaryDigests.get(asset.name);
    if (!expectedDigest || asset.digest !== expectedDigest) {
      throw new Error(`${newTag}: binary asset digest changed for ${asset.name}`);
    }
  }

  if (refObject(oldTag)) {
    ghJson(`repos/${repo}/git/refs/tags/${oldTag}`, { method: "DELETE" });
  }
  if (refObject(oldTag)) throw new Error(`${oldTag}: legacy tag still exists after deletion`);
  console.log(`migrated and verified ${oldTag} -> ${newTag}`);
}

if (verifyOnly) console.log("all published releases use canonical 0.x versions");
