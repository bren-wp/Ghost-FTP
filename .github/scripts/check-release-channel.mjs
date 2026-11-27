import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

// Release classification is independent of signing. Historical tags keep their original policy.
export function classifyReleaseChannel(channel) {
  if (channel === "preview") {
    return { prerelease: true, titleSuffix: " Preview" };
  }
  if (channel === "stable") {
    return { prerelease: false, titleSuffix: "" };
  }
  throw new Error(
    `Unsupported release channel ${JSON.stringify(channel)}; expected preview or stable.`
  );
}

export function requiresStable(version) {
  if (!/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(version)) throw new Error("Invalid release version");
  const [major, minor, patch] = version.split(".").map(Number);
  return major > 0 || minor > 99 || (minor === 99 && patch >= 90);
}

export function checkReleaseChannel(file = path.join(root, "version.json")) {
  const config = JSON.parse(fs.readFileSync(file, "utf8"));
  const result = classifyReleaseChannel(config.channel);
  if (!/^\d+\.\d+\.\d+$/.test(config.version)) {
    throw new Error("Release channel version must be canonical SemVer.");
  }
  if (requiresStable(config.version) && result.prerelease) throw new Error("New releases must use the stable channel");
  return { ...result, version: config.version };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const status = checkReleaseChannel();
    console.log(`Release distribution channel verified: v${status.version} is ${status.prerelease ? 'Preview/prerelease' : 'stable (signing independent)'}.`);
  } catch (error) {
    console.error(`Release channel security gate failed: ${error.message}`);
    process.exitCode = 1;
  }
}
