#!/usr/bin/env node
// Exact source-token checks for the maintained Windows/Linux/macOS clients.
// Source checks cannot substitute for native screenshots or interactive tests.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (p) => fs.readFileSync(path.join(root, p), "utf8");
const tokens = JSON.parse(read("docs/branding/premium-ui-tokens.json"));
const desktop = read("ghostftp-desktop/src/styles.css");
const mac = read("macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift");
const brand = read("ghostftp-desktop/branding/ghostftp-logo.svg");
if (!desktop.includes("width: 214px !important") ||
    !desktop.includes("min-height: 43px") ||
    !desktop.includes("--accent: 56 171 255;") ||
    !mac.includes(".frame(minWidth: 840, minHeight: 560)") ||
    !mac.includes("GhostPremiumPalette.accent") ||
    !brand.includes("MORE THAN TRANSFER. TOTAL CONTROL.") ||
    tokens.referenceCounts.windows !== 30 ||
    tokens.referenceCounts.linux !== 29 ||
    tokens.macos.referenceScreenCount !== 0) {
  throw Error("Windows/Linux/macOS identity, layout or reference inventory drift");
}
console.log("Ghost FTP Windows/Linux/macOS brand and reference geometry checks: PASS.");
console.log("59 Windows/Linux concepts require real native pixel comparison.");
