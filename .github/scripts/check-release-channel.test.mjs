import assert from "node:assert/strict";
import test from "node:test";
import { classifyReleaseChannel, checkReleaseChannel } from "./check-release-channel.mjs";

test("development source must publish as an explicit GitHub prerelease", () => {
  assert.deepEqual(classifyReleaseChannel("preview"), {
    prerelease: true,
    titleSuffix: " Preview",
  });
  const state = checkReleaseChannel();
  assert.equal(state.prerelease, true);
  assert.match(state.version, /^\d+\.\d+\.\d+$/);
});

test("stable release is blocked while production signatures and lifecycle proofs are missing", () => {
  assert.throws(() => classifyReleaseChannel("stable"), /Stable release refused/);
});

test("missing, unknown and deceptive release channels cannot silently become stable", () => {
  for (const channel of [undefined, null, "", "production", "Stable", "preview ", "rc"]) {
    assert.throws(() => classifyReleaseChannel(channel), /Unsupported release channel/);
  }
});
