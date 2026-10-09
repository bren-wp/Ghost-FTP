import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { randomFillSync } from "node:crypto";
import { runInNewContext } from "node:vm";
import ts from "typescript";

const source = readFileSync(new URL("../src/lib/password.ts", import.meta.url), "utf8");
const output = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
}).outputText;

const module = { exports: {} };
runInNewContext(output, {
  module,
  exports: module.exports,
  Uint32Array,
  crypto: {
    getRandomValues: (buf) => randomFillSync(buf),
  },
}, { timeout: 2000 });
const generatePassword = module.exports.generatePassword;

for (const length of [undefined, 0, 8, 20, 64, 128, 512]) {
  const password = generatePassword(length);
  assert.equal(password.length, Math.max(8, length ?? 20));
  assert.match(password, /[a-z]/);
  assert.match(password, /[A-Z]/);
  assert.match(password, /[2-9]/);
  assert.match(password, /[!@#%^*_-+=?.]/);
}
for (const invalid of [NaN, Infinity, -Infinity, 8.5, 513, Number.MAX_SAFE_INTEGER]) {
  assert.throws(() => generatePassword(invalid), /Password length/);
}
assert.notEqual(generatePassword(32), generatePassword(32), "passwords must be unpredictable");
console.log("Ghost FTP password generator bounded-length regressions passed.");
