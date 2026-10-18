import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { validateRuntimeEvidence } from "./check-runtime-75-evidence.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const reference = JSON.parse(readFileSync(path.join(root, "docs/qa/PREMIUM_REFERENCE_75.json"), "utf8"));
const baseline = JSON.parse(readFileSync(path.join(root, "docs/qa/PREMIUM_REFERENCE_EVIDENCE.json"), "utf8"));
const copy = () => structuredClone(baseline);

test("every reference is present and no concept art is reported as accepted runtime UI", () => {
  const result = validateRuntimeEvidence(reference, baseline, root);
  assert.deepEqual(result, { total: 75, pass: 0, fail: 0, unverified: 75 });
});

test("missing, duplicate, reordered or falsely counted references are rejected", () => {
  const missing = copy();
  missing.screens.pop();
  assert.throws(() => validateRuntimeEvidence(reference, missing, root), /every screen/);
  const reversed = copy();
  [reversed.screens[0], reversed.screens[1]] = [reversed.screens[1], reversed.screens[0]];
  assert.throws(() => validateRuntimeEvidence(reference, reversed, root), /missing, reordered or duplicate/);
});

test("an unverified entry cannot carry screenshot or approval claims", () => {
  const a = copy();
  a.screens[0].capture = { path: "docs/assets/screenshots/ghostftp-main-file-manager.webp" };
  assert.throws(() => validateRuntimeEvidence(reference, a, root), /unverified screen cannot claim a capture/);
  const b = copy();
  b.screens[0].review = { approved: true };
  assert.throws(() => validateRuntimeEvidence(reference, b, root), /unverified screen cannot claim approval/);
});

test("a PASS without source SHA and real immutable runtime PNG evidence is blocked", () => {
  const a = copy();
  a.screens[0].status = "pass";
  assert.throws(() => validateRuntimeEvidence(reference, a, root), /missing actual screenshot proof/);
  a.screens[0].capture = { path: "docs/qa/runtime-captures/windows/files.png" };
  assert.throws(() => validateRuntimeEvidence(reference, a, root), /missing exact built-source SHA/);
  a.screens[0].capture.source_sha = "a".repeat(40);
  a.screens[0].capture.sha256 = "b".repeat(64);
  a.screens[0].capture.viewport = [1290, 852];
  assert.throws(() => validateRuntimeEvidence(reference, a, root), /ENOENT/);
});

test("unsafe screenshot paths, mismatched viewports and unsupported states fail closed", () => {
  const a = copy();
  a.screens[0].status = "accepted";
  assert.throws(() => validateRuntimeEvidence(reference, a, root), /invalid status/);
  a.screens[0].status = "fail";
  a.screens[0].capture = {
    source_sha: "a".repeat(40),
    sha256: "b".repeat(64),
    viewport: [1290, 852],
    path: "docs/assets/screenshots/ghostftp-main-file-manager.webp",
  };
  assert.throws(() => validateRuntimeEvidence(reference, a, root), /untrusted screenshot location/);
  a.screens[0].capture.path = "docs/qa/runtime-captures/windows/../other.png";
  assert.throws(() => validateRuntimeEvidence(reference, a, root), /untrusted screenshot location/);
  a.screens[0].capture.path = "docs/qa/runtime-captures/windows/files.png";
  a.screens[0].capture.viewport = [800, 600];
  assert.throws(() => validateRuntimeEvidence(reference, a, root), /wrong reference viewport/);
});
