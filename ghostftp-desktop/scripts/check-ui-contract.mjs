import fs from "node:fs";

const read = (path) => fs.readFileSync(path, "utf8");
const failures = [];

const workspaceFiles = [
  "src/components/SiteManagerDialog.tsx",
  "src/components/TransferCenterDialog.tsx",
  "src/components/Settings.tsx",
  "src/components/AboutDialog.tsx",
];

const transientFiles = [
  "src/components/QuickConnectionDialog.tsx",
  "packages/file-ui/src/components/PropertiesModal.tsx",
];

for (const file of workspaceFiles) {
  const source = read(file);
  if (!source.includes("ghost-workspace-view")) {
    failures.push(`${file}: primary view must use ghost-workspace-view`);
  }
  if (source.includes("fixed inset-0 z-modal")) {
    failures.push(`${file}: primary workspace view must not be a modal overlay`);
  }
  if (source.includes('aria-modal="true"')) {
    failures.push(`${file}: primary workspace view must not claim modal semantics`);
  }
}

for (const file of transientFiles) {
  const source = read(file);
  if (!source.includes("ghost-transient-overlay")) {
    failures.push(`${file}: transient view must use the shared in-app overlay shell`);
  }
}

const criticalFiles = [
  "src/components/TitleBar.tsx",
  "src/components/ReferenceWindowChrome.tsx",
  "src/components/SiteManagerDialog.tsx",
  "src/components/TransferCenterDialog.tsx",
  "src/components/Settings.tsx",
  "src/components/AboutDialog.tsx",
  "src/components/QuickConnectionDialog.tsx",
  "packages/file-ui/src/components/PropertiesModal.tsx",
];

for (const file of criticalFiles) {
  const source = read(file);

  if (/window\.open\s*\(/.test(source) || /target\s*=\s*["']_blank["']/.test(source)) {
    failures.push(`${file}: browser popup/new-tab navigation is not allowed in the production shell`);
  }

  const tagSafeSource = source.replace(/=>/g, "__ARROW__");
  const buttons = tagSafeSource.match(/<button\b[\s\S]*?>/g) ?? [];
  for (const tag of buttons) {
    const actionable =
      /onClick\s*=/.test(tag) ||
      /type\s*=\s*["']submit["']/.test(tag) ||
      /disabled\s*=\s*\{true\}/.test(tag);
    if (!actionable) {
      const compact = tag.replace(/\s+/g, " ").slice(0, 180);
      failures.push(`${file}: button without an explicit click/submit contract: ${compact}`);
    }
  }
}

const titleBar = read("src/components/TitleBar.tsx");
for (const required of [
  "Quick Connect",
  "Site Manager",
  "New Folder",
  "Properties",
  "Settings",
]) {
  if (!titleBar.includes(required)) failures.push(`TitleBar missing required action: ${required}`);
}

const siteManager = read("src/components/SiteManagerDialog.tsx");
for (const required of ["Import", "Export", "New Site", "Connect", "Test Connection"]) {
  if (!siteManager.includes(required)) failures.push(`Site Manager missing required action: ${required}`);
}

const transferCenter = read("src/components/TransferCenterDialog.tsx");
for (const required of ["Add Transfer", "Pause All", "Clear Finished", "Retry"]) {
  if (!transferCenter.includes(required)) failures.push(`Transfer Center missing required action: ${required}`);
}

const settings = read("src/components/Settings.tsx");
for (const required of ["Reset to Defaults", "Cancel", "Apply"]) {
  if (!settings.includes(required)) failures.push(`Preferences missing required action: ${required}`);
}

const quick = read("src/components/QuickConnectionDialog.tsx");
for (const required of ["Test Connection", "Save Profile", "Connect"]) {
  if (!quick.includes(required)) failures.push(`New Connection missing required action: ${required}`);
}

const props = read("packages/file-ui/src/components/PropertiesModal.tsx");
for (const required of ["Open Containing Folder", "Duplicate", "Apply", "Checksums"]) {
  if (!props.includes(required)) failures.push(`File Properties missing required action: ${required}`);
}

if (failures.length) {
  console.error("Ghost FTP UI contract failed:\n" + failures.map((x) => " - " + x).join("\n"));
  process.exit(1);
}

console.log("Ghost FTP UI contract OK: single-shell views, transient overlays and critical click handlers are present.");
