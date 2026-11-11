#!/usr/bin/env node
// Verify Windows/Linux brand tokens in maintained sources; not pixel comparison.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = file => fs.readFileSync(path.join(root, file), "utf8");
const tokens = JSON.parse(read("docs/branding/premium-ui-tokens.json"));
const css = read("ghostftp-desktop/src/styles.css");
const logo = read("ghostftp-desktop/branding/ghostftp-logo.svg");
if (!css.includes("width: 214px !important") ||
    !css.includes("min-height: 43px") ||
    !css.includes("--accent: 56 171 255;") ||
    !logo.includes("MORE THAN TRANSFER. TOTAL CONTROL.") ||
    tokens.referenceCounts.windows !== 30 ||
    tokens.referenceCounts.linux !== 29)
  throw Error("Windows/Linux branding or reference geometry drift");
console.log("Windows/Linux desktop design source checks: PASS; native visual acceptance is pending.");
