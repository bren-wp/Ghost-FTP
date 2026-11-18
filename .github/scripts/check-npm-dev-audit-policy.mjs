// Fail-closed policy for the npm development-dependency audit.
// Only the specifically patched picomatch advisory may be conditionally accepted.
const rootAdvisories = new Map([
  ["picomatch", "https://github.com/advisories/GHSA-fc5q-39xc-vjpf"],
]);
const allowedAffectedPackages = new Set([
  "@vitejs/plugin-react", "anymatch", "fdir", "picomatch", "readdirp",
  "sucrase", "tinyglobby", "vite", "chokidar", "fast-glob",
  "micromatch", "tailwindcss",
]);
const isSevere = (finding) =>
  finding?.severity === "high" || finding?.severity === "critical";

export function evaluateNpmDevAudit(audit, lock) {
  if (!audit || audit.auditReportVersion !== 2 || audit.error ||
      !audit.metadata?.vulnerabilities ||
      !audit.vulnerabilities || typeof audit.vulnerabilities !== "object" ||
      Array.isArray(audit.vulnerabilities)) {
    throw new Error("npm audit failed or returned an incomplete report; refuse a clean verdict");
  }
  if (!lock?.packages || typeof lock.packages !== "object") {
    throw new Error("npm lockfile packages are missing");
  }

  const vulnerabilities = audit.vulnerabilities;
  const severe = Object.entries(vulnerabilities).filter(([, value]) => isSevere(value));
  if (severe.length === 0) return { affected: [], roots: [] };

  const names = new Set(severe.map(([name]) => name));
  for (const name of names) {
    if (!allowedAffectedPackages.has(name)) {
      throw new Error("Unexpected high/critical npm vulnerability package: " + name);
    }
  }

  const seenRoots = new Set();
  for (const [name, finding] of severe) {
    if (!Array.isArray(finding.via) || finding.via.length === 0) {
      throw new Error("Missing npm advisory path for " + name);
    }
    if (!Array.isArray(finding.nodes) || finding.nodes.length === 0) {
      throw new Error("Missing npm lockfile paths for " + name);
    }

    for (const via of finding.via) {
      if (typeof via === "string") {
        const transitive = vulnerabilities[via];
        if (!transitive) {
          throw new Error("Unresolved npm advisory dependency: " + name + " -> " + via);
        }
        if (!isSevere(transitive)) continue;
        if (!names.has(via) || !allowedAffectedPackages.has(via)) {
          throw new Error("Unexpected high/critical npm advisory path: " + name + " -> " + via);
        }
        continue;
      }

      const recognized = rootAdvisories.get(via?.name);
      if (!recognized || via.url !== recognized || via.severity !== "high") {
        throw new Error("Unexpected direct npm advisory for " + name + ": " +
          JSON.stringify({ name: via?.name, severity: via?.severity, url: via?.url }));
      }
      seenRoots.add(via.name);
    }

    for (const node of finding.nodes) {
      if (typeof node !== "string" || lock.packages[node]?.dev !== true) {
        throw new Error("High/critical npm finding is not proven dev-only: " + name +
          " at " + String(node));
      }
    }
  }

  if (!seenRoots.has("picomatch")) {
    throw new Error("No independently verified, locally patched advisory root found");
  }
  return { affected: [...names].sort(), roots: [...seenRoots].sort() };
}
