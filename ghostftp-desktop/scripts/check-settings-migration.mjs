import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import ts from "typescript";

// Compile and execute the application's real migration implementation, while
// substituting only IPC, storage and hydration dependencies.
const source = readFileSync(new URL("../src/lib/secretMigration.ts", import.meta.url), "utf8");
const output = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
}).outputText;

function harness({ writeMode = "exact", existing = {}, storageEntries } = {}) {
  const storage = new Map(Object.entries(storageEntries ?? {
    "ghostftp.settings.v1": JSON.stringify({ theme: "nord", showHiddenFiles: true }),
  }));
  let db = { ...existing };
  let hydrations = 0;
  let writes = 0;
  const ipc = {
    settingsGetAll: async () => ({ ...db }),
    settingsSetAll: async (entries) => {
      writes++;
      if (writeMode === "exact") db = { ...db, ...entries };
      if (writeMode === "wrong") db = { ...db, ...entries, theme: JSON.stringify("dark") };
      if (writeMode === "partial") db = { ...db, showHiddenFiles: entries.showHiddenFiles, unrelated: "extra" };
      if (writeMode === "throw") throw new Error("simulated write failure");
    },
    settingsSet: async () => {},
  };
  const localStorage = {
    getItem: (key) => storage.get(key) ?? null,
    removeItem: (key) => { storage.delete(key); },
    key: (index) => [...storage.keys()][index] ?? null,
    get length() { return storage.size; },
  };
  const module = { exports: {} };
  const context = {
    module,
    exports: module.exports,
    localStorage,
    console: { warn() {} },
    require: (name) => {
      if (name === "@/lib/ipc") return { ipc };
      if (name === "@/stores/settingsStore") return {
        SETTINGS_KEYS: ["theme", "showHiddenFiles"],
        hydrateFromDb: async () => { hydrations++; },
      };
      if (name === "@/lib/errors") return { messageOf: () => "redacted error" };
      throw new Error("Unmocked dependency: " + name);
    },
  };
  runInNewContext(output, context, { timeout: 2000 });
  return {
    migrate: () => module.exports.runSettingsMigration(),
    hasBlob: () => storage.has("ghostftp.settings.v1"),
    has: (key) => storage.has(key),
    reads: () => ({ ...db }),
    writes: () => writes,
    hydrations: () => hydrations,
  };
}

{
  const h = harness();
  await h.migrate();
  assert.equal(h.hasBlob(), false, "remove legacy settings only after exact DB readback");
  assert.equal(h.reads().theme, JSON.stringify("nord"));
  assert.equal(h.reads().showHiddenFiles, "true");
  assert.equal(h.hydrations(), 1);
}
for (const writeMode of ["wrong", "partial"]) {
  const h = harness({ writeMode });
  await h.migrate();
  assert.equal(h.hasBlob(), true, `retain backup for ${writeMode} readback`);
  assert.equal(h.hydrations(), 0, "never hydrate partially migrated values");
}
{
  const h = harness({ writeMode: "throw" });
  await assert.rejects(h.migrate(), /simulated write failure/);
  assert.equal(h.hasBlob(), true, "retain backup when IPC fails");
}
{
  const h = harness({ existing: { theme: JSON.stringify("light") } });
  await h.migrate();
  assert.equal(h.writes(), 0, "existing DB settings override legacy storage");
  assert.equal(h.hasBlob(), true, "do not erase old data without migration verification");
}
{
  const h = harness({
    writeMode: "partial",
    storageEntries: {
      "ghostftp.settings.v1": JSON.stringify({ theme: "nord", showHiddenFiles: true }),
      "ghostftp.notifications.v1": "sensitive",
      "ghostftp.term-history.v1:example": "secret",
    },
  });
  await h.migrate();
  assert.equal(h.has("ghostftp.notifications.v1"), false);
  assert.equal(h.has("ghostftp.term-history.v1:example"), false);
  assert.equal(h.hasBlob(), true, "do not confuse privacy purge with settings migration");
}

for (const legacyValue of ["{corrupted", "null", '"unexpected string"', "[1,2,3]"]) {
  const h = harness({ storageEntries: { "ghostftp.settings.v1": legacyValue } });
  await assert.doesNotReject(h.migrate(), "bad legacy JSON must not abort application startup");
  assert.equal(h.hasBlob(), true, "preserve damaged source for later recovery");
  assert.equal(h.writes(), 0, "do not write malformed settings into the database");
  assert.equal(h.hydrations(), 0);
}

console.log("Ghost FTP settings migration persistence and privacy regressions passed.");
