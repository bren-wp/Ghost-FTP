import fs from "node:fs";
import ts from "typescript";

const read = (path) => fs.readFileSync(path, "utf8");
const failures = [];

function walkSource(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = `${dir}/${entry.name}`;
    if (entry.isDirectory()) out.push(...walkSource(full));
    else if (/\.(ts|tsx)$/.test(entry.name)) out.push(full);
  }
  return out;
}


function jsxAttribute(node, name) {
  return node.attributes.properties.find(
    (attr) => ts.isJsxAttribute(attr) && attr.name.text === name
  );
}

function hasJsxSpread(node) {
  return node.attributes.properties.some((attr) => ts.isJsxSpreadAttribute(attr));
}

function expressionFromAttribute(attr) {
  if (!attr || !attr.initializer) return null;
  if (ts.isJsxExpression(attr.initializer)) return attr.initializer.expression ?? null;
  return null;
}

function isEmptyHandler(expr) {
  if (!expr) return false;
  if (ts.isArrowFunction(expr) || ts.isFunctionExpression(expr)) {
    return ts.isBlock(expr.body) && expr.body.statements.length === 0;
  }
  return false;
}

const auditedHandlerNames = [
  "onClick",
  "onChange",
  "onSubmit",
  "onPointerDown",
  "onPointerUp",
  "onMouseDown",
  "onKeyDown",
  "onKeyUp",
];

const interactiveAriaRoles = new Set([
  "button",
  "menuitem",
  "tab",
  "checkbox",
  "switch",
]);

function auditClickableTsx(file) {
  const sourceText = read(file);
  const sourceFile = ts.createSourceFile(
    file,
    sourceText,
    ts.ScriptTarget.Latest,
    true,
    file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS
  );

  const visit = (node) => {
    if (ts.isCatchClause(node) && node.block.statements.length === 0) {
      const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
      failures.push(`${file}:${pos.line + 1}: empty catch block hides a production error`);
    }

    if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
      const tag = node.tagName.getText(sourceFile);

      for (const handlerName of auditedHandlerNames) {
        const handler = jsxAttribute(node, handlerName);
        if (isEmptyHandler(expressionFromAttribute(handler))) {
          const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
          failures.push(`${file}:${pos.line + 1}: ${tag} has an empty ${handlerName} handler`);
        }
      }

      const roleAttr = jsxAttribute(node, "role");
      const role =
        roleAttr?.initializer && ts.isStringLiteral(roleAttr.initializer)
          ? roleAttr.initializer.text
          : null;
      if (
        role &&
        interactiveAriaRoles.has(role) &&
        !["button", "input", "select", "textarea", "a"].includes(tag)
      ) {
        const hasPointerAction = Boolean(
          jsxAttribute(node, "onClick") ||
          jsxAttribute(node, "onPointerDown") ||
          jsxAttribute(node, "onMouseDown") ||
          hasJsxSpread(node)
        );
        if (!hasPointerAction) {
          const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
          failures.push(`${file}:${pos.line + 1}: role="${role}" element has no pointer/click contract`);
        }
        if (
          (role === "button" || role === "tab") &&
          !jsxAttribute(node, "onKeyDown") &&
          !hasJsxSpread(node)
        ) {
          const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
          failures.push(`${file}:${pos.line + 1}: role="${role}" element has no keyboard activation contract`);
        }
      }

      if (tag === "button") {
        const onClick = jsxAttribute(node, "onClick");
        const onPointerDown = jsxAttribute(node, "onPointerDown");
        const onMouseDown = jsxAttribute(node, "onMouseDown");
        const type = jsxAttribute(node, "type");
        const typeText = type?.initializer && ts.isStringLiteral(type.initializer)
          ? type.initializer.text
          : null;
        const hasAction =
          Boolean(onClick || onPointerDown || onMouseDown) ||
          typeText === "submit" ||
          typeText === "reset" ||
          hasJsxSpread(node);

        if (!hasAction) {
          const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
          failures.push(`${file}:${pos.line + 1}: button has no click/pointer/submit contract`);
        }

      }

      if (tag === "a") {
        const href = jsxAttribute(node, "href");
        if (href?.initializer && ts.isStringLiteral(href.initializer)) {
          const hrefText = href.initializer.text.trim().toLowerCase();
          if (hrefText === "#") {
            const onClick = jsxAttribute(node, "onClick");
            if (!onClick && !hasJsxSpread(node)) {
              const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
              failures.push(`${file}:${pos.line + 1}: href="#" anchor has no click contract`);
            }
          }
          if (hrefText.startsWith("javascript:")) {
            const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
            failures.push(`${file}:${pos.line + 1}: javascript: anchors are not allowed`);
          }
        }
      }

      if (tag === "input" || tag === "select" || tag === "textarea") {
        const controlled = Boolean(
          jsxAttribute(node, "value") ||
          jsxAttribute(node, "checked")
        );
        const onChange = jsxAttribute(node, "onChange");
        const readOnly = jsxAttribute(node, "readOnly");
        const disabled = jsxAttribute(node, "disabled");
        if (controlled && !onChange && !readOnly && !disabled && !hasJsxSpread(node)) {
          const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
          failures.push(`${file}:${pos.line + 1}: controlled ${tag} has no onChange/readOnly/disabled contract`);
        }
      }
    }
    ts.forEachChild(node, visit);
  };

  visit(sourceFile);
}

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
  if (!source.includes("useDialog(")) {
    failures.push(`${file}: transient view must use the shared Escape/focus dialog contract`);
  }
}

