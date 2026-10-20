// Enforce the published Ghost FTP release train. This module has no IO so it
// can be regression-tested independently of release metadata / CI state.
const semver = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

function parts(value) {
  const match = semver.exec(value);
  if (!match) throw new Error(`Invalid semantic version: ${value}`);
  return match.slice(1).map(Number);
}

/**
 * 0.30.21 -> 0.90.0: one authorized migration to the new train.
 * 0.95.53 -> 0.99.0: explicitly authorized jump to the final Preview series.
 * 0.99.0 -> 0.99.1 -> ... -> 0.99.20 -> 1.0.0.
 * 1.0.0: production-stable only after separately verified release gates.
 * 1.x: next minor for features, next patch for fixes.
 */
export function validateVersionTrain(previous, next) {
  const [major, minor, patch] = parts(previous);
  const [nextMajor, nextMinor, nextPatch] = parts(next);
  const patchFix = nextMajor === major && nextMinor === minor && nextPatch === patch + 1;
  const minorFeature = nextMajor === major && nextMinor === minor + 1 && nextPatch === 0;

  let valid = false;
  if (previous === "0.30.21") {
    valid = next === "0.90.0";
  } else if (previous === "0.95.53") {
    valid = next === "0.99.0";
  } else if (major === 0 && minor >= 90 && minor < 99) {
    valid = patchFix || minorFeature;
  } else if (major === 0 && minor === 99) {
    valid = (patch < 20 && patchFix) ||
      (patch === 20 && nextMajor === 1 && nextMinor === 0 && nextPatch === 0);
  } else if (major >= 1) {
    valid = patchFix || minorFeature;
  }

  if (!valid) {
    throw new Error(
      `Invalid Ghost FTP release train ${previous} -> ${next}; ` +
      "use the explicitly authorized 0.95.53 -> 0.99.0 migration, " +
      "then patch sequentially through 0.99.20 before 1.0.0 and production acceptance.",
    );
  }
}
