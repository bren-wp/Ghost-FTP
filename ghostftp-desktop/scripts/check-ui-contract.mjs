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
    if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
      const tag = node.tagName.getText(sourceFile);

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

        const clickExpr = expressionFromAttribute(onClick);
        if (isEmptyHandler(clickExpr)) {
          const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
          failures.push(`${file}:${pos.line + 1}: button has an empty onClick handler`);
        }
      }

      if (tag === "a") {
        const href = jsxAttribute(node, "href");
        if (href?.initializer && ts.isStringLiteral(href.initializer) && href.initializer.text === "#") {
          const onClick = jsxAttribute(node, "onClick");
          if (!onClick && !hasJsxSpread(node)) {
            const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
            failures.push(`${file}:${pos.line + 1}: href="#" anchor has no click contract`);
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
        if (isEmptyHandler(expressionFromAttribute(onChange)) && !readOnly && !disabled) {
          const pos = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
          failures.push(`${file}:${pos.line + 1}: ${tag} has an empty onChange handler`);
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

for (const file of [...walkSource("src"), ...walkSource("packages/file-ui/src")]) {
  const source = read(file);
  if (file.endsWith(".tsx")) auditClickableTsx(file);
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

const sidebar = read("src/components/ReferenceSiteSidebar.tsx");
for (const required of ["Transfer Center", "File Manager", "Sync & Backup", "Cloud Storage", "Schedules", "Activity Logs", "Settings"]) {
  if (!sidebar.includes(required)) failures.push(`Reference sidebar missing required navigation: ${required}`);
}
if (!sidebar.includes('openDialog("sync")')) failures.push("Sync & Backup must open the real in-app sync workspace.");
for (const route of ['openDialog("cloudStorage")', 'openDialog("schedules")', 'openDialog("activityLogs")']) {
  if (!sidebar.includes(route)) failures.push(`Sidebar must use purpose-specific in-app route: ${route}`);
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

const siteManager = read("src/components/SiteManagerDialog.tsx");
for (const required of ["Import", "Export", "New Site", "Connect", "Test Connection"]) {
  if (!siteManager.includes(required)) failures.push(`Site Manager missing required action: ${required}`);
}

const transferCenter = read("src/components/TransferCenterDialog.tsx");
for (const required of ["Add Transfer", "Schedule", "Transfer Scheduler", "Set Schedule", "Priority", "Clear Completed", "More", "Pause All", "Retry", "Paused", "All Directions", "Any Time"]) {
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

console.log("Ghost FTP UI contract OK: single-shell views, transient overlays, popup guards and TSX click contracts are valid.");
