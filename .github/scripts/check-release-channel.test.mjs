import assert from "node:assert/strict";
import test from "node:test";
import { classifyReleaseChannel, checkReleaseChannel, requiresStable } from "./check-release-channel.mjs";

test("historical Preview classification is explicit; current source is Stable", () => {
  assert.deepEqual(classifyReleaseChannel("preview"), {
    prerelease: true,
    titleSuffix: " Preview",
  });
  const state = checkReleaseChannel();
  assert.equal(state.prerelease, false);
  assert.match(state.version, /^\d+\.\d+\.\d+$/);
});

test("stable distribution does not require signatures; transition is versioned", () => {
  assert.deepEqual(classifyReleaseChannel("stable"), { prerelease: false, titleSuffix: "" });
  assert.equal(requiresStable("0.99.89"), false);
  assert.equal(requiresStable("0.99.90"), true);
  assert.equal(requiresStable("1.0.0"), true);
});

test("missing, unknown and deceptive release channels cannot silently become stable", () => {
  for (const channel of [undefined, null, "", "production", "Stable", "preview ", "rc"]) {
    assert.throws(() => classifyReleaseChannel(channel), /Unsupported release channel/);
  }
});
