#!/usr/bin/env node
// Promote ONLY genuine, validated Windows runtime captures from the exact
// successful native GitHub Actions QA artifact. No image synthesis or mock UI.
import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";

const root = process.cwd();
const input = path.resolve(process.argv[2] || "qa/windows");
const version = JSON.parse(fs.readFileSync("version.json","utf8")).version;
const sourceSha = process.env.GHOSTFTP_CAPTURE_COMMIT || "";
const runId = process.env.GITHUB_RUN_ID || "";
if (!/^0\.95\.\d+$/.test(version) || !/^[a-f0-9]{40}$/.test(sourceSha) || !/^\d+$/.test(runId)) {
  throw new Error("Missing exact, trusted native QA provenance.");
}
const mapping = {
  "ghostftp-native-files.png": "Native-Window",
  "ghostftp-native-new-connection.png": "New-Connection",
  "ghostftp-native-sites.png": "Site-Manager",
  "ghostftp-native-transfers.png": "Transfer-Center",
  "ghostftp-native-settings.png": "Preferences",
  "ghostftp-native-file-properties.png": "File-Properties",
  "ghostftp-native-about.png": "About"
};
const dest = path.join(root,"docs/assets/screenshots");
const manifestFile = path.join(dest,"README-NATIVE-CAPTURE.json");
if (fs.existsSync(manifestFile)) {
  const prior = JSON.parse(fs.readFileSync(manifestFile,"utf8"));
  if (prior.version === version) {
    console.log("Real screenshot upgrade already promoted for this version.");
    process.exit(0);
  }
}
const images = {};
const unique = new Set();
for (const [name,label] of Object.entries(mapping)) {
  const srcPath = path.join(input,`GhostFTP-Windows-Preview-${label}.png`);
  const reportPath = path.join(input,`GhostFTP-Windows-Preview-${label}.txt`);
  if (!fs.existsSync(srcPath) || !fs.existsSync(reportPath)) {
    throw new Error(`Missing real runtime QA image/report for ${label}`);
  }
  const report = fs.readFileSync(reportPath,"utf8");
  if (!report.includes("WindowBounds=1290x852") ||
      !report.includes("Executable=") ||
      !report.includes("CaptureMethod=") ||
      !report.includes(`ImageBytes=${fs.statSync(srcPath).size}`)) {
    throw new Error(`Invalid native QA image report for ${label}`);
  }
  const bytes = fs.readFileSync(srcPath);
  if (bytes.length < 30000 || !bytes.subarray(0,8).equals(
    Buffer.from([137,80,78,71,13,10,26,10]))) {
    throw new Error(`Invalid PNG runtime screenshot for ${label}`);
  }
  if (bytes.readUInt32BE(16) !== 1290 || bytes.readUInt32BE(20) !== 852) {
    throw new Error(`Wrong native viewport for ${label}`);
  }
  const sha256 = crypto.createHash("sha256").update(bytes).digest("hex");
  if (unique.has(sha256)) throw new Error(`Duplicate captured UI screen: ${label}`);
  unique.add(sha256);
  images[`docs/assets/screenshots/${name}`] = sha256;
}
for (const [name,label] of Object.entries(mapping)) {
  fs.copyFileSync(path.join(input,`GhostFTP-Windows-Preview-${label}.png`),path.join(dest,name));
}
fs.writeFileSync(manifestFile,JSON.stringify({
  version,
  platform:"Windows",
  capture_source_sha: sourceSha,
  github_actions_run_id: runId,
  provenance:"Ghost FTP native build, real Windows WebView2 compositor QA, 1290x852",
  images
},null,2)+"\n");
const readmePath=path.join(root,"README.md");
let readme=fs.readFileSync(readmePath,"utf8");
const label=`Seven authentic Ghost FTP Windows ${version} running-application screenshots (1290×852), captured in [GitHub Actions native QA](https://github.com/bren-wp/Ghost-FTP/actions/runs/${runId}) at source commit \\`${sourceSha.slice(0,12)}\\`. Each PNG is SHA-256 pinned in [the capture manifest](docs/assets/screenshots/README-NATIVE-CAPTURE.json). No concept artwork or generated screenshots.`;
readme=readme.replace(/<p align="center"><sub>[^<]*<\/sub><\/p>/, `<p align="center"><sub>${label}</sub></p>`);
fs.writeFileSync(readmePath,readme);
console.log(`Promoted ${Object.keys(mapping).length} genuine Windows ${version} QA screenshots, run ${runId}.`);
