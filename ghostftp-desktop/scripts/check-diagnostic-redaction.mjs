import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import ts from "typescript";

// Exercise the shipped TS implementation, rather than a second test-only
// redactor. TypeScript is already a pinned development dependency.
const source = readFileSync(new URL("../src/lib/redact.ts", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
});
const exported = {};
runInNewContext(compiled.outputText, { exports: exported }, { timeout: 2000 });
const redact = exported.redactSensitiveText;

assert.equal(typeof redact, "function");
assert.equal(redact("password=TopSecret42"), "password=<redacted>");
assert.equal(
  redact("https://ftp.example.invalid/download?token=VeryPrivate&file=sample"),
  "https://ftp.example.invalid/download?token=<redacted>&file=sample",
);
assert.equal(redact("Authorization: Bearer aBcDe123"), "Authorization: <redacted>");
assert.equal(redact('{"password":"private-value"}'), '{"password":"<redacted>"}');
assert.equal(redact("token=abc" + "x".repeat(65_526)).startsWith("token=<redacted>"), true);
assert.equal(redact("x".repeat(65_536)).length, 65_536);
assert.equal(redact("x".repeat(65_537)), "[diagnostic omitted: oversized response]");
assert.equal(redact("/Users/alice/Documents/secret.txt"), "/Users/<user>/Documents/secret.txt");
assert.equal(redact("/home/bob/notes.txt"), "/home/<user>/notes.txt");
assert.equal(redact("C:\\Users\\alice\\notes.txt"), "C:\\Users\\<user>\\notes.txt");
const cutoff = redact("password=topsecret-with-a-long-value", 12);
assert.equal(cutoff, "password=<re");
assert.ok(!cutoff.includes("topsecret"), "must redact before display truncation");
const key = redact("-----BEGIN OPENSSH PRIVATE KEY-----\nsecret-key-material\n(no end marker)");
assert.ok(!key.includes("secret-key-material"), "unterminated private key must fail closed");
assert.ok(key.includes("<redacted private key>"));
const tooLarge = redact("token=private-" + "x".repeat(70_000), 600);
assert.equal(tooLarge, "[diagnostic omitted: oversized response]");
assert.ok(!tooLarge.includes("private"), "oversized server messages cannot leak content");

console.log("Diagnostic redaction security regressions passed.");
