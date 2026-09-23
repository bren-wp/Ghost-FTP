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
  "<WorkspaceErrorBoundary",
  "onReturnToFiles={showFiles}",
]) {
  if (!app.includes(required)) failures.push(`App missing single-window shell contract: ${required}`);
}
if (app.includes("standaloneDialog") || app.includes("<Suspense")) {
  failures.push("App must not hide the main shell or lazy-swap primary workspaces.");
}
if (!fs.existsSync("src/components/WorkspaceErrorBoundary.tsx")) {
  failures.push("Primary workspaces need a local error boundary so one render failure cannot replace the persistent app shell.");
}
for (const required of [
  "Couldn't initialize Sync & Backup",
  "Couldn't initialize application settings",
  "Couldn't initialize transfer activity",
  "Couldn't initialize app updates",
  "Couldn't register Ghost FTP deep-link listener",
  "useTransfers.getState().loadInitial()",
  "useTransfers.getState().initListeners()",
  "useUpdater.getState()",
  ".init()",
]) {
  if (!app.includes(required)) failures.push(`App startup/deep-link/transfer lifecycle must remain observable: ${required}`);
}
if (app.includes("<TransferQueue") || app.includes('from "./components/TransferQueue"')) {
  failures.push("Files must not reintroduce a duplicate transfer-management panel.");
}
if (fs.existsSync("src/components/TransferQueue.tsx")) {
  failures.push("Obsolete duplicate TransferQueue component must not return; use the Transfers workspace.");
}

if (fs.existsSync("src/components/ServerRail.tsx")) {
  failures.push("Obsolete duplicate ServerRail must not return; primary navigation lives in ReferenceSiteSidebar.");
}
const settingsStore = read("src/stores/settingsStore.ts");
for (const forbidden of [
  "railExpanded",
  "railCollapsedGroups",
  "setRailExpanded",
  "toggleRailGroup",
  "autoOpenTransferPanel",
  "setAutoOpenTransferPanel",
  "fileAssociations",
  "setFileAssociations",
]) {
  if (settingsStore.includes(forbidden)) failures.push(`Removed duplicate-shell setting returned: ${forbidden}`);
}

const transfersStore = read("src/stores/transfersStore.ts");
for (const required of ["rateById", "liveTransferRate", "RATE_STALE_AFTER_MS", "rateFromEvent"]) {
  if (!transfersStore.includes(required)) {
    failures.push(`Transfers store missing real progress-derived rate telemetry: ${required}`);
  }
}
for (const forbidden of ["panelOpen", "togglePanel", "setPanelOpen", "autoOpenTransferPanel"]) {
  if (transfersStore.includes(forbidden)) failures.push(`Transfers store still contains obsolete Files-panel state: ${forbidden}`);
}

