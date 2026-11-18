import assert from "node:assert/strict";
import test from "node:test";
import { evaluateNpmDevAudit } from "./check-npm-dev-audit-policy.mjs";

const root = {
  name: "picomatch",
  url: "https://github.com/advisories/GHSA-fc5q-39xc-vjpf",
  severity: "high",
};
function fixture() {
  return {
    audit: {
      auditReportVersion: 2,
      metadata: { vulnerabilities: { total: 2, high: 2 } },
      vulnerabilities: {
        picomatch: { severity: "high", via: [root], nodes: ["node_modules/picomatch"] },
        vite: { severity: "high", via: ["picomatch"], nodes: ["node_modules/vite"] },
      },
    },
    lock: { packages: {
      "node_modules/picomatch": { dev: true },
      "node_modules/vite": { dev: true },
    } },
  };
}

test("accept only the exact patched, dev-only advisory topology", () => {
  const { audit, lock } = fixture();
  assert.deepEqual(evaluateNpmDevAudit(audit, lock), {
    affected: ["picomatch", "vite"], roots: ["picomatch"],
  });
});

test("reject a withdrawn/unmitigated braces exception even when dev-only", () => {
  const { audit, lock } = fixture();
  audit.vulnerabilities.braces = {
    severity: "high",
    via: [{ name: "braces", severity: "high", url:
      "https://github.com/advisories/GHSA-vfj7-8cjw-p6xm" }],
    nodes: ["node_modules/braces"],
  };
  lock.packages["node_modules/braces"] = { dev: true };
  assert.throws(() => evaluateNpmDevAudit(audit, lock), /Unexpected high/);
});

test("reject unexpected security advisories and unexpected transitive roots", () => {
  const { audit, lock } = fixture();
  audit.vulnerabilities.picomatch.via = [{
    name: "picomatch", severity: "high", url: "https://example.invalid/advisory",
  }];
  assert.throws(() => evaluateNpmDevAudit(audit, lock), /Unexpected direct/);
  audit.vulnerabilities.picomatch.via = [root];
  audit.vulnerabilities.vite.via = ["unknown"];
  assert.throws(() => evaluateNpmDevAudit(audit, lock), /Unresolved npm/);
});

test("reject production dependencies and an untraceable vulnerability", () => {
  const { audit, lock } = fixture();
  lock.packages["node_modules/picomatch"].dev = false;
  assert.throws(() => evaluateNpmDevAudit(audit, lock), /not proven dev-only/);
  lock.packages["node_modules/picomatch"].dev = true;
  audit.vulnerabilities.picomatch.nodes = [];
  assert.throws(() => evaluateNpmDevAudit(audit, lock), /Missing npm lockfile paths/);
  audit.vulnerabilities.picomatch.nodes = ["node_modules/picomatch"];
  audit.vulnerabilities.vite.via = [];
  assert.throws(() => evaluateNpmDevAudit(audit, lock), /Missing npm advisory path/);
});

test("reject audit command failures masquerading as clean results", () => {
  const { audit, lock } = fixture();
  const malformed = [
    { error: { code: "ENOAUDIT", summary: "registry unavailable" } },
    { auditReportVersion: 2, vulnerabilities: {} },
    { auditReportVersion: 2, metadata: { vulnerabilities: { total: 0 } } },
  ];
  for (const report of malformed) {
    assert.throws(() => evaluateNpmDevAudit(report, lock), /incomplete report/);
  }
  audit.vulnerabilities = {};
  audit.metadata.vulnerabilities = { total: 0, high: 0 };
  assert.deepEqual(evaluateNpmDevAudit(audit, lock), { affected: [], roots: [] });
});
