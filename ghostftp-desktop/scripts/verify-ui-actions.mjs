import fs from "node:fs";
import path from "node:path";
import ts from "typescript";

const root = path.resolve("src");
const primaryViews = new Set(["settings", "siteManager", "transferCenter", "about"]);
const allowedSecondaryWindows = new Set(["lib/popout.ts"]);
const failures = [];
let buttonCount = 0;
let menuCount = 0;

function walk(dir) {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return walk(full);
    return /\.(ts|tsx)$/.test(entry.name) ? [full] : [];
  });
}

function attrMap(attrs) {
  const map = new Map();
  for (const prop of attrs.properties) {
    if (ts.isJsxAttribute(prop)) map.set(prop.name.text, prop.initializer ?? true);
  }
  return map;
}

function stringArg(call, index = 0) {
  const arg = call.arguments[index];
  return arg && (ts.isStringLiteral(arg) || ts.isNoSubstitutionTemplateLiteral(arg))
    ? arg.text
    : null;
}

for (const file of walk(root)) {
  const rel = path.relative(root, file).replaceAll(path.sep, "/");
  const source = fs.readFileSync(file, "utf8");
  const sf = ts.createSourceFile(
    file,
    source,
    ts.ScriptTarget.Latest,
    true,
    file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS
  );

  function visit(node) {
    if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
      const tag = node.tagName.getText(sf);
      if (tag === "button") {
        buttonCount++;
        const attrs = attrMap(node.attributes);
        const typeInit = attrs.get("type");
        const typeText =
          typeInit && ts.isStringLiteral(typeInit) ? typeInit.text :
          typeInit && ts.isJsxExpression(typeInit) ? typeInit.expression?.getText(sf) : "";
        const hasAction =
          attrs.has("onClick") ||
          attrs.has("onPointerDown") ||
          attrs.has("onMouseDown") ||
          typeText === "submit" ||
          typeText === "reset";
        if (!hasAction) {
          const pos = sf.getLineAndCharacterOfPosition(node.getStart(sf));
          failures.push(`${rel}:${pos.line + 1} button has no click/submit action`);
        }
        if (attrs.has("aria-haspopup")) {
          menuCount++;
          if (!attrs.has("aria-expanded")) {
            const pos = sf.getLineAndCharacterOfPosition(node.getStart(sf));
            failures.push(`${rel}:${pos.line + 1} popup trigger is missing aria-expanded`);
          }
        }
      }
    }

    if (ts.isCallExpression(node)) {
      const callee = node.expression.getText(sf);
      if (callee.endsWith("openDialog")) {
        const target = stringArg(node);
        if (target && primaryViews.has(target)) {
          const pos = sf.getLineAndCharacterOfPosition(node.getStart(sf));
          failures.push(`${rel}:${pos.line + 1} primary view "${target}" is still opened as a dialog`);
        }
      }
    }

    if (ts.isNewExpression(node) && node.expression.getText(sf).includes("WebviewWindow")) {
      if (!allowedSecondaryWindows.has(rel)) {
        const pos = sf.getLineAndCharacterOfPosition(node.getStart(sf));
        failures.push(`${rel}:${pos.line + 1} creates an unexpected secondary native window`);
      }
    }

    ts.forEachChild(node, visit);
  }
  visit(sf);

  if (/window\.open\s*\(/.test(source) || /target\s*=\s*["']_blank["']/.test(source)) {
    failures.push(`${rel}: browser popup navigation is not allowed inside Ghost FTP`);
  }
}

const appSource = fs.readFileSync(path.join(root, "App.tsx"), "utf8");
for (const view of ["files", "siteManager", "transferCenter", "settings", "about"]) {
  if (!appSource.includes(`view === "${view}"`)) {
    failures.push(`App.tsx does not render integrated view "${view}"`);
  }
}

if (failures.length) {
  console.error("Ghost FTP UI action audit FAILED:");
  for (const failure of failures) console.error(" -", failure);
  process.exit(1);
}

console.log(`Ghost FTP UI action audit OK: ${buttonCount} buttons, ${menuCount} menu triggers, integrated primary views verified.`);