const layoutStore = read("src/stores/layoutStore.ts");
for (const forbidden of ["cloudStorage", "schedules", "activityLogs"]) {
  for (const [label, source] of [
    ["layout store", layoutStore],
    ["app shell", app],
    ["title bar", read("src/components/TitleBar.tsx")],
    ["primary sidebar", read("src/components/ReferenceSiteSidebar.tsx")],
  ]) {
    if (source.includes(forbidden)) {
      failures.push(`${label} still contains obsolete hidden workspace alias: ${forbidden}`);
    }
  }
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
  'label="Local"',
  "New Folder",
  "Properties",
  "ghost-simple-header",
  "ghost-toolbar-more",
  'aria-haspopup="menu"',
  'aria-label="More file actions"',
  'fileAction("refresh", effectivePane)',
  'fileAction("newFolder", effectivePane)',
  'fileAction("rename", effectivePane)',
  'fileAction("delete", effectivePane)',
  'fileAction("properties", effectivePane)',
  'new CustomEvent("ghostftp:pick-upload")',
]) {
  if (!titleBar.includes(required)) failures.push(`Simplified header missing contextual action: ${required}`);
}
for (const forbidden of [
  "English (English)",
  "Quick Connect",
  "Application menu",
  "Bookmarks",
  "Tools",
  "Help Center",
  "ghost-menu-popover",
  "ghost-back-to-files",
  "Choose a site",
  '<Tool icon={<Pencil',
  '<Tool icon={<Trash2',
  '<Tool icon={<Info',
]) {
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

const filePane = read("packages/file-ui/src/components/FilePane.tsx");
if (!filePane.includes("Open Sites and choose a saved connection, or create a new connection.")) {
  failures.push("Remote FilePane empty state must point users to the simplified Sites/New connection navigation.");
}
if (filePane.includes("Pick a server in the left rail")) {
  failures.push("Remote FilePane still references the removed server rail.");
}

const profileEditor = read("src/components/ProfileEditor.tsx");
if (profileEditor.includes("void connectProfile(id);")) {
  failures.push("Profile pairing must not leave the post-pair connection promise unhandled.");
}
if (!profileEditor.includes("connectProfile(id).catch")) {
  failures.push("Profile pairing must catch a failed post-pair connection attempt.");
}

const newConnection = read("src/components/QuickConnectionDialog.tsx");
for (const required of [
  "Save this connection in Sites",
  "Host / Address",
  "ftp.your-domain.tld or 192.0.2.10",
  "Advanced Settings",
  "Test Connection",
  "Authentication",
  "Private key",
  "ghost-auth-choice",
  "cancelLabel",
]) {
  if (!newConnection.includes(required)) failures.push(`New Connection missing simplified form contract: ${required}`);
}
for (const forbidden of ["Quick Connect", "Save as Profile", "ghost-new-connection-mode", "Use private key (SSH)"]) {
  if (newConnection.includes(forbidden)) failures.push(`New Connection reintroduced duplicate mode UI: ${forbidden}`);
}

const commands = read("src/lib/commands.tsx");
for (const required of ['id: "open-transfers"', 'title: "Open Transfers"', 'openDialog("transferCenter")']) {
  if (!commands.includes(required)) failures.push(`Command palette must route transfer access to the Transfers workspace: ${required}`);
}
for (const forbidden of ["Toggle Transfer Panel", "togglePanel"]) {
  if (commands.includes(forbidden)) failures.push(`Command palette still references removed transfer panel behavior: ${forbidden}`);
}
if (!commands.includes("openNewConnection()")) {
  failures.push("Command palette New Connection must use openNewConnection so closing returns to its caller.");
}
if (commands.includes('openDialog("newConnection")')) {
  failures.push("Command palette must not bypass New Connection return-navigation state.");
}

const appShell = read("src/App.tsx");
if (!appShell.includes('"Back to Sites"')) {
  failures.push("Site Manager New Site must expose an explicit return path back to Sites.");
}

for (const obsolete of [
  "src-tauri/src/cli_updater.rs",
  "src/stores/cliUpdaterStore.ts",
  "src/components/CliUpdatePrompt.tsx",
  "src/components/CliUpdaterSettings.tsx",
]) {
  if (fs.existsSync(obsolete)) {
    failures.push(`Dead desktop CLI updater surface must not return: ${obsolete}`);
  }
}
for (const [label, source] of [
  ["Tauri app state", read("src-tauri/src/lib.rs")],
  ["frontend IPC", read("src/lib/ipc.ts")],
  ["frontend types", read("src/lib/types.ts")],
]) {
  for (const forbidden of [
    "cli_updater",
    "cliUpdaterStatus",
    "CliUpdateMode",
    "CliStatus",
    "onCliUpdaterStatus",
  ]) {
    if (source.includes(forbidden)) {
      failures.push(`${label} still contains removed desktop CLI updater residue: ${forbidden}`);
    }
  }
}

const cliMain = read("src-tauri/ghostftp-cli/src/main.rs");
for (const required of [
  "automatic ghostftp-cli replacement is disabled until signed package verification is available",
  "if check {",
]) {
  if (!cliMain.includes(required)) {
    failures.push(`ghostftp-cli self-update must remain read-only/fail-closed: ${required}`);
  }
}
for (const forbidden of [
  "fn download_bytes(",
  "fn swap_binary_at(",
  "Downloading {asset}",
]) {
  if (cliMain.includes(forbidden)) {
    failures.push(`ghostftp-cli must not restore unsigned executable replacement: ${forbidden}`);
  }
}

const appErrorBoundary = read("src/components/AppErrorBoundary.tsx");
if (appErrorBoundary.includes("Your files and server data were not modified")) {
  failures.push("Global recovery UI must not make an unverifiable claim about side effects from an operation that was already running.");
}
for (const [label, source] of [
  ["App error boundary", appErrorBoundary],
  ["Workspace error boundary", read("src/components/WorkspaceErrorBoundary.tsx")],
]) {
  if (!source.includes("redactSensitiveText")) {
    failures.push(`${label} must use central credential redaction before logging diagnostics.`);
  }
}

const mainEntry = read("src/main.tsx");
for (const forbidden of [
  "TerminalWindow",
  "sweepStalePopoutBuffers",
  'view === "terminal"',
  "popped-out terminals",
  "Plan 12",
]) {
  if (mainEntry.includes(forbidden)) failures.push(`Main entry still contains secondary-window/development residue: ${forbidden}`);
}

const terminal = read("src/components/Terminal.tsx");
for (const forbidden of ["openTerminalWindow", "Pop out active pane", "PictureInPicture2", "popoutBufferKey"]) {
  if (terminal.includes(forbidden)) failures.push(`Terminal still exposes a secondary-window action: ${forbidden}`);
}

const terminalRegistry = read("src/lib/terminalRegistry.ts");
for (const forbidden of ["setHandedOff", "handedOff", "SerializeAddon", "popout"]) {
  if (terminalRegistry.includes(forbidden)) failures.push(`Terminal registry still contains obsolete secondary-window handoff residue: ${forbidden}`);
}
const packageJson = read("package.json");
const packageLock = read("package-lock.json");
for (const [label, source] of [["package.json", packageJson], ["package-lock.json", packageLock]]) {
  if (source.includes("@xterm/addon-serialize")) {
    failures.push(`${label}: unused terminal serialize dependency returned after popout removal`);
  }
}


const terminalSuggestions = read("src/lib/termSuggest.ts");
for (const forbidden of ["localStorage", "ghostftp.term-history.v1:", "HISTORY_PREFIX"]) {
  if (terminalSuggestions.includes(forbidden)) {
    failures.push(`Terminal suggestion history must remain process-memory only: ${forbidden}`);
  }
}
for (const required of ["SENSITIVE_COMMAND_PATTERNS", "looksSensitiveCommand", "if (looksSensitiveCommand(cmd)) return"]) {
  if (!terminalSuggestions.includes(required)) {
    failures.push(`Terminal suggestion history is missing sensitive-command filtering: ${required}`);
  }
}

const toastStore = read("src/stores/toastStore.ts");
for (const forbidden of ["localStorage", "ghostftp.notifications.v1", "saveHistory(", "loadHistory("]) {
  if (toastStore.includes(forbidden)) {
    failures.push(`Notification history must remain session-only: ${forbidden}`);
  }
}
for (const required of ["redactSensitiveText(title", "redactSensitiveText(message"]) {
  if (!toastStore.includes(required)) {
    failures.push(`Notification text must pass through credential redaction: ${required}`);
  }
}

const secretMigration = read("src/lib/secretMigration.ts");
for (const required of [
  "purgeLegacySensitiveBrowserState",
  "ghostftp.notifications.v1",
  "ghostftp.term-history.v1:",
  "localStorage.removeItem",
]) {
  if (!secretMigration.includes(required)) {
    failures.push(`Startup migration must purge legacy sensitive browser state: ${required}`);
  }
}

const redaction = read("src/lib/redact.ts");
for (const required of [
  "SECRET_QUERY",
  "SECRET_ASSIGNMENT",
  "SECRET_FLAG",
  "AUTH_HEADER",
  "JSON_SECRET",
  "BEARER_TOKEN",
  "PRIVATE_KEY_BLOCK",
]) {
  if (!redaction.includes(required)) {
    failures.push(`Central credential redaction is incomplete: ${required}`);
  }
}

const settings = read("src/components/Settings.tsx");
for (const [label, source] of [
  ["Settings", settings],
  ["Help & About", read("src/components/AboutDialog.tsx")],
  ["Transfer Center", read("src/components/TransferCenterDialog.tsx")],
  ["Site Manager", read("src/components/SiteManagerDialog.tsx")],
]) {
  if (!source.includes("trapFocus: false")) {
    failures.push(`${label} primary workspace must not trap keyboard focus away from persistent navigation`);
  }
}

for (const required of ["Reset to Defaults", "Done", "Primary Language", "ghost-settings-tabs", "ghost-settings-tab-label", "aria-label={label}", "title={label}", "LanguagePanel", "Advanced", "Concurrent Transfers", "Speed Limit (KiB/s)", "Max Retry Attempts"]) {
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
  "Open transfer queue automatically",
  "autoOpenTransferPanel",
  'section="updates"',
  'section="sync"',
  'section="integrations"',
  'section="shortcuts"',
]) {
  if (settings.includes(forbidden)) failures.push(`Settings still duplicates application/settings navigation: ${forbidden}`);
}

