#!/usr/bin/env node
// Source-level desktop palette and reference contract, not pixel-parity proof.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (file) => fs.readFileSync(path.join(root, file), "utf8");
const tokens = JSON.parse(read("docs/branding/premium-ui-tokens.json"));
const contract = JSON.parse(read("docs/qa/premium-75-screen-contract.json"));
const css = read("ghostftp-desktop/src/styles.css");
const mac = read("macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift");
const macCloud = read("macos/Sources/GhostFTPMacApp/Views/CloudWorkspace.swift");
const errors = [];
const requireSource = (text, needle, label) => {
  if (!text.includes(needle)) errors.push(label);
};
const rgb = (hex) => [0, 2, 4].map(index => parseInt(hex.slice(index + 1, index + 3), 16));
for (const [semantic, cssName] of [
  ["background", "--bg"], ["panel", "--bg-panel"], ["raised", "--bg-subtle"],
  ["accent", "--accent"], ["success", "--success"], ["danger", "--danger"],
]) {
  const [r, g, b] = rgb(tokens.colors[semantic]);
  requireSource(css, `${cssName}: ${r} ${g} ${b};`, `Desktop ${semantic} palette mismatch`);
  const macName = semantic === "panel" ? "sidebar" : semantic;
  requireSource(mac, `static let ${macName} = Color(red: ${r}/255, green: ${g}/255, blue: ${b}/255)`,
    `macOS ${semantic} palette mismatch`);
}
for (const semantic of ["navy", "slate", "cyan"]) {
  requireSource(css, `--ghost-premium-${semantic}: ${tokens.colors[semantic]};`,
    `Desktop ${semantic} token mismatch`);
}
requireSource(mac, "static let slate = Color(red: 19/255, green: 45/255, blue: 82/255)", "macOS selected surface mismatch");
requireSource(css, "width: 214px !important", "Windows/Linux sidebar width mismatch");
requireSource(css, "min-height: 43px", "Windows/Linux navigation row mismatch");
requireSource(mac, ".listRowBackground(", "macOS list selection background absent");
requireSource(macCloud, 'Button(L10n.text("Cancel transfer"))', "macOS cancel action missing");
requireSource(macCloud, ".buttonStyle(.borderedProminent)", "macOS primary cloud button missing");
for (const [platform, count] of Object.entries({windows:30, linux:29})) {
  if (tokens.referenceCounts[platform] !== count || contract.platforms?.[platform]?.length !== count) {
    errors.push(`${platform} reference count mismatch`);
  }
  if (contract.platforms?.[platform]?.some(entry => entry.visual_acceptance !== "pending")) {
    errors.push(`${platform} requires native screenshot acceptance`);
  }
}
if (tokens.macos.referenceScreenCount !== 0 || contract.macos?.referenceCount !== 0) {
  errors.push("Cannot invent a macOS reference screenshot");
}
if (errors.length) {
  console.error("Ghost FTP desktop UI token audit failed:");
  errors.forEach(error => console.error(" - " + error));
  process.exitCode = 1;
} else {
  console.log("Windows/Linux/macOS premium design tokens pass source checks; 59 desktop screenshots remain unverified.");
}
