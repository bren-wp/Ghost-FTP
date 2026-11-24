import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import ts from "typescript";

// Execute the production code, not a duplicated algorithm in the test.
const source = readFileSync(new URL("../src/lib/siteDuplicates.ts", import.meta.url), "utf8");
const code = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
}).outputText;
const module = { exports: {} };
runInNewContext(code, { module, exports: module.exports, Map, Set, JSON, Number }, { timeout: 2000 });
const { savedSiteIdentity, matchingSavedSites, duplicateSavedSiteIds } = module.exports;

const site = (id, overrides = {}) => ({
  id, name: id, protocol: "sftp", host: "ftp.example.org", port: 22,
  username: "Owner", auth: { kind: "password", password: "never compare or log me" },
  ...overrides,
});
const sites = [
  site("one"),
  site("two", { host: " FTP.Example.ORG. ", auth: { kind: "agent" }, name: "Second alias" }),
  site("three", { protocol: "ftp" }),
  site("four", { port: 2222 }),
  site("five", { username: "owner" }), // account name case may be significant
  site("six", { host: "" }), // incomplete saved item is never labeled duplicate
  site("seven", { username: "" }),
  site("eight", { host: "other.example.org" }),
];
assert.equal(savedSiteIdentity(sites[0]), savedSiteIdentity(sites[1]), "ignore DNS case / trailing dot");
assert.equal(matchingSavedSites(sites[0], sites).map((s) => s.id).join(","), "two");
assert.equal(matchingSavedSites(sites[1], sites).map((s) => s.id).join(","), "one");
assert.equal([...duplicateSavedSiteIds(sites)].sort().join(","), "one,two");
assert.equal(duplicateSavedSiteIds([site("one")]).size, 0, "same profile must not match itself");
assert.equal(savedSiteIdentity(site("bad", { port: NaN })), null);
assert.equal(savedSiteIdentity(site("bad", { port: 70000 })), null);
assert.equal(savedSiteIdentity(site("bad", { username: " " })), null);
assert.equal(matchingSavedSites(site("unsaved", { host: " " }), sites).length, 0);
const many = [site("x"), site("y"), site("z")];
assert.equal([...duplicateSavedSiteIds(many)].sort().join(","), "x,y,z", "every member must be surfaced");
assert.equal(matchingSavedSites({ ...many[0], host: "different.org" }, many).length, 0,
  "editing an existing connection must update the duplicate preview without persisting data");
assert.equal(
  savedSiteIdentity(site("a", { host: "a|b", username: "c" })) ===
  savedSiteIdentity(site("b", { host: "a", username: "b|c" })),
  false, "delimiter-bearing identities must not collide"
);

const ui = readFileSync(new URL("../src/components/SiteManagerDialog.tsx", import.meta.url), "utf8");
const viewSource = readFileSync(new URL("../src/lib/siteViewModel.ts", import.meta.url), "utf8");
assert.match(viewSource, /view === "duplicates" && !duplicateIds\.has\(profile\.id\)/);
assert.match(ui, /label="Duplicates" count=\{counts\.duplicates\}/);
assert.match(ui, /matchingSites\.length > 0/, "visible advisory when a connection matches");
assert.match(viewSource, /\$\{profile\.username\}/, "Sites search must include the server account name");
console.log("Ghost FTP saved-site duplicate discovery and UI wiring regressions passed.");

const { outputText } = ts.transpileModule(viewSource, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
});
const { filterAndSortSites, countSiteViews, validLastUsed } = await import(
  "data:text/javascript;base64," + Buffer.from(outputText).toString("base64")
);
const inventory = [
  site("ten", { name: "Server 10", lastUsed: NaN }),
  site("two", { name: "Server 2", lastUsed: 200, favorite: true }),
  site("cloud", { protocol: "s3", lastUsed: Infinity }),
  site("old", { lastUsed: 100, bookmarked: true }),
];
Object.freeze(inventory);
const duplicates = new Set(["ten", "two"]);
const connected = new Set(["two", "ephemeral"]);
const select = (query, view, field = "name", direction = "asc") =>
  filterAndSortSites(inventory, query, view, field, direction, duplicates, connected);
assert.deepEqual(countSiteViews(inventory, duplicates, connected),
  { all: 4, favorites: 1, recent: 2, bookmarks: 1, duplicates: 2, cloud: 1, connected: 1 });
assert.deepEqual(select("  SERVER  ", "all").map(s => s.id), ["two", "ten"]);
assert.deepEqual(select("OWNER", "connected").map(s => s.id), ["two"]);
assert.deepEqual(select("never compare or log me", "all"), []);
assert.deepEqual(select("", "recent", "lastUsed", "desc").map(s => s.id), ["two", "old"]);
assert.deepEqual(select("", "all", "lastUsed", "asc").map(s => s.id), ["old", "two", "ten", "cloud"]);
assert.deepEqual(select("", "all", "lastUsed", "desc").map(s => s.id), ["two", "old", "ten", "cloud"]);
assert.strictEqual(select("", "connected")[0], inventory[1]);
assert.equal(inventory[0].id, "ten");
for (const value of [undefined, 0, -1, NaN, Infinity, Number.MAX_VALUE]) assert.equal(validLastUsed(value), null);
assert.ok(ui.includes('label="Connected" count={counts.connected}'));
assert.ok(ui.includes('if (actionRef.current) return false;'));
assert.ok(ui.includes('revealedPasswordId === draft.id'));
assert.ok(ui.includes('Disconnect this site before deleting it.'));
assert.ok(ui.includes('aria-label="Clear site search"'));
console.log("Sites connected filter, stable sorting, credential privacy and action guard regressions passed.");

assert.ok(ui.includes("selected?.id === draft.id && revealedPasswordId === draft.id"), "password visibility must match both selected profile and draft");
assert.ok(ui.includes("selected && draft && selected.id === draft.id ? ("), "never render an old profile's fields beside a newly selected site's actions");
assert.ok(ui.includes('if (!draft || selected?.id !== draft.id || !editing || actionRef.current) return;'), "save must target the current editable draft");
assert.ok(ui.includes('<fieldset disabled={Boolean(action)}'), "profile fields must not change during an in-flight save");
