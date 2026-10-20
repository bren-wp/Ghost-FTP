#!/usr/bin/env node
// Import only original Windows v0.95.51 QA pixels from the successful native run.
import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";

const sourceDir = process.argv[2];
if (!sourceDir || !fs.statSync(sourceDir).isDirectory()) {
  throw new Error("Expected directory of downloaded native Windows QA artifacts");
}
const sourceSHA = "1d6d5683c0fc4dee8e4b9c6de1750599db33fc4f";
const runId = "39484945427";
const mapping = [
  ["Native-Window", "files", "main", "703716c2a5b8df6ec2cc0a97c2eca5b4e7e2e9f07ae9f5e499dae85761e43849"],
  ["New-Connection", "new-connection", "newConnection", "801f2c987da65f0afe45841fc5cb9005c8bda673ee2b069dabfe0a6a8a4aa865"],
  ["Site-Manager", "sites", "siteManager", "c87184d252ee868ab52884667b3af135f3cb217c81ec10e97d1415e050b3fb42"],
  ["Transfer-Center", "transfers", "transferCenter", "ec1f2f98721a7b548f5cd03efcf909510abc3caa4462acf027eb0fc8777b0796"],
  ["Preferences", "settings", "settings", "7ce68fe18370e2eb8b541b1d12400645d1a08070efc11b2a757bbf534e447d39"],
  ["File-Properties", "file-properties", "properties", "7e9b1b2775eb850b57b0a114993c5faab746a11be88805e6f9115dc5c653b12a"],
  ["About", "about", "about", "a4535487c3307058af611831b6460c008d6716279aff8570e005f36968c076a6"]
];
const images = {};
for (const [capture, dest, view, digest] of mapping) {
  const filename = "GhostFTP-Windows-Preview-" + capture;
  const pngPath = path.join(sourceDir, filename + ".png");
  const metadata = fs.readFileSync(path.join(sourceDir, filename + ".txt"), "utf8");
  const png = fs.readFileSync(pngPath);
  if (png.length < 20_000 || png.subarray(0,8).toString("hex") !== "89504e470d0a1a0a" ||
      png.readUInt32BE(16) !== 1290 || png.readUInt32BE(20) !== 852 ||
      !(metadata.includes("View=" + view + "\r") || metadata.includes("View=" + view + "\n")) ||
      !metadata.includes("WindowTitle=Ghost FTP") ||
      !metadata.includes("CaptureMethod=") || !metadata.includes("Executable=")) {
    throw new Error("Unverified compositor QA capture " + filename);
  }
  const actual = createHash("sha256").update(png).digest("hex");
  if (actual !== digest) throw new Error("Native QA screenshot hash mismatch: " + filename);
  const target = "docs/assets/screenshots/ghostftp-native-" + dest + ".png";
  fs.copyFileSync(pngPath, target);
  images[target] = digest;
}
fs.writeFileSync("docs/assets/screenshots/README-NATIVE-CAPTURE.json",
  JSON.stringify({
    version: "0.95.51", platform: "Windows", capture_source_sha: sourceSHA,
    github_actions_run_id: runId,
    provenance: "Ghost FTP native build, real Windows WebView2 compositor QA, 1290x852",
    images
  }, null, 2) + "\n");
const original = fs.readFileSync("README.md","utf8");
const oldCaption = /<p align="center"><sub>Seven authentic Ghost FTP Windows 0\.95\.18 running-application screenshots[\s\S]*?No concept artwork or generated screenshots\.<\/sub><\/p>/;
if (!oldCaption.test(original)) throw new Error("Old README screenshot caption not found");
const replacement = '<p align="center"><sub>Seven authentic Ghost FTP Windows 0.95.51 native WebView2 screenshots (1290×852), captured in <a href="https://github.com/bren-wp/Ghost-FTP/actions/runs/'+runId+'">successful Windows QA run '+runId+'</a> from source SHA <code>'+sourceSHA+'</code>. Original screenshots are SHA-256 pinned in <a href="docs/assets/screenshots/README-NATIVE-CAPTURE.json">the capture manifest</a>. These are actual running application pixels, not generated UI mockups.</sub></p>';
fs.writeFileSync("README.md", original.replace(oldCaption,replacement));
console.log("Imported seven real verified native Windows QA captures from run " + runId);
