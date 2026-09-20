import fs from "node:fs";

const file = process.argv[2];
if (!file) throw new Error("Usage: node verify-manifest.mjs <manifest.json>");
const manifest = JSON.parse(fs.readFileSync(file, "utf8"));

if (!["preview", "stable"].includes(manifest.channel)) throw new Error("Invalid update channel");
if (typeof manifest.version !== "string" || !manifest.version) throw new Error("Missing version");
if (typeof manifest.pub_date !== "string" || Number.isNaN(Date.parse(manifest.pub_date))) throw new Error("Invalid pub_date");
if (typeof manifest.minimum_supported_version !== "string" || !manifest.minimum_supported_version) throw new Error("Missing minimum_supported_version");

for (const platform of ["windows-x86_64", "linux-x86_64"]) {
  const pkg = manifest.platforms?.[platform];
  if (!pkg) throw new Error(`Missing ${platform}`);
  if (typeof pkg.url !== "string" || !pkg.url.startsWith("https://")) throw new Error(`${platform} URL must use HTTPS`);
  if (typeof pkg.signature !== "string" || pkg.signature.length < 16) throw new Error(`${platform} signature missing`);
}
console.log(`Ghost FTP update manifest OK: ${manifest.channel} ${manifest.version}`);
