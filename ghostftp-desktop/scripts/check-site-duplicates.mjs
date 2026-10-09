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
assert.match(ui, /view === "duplicates" && !duplicateIds\.has\(profile\.id\)/);
assert.match(ui, /label="Duplicates" count=\{duplicateIds\.size\}/);
assert.match(ui, /matchingSites\.length > 0/, "visible advisory when a connection matches");
console.log("Ghost FTP saved-site duplicate discovery and UI wiring regressions passed.");
