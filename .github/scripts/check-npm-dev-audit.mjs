import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

// The picomatch exception applies ONLY to the locally patched, tested copy.
// The npm audit is still executed and every unrecognized root advisory fails.
const allowedRootUrls = new Map([
  ["braces", "https://github.com/advisories/GHSA-vfj7-8cjw-p6xm"],
  ["picomatch", "https://github.com/advisories/GHSA-fc5q-39xc-vjpf"],
]);
const allowedPackages = new Set([
  "@vitejs/plugin-react",
  "anymatch",
  "braces",
  "fdir",
  "picomatch",
  "readdirp",
  "sucrase",
  "tinyglobby",
  "vite",
  "chokidar",
  "fast-glob",
  "micromatch",
  "tailwindcss",
]);

let raw = "";
try {
  raw = execFileSync("npm", ["audit", "--json"], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
} catch (error) {
  raw = error?.stdout?.toString?.() ?? "";
  if (!raw.trim()) {
    throw error;
  }
}

const audit = JSON.parse(raw);
const vulnerabilities = audit.vulnerabilities ?? {};
const lock = JSON.parse(readFileSync(new URL("../../ghostftp-desktop/package-lock.json", import.meta.url), "utf8"));
const packages = lock.packages ?? {};

const severe = Object.entries(vulnerabilities).filter(([, value]) =>
  value?.severity === "high" || value?.severity === "critical"
);

// Fail-closed local source mitigation. npm reports published tarball versions,
// not the verified on-disk patch applied after their integrity-checked install.
// If the patch is missing, modified, or regresses, THIS audit must fail too.
execFileSync(process.execPath, [fileURLToPath(new URL("../../ghostftp-desktop/scripts/patch-picomatch-redos.mjs", import.meta.url)), "--verify"], {
  stdio: "inherit",
  timeout: 15000,
});

if (severe.length === 0) {
  console.log("npm dev audit policy OK: no high/critical development vulnerabilities.");
  process.exit(0);
}

const names = new Set(severe.map(([name]) => name));
const unexpectedNames = [...names].filter((name) => !allowedPackages.has(name));
if (unexpectedNames.length > 0) {
  throw new Error(
    `Unexpected high/critical npm vulnerability package(s): ${unexpectedNames.join(", ")}`
  );
}

let sawExpectedRoot = false;
for (const [name, finding] of severe) {
  for (const via of finding.via ?? []) {
    if (typeof via === "string") {
      const transitive = vulnerabilities[via];
      const transitiveIsSevere =
        transitive?.severity === "high" || transitive?.severity === "critical";

      // npm propagates lower-severity dependency findings into an aggregate
      // high-severity parent. This policy blocks high/critical development
      // advisories; a moderate/low intermediary must not be reclassified as
      // high merely because the same parent also reaches the allowed braces
      // advisory through another dependency path.
      if (transitive && !transitiveIsSevere) {
        continue;
      }

      if (!names.has(via) || !allowedPackages.has(via)) {
        throw new Error(`Unexpected high/critical transitive npm advisory path for ${name}: ${via}`);
      }
      continue;
    }

    const recognizedRootUrl = allowedRootUrls.get(via?.name);
    if (!recognizedRootUrl || via?.url !== recognizedRootUrl || via?.severity !== "high") {
      throw new Error(
        `Unexpected direct npm advisory for ${name}: ${JSON.stringify({
          name: via?.name,
          severity: via?.severity,
          url: via?.url,
        })}`
      );
    }
    sawExpectedRoot = true;
  }

  for (const node of finding.nodes ?? []) {
    const entry = packages[node];
    if (!entry || entry.dev !== true) {
      throw new Error(
        `High/critical npm finding ${name} is not proven dev-only at lockfile node ${node}`
      );
    }
  }
}

if (!sawExpectedRoot) {
  throw new Error("No recognized direct npm advisory root was present in npm audit output.");
}

const reported = [...names].sort();
console.warn(
  `Known dev-only npm advisory accepted only after local CVE patch verification: ${[...allowedRootUrls.values()].join(", ")}; affected chain: ${reported.join(", ")}`
);
console.warn(
  "Runtime dependencies remain separately blocked by npm audit --omit=dev --audit-level=high."
);
