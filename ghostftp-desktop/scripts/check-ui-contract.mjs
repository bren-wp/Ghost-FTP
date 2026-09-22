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

const interactiveAriaRoles = new Set(["button", "menuitem", "tab", "checkbox", "switch"]);

function auditFireAndForgetPromises(file) {
  const sourceText = read(file);
  const sourceFile = ts.createSourceFile(
    file,
    sourceText,
    ts.ScriptTarget.Latest,
    true,
    file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS
  );

  const visit = (node) => {
    if (
      ts.isCallExpression(node) &&
      ts.isPropertyAccessExpression(node.expression) &&
      node.expression.name.text === "then" &&
      node.arguments.length < 2
    ) {
      let cursor = node;
      let top = node;
      let handled = false;

      while (cursor.parent) {
        const parent = cursor.parent;
        if (
          ts.isPropertyAccessExpression(parent) &&
          parent.expression === cursor &&
          parent.name.text === "catch" &&
          ts.isCallExpression(parent.parent) &&
          parent.parent.expression === parent
        ) {
          handled = true;
          top = parent.parent;
          break;
        }
        if (
          ts.isPropertyAccessExpression(parent) ||
          ts.isCallExpression(parent) ||
          ts.isParenthesizedExpression(parent)
        ) {
          top = parent;
          cursor = parent;
          continue;
        }
        break;
      }

      const standalone =
        ts.isExpressionStatement(top.parent) ||
        (ts.isVoidExpression(top.parent) && ts.isExpressionStatement(top.parent.parent));

      if (standalone && !handled) {
        const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
        failures.push(
          `${file}:${pos.line + 1}: fire-and-forget .then() chain has no rejection handler`
        );
      }
    }
    ts.forEachChild(node, visit);
  };

  visit(sourceFile);
}

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
        const typeText =
          type?.initializer && ts.isStringLiteral(type.initializer)
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

      if (tag === "input" || tag === "select" || tag === "textarea") {
        const controlled = Boolean(jsxAttribute(node, "value") || jsxAttribute(node, "checked"));
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
    failures.push(`${file}: primary workspace must render inside the shared main-window workspace`);
  }
  if (source.includes('aria-modal="true"') || source.includes("fixed inset-0 z-modal")) {
    failures.push(`${file}: primary workspace must not render as a separate/modal application window`);
  }
  if (!source.includes("useDialog(")) {
    failures.push(`${file}: workspace must keep the shared Escape/focus close contract`);
  }
}

for (const file of transientFiles) {
  const source = read(file);
  if (!source.includes("ghost-transient-overlay")) {
    failures.push(`${file}: transient editor must use the shared in-app overlay`);
  }
  if (!source.includes("useDialog(")) {
    failures.push(`${file}: transient editor must use the shared dialog focus contract`);
  }
}

