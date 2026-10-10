#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const source = (file) => fs.readFileSync(path.join(root, file), "utf8");
const chrome = source("ghostftp-desktop/src/components/PremiumFilesChrome.tsx");
const app = source("ghostftp-desktop/src/App.tsx");
const css = source("ghostftp-desktop/src/styles.css");
const violations = [];
const requireText = (actual, text, error) => {
  if (!actual.includes(text)) violations.push(error);
};
for (const component of ["PremiumFilesChrome", "PremiumLiveTransferStrip"]) {
  requireText(app, `<${component} />`, `${component} is missing from real Files workspace`);
}
requireText(chrome, 'openNewConnection({', "Quick connection does not hand off to validated editor");
requireText(chrome, 'type="submit"', "Quick connection cannot submit from keyboard");
requireText(chrome, 'event.preventDefault()', "Quick connection submits an invalid native form");
requireText(chrome, 'PROTOCOL_DEFAULT_PORT[value]', "Protocol change does not update default port");
requireText(chrome, 'state.activeSessionId', "Connected status not bound to real session");
requireText(chrome, 'state.byId', "Transfer rows are not bound to live backend transfer store");
requireText(chrome, 'openDialog("transferCenter")', "Transfer strip does not open working transfer center");
requireText(chrome, 'No transfers in progress', "No honest empty transfer state");
for (const cssClass of [
  "ghost-premium-files-heading",
  "ghost-premium-quickconnect",
  "ghost-premium-connection-state",
  "ghost-premium-transfer-strip",
  "ghost-premium-transfer-progress",
]) {
  requireText(css, `.${cssClass}`, `Missing genuine Files reference styling: ${cssClass}`);
}
for (const dangerous of ['type="password"', 'localStorage', 'sessionStorage', 'fakeTransfer', 'mockTransfer', 'example.com']) {
  if (chrome.includes(dangerous)) violations.push(`Credential/demo logic in inline quick-connect: ${dangerous}`);
}
if (violations.length > 0) {
  console.error("Premium Windows/Linux Files UI contract failed:");
  for (const violation of violations) console.error(" - " + violation);
  process.exit(1);
}
console.log("Real Windows/Linux Files header, secure quick-connect handoff and live transfer strip: contract passed. Pixel acceptance still pending.");
