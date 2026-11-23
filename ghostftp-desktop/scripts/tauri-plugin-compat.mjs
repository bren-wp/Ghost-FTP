// Check Tauri native/JavaScript plugin compatibility before native packaging.
// Both sides must share a major/minor line; a successful web build is insufficient.
export const PLUGIN_PAIRS = Object.freeze([
  ["@tauri-apps/plugin-dialog", "tauri-plugin-dialog"],
  ["@tauri-apps/plugin-notification", "tauri-plugin-notification"],
  ["@tauri-apps/plugin-process", "tauri-plugin-process"],
  ["@tauri-apps/plugin-updater", "tauri-plugin-updater"],
]);

function versionParts(v, label) {
  const parts = String(v ?? "").match(/^(\d+)\.(\d+)\.(\d+)$/);
  if (!parts) throw new Error(`${label} has no valid locked semver: ${v}`);
  return [Number(parts[1]), Number(parts[2]), Number(parts[3])];
}

export function assertTauriPluginCompatibility(pkg, npmLock, cargoLock) {
  const result = [];
  for (const [jsName, rustName] of PLUGIN_PAIRS) {
    const spec = pkg.dependencies?.[jsName];
    // Restrict minor upgrades until the Rust lockfile is updated in lockstep.
    const range = /^~(\d+\.\d+\.\d+)$/.exec(spec ?? "");
    if (!range) {
      throw new Error(`${jsName} must use a patch-only ~MAJOR.MINOR.PATCH range; found ${spec}`);
    }
    if (npmLock.packages?.[""]?.dependencies?.[jsName] !== spec) {
      throw new Error(`${jsName} package-lock root differs from package.json`);
    }
    const locked = npmLock.packages?.[`node_modules/${jsName}`]?.version;
    const jsParts = versionParts(locked, jsName);
    const minParts = versionParts(range[1], jsName + " range");
    if (jsParts[0] !== minParts[0] || jsParts[1] !== minParts[1] || jsParts[2] < minParts[2]) {
      throw new Error(`${jsName} locked version ${locked} violates patch-only range ${spec}`);
    }
    const escaped = rustName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const match = cargoLock.match(new RegExp('\\[\\[package\\]\\]\\r?\\nname = "' + escaped + '"\\r?\\nversion = "(\\d+\\.\\d+\\.\\d+)"'));
    if (!match) throw new Error(`Missing locked Rust crate ${rustName}`);
    const rustParts = versionParts(match[1], rustName);
    if (jsParts[0] !== rustParts[0] || jsParts[1] !== rustParts[1]) {
      throw new Error(`Tauri plugin mismatch: ${rustName} ${match[1]} vs ${jsName} ${locked}`);
    }
    result.push(`${rustName}: Rust ${match[1]} / JS ${locked}`);
  }
  return result;
}
