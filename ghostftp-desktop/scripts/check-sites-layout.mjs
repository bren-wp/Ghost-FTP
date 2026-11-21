#!/usr/bin/env node
// Ghost FTP by Brendigo — Sites layout and action regression contract
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
const root = new URL("../", import.meta.url);
const read = (name) => readFileSync(new URL(name, root), "utf8");
const ui = read("src/components/SiteManagerDialog.tsx");
const onboarding = read("src/components/SiteManagerEmptyState.tsx");
const css = read("src/styles/site-manager-layout.css");
const entry = read("src/main.tsx");
assert.match(ui, /import \{ SiteManagerEmptyState \} from "\.\/SiteManagerEmptyState"/);
assert.match(ui, /profiles\.length === 0 && !query \? \(/);
const emptyBranch = ui.split("profiles.length === 0 && !query ? (")[1]?.split(") : (")[0];
assert.ok(emptyBranch?.includes("<SiteManagerEmptyState"), "Empty sites must use the dedicated onboarding component");
assert.ok(!emptyBranch.includes("No site selected"), "Empty state must not repeat an unselected-details panel");
assert.match(ui, /ghost-site-manager-workspace--populated/);
assert.doesNotMatch(ui, /xl:grid-cols-\[/);
assert.match(onboarding, /ghost-site-manager-workspace--empty/);
for (const label of ["Add your first site", "New Site", "Connect Cloud", "Import Sites"]) {
  assert.ok(onboarding.includes(label), "Missing onboarding label: " + label);
}
for (const callback of ["onNewSite", "onConnectCloud", "onImportSites"]) {
  assert.ok(onboarding.includes(`onClick={${callback}}`), "Missing wired action: " + callback);
}
const legacy = read("src/styles.css");
for (const obsolete of [
  "grid-template-columns: 242px minmax(0,1fr) 348px",
  "grid-template-columns: 190px minmax(430px,1fr) 300px",
  "grid-template-columns: 132px minmax(390px,1fr) 276px",
  "grid-template-columns: 196px minmax(420px, 1fr) 304px",
]) {
  assert.ok(!legacy.includes(obsolete), "Retired three-column Sites rule returned: " + obsolete);
}
assert.match(entry, /import "\.\/styles\.css";\s*import "\.\/styles\/site-manager-layout\.css";/);
assert.match(css, /container:\s*sites-workspace\s*\/\s*inline-size/);
assert.match(css, /ghost-site-manager-workspace--empty\s*\{/);
assert.match(css, /ghost-site-manager-workspace--populated\s*\{/);
assert.match(css, /grid-template-columns:\s*minmax\(320px,\s*42%\) minmax\(0,\s*1fr\)\s*!important/);
assert.match(css, /@container sites-workspace \(max-width: 940px\)/);
assert.match(css, /@container sites-workspace \(max-width: 590px\)/);
assert.match(css, /\.ghost-sites-empty-actions button:focus-visible/);
assert.doesNotMatch(css, /grid-template-columns:[^;]*242px[^;]*348px/);
console.log("Sites responsive panel, empty-state actions and keyboard contracts passed.");
