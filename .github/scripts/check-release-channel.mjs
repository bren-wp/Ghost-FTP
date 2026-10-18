import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

/**
 * Current published production payload deliberately contains an unsigned
 * Android APK/AAB and an ad-hoc signed macOS Preview. Stable distribution
 * MUST remain disabled until OS signers, installer upgrade proofs, Play
 * acceptance and notarization are independently enforced in release CI.
 */
export function classifyReleaseChannel(channel) {
  if (channel === "preview") {
    return { prerelease: true, titleSuffix: " Preview" };
  }
  if (channel === "stable") {
    throw new Error(
      "Stable release refused: Windows Authenticode, Android Play signing, " +
        "macOS Developer ID notarization and verified installer upgrades " +
        "are not mandatory release gates. Publish Preview only."
    );
  }
  throw new Error(
    `Unsupported release channel ${JSON.stringify(channel)}; expected preview.`
  );
}

export function checkReleaseChannel(file = path.join(root, "version.json")) {
  const config = JSON.parse(fs.readFileSync(file, "utf8"));
  const result = classifyReleaseChannel(config.channel);
  if (!/^\d+\.\d+\.\d+$/.test(config.version)) {
    throw new Error("Release channel version must be canonical SemVer.");
  }
  return { ...result, version: config.version };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const status = checkReleaseChannel();
    console.log(`Release distribution channel verified: v${status.version} is Preview/prerelease, not stable.`);
  } catch (error) {
    console.error(`Release channel security gate failed: ${error.message}`);
    process.exitCode = 1;
  }
}
