import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import ts from "typescript";

// Exercise the production selector, not a duplicated heuristic.
const source = readFileSync(new URL("../src/lib/transferFailureNavigation.ts", import.meta.url), "utf8");
const { outputText, diagnostics } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.equal(diagnostics?.length ?? 0, 0);
const { mostRecentFailedTransferId } = await import(
  "data:text/javascript;base64," + Buffer.from(outputText).toString("base64")
);
const t = (id, status, startedAt) => Object.freeze({ id, status, startedAt });
assert.equal(mostRecentFailedTransferId([]), null);
assert.equal(mostRecentFailedTransferId([t("canceled", "canceled", 300)]), null);
assert.equal(mostRecentFailedTransferId([t("ok", "done", 100), t("queued", "queued", 200)]), null);
assert.equal(mostRecentFailedTransferId([t("older", "error", 10), t("new", "error", 40), t("running", "transferring", 90)]), "new");
assert.equal(mostRecentFailedTransferId([t("first", "error", 10), t("second", "error", 10)]), "first");
assert.equal(mostRecentFailedTransferId([t("broken", "error", NaN), t("valid", "error", 0)]), "valid");
assert.equal(mostRecentFailedTransferId([t("broken", "error", Infinity), t("broken2", "error", NaN)]), "broken");
const frozen = Object.freeze([t("first", "error", 11), t("second", "error", 12)]);
assert.equal(mostRecentFailedTransferId(frozen), "second");
assert.equal(frozen[0].startedAt, 11, "no native record mutation");

// Static UI contract checks only: real keyboard/5–8 user studies remain required.
const ui = readFileSync(new URL("../src/components/TransferCenterDialog.tsx", import.meta.url), "utf8");
for (const required of [
  "onClick={reviewLatestFailure}",
  'filter === "failed" ? reviewLatestFailure() : setTab(filter)',
  "focusedFailureRowRef.current?.scrollIntoView",
  'setTab("failed");',
  'setDirectionFilter("all");',
  'setTimeFilter("all");',
  'setQuery("");',
  "messageOf(transfer.error)",
  "Retry this transfer",
  'role="group"',
  "sticky top-0 z-10",
]) {
  assert.ok(ui.includes(required), `Transfer Center failure review contract missing: ${required}`);
}
console.log("UI-05 failure review: newest error, invalid time, focus path, sanitized diagnostics and retry contract OK.");
