import fs from "node:fs";

const [file, ...rest] = process.argv.slice(2);
if (!file) throw new Error("Usage: node verify-manifest.mjs <file> [--expected-version=x.y.z]");
const options = Object.fromEntries(rest.map((part) => {
  const i = part.indexOf("=");
  if (i < 0) throw new Error(`Expected --key=value, received: ${part}`);
  return [part.slice(0, i).replace(/^--/, ""), part.slice(i + 1)];
}));

const manifest = JSON.parse(fs.readFileSync(file, "utf8"));
const allowedTop = new Set(["version", "notes", "pub_date", "platforms"]);
for (const key of Object.keys(manifest)) {
  if (!allowedTop.has(key)) throw new Error(`Unsupported top-level update field: ${key}`);
}
if (!/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(manifest.version ?? "")) {
  throw new Error("Missing or invalid version");
}
if (options["expected-version"] && manifest.version !== options["expected-version"]) {
  throw new Error(`Expected version ${options["expected-version"]}, received ${manifest.version}`);
}
if (manifest.pub_date !== undefined && Number.isNaN(Date.parse(manifest.pub_date))) {
  throw new Error("Invalid pub_date");
}
if (manifest.notes !== undefined && typeof manifest.notes !== "string") {
  throw new Error("notes must be a string");
}

const platformNames = ["windows-x86_64", "linux-x86_64"];
if (!manifest.platforms || typeof manifest.platforms !== "object" || Array.isArray(manifest.platforms)) {
  throw new Error("Missing platforms");
}
for (const key of Object.keys(manifest.platforms)) {
  if (!platformNames.includes(key)) throw new Error(`Unsupported platform: ${key}`);
}
for (const platform of platformNames) {
  const pkg = manifest.platforms[platform];
  if (!pkg || typeof pkg !== "object" || Array.isArray(pkg)) throw new Error(`Missing ${platform}`);
  const keys = Object.keys(pkg);
  if (keys.length !== 2 || !keys.includes("url") || !keys.includes("signature")) {
    throw new Error(`${platform} must contain only url and signature`);
  }
  const url = new URL(pkg.url);
  if (url.protocol !== "https:") throw new Error(`${platform} URL must use HTTPS`);
  if (typeof pkg.signature !== "string" || pkg.signature.trim().length < 16 || /REPLACE_WITH/i.test(pkg.signature)) {
    throw new Error(`${platform} signature missing or placeholder`);
  }
}
console.log(`Ghost FTP desktop update response OK: ${manifest.version}`);