const notificationToggleCount = (settings.match(/<DesktopNotificationsToggle\s*\/>/g) || []).length;
if (notificationToggleCount !== 1) {
  failures.push(`Settings must expose desktop notifications in exactly one section; found ${notificationToggleCount}`);
}
if (!settings.includes('advanced: "Terminal, system integrations and keyboard shortcuts."')) {
  failures.push("Advanced Settings description must match its Terminal/Integrations/Shortcuts grouping.");
}
if (!settings.includes('<div className="xl:col-span-2"><TerminalCard/></div>')) {
  failures.push("Terminal settings must live under Advanced rather than Transfers.");
}
for (const forbidden of ["File associations", "setFileAssociations", "fileAssociations", "source/dev builds"]) {
  if (settings.includes(forbidden)) failures.push(`Settings still exposes a nonfunctional/development-only control: ${forbidden}`);
}
for (const required of [
  '<StatusRow label="No tracking"/>',
  '<StatusRow label="No analytics or telemetry"/>',
  '<StatusRow label="Credentials stored with the operating system keychain"/>',
]) {
  if (!settings.includes(required)) failures.push(`Security settings must present fixed protections as status, not fake toggles: ${required}`);
}

for (const file of walkSource("src/components")) {
  if (!file.endsWith(".tsx") || file === "src/components/Settings.tsx") continue;
  const source = read(file);
  if (source.includes('aria-label="Language"') || source.includes("English (English)")) {
    failures.push(`${file}: application language selector must live in Settings only`);
  }
}

