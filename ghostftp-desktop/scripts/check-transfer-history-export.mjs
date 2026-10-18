import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import ts from "typescript";

// Test actual production implementation; no second copy of export logic.
const source = readFileSync(new URL("../src/lib/transferHistoryExport.ts", import.meta.url), "utf8");
const { outputText, diagnostics } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.equal(diagnostics?.length ?? 0, 0);
const { runTransferHistoryExport } = await import(
  "data:text/javascript;base64," + Buffer.from(outputText).toString("base64")
);

const outcomes = [];
const save = (count) => outcomes.push(["saved", count]);
const fail = (error) => outcomes.push(["failed", String(error?.message ?? error)]);
const neverExport = async () => { throw new Error("export must not start"); };

assert.equal(
  await runTransferHistoryExport(async () => null, neverExport, save, fail),
  "cancelled", "dialog cancel should be a no-op"
);
assert.equal(outcomes.length, 0, "cancel must not produce a toast");
assert.equal(
  await runTransferHistoryExport(async () => { throw new Error("OS save dialog unavailable"); },
    neverExport, save, fail),
  "failed", "picker rejection must be caught instead of unhandled"
);
assert.deepEqual(outcomes.at(-1), ["failed", "OS save dialog unavailable"]);

let invokedPath = null;
assert.equal(
  await runTransferHistoryExport(async () => "/tmp/ghostftp-history.csv",
    async (path) => { invokedPath = path; return 2; }, save, fail),
  "saved"
);
assert.equal(invokedPath, "/tmp/ghostftp-history.csv");
assert.deepEqual(outcomes.at(-1), ["saved", 2]);

assert.equal(
  await runTransferHistoryExport(async () => "/tmp/out.csv",
    async () => { throw new Error("disk full"); }, save, fail),
  "failed", "filesystem/IPC rejection must also be caught"
);
assert.deepEqual(outcomes.at(-1), ["failed", "disk full"]);

const ui = readFileSync(new URL("../src/components/TransferCenterDialog.tsx", import.meta.url), "utf8");
assert.ok(ui.includes("await runTransferHistoryExport("));
assert.ok(ui.includes("(error) => toastError(error, \"Couldn't export transfer history\")"));
console.log("Transfer export rejects safely on picker errors and native filesystem failures.");
