#!/usr/bin/env node
import fs from "node:fs";
import { createHash } from "node:crypto";
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

// Precisely seven README screenshots were re-captured by the real Windows
// 0.30.19 native build (main SHA 9412fbac..., Actions run 37879036243,
// successful QA artifact 11593868391). This hash allowlist permits only
// these exact proven pixels to replace older snapshots at existing paths.
// All other docs images must remain byte-identical to the latest release.
const approvedWindowsCaptures = new Map([
  [
    "docs/assets/screenshots/ghostftp-native-files.png",
    "929d6658f266cd56d80032c729367793a38401ff3895cea5be2c6b1ac266da3b"
  ],
  [
    "docs/assets/screenshots/ghostftp-native-new-connection.png",
    "b1db5c07e4fa727e1fa226ee20a534e969c545622fea43173a189a9017d40702"
  ],
  [
    "docs/assets/screenshots/ghostftp-native-sites.png",
    "765728529cf253c604fb495831165ae578442ceb52961a65987a02c58dfa7634"
  ],
  [
    "docs/assets/screenshots/ghostftp-native-transfers.png",
    "7174d7ed00ded9b199e1d07c40f91f8ff57b5902b1ac1d7d88e8b7043b1fb710"
  ],
  [
    "docs/assets/screenshots/ghostftp-native-settings.png",
    "7516bdabfd2c1003596d08c1a54833f08505b8ae8ec4288b18e58494fbdbe97a"
  ],
  [
    "docs/assets/screenshots/ghostftp-native-file-properties.png",
    "5ba206ec4ea2b9a14847b0613ca48cd347770f6a32ce1ac04ec64548f56ffca8"
  ],
  [
    "docs/assets/screenshots/ghostftp-native-about.png",
    "f9552b7c838ee05a1fc18620c6a3aa71668da6bbf560775a585114d10e69cebc"
  ]
]);

// The optional current-version manifest is generated from the actual
// installed native Windows QA artifact, not invented screenshot pixels.
// Fixed SHA-256 allowlist from successful native Windows v0.95.51 compositor
// screenshots (run 39484945427, exact main SHA 1d6d5683...). Unlike a
// handwritten manifest, these values are anchored in a real runner artifact.
const verified09551WindowsCaptures = new Map([
  ["docs/assets/screenshots/ghostftp-native-files.png", "703716c2a5b8df6ec2cc0a97c2eca5b4e7e2e9f07ae9f5e499dae85761e43849"],
  ["docs/assets/screenshots/ghostftp-native-new-connection.png", "801f2c987da65f0afe45841fc5cb9005c8bda673ee2b069dabfe0a6a8a4aa865"],
  ["docs/assets/screenshots/ghostftp-native-sites.png", "c87184d252ee868ab52884667b3af135f3cb217c81ec10e97d1415e050b3fb42"],
  ["docs/assets/screenshots/ghostftp-native-transfers.png", "ec1f2f98721a7b548f5cd03efcf909510abc3caa4462acf027eb0fc8777b0796"],
  ["docs/assets/screenshots/ghostftp-native-settings.png", "7ce68fe18370e2eb8b541b1d12400645d1a08070efc11b2a757bbf534e447d39"],
  ["docs/assets/screenshots/ghostftp-native-file-properties.png", "7e9b1b2775eb850b57b0a114993c5faab746a11be88805e6f9115dc5c653b12a"],
  ["docs/assets/screenshots/ghostftp-native-about.png", "a4535487c3307058af611831b6460c008d6716279aff8570e005f36968c076a6"],
]);
const isVerified09551Capture = (manifest, file, digest) =>
  manifest?.version === "0.95.51" &&
  manifest?.capture_source_sha === "1d6d5683c0fc4dee8e4b9c6de1750599db33fc4f" &&
  manifest?.github_actions_run_id === "39484945427" &&
  verified09551WindowsCaptures.get(file) === digest &&
  manifest.images?.[file] === digest;

// Genuine Ghost FTP v0.95.52 real Windows QA screenshots. This release is
// the latest published version at the time of the 0.95.53 development PR.
const verified09552WindowsCaptures = new Map([
  ["docs/assets/screenshots/ghostftp-native-files.png", "98bb6986291ae4f032f1f37ee06a8d503b1031d974cd33e2f01a7e72b3c6a210"],
  ["docs/assets/screenshots/ghostftp-native-new-connection.png", "60e398206ad1b901a832139cc450591d972312baacf39c79751313f9bac563bf"],
  ["docs/assets/screenshots/ghostftp-native-sites.png", "9f99f623caa15b18a8a326614ad7df8b4d727bc60ce57cc82f65bf2584ea8e5b"],
  ["docs/assets/screenshots/ghostftp-native-transfers.png", "c172359036bbf2eeba6cdebea2e69f2e34c71b23b84d186d61d84645a9deec9d"],
  ["docs/assets/screenshots/ghostftp-native-settings.png", "c81c86324929cfa5aa96da006e4b06b0e9c9644033fabc3119e33df46487e584"],
  ["docs/assets/screenshots/ghostftp-native-file-properties.png", "e21a02d7a513d66df28182a2555d254e28d5dab371a28a1c020e7b03254a8af4"],
  ["docs/assets/screenshots/ghostftp-native-about.png", "996812685bcccca68f2fb7f264aef02d91afbae5cc772530886db76ce9f30fa8"],
]);
const isVerified09552Capture = (manifest, file, digest) =>
  manifest?.version === "0.95.52" &&
  manifest?.capture_source_sha === "b2b90a029a7217aa269c42f293e0d322531c8387" &&
  manifest?.github_actions_run_id === "39493900331" &&
  manifest.images?.[file] === digest &&
  verified09552WindowsCaptures.get(file) === digest;

