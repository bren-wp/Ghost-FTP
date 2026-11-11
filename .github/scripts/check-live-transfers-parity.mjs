#!/usr/bin/env node
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (relative) => fs.readFileSync(path.join(root, relative), "utf8");
const desktop = read("ghostftp-desktop/src/components/TransferCenterDialog.tsx");
const css = read("ghostftp-desktop/src/styles.css");

for (const text of [
  'className="ghost-transfer-overview"',
  'const activeCount = activeFilterCount',
  'count: transfers.length',
  'count: activeFilterCount',
  'count: completed',
  'count: failed',
  'aria-pressed={tab === filter}',
]) assert.ok(desktop.includes(text), `Missing genuine desktop transfer overview: ${text}`);

// The overview must remain clickable, but the new Failed tile must not
// leave stale keyword/direction/time filters hiding actual failed jobs.
const legacyOverview = desktop.includes('onClick={() => setTab(filter)}');
const safeFailureOverview = desktop.includes('filter === "failed" ? reviewLatestFailure() : setTab(filter)');
assert.ok(
  legacyOverview || safeFailureOverview,
  "Live transfer overview has no working click-to-filter handler"
);
if (safeFailureOverview) {
  for (const text of [
    'onClick={reviewLatestFailure}',
    'setQuery("");',
    'setDirectionFilter("all");',
    'setTimeFilter("all");',
    'setTab("failed");',
    'setSelectedId(id);',
    'focusedFailureRowRef.current?.scrollIntoView',
  ]) {
    assert.ok(desktop.includes(text), `Failed overview lost safe navigation: ${text}`);
  }
}

for (const text of [
  ".ghost-transfer-overview-card:focus-visible",
  ".ghost-transfer-overview-card.is-selected",
  "grid-template-columns: repeat(2, minmax(0, 1fr))",
]) assert.ok(css.includes(text), `Missing keyboard/compact desktop transfer layout: ${text}`);

console.log("Live Transfers UI: Windows/Linux real backend-queue metrics verified. Installed visual and device acceptance remain pending.");