for (const file of [...walkSource("src"), ...walkSource("packages/file-ui/src")]) {
  const source = read(file);
  auditFireAndForgetPromises(file);
  if (file.endsWith(".tsx") || file.endsWith(".ts")) auditClickableTsx(file);
  if (/onClick\s*:\s*\(\)\s*=>\s*\{\s*\}/.test(source)) {
    failures.push(`${file}: contains a fake no-op onClick handler`);
  }
  if (/\.catch\s*\(\s*\(\s*\)\s*=>\s*\{\s*\}\s*\)/.test(source)) {
    failures.push(`${file}: contains a silent rejected-promise handler`);
  }
  if (/catch\s*\{\s*\}/.test(source)) {
    failures.push(`${file}: contains an empty catch block`);
  }
  if (/window\.open\s*\(/.test(source) || /target\s*=\s*["']_blank["']/.test(source)) {
    failures.push(`${file}: browser popup/new-tab navigation is not allowed inside Ghost FTP`);
  }
  if (/new\s+WebviewWindow\s*\(/.test(source)) {
    failures.push(`${file}: secondary Tauri application windows are forbidden`);
  }
  if (/[\u3400-\u9fff]/u.test(source)) {
    failures.push(`${file}: unexpected CJK text found in production UI source`);
  }
}

const app = read("src/App.tsx");
for (const required of [
  "<TitleBar />",
  "<ReferenceSiteSidebar />",
  "ghost-app-body",
  "ghost-content-shell",
  "workspaceFor(",
  "fileManager && <TransferQueue />",
]) {
  if (!app.includes(required)) failures.push(`App missing single-window shell contract: ${required}`);
}
if (app.includes("standaloneDialog") || app.includes("<Suspense")) {
  failures.push("App must not hide the main shell or lazy-swap primary workspaces.");
}
for (const required of [
  "Couldn't initialize Sync & Backup",
  "Couldn't initialize application settings",
  "Couldn't register Ghost FTP deep-link listener",
]) {
  if (!app.includes(required)) failures.push(`App startup/deep-link failure must remain observable: ${required}`);
}

if (fs.existsSync("src/components/ServerRail.tsx")) {
  failures.push("Obsolete duplicate ServerRail must not return; primary navigation lives in ReferenceSiteSidebar.");
}
const settingsStore = read("src/stores/settingsStore.ts");
for (const forbidden of ["railExpanded", "railCollapsedGroups", "setRailExpanded", "toggleRailGroup"]) {
  if (settingsStore.includes(forbidden)) failures.push(`Removed duplicate-rail setting returned: ${forbidden}`);
}

const sidebar = read("src/components/ReferenceSiteSidebar.tsx");
for (const required of ["New connection", "Files", "Sites", "Transfers", "Sync & Backup", "Settings", "Help & About", "aria-label={label}", "title={label}"]) {
  if (!sidebar.includes(required)) failures.push(`Primary sidebar missing: ${required}`);
}
for (const forbidden of ["Secure Connections", "Fast Transfers", "Modern Interface", "Cross-Platform", "Built for Creators", "Schedules", "Activity Logs", 'label="Cloud Storage"']) {
  if (sidebar.includes(forbidden)) failures.push(`Primary sidebar still contains duplicate/noisy navigation: ${forbidden}`);
}

const siteManagerCloud = read("src/components/SiteManagerDialog.tsx");
for (const required of ['view === "cloud"', 'label="Cloud"', "s3", "azure", "gcs"]) {
  if (!siteManagerCloud.includes(required)) failures.push(`Sites must retain discoverable cloud filtering: ${required}`);
}

const titleBar = read("src/components/TitleBar.tsx");
for (const required of [
  "Choose a site",
  'label="Local"',
  "New Folder",
  "Properties",
  "ghost-simple-header",
  'fileAction("refresh", effectivePane)',
  'fileAction("newFolder", effectivePane)',
  'fileAction("rename", effectivePane)',
  'fileAction("delete", effectivePane)',
  'fileAction("properties", effectivePane)',
  'new CustomEvent("ghostftp:pick-upload")',
]) {
  if (!titleBar.includes(required)) failures.push(`Simplified header missing contextual action: ${required}`);
}
for (const forbidden of ["English (English)", "Quick Connect", "Application menu", "Bookmarks", "Tools", "Help Center", "ghost-menu-popover", "ghost-back-to-files"]) {
  if (titleBar.includes(forbidden)) failures.push(`Header still contains duplicate navigation/language control: ${forbidden}`);
}

const fileBrowser = read("src/components/FileBrowser.tsx");
for (const required of [
  'window.addEventListener("ghostftp:pick-upload"',
  'setBrowseLocal(true)',
  'setBrowseLocal(false)',
]) {
  if (!fileBrowser.includes(required)) failures.push(`Single-pane File Browser missing navigation/upload bridge: ${required}`);
}

const newConnection = read("src/components/QuickConnectionDialog.tsx");
for (const required of [
  "Save this connection in Sites",
  "Host / Address",
  "ftp.your-domain.tld or 192.0.2.10",
  "Advanced Settings",
  "Test Connection",
]) {
  if (!newConnection.includes(required)) failures.push(`New Connection missing simplified form contract: ${required}`);
}
for (const forbidden of ["Quick Connect", "Save as Profile", "ghost-new-connection-mode"]) {
  if (newConnection.includes(forbidden)) failures.push(`New Connection reintroduced duplicate mode UI: ${forbidden}`);
}

const commands = read("src/lib/commands.tsx");
if (!commands.includes("openNewConnection()")) {
  failures.push("Command palette New Connection must use openNewConnection so closing returns to its caller.");
}
if (commands.includes('openDialog("newConnection")')) {
  failures.push("Command palette must not bypass New Connection return-navigation state.");
}

const mainEntry = read("src/main.tsx");
for (const forbidden of ["TerminalWindow", "sweepStalePopoutBuffers", 'view === "terminal"']) {
  if (mainEntry.includes(forbidden)) failures.push(`Main entry still supports a secondary app window: ${forbidden}`);
}

const terminal = read("src/components/Terminal.tsx");
for (const forbidden of ["openTerminalWindow", "Pop out active pane", "PictureInPicture2", "popoutBufferKey"]) {
  if (terminal.includes(forbidden)) failures.push(`Terminal still exposes a secondary-window action: ${forbidden}`);
}

const settings = read("src/components/Settings.tsx");
for (const required of ["Reset to Defaults", "Done", "Primary Language", "ghost-settings-tabs", "LanguagePanel", "Advanced", "Concurrent Transfers", "Speed Limit (KiB/s)", "Max Retry Attempts"]) {
  if (!settings.includes(required)) failures.push(`Settings missing simplified contract: ${required}`);
}
for (const required of ['const syncOnly = initialSection === "sync"', "{!syncOnly && (", "{!syncOnly && <button"]) {
  if (!settings.includes(required)) failures.push(`Sync workspace must not expose the general Settings navigation/reset controls: ${required}`);
}
for (const forbidden of [
  "ReferenceWindowTitlebar",
  "GeneralGrid",
  "GeneralPerformanceCard",
  "GeneralIntegrationsCard",
  "GeneralUpdatesCard",
  'section="updates"',
  'section="sync"',
  'section="integrations"',
  'section="shortcuts"',
]) {
  if (settings.includes(forbidden)) failures.push(`Settings still duplicates application/settings navigation: ${forbidden}`);
}

for (const file of walkSource("src/components")) {
  if (!file.endsWith(".tsx") || file === "src/components/Settings.tsx") continue;
  const source = read(file);
  if (source.includes('aria-label="Language"') || source.includes("English (English)")) {
    failures.push(`${file}: application language selector must live in Settings only`);
  }
}

const about = read("src/components/AboutDialog.tsx");
for (const required of ["ghost-about-tabs", "PrivacyContent", "Ghost FTP Updates", "Ghost FTP Help Center"]) {
  if (!about.includes(required)) failures.push(`Help & About missing in-app section: ${required}`);
}
for (const forbidden of ["openOfficialUrl", "ReferenceWindowTitlebar", "Visit ghostftp.com"]) {
  if (about.includes(forbidden)) failures.push(`Help & About still escapes the single-window shell: ${forbidden}`);
}

const siteManager = read("src/components/SiteManagerDialog.tsx");
for (const required of ["ghost-site-filterbar", "FilterChip", "Import", "Export", "New Site", "Connect", "Test Connection"]) {
  if (!siteManager.includes(required)) failures.push(`Site Manager missing simplified/action contract: ${required}`);
}
for (const forbidden of ["ReferenceMenuTitlebar", "ReferenceActionRow", 'grid-cols-[242px_minmax(0,1fr)_356px]']) {
  if (siteManager.includes(forbidden)) failures.push(`Site Manager still contains duplicate nested navigation: ${forbidden}`);
}

const transferCenter = read("src/components/TransferCenterDialog.tsx");
for (const required of ["Add Transfer", "Schedule", "Transfer Scheduler", "Set Schedule", "Clear Completed", "Pause All", "Retry"]) {
  if (!transferCenter.includes(required)) failures.push(`Transfer Center missing action: ${required}`);
}
for (const forbidden of ["TransferCenterTitlebar", "ghost-transfer-language", "English (English)", "ReferenceWindowControls"]) {
  if (transferCenter.includes(forbidden)) failures.push(`Transfer Center still contains duplicate app navigation/language control: ${forbidden}`);
}

const props = read("packages/file-ui/src/components/PropertiesModal.tsx");
for (const required of ["Open Containing Folder", "Duplicate", "Apply", "Checksums"]) {
  if (!props.includes(required)) failures.push(`File Properties missing required action: ${required}`);
}

const styles = read("src/styles.css");
for (const required of [
  "RC16 single-window simplified navigation",
  ".ghost-primary-sidebar {",
  ".ghost-sidebar-new {",
  ".ghost-content-shell {",
  ".ghost-settings-tabs,",
  ".ghost-about-tabs,",
  ".ghost-site-filterbar {",
  ".ghost-workspace-view.ghost-standalone-view {",
]) {
  if (!styles.includes(required)) failures.push(`Styles missing RC16 simplified-shell contract: ${required}`);
}

const capability = read("src-tauri/capabilities/default.json");
if (!capability.includes('"windows": ["main"]')) {
  failures.push("Tauri capability scope must be restricted to the single main window.");
}
if (capability.includes("terminal-*")) {
  failures.push("Tauri capability scope still allows secondary terminal windows.");
}

const nativeBuildWorkflow = read("../.github/workflows/ghostftp-build.yml");
for (const required of ["QuantizedColors", "EdgeRatio", "byte-identical", "blank-or-structureless"]) {
  if (!nativeBuildWorkflow.includes(required)) failures.push(`Windows native QA missing structural blank-frame guard: ${required}`);
}

if (!styles.includes("@media (max-width: 760px)")) {
  failures.push("Primary navigation must keep labels until a truly narrow viewport.");
}
for (const required of [
  "grid-template-rows: minmax(180px, 42%) minmax(0, 58%);",
  "display: block !important;",
]) {
  if (!styles.includes(required)) failures.push(`Narrow Site Manager must retain editable connection details: ${required}`);
}

if (failures.length) {
  console.error("Ghost FTP UI contract failed:\n" + failures.map((x) => " - " + x).join("\n"));
  process.exit(1);
}

console.log("Ghost FTP UI contract OK: one main window, one primary navigation surface, in-app workspaces and actionable controls.");