if (fs.existsSync("src/components/UpdatePrompt.tsx")) {
  failures.push("Updates must remain inside Help & About; obsolete global UpdatePrompt must not return.");
}
const updaterStore = read("src/stores/updaterStore.ts");
for (const forbidden of ["dismissed:", "dismiss: () =>"]) {
  if (updaterStore.includes(forbidden)) failures.push(`Updater store still contains obsolete global-prompt state: ${forbidden}`);
}

const about = read("src/components/AboutDialog.tsx");
for (const required of ["ghost-about-tabs", "PrivacyContent", "Ghost FTP Updates", "Ghost FTP Help Center"]) {
  if (!about.includes(required)) failures.push(`Help & About missing in-app section: ${required}`);
}
for (const forbidden of ["openOfficialUrl", "ReferenceWindowTitlebar", "Visit ghostftp.com"]) {
  if (about.includes(forbidden)) failures.push(`Help & About still escapes the single-window shell: ${forbidden}`);
}

const siteManager = read("src/components/SiteManagerDialog.tsx");
for (const required of [
  "ghost-site-filterbar",
  "FilterChip",
  "Import",
  "Export",
  "New Site",
  "Connect",
  "Test Connection",
  "ghost-sites-empty",
  "Add your first site",
  "Import Sites",
  "profiles.length === 0 && !query",
]) {
  if (!siteManager.includes(required)) failures.push(`Site Manager missing simplified/action contract: ${required}`);
}
for (const forbidden of ["ReferenceMenuTitlebar", "ReferenceActionRow", 'grid-cols-[242px_minmax(0,1fr)_356px]']) {
  if (siteManager.includes(forbidden)) failures.push(`Site Manager still contains duplicate nested navigation: ${forbidden}`);
}

const transferCenter = read("src/components/TransferCenterDialog.tsx");
for (const required of [
  "liveTransferRate",
  "rate={liveTransferRate(transfer, rateById[transfer.id])}",
  'aria-label="Transfers"',
  "<div className=\"text-xl font-semibold\">Transfers</div>",
  "Add Transfer",
  "Show Details",
  "Hide Details",
  "Schedule Transfer…",
  "Transfer Scheduler",
  "Set Schedule",
  "Clear Completed",
  "Pause All",
  "Retry",
  "detailsOpen",
  'active={tab === "active"}',
  'aria-label="Transfer direction filter"',
  'aria-label="Transfer time filter"',
]) {
  if (!transferCenter.includes(required)) failures.push(`Transfer Center missing action/filter contract: ${required}`);
}
if (!transferCenter.includes('filtered.length === 0 ? (\n            <Empty />')) {
  failures.push("Transfer Center empty state must render outside the wide transfer table so compact windows do not inherit table overflow.");
}
if (transferCenter.includes('<div className="min-w-[920px]">\n            {filtered.length === 0 ?')) {
  failures.push("Transfer Center empty state must not be wrapped in the 920px transfer-table width.");
}

for (const forbidden of [
  "function speedOf(transfer: Transfer)",
  "transfer.transferred / seconds",
  "previousTotals",
  "TransferCenterTitlebar",
  "ghost-transfer-language",
  "English (English)",
  "ReferenceWindowControls",
  'aria-label="Transfer status filter"',
  'active={tab === "upload"}',
  'active={tab === "download"}',
  'active={tab === "paused"}',
  'initialFocus?: "scheduler" | "log"',
  "setConcurrency",
  "setThrottle",
  "Concurrent</span>",
  "Throttle</span>",
]) {
  if (transferCenter.includes(forbidden)) failures.push(`Transfer Center still contains duplicate app navigation/filter control: ${forbidden}`);
}

