#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const settings = fs.readFileSync(path.join(root, "ghostftp-desktop/src/components/Settings.tsx"), "utf8");
const styles = fs.readFileSync(path.join(root, "ghostftp-desktop/src/styles.css"), "utf8");
const errors = [];
for (const name of ["appearance", "language", "transfers", "connection", "security", "advanced"]) {
  if (!settings.includes(`<Nav section="${name}"`)) errors.push(`Missing real Settings category: ${name}`);
}
for (const [source, pattern, explanation] of [
  [settings, 'aria-label="Preferences categories"', "Missing semantic Preferences category navigation"],
  [settings, 'ghost-settings-premium-layout', "Settings is not wired to reference desktop layout"],
  [settings, 'syncOnly ? "" : "ghost-settings-premium-layout"', "Standalone Sync should not render empty navigation rail"],
  [settings, 'aria-current={active ? "page" : undefined}', "Settings selection is not exposed to assistive technology"],
  [styles, '@media (min-width: 900px) and (min-height: 520px)', "Desktop-only Settings viewport guard missing"],
  [styles, '.ghost-settings-premium-layout > .ghost-preferences-nav.ghost-settings-tabs', "Reference sidebar selector missing"],
  [styles, 'flex-direction: column !important;', "Desktop Settings sidebar not vertical"],
  [styles, '.ghost-settings-premium-layout > main', "Settings main content is not given remaining width"],
  [styles, '@media (max-width: 560px)', "Compact/mobile layout fallback missing"],
]) {
  if (!source.includes(pattern)) errors.push(explanation);
}
if (/placeholder settings|mock-settings|fake-connection/i.test(settings)) {
  errors.push("Unexpected placeholder content in Settings");
}
if (errors.length) {
  for (const error of errors) console.error(error);
  process.exit(1);
}
console.log("Premium Windows/Linux Settings: real sections, semantic desktop rail and compact layout contracts passed. Pixel acceptance requires native QA screenshots.");
