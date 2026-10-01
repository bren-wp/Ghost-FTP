#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const assetsDir = path.join(root, "dist", "assets");
const maxBytes = 500 * 1024;

if (!fs.existsSync(assetsDir)) {
  console.error("Bundle budget failed: dist/assets does not exist. Run vite build first.");
  process.exit(1);
}

const jsFiles = fs.readdirSync(assetsDir)
  .filter((name) => name.endsWith(".js"))
  .map((name) => {
    const full = path.join(assetsDir, name);
    return { name, bytes: fs.statSync(full).size };
  })
  .sort((a, b) => b.bytes - a.bytes);

if (jsFiles.length === 0) {
  console.error("Bundle budget failed: no production JavaScript chunks found.");
  process.exit(1);
}

const oversized = jsFiles.filter((file) => file.bytes > maxBytes);
if (oversized.length) {
  console.error(`Bundle budget failed: JavaScript chunks must be <= ${maxBytes} bytes (500 KiB).`);
  for (const file of oversized) {
    console.error(` - ${file.name}: ${file.bytes} bytes`);
  }
  process.exit(1);
}

console.log("Ghost FTP production bundle budget OK");
for (const file of jsFiles.slice(0, 12)) {
  console.log(` - ${file.name}: ${file.bytes} bytes`);
}