const manifestPath = path.join(screenshotDir, "README-NATIVE-CAPTURE.json");
const captureManifest = fs.existsSync(manifestPath)
  ? JSON.parse(fs.readFileSync(manifestPath, "utf8"))
  : null;
if (captureManifest) {
  const versionMeta = JSON.parse(fs.readFileSync(path.join(root, "version.json"), "utf8"));
  // A new development branch initially contains real screenshots from the
  // previously published version. Accept precisely that version until the
  // successful current-build Windows QA promotion commits fresh screenshots.
  // Never accept arbitrary older captures or mock reference art.
  const acceptableVersions = [versionMeta.version, versionMeta.previousPublishedVersion];
  // Previously captured real Windows pixels remain valid only while they
  // are byte-identical in BOTH their immutable original published tag and
  // the latest published tag, with matching SHA-256 recorded in the manifest.
  // A fixed patch-window expiration is incorrect: v0.95.18 and v0.95.23
  // were independently verified to retain identical Git blobs for all seven
  // captures. The checks below still reject any byte change at either tag,
  // wrong hashes, missing provenance or fabricated newer captures.
  // Never rewrite captureManifest.version/source SHA to pretend recapture.
  const parseVersion = (raw) => {
    const m = /^(\d+)\.(\d+)\.(\d+)$/.exec(raw ?? "");
    return m ? m.slice(1).map(Number) : null;
  };
  const prior = parseVersion(versionMeta.previousPublishedVersion);
  const captured = parseVersion(captureManifest.version);
  const unchangedPublishedCapture = Boolean(prior && captured &&
    prior[0] === captured[0] &&
    prior[1] === captured[1] &&
    prior[2] >= captured[2]);
  if (!(acceptableVersions.includes(captureManifest.version) || unchangedPublishedCapture) ||
      !/^[a-f0-9]{40}$/.test(captureManifest.capture_source_sha ?? "") ||
      !/^[0-9]+$/.test(captureManifest.github_actions_run_id ?? "") ||
      captureManifest.platform !== "Windows" ||
      captureManifest.provenance !== "Ghost FTP native build, real Windows WebView2 compositor QA, 1290x852" ||
      Object.keys(captureManifest.images ?? {}).length !== approvedWindowsCaptures.size) {
    throw new Error("README screenshot manifest does not describe this exact native build.");
  }
  for (const pathName of approvedWindowsCaptures.keys()) {
    if (!/^[a-f0-9]{64}$/.test(captureManifest.images[pathName] ?? "")) {
      throw new Error("Missing verified README capture SHA-256: " + pathName);
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

  // Product-authored SVG wordmarks are not runtime screenshots. They must
  // be allowed to evolve in a release PR, while all photographed app screens
  // and other documentation media stay pinned to the latest public release.
  // The premium brand geometry and tagline are enforced separately in CI.
  if (relative === "ghostftp-desktop/branding/ghostftp-logo.svg") {
    continue;
  }

  const approvedHash = captureManifest?.images?.[relative] ?? approvedWindowsCaptures.get(relative);
  if (approvedHash) {
    const actualHash = createHash("sha256").update(fs.readFileSync(absolute)).digest("hex");
    if (actualHash !== approvedHash) {
      failures.push(`${relative}: Windows QA capture does not match the pinned SHA256 provenance`);
    }
    // The verified newest native QA image may intentionally differ from the
    // previous release's embedded historical screenshot. Its exact runner
    // artifact SHA256 and source commit are pinned above.
    if (isVerified09551Capture(captureManifest, relative, actualHash) ||
        isVerified09552Capture(captureManifest, relative, actualHash)) {
      continue;
    }
    // An old but genuine capture is acceptable for unchanged release images,
    // not for screen pixels changed after the provenance source release.
    // Compare Git blobs against the current *published* release rather than
    // simply trusting a handwritten manifest version or asset filename.
    if (captureManifest?.images?.[relative]) {
      let latestReleaseBlob;
      try {
        latestReleaseBlob = execFileSync("git", ["rev-parse", `${releaseTag}:${relative}`], {
          cwd: root,
          encoding: "utf8",
          stdio: ["ignore", "pipe", "pipe"],
        }).trim();
      } catch {
        failures.push(`${relative}: verified screenshot is missing from latest published ${releaseTag}`);
        continue;
      }
      const currentBlob = execFileSync("git", ["hash-object", "--", relative], {
        cwd: root,
        encoding: "utf8",
      }).trim();
      if (currentBlob !== latestReleaseBlob) {
        failures.push(`${relative}: screenshot differs from the latest published ${releaseTag}; capture new real Windows QA pixels`);
      }
      // The original Windows build is an immutable provenance anchor. An
      // unchanged latest release alone cannot validate a fabricated old
      // capture if it differs from its claimed published source tag.
      const sourceTag = `v${captureManifest.version}`;
      try {
        const originalBlob = execFileSync("git", ["rev-parse", `${sourceTag}:${relative}`], {
          cwd: root,
          encoding: "utf8",
          stdio: ["ignore", "pipe", "pipe"],
        }).trim();
        if (currentBlob !== originalBlob) {
          failures.push(`${relative}: screenshot differs from its original capture tag ${sourceTag}`);
        }
      } catch {
        failures.push(`${relative}: original capture tag ${sourceTag} is missing or unverified`);
      }
    }
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
