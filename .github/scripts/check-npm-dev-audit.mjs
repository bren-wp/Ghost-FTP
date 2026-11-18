import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { evaluateNpmDevAudit } from "./check-npm-dev-audit-policy.mjs";

// Always call npm audit: advisory exceptions must never silence a failed query.
let raw = "";
try {
  raw = execFileSync("npm", ["audit", "--json"], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
} catch (error) {
  raw = error?.stdout?.toString?.() ?? "";
  if (!raw.trim()) throw error;
}
const audit = JSON.parse(raw);
const lock = JSON.parse(readFileSync(
  new URL("../../ghostftp-desktop/package-lock.json", import.meta.url), "utf8"
));

// Always verify all three locally patched parser copies, even when the
// vulnerability database returns no advisories for a withdrawn GHSA.
execFileSync(process.execPath, [
  fileURLToPath(new URL("../../ghostftp-desktop/scripts/patch-picomatch-redos.mjs", import.meta.url)),
  "--verify",
], { stdio: "inherit", timeout: 15000 });

const result = evaluateNpmDevAudit(audit, lock);
if (result.affected.length === 0) {
  console.log("npm dev audit policy OK: no high/critical development vulnerabilities.");
} else {
  console.warn("Locally patched picomatch exception (GHSA-fc5q-39xc-vjpf): " +
    result.affected.join(", "));
  console.warn("The original npm tarball remains unmodified; runtime dependencies are audited separately.");
}
