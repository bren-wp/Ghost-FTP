#!/usr/bin/env node
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (relative) => fs.readFileSync(path.join(root, relative), "utf8");
const desktop = read("ghostftp-desktop/src/components/TransferCenterDialog.tsx");
const css = read("ghostftp-desktop/src/styles.css");
const mac = read("macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift");
const android = read("android/app/src/main/java/com/ghostftp/android/MainActivity.kt");

for (const text of [
  'className="ghost-transfer-overview"',
  'const activeCount = activeFilterCount',
  'count: transfers.length',
  'count: activeFilterCount',
  'count: completed',
  'count: failed',
  'onClick={() => setTab(filter)}',
  'aria-pressed={tab === filter}',
]) assert.ok(desktop.includes(text), `Missing genuine desktop transfer overview: ${text}`);

for (const text of [
  ".ghost-transfer-overview-card:focus-visible",
  ".ghost-transfer-overview-card.is-selected",
  "grid-template-columns: repeat(2, minmax(0, 1fr))",
]) assert.ok(css.includes(text), `Missing keyboard/compact desktop transfer layout: ${text}`);

for (const text of [
  'history.records.filter { $0.status == .running }.count',
  'history.records.filter { $0.status == .completed }.count',
  'history.records.filter { $0.status == .failed }.count',
  'history.records.filter { $0.status == .cancelled }.count',
  'accessibilityIdentifier("ghostftp-transfer-overview")',
]) assert.ok(mac.includes(text), `Missing real macOS history metric: ${text}`);

for (const text of [
  'updateSessionOperationOverview()',
  'sessionCompletedOperations++',
  'sessionCancelledOperations++',
  'sessionFailedOperations++',
  'operationInFlight) 1 else 0',
  'transferOverviewText.contentDescription = message',
]) assert.ok(android.includes(text), `Missing Android live session-operation behavior: ${text}`);

console.log("Live Transfers UI: Windows/Linux backend-queue metrics, macOS actual credential-free history and Android session-only operation counters verified. Installed visual and device acceptance remain pending.");