for (const file of workspaceFiles) {
  const source = read(file);
  if (!source.includes("useDialog(")) {
    failures.push(`${file}: workspace view must keep the shared Escape/focus close contract`);
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
  "src/components/DualPaneBrowser.tsx",
  "src/components/ProfileEditor.tsx",
  "src/components/ServerRail.tsx",
  "src/lib/commands.tsx",
  "packages/file-ui/src/components/PropertiesModal.tsx",
];

for (const file of [...walkSource("src"), ...walkSource("packages/file-ui/src")]) {
  const source = read(file);
  if (file.endsWith(".tsx") || file.endsWith(".ts")) auditClickableTsx(file);
  if (/onClick\s*:\s*\(\)\s*=>\s*\{\s*\}/.test(source)) {
    failures.push(`${file}: contains a fake no-op onClick handler`);
  }
  if (/\.catch\s*\(\s*\(\s*\)\s*=>\s*\{\s*\}\s*\)/.test(source)) {
    failures.push(`${file}: contains a silent rejected-promise handler`);
  }
  if (/[\u3400-\u9fff]/u.test(source)) {
    failures.push(`${file}: unexpected CJK text found in the production English/Balkan source UI`);
  }
  if (/window\.open\s*\(/.test(source) || /target\s*=\s*["']_blank["']/.test(source)) {
    failures.push(`${file}: popup/new-tab navigation is not allowed inside Ghost FTP`);
  }
  if (/new\s+WebviewWindow\s*\(/.test(source) && file !== "src/lib/popout.ts") {
    failures.push(`${file}: secondary Tauri windows are restricted to the explicit Terminal pop-out implementation`);
  }
}

for (const file of criticalFiles) {
  const source = read(file);

  if (/window\.open\s*\(/.test(source) || /target\s*=\s*["']_blank["']/.test(source)) {
    failures.push(`${file}: browser popup/new-tab navigation is not allowed in the production shell`);
  }
  if (/\.catch\s*\(\s*\(\s*\)\s*=>\s*\{\s*\}\s*\)/.test(source)) {
    failures.push(`${file}: critical UI contains a silent rejected-promise handler`);
  }

  // Critical actions are checked explicitly below. Avoid regex-parsing JSX
  // opening tags here because TypeScript generics inside handlers contain ">"
  // characters and would create false dead-button reports.

}

const appShell = read("src/App.tsx");
if (appShell.includes("openTerminalWindow(")) {
  failures.push("src/App.tsx: terminal deep links must remain docked inside the single Ghost FTP window");
}
if (!appShell.includes("setTerminalOpen(true)")) {
  failures.push("src/App.tsx: single-window terminal deep link must open the in-app terminal dock");
}
if (appShell.includes("lazy(") || appShell.includes("<Suspense")) {
  failures.push("src/App.tsx: primary Ghost FTP views must not use lazy/Suspense transitions that can flash or blank the workspace");
}
for (const required of [
  "Couldn't initialize Sync & Backup",
  "Couldn't initialize application settings",
  "Couldn't register Ghost FTP deep-link listener",
]) {
  if (!appShell.includes(required)) {
    failures.push(`src/App.tsx: startup/deep-link failure must stay observable: ${required}`);
  }
}

const sidebar = read("src/components/ReferenceSiteSidebar.tsx");
for (const required of ["Transfer Center", "File Manager", "Sync & Backup", "Cloud Storage", "Schedules", "Activity Logs", "Settings"]) {
  if (!sidebar.includes(required)) failures.push(`Reference sidebar missing required navigation: ${required}`);
}
if (!sidebar.includes('openDialog("sync")')) failures.push("Sync & Backup must open the real in-app sync workspace.");
for (const note of ["Secure Connections", "Fast Transfers", "Modern Interface", "Cross-Platform", "Built for Creators"]) {
  if (!sidebar.includes(note)) failures.push(`Reference sidebar missing capability note: ${note}`);
}
for (const route of ['openDialog("cloudStorage")', 'openDialog("schedules")', 'openDialog("activityLogs")']) {
  if (!sidebar.includes(route)) failures.push(`Sidebar must use purpose-specific in-app route: ${route}`);
}

const app = read("src/App.tsx");
for (const required of [
  'className="ghost-file-manager-body flex min-h-0 flex-1 overflow-hidden"',
  'className="ghost-file-manager-right flex min-w-0 flex-1 flex-col"',
  "<TransferQueue />",
]) {
  if (!app.includes(required)) failures.push(`App missing newest-reference shell contract: ${required}`);
}
const sidebarIndex = app.indexOf("<ReferenceSiteSidebar />");
const rightIndex = app.indexOf("ghost-file-manager-right");
const transferIndex = app.indexOf("<TransferQueue />");
if (sidebarIndex < 0 || rightIndex < sidebarIndex || transferIndex < rightIndex) {
  failures.push("File Manager must keep the site rail beside the workspace + transfer band, matching the newest reference.");
}

const titleBar = read("src/components/TitleBar.tsx");
if (titleBar.includes("openOfficialUrl")) failures.push("Top Help menu must stay inside the Ghost FTP app.");
for (const required of [
  "Quick Connect",
  "Site Manager",
  "New Folder",
  "Properties",
  "Settings",
]) {
  if (!titleBar.includes(required)) failures.push(`TitleBar missing required action: ${required}`);
}
for (const route of ['openDialog("help")', 'openDialog("updates")', 'openDialog("about")']) {
  if (!titleBar.includes(route)) failures.push(`TitleBar must use in-app workspace route: ${route}`);
}
if (!titleBar.includes('useState<Protocol>("sftp")')) {
  failures.push("TitleBar Quick Connect must default to SFTP like the approved reference.");
}
if (!titleBar.includes("useState(22)")) {
  failures.push("TitleBar Quick Connect must default to port 22 like the approved reference.");
}

const siteManager = read("src/components/SiteManagerDialog.tsx");
for (const required of ["Import", "Export", "New Site", "Connect", "Test Connection"]) {
  if (!siteManager.includes(required)) failures.push(`Site Manager missing required action: ${required}`);
}
for (const required of ["saveDialog(", "ipc.exportProfiles(", "Couldn't export sites"]) {
  if (!siteManager.includes(required)) failures.push(`Site Manager missing native export/error contract: ${required}`);
}

const transferCenter = read("src/components/TransferCenterDialog.tsx");
for (const required of ["Add Transfer", "Schedule", "Transfer Scheduler", "Set Schedule", "Priority", "Concurrent", "Throttle", "Clear Completed", "More", "Pause All", "Retry", "Paused", "All Directions", "Any Time", "runBackendAction"]) {
  if (!transferCenter.includes(required)) failures.push(`Transfer Center missing required action: ${required}`);
}

const settings = read("src/components/Settings.tsx");
for (const required of ["Reset to Defaults", "Cancel", "Apply"]) {
  if (!settings.includes(required)) failures.push(`Preferences missing required action: ${required}`);
}
for (const required of ["GeneralPerformanceCard", "Concurrent Transfers", "Speed Limit (KiB/s)", "Max Retry Attempts"]) {
  if (!settings.includes(required)) failures.push(`Preferences missing reference Performance contract: ${required}`);
}

const quick = read("src/components/QuickConnectionDialog.tsx");
for (const required of ["Test Connection", "Save Profile", "Connect"]) {
  if (!quick.includes(required)) failures.push(`New Connection missing required action: ${required}`);
}

const filePane = read("packages/file-ui/src/components/FilePane.tsx");
for (const required of ["Type", "Permissions", "JavaScript File", "Markdown File", "ENV File"]) {
  if (!filePane.includes(required)) failures.push(`FilePane missing reference metadata contract: ${required}`);
}
for (const required of ["Couldn't copy path", "Couldn't copy name"]) {
  if (!filePane.includes(required)) failures.push(`FilePane clipboard errors must be visible: ${required}`);
}

const styles = read("src/styles.css");
for (const required of [
  "flex: 0 0 175px !important;",
  "height: 65px !important;",
  "height: 60px !important;",
  "flex: 0 0 423px !important;",
  "width: 204px !important;",
  "flex: 0 0 214px !important;",
  "flex: 0 0 40px !important;",
  "grid-template-columns: minmax(0, 1.55fr) minmax(360px, 1fr) !important;",
]) {
  if (!styles.includes(required)) failures.push(`Styles missing newest-reference 1290x852 geometry: ${required}`);
}
for (const required of [
  ".ghost-reference-statusbar {",
  "display: flex !important;",
  "align-items: center !important;",
]) {
  if (!styles.includes(required)) failures.push(`Styles missing status-bar reference contract: ${required}`);
}

for (const required of [
  "RC14 compact-height guard",
  "@media (max-width: 1289px), (max-height: 851px)",
  "max-height: none !important;",
  "flex: 0 1 clamp(132px, 24dvh, 190px) !important;",
]) {
  if (!styles.includes(required)) failures.push(`Styles missing compact-height clipping guard: ${required}`);
}

for (const required of [
  "RC14 transfer-center fidelity",
  "flex: 0 0 210px;",
  "flex: 0 0 178px;",
]) {
  if (!styles.includes(required)) failures.push(`Styles missing Transfer Center table-space fidelity guard: ${required}`);
}

for (const required of [
  "RC15 reference-density pass",
  "grid-template-rows: repeat(4, minmax(0, 1fr));",
  "grid-template-rows: minmax(344px, 1.15fr) minmax(190px, .85fr);",
  "grid-template-rows: minmax(360px, 1.15fr) minmax(180px, .85fr);",
]) {
  if (!styles.includes(required)) failures.push(`Styles missing RC15 standalone reference-density guard: ${required}`);
}

const nativeBuildWorkflow = read("../.github/workflows/ghostftp-build.yml");
for (const required of ["QuantizedColors", "EdgeRatio", "byte-identical", "blank-or-structureless"]) {
  if (!nativeBuildWorkflow.includes(required)) failures.push(`Windows native QA missing structural blank-frame guard: ${required}`);
}

const props = read("packages/file-ui/src/components/PropertiesModal.tsx");
for (const required of ["Open Containing Folder", "Duplicate", "Apply", "Checksums"]) {
  if (!props.includes(required)) failures.push(`File Properties missing required action: ${required}`);
}

if (failures.length) {
  console.error("Ghost FTP UI contract failed:\n" + failures.map((x) => " - " + x).join("\n"));
  process.exit(1);
}

console.log("Ghost FTP UI contract OK: single-shell views, transient overlays, popup guards and TSX click contracts are valid.");
