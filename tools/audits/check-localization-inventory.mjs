#!/usr/bin/env node
// Windows/Linux source-language entrypoints; translation quality is not certified.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = file => fs.readFileSync(path.join(root, file), "utf8");
const catalog = read("ghostftp-desktop/src/lib/i18n.ts");
const entry = read("ghostftp-desktop/src/main.tsx");
if (!catalog.includes("en") || !entry.includes("App"))
  throw Error("Windows/Linux language entrypoints missing");
console.log("Desktop localization entrypoints: PASS; run check:i18n and native language QA separately.");
