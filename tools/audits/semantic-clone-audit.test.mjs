import test from "node:test";
import assert from "node:assert/strict";
import { scanEntries, tokenize } from "./semantic-clone-audit.mjs";

const fixture = name => [
  "let " + name + " = readValue();",
  "if (" + name + " > 7) {",
  "  " + name + " = normalize(" + name + ");",
  "  return " + name + ";",
  "}",
  "return null;",
].join("\n");
const entry = (file, source, language = "ts") => ({ file, source, language });

test("renamed identifiers and comments do not hide structural clones", () => {
  const a = fixture("first");
  const b = "// unrelated comment\n" + fixture("second");
  const width = tokenize(a, "ts").length;
  const report = scanEntries([entry("a.ts", a), entry("b.ts", b)], { width, span: 1 });
  assert.equal(report.candidateGroups, 1);
  assert.deepEqual(report.candidates[0].locations.map(x => x.file), ["a.ts", "b.ts"]);
  assert.equal(report.candidates[0].locations[1].startLine, 2);
});

test("literals, calls and language boundaries are retained", () => {
  const a = fixture("x");
  const width = tokenize(a, "ts").length;
  for (const other of [entry("b.ts", a.replace("7", "8")), entry("b.ts", a.replace("readValue", "writeValue")), entry("b.rs", a, "rs")]) {
    assert.equal(scanEntries([entry("a.ts", a), other], { width, span: 1 }).candidateGroups, 0);
  }
});

test("same file nonoverlapping clones and prefixed copies are found", () => {
  const a = fixture("first");
  const b = fixture("second");
  const report = scanEntries([entry("a.ts", a + "\n\n" + b)], { width: tokenize(a, "ts").length, span: 1 });
  assert.ok(report.candidates.some(group => group.locations.length === 2));
  assert.equal(scanEntries([], {}).candidateGroups, 0);
});
