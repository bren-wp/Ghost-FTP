import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync, lstatSync, realpathSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const shaPattern = /^[0-9a-f]{40}$/;
const hex256 = /^[0-9a-f]{64}$/;
const statuses = new Set(["unverified", "pass", "fail"]);

export function validateRuntimeEvidence(reference, evidence, root = projectRoot) {
  assert.equal(reference.type, "concept-reference-only", "references must remain design concepts");
  assert.equal(reference.total, 75, "expected the reviewed 75-screen contract");
  assert.equal(evidence.schema_version, 1, "unknown runtime-evidence schema");
  assert.equal(evidence.reference, "docs/qa/PREMIUM_REFERENCE_75.json");
  assert.ok(Array.isArray(evidence.screens), "missing per-screen evidence");
  const expected = Object.entries(reference.screens).flatMap(([platform, names]) =>
    names.map((name) => ({ id: platform + "/" + name, platform, viewport: reference.screen_size[platform] }))
  );
  assert.equal(expected.length, 75, "reference screen count mismatch");
  assert.equal(evidence.screens.length, expected.length, "every screen needs one evidence entry");
  let verified = 0;
  let failed = 0;
  for (let i = 0; i < expected.length; i++) {
    const spec = expected[i];
    const entry = evidence.screens[i];
    assert.equal(entry.id, spec.id, "missing, reordered or duplicate screen " + spec.id);
    assert.ok(statuses.has(entry.status), "invalid status for " + spec.id);
    if (entry.status === "unverified") {
      assert.equal(entry.capture, null, "unverified screen cannot claim a capture: " + spec.id);
      assert.equal(entry.review, null, "unverified screen cannot claim approval: " + spec.id);
      continue;
    }
    assert.ok(entry.capture && typeof entry.capture === "object", "missing actual screenshot proof for " + spec.id);
    const capture = entry.capture;
    assert.ok(shaPattern.test(capture.source_sha ?? ""), "missing exact built-source SHA for " + spec.id);
    assert.ok(hex256.test(capture.sha256 ?? ""), "missing screenshot SHA-256 for " + spec.id);
    assert.deepEqual(capture.viewport, spec.viewport, "wrong reference viewport for " + spec.id);
    assert.ok(
      typeof capture.path === "string" &&
      capture.path.startsWith("docs/qa/runtime-captures/" + spec.platform + "/") &&
      capture.path.endsWith(".png") &&
      !capture.path.includes("..") && !capture.path.includes("\\"),
      "untrusted screenshot location for " + spec.id
    );
    const location = path.join(root, capture.path);
    assert.ok(lstatSync(location).isFile() && !lstatSync(location).isSymbolicLink(), "screenshot must be a regular PNG");
    const resolved = realpathSync(location);
    assert.ok(resolved.startsWith(realpathSync(path.join(root, "docs/qa/runtime-captures")) + path.sep), "screenshot leaves capture root");
    const bytes = readFileSync(location);
    assert.equal(createHash("sha256").update(bytes).digest("hex"), capture.sha256, "screenshot hash drift for " + spec.id);
    assert.ok(bytes.length >= 24 && bytes.subarray(0, 8).equals(Buffer.from("89504e470d0a1a0a", "hex")), "not a PNG");
    assert.equal(bytes.subarray(12, 16).toString("ascii"), "IHDR", "invalid PNG header");
    assert.deepEqual([bytes.readUInt32BE(16), bytes.readUInt32BE(20)], spec.viewport, "PNG dimensions differ from viewport");
    assert.ok(entry.review && typeof entry.review === "object", "missing human+interaction review for " + spec.id);
    assert.ok(typeof entry.review.reviewer === "string" && entry.review.reviewer.trim().length > 1);
    assert.ok(!Number.isNaN(Date.parse(entry.review.reviewed_at ?? "")), "missing review timestamp");
    assert.equal(entry.review.keyboard_checked, true, "keyboard QA not recorded for " + spec.id);
    assert.equal(entry.review.accessibility_checked, true, "accessibility QA not recorded for " + spec.id);
    assert.equal(entry.review.interactions_checked, true, "interaction QA not recorded for " + spec.id);
    if (entry.status === "pass") {
      assert.equal(entry.review.approved, true, "unapproved screen cannot be PASS");
      verified++;
    } else {
      assert.ok(typeof entry.review.discrepancy === "string" && entry.review.discrepancy.trim(), "FAIL must state actual discrepancy");
      failed++;
    }
  }
  return { total: expected.length, pass: verified, fail: failed, unverified: expected.length - verified - failed };
}

function loadAndValidate() {
  const reference = JSON.parse(readFileSync(path.join(projectRoot, "docs/qa/PREMIUM_REFERENCE_75.json"), "utf8"));
  const evidence = JSON.parse(readFileSync(path.join(projectRoot, "docs/qa/PREMIUM_REFERENCE_EVIDENCE.json"), "utf8"));
  return validateRuntimeEvidence(reference, evidence);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const result = loadAndValidate();
    console.log("75-screen runtime evidence: " + JSON.stringify(result) + ". Unverified is NOT a UI PASS.");
  } catch (error) {
    console.error("Runtime visual evidence contract FAILED: " + error.message);
    process.exitCode = 1;
  }
}