const props = read("packages/file-ui/src/components/PropertiesModal.tsx");
for (const required of ["Open Containing Folder", "Duplicate", "Apply", "Checksums"]) {
  if (!props.includes(required)) failures.push(`File Properties missing required action: ${required}`);
}

const statusBar = read("src/components/ReferenceStatusBar.tsx");
for (const required of [
  "liveTransferRate",
  "rateById",
  'openDialog("transferCenter")',
  "ghost-status-transfer-link",
  'aria-label="Open Transfers"',
]) {
  if (!statusBar.includes(required)) failures.push(`Files transfer status must link to the single Transfers workspace: ${required}`);
}
if (statusBar.includes("transfer.transferred / elapsed")) {
  failures.push("Files status bar must not regress to average-since-start transfer speed.");
}

const styles = read("src/styles.css");
for (const required of [
  "RC16 single-window simplified navigation",
  "RC16 post-QA geometry corrections",
  "RC16 final Files workspace fill",
  "RC16 minimum-window responsive corrections",
  ".ghost-settings-tab-label {",
  ".ghost-primary-sidebar {",
  ".ghost-sidebar-new {",
  ".ghost-content-shell {",
  ".ghost-settings-tabs,",
  ".ghost-about-tabs,",
  ".ghost-site-filterbar {",
  ".ghost-workspace-view.ghost-standalone-view {",
  ".ghost-sites-empty {",
  ".ghost-sites-empty-card {",
  ".ghost-transfer-filters > .relative {",
  ".ghost-status-transfer-link {",
  "height: auto !important;",
  "max-height: none !important;",
  "flex: 1 1 auto !important;",
]) {
  if (!styles.includes(required)) failures.push(`Styles missing RC16 simplified-shell contract: ${required}`);
}

const nativeShell = read("src-tauri/src/lib.rs");
for (const required of [
  ".inner_size(1290.0, 852.0)",
  ".min_inner_size(480.0, 600.0)",
  ".prevent_overflow_with_margin(tauri::LogicalSize::new(32.0, 32.0))",
  ".center()",
  ".resizable(true)",
  ".maximized(false)",
  ".fullscreen(false)",
]) {
  if (!nativeShell.includes(required)) failures.push(`Default native window contract missing: ${required}`);
}
for (const forbidden of [".maximized(true)", ".fullscreen(true)"]) {
  if (nativeShell.includes(forbidden)) failures.push(`Default native window must remain non-maximized: ${forbidden}`);
}

const capability = read("src-tauri/capabilities/default.json");
if (!capability.includes('"windows": ["main"]')) {
  failures.push("Tauri capability scope must be restricted to the single main window.");
}
if (capability.includes("terminal-*")) {
  failures.push("Tauri capability scope still allows secondary terminal windows.");
}

const nativeBuildWorkflow = read("../.github/workflows/ghostftp-build.yml");
for (const required of ["QuantizedColors", "EdgeRatio", "byte-identical", "blank-or-structureless", "-Minimum.png", "minimum responsive viewport"]) {
  if (!nativeBuildWorkflow.includes(required)) failures.push(`Windows native QA missing structural blank-frame guard: ${required}`);
}

if (!styles.includes("@media (max-width: 760px)")) {
  failures.push("Primary navigation must keep labels until a truly narrow viewport.");
}
if (!styles.includes("@media (max-width: 620px)") || !styles.includes("flex-direction: column !important;")) {
  failures.push("Dual-pane Files must stack at near-minimum window widths instead of squeezing both panes horizontally.");
}
if (!styles.includes("RC16 compact Transfers filter reflow") ||
    !/\.ghost-transfer-filters\s*\{[\s\S]*?flex-wrap:\s*wrap;[\s\S]*?overflow-x:\s*hidden\s*!important;/.test(
      styles.slice(styles.indexOf("RC16 compact Transfers filter reflow"))
    )) {
  failures.push("Compact Transfers filters must wrap without horizontal filter scrolling.");
}
for (const required of [
  '.ghost-transfer-filters select[aria-label="Transfer direction filter"]',
  '.ghost-transfer-filters select[aria-label="Transfer time filter"]',
]) {
  if (!styles.includes(required)) failures.push(`Compact Transfers filters missing responsive selector: ${required}`);
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
