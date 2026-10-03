import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";

const allowedRootUrl = "https://github.com/advisories/GHSA-vfj7-8cjw-p6xm";
const allowedPackages = new Set([
  "braces",
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
      if (!names.has(via) || !allowedPackages.has(via)) {
        throw new Error(`Unexpected transitive npm advisory path for ${name}: ${via}`);
      }
      continue;
    }

    if (via?.url !== allowedRootUrl || via?.name !== "braces" || via?.severity !== "high") {
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
  throw new Error("Expected braces advisory root was not present in npm audit output.");
}

const reported = [...names].sort();
console.warn(
  `Known dev-only npm advisory temporarily accepted: ${allowedRootUrl}; affected chain: ${reported.join(", ")}`
);
console.warn(
  "Runtime dependencies remain separately blocked by npm audit --omit=dev --audit-level=high."
);
