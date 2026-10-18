import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import ts from "typescript";

const source = readFileSync(new URL("../src/lib/codeSearch.ts", import.meta.url), "utf8");
const { outputText, diagnostics } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.equal(diagnostics?.length ?? 0, 0);
const { replaceNextLiteral, replaceAllLiteral } = await import(
  "data:text/javascript;base64," + Buffer.from(outputText).toString("base64")
);

assert.equal(replaceAllLiteral("ab", "", "z").content, "ab", "empty query is a no-op");
assert.equal(replaceAllLiteral("Foo foo FOo", "foo", "$&").content, "$& $& $&",
  "replacement is literal, not JS replacement template syntax");
assert.equal(replaceAllLiteral("a+b a+b a?b", "a+b", "X").content, "X X a?b",
  "query is escaped, never treated as a regex");
assert.equal(replaceAllLiteral("<tag>(one)</tag>", "(one)", "a\\b").content, "<tag>a\\b</tag>");
assert.equal(replaceAllLiteral("😀 😀", "😀", "OK").count, 2);
assert.equal(replaceAllLiteral("aaaa", "aa", "x").content, "xx", "non-overlapping replacement");

const current = replaceNextLiteral("test TEST test", "test", "X", 5, 9);
assert.equal(current.content, "test X test", "matching selection is replaced exactly once");
assert.equal(current.cursor, 6);
const next = replaceNextLiteral("test TEST test", "test", "X", 4, 4);
assert.equal(next.content, "test X test", "collapsed cursor finds next match");
const wrap = replaceNextLiteral("test TEST test", "test", "X", 14, 14);
assert.equal(wrap.content, "X TEST test", "search wraps after end");
assert.equal(replaceNextLiteral("nothing", "xyz", "x", 0, 0).count, 0);

const original = "source with variable name, no changes";
const replaced = replaceAllLiteral(original, "VARIABLE", "$1");
assert.equal(original, "source with variable name, no changes", "input is immutable");
assert.equal(replaced.content, "source with $1 name, no changes");

const ui = readFileSync(new URL("../src/components/BuiltinCodeEditor.tsx", import.meta.url), "utf8");
for(const contract of ["replaceNextLiteral(", "replaceAllLiteral(", "Replace next match", "Replace all matches", "setContent(result.content)"]) {
  assert.ok(ui.includes(contract), `Editor replace UI contract missing: ${contract}`);
}
console.log("EDIT-03 literal replace: case, regex metacharacters, Unicode, cursor, wrapping and safe replacement OK.");
