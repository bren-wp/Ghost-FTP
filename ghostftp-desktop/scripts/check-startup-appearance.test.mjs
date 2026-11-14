import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const read = (relative) => fs.readFileSync(path.join(ROOT, relative), "utf8");

function startupIssues({ native, entry, html, permissions }) {
  const issues = [];
  const created = native.slice(
    native.indexOf("tauri::WebviewWindowBuilder::new(app, \"main\""),
    native.indexOf("window_builder.build()?;")
  );
  if (!created.includes(".visible(false)")) {
    issues.push("native main window must start hidden");
  }
  if (!created.includes(".initialization_script(&init_script)")) {
    issues.push("native settings snapshot must be injected before first paint");
  }
  if (!entry.includes("function RevealNativeWindowAfterMount(")
      || !entry.includes("<RevealNativeWindowAfterMount>")
      || !entry.includes("useEffect(() => {")
      || !entry.includes("getCurrentWindow().show()")) {
    issues.push("React shell must reveal the window after mounting");
  }
  if (!entry.includes("if (!isTauri()) return;")) {
    issues.push("web preview must not call native reveal");
  }
  if (entry.includes("requestAnimationFrame(")) {
    issues.push("hidden native webviews may suppress requestAnimationFrame");
  }
  if (!permissions.includes("core:window:allow-show")
      || !permissions.includes("core:window:default")) {
    issues.push("main window must have narrowly scoped native show permission");
  }
  if (!html.includes("background-color: #04111e")
      || !html.includes('html[data-theme="light"]')
      || !html.includes('html[data-theme="ghost-ice"]')) {
    issues.push("startup document must have pre-bundle dark/light color fallbacks");
  }
  return issues;
}

function sources() {
  return {
    native: read("src-tauri/src/lib.rs"),
    entry: read("src/main.tsx"),
    html: read("index.html"),
    permissions: JSON.parse(read("src-tauri/capabilities/default.json")).permissions,
  };
}

test("main window remains hidden until the React shell is mounted and themed", () => {
  assert.deepEqual(startupIssues(sources()), []);
});

test("startup gate fails closed if the native webview becomes visible immediately", () => {
  const source = sources();
  assert.ok(startupIssues({
    ...source,
    native: source.native.replace(".visible(false)", ".visible(true)"),
  }).some(x => x.includes("hidden")));
});

test("startup gate prevents an invisible window caused by a missing native show permission", () => {
  const source = sources();
  assert.ok(startupIssues({
    ...source,
    permissions: source.permissions.filter(x => x !== "core:window:allow-show"),
  }).some(x => x.includes("permission")));
});

test("startup gate prevents reintroducing WebView dark/light flash or RAF deadlock", () => {
  const source = sources();
  assert.ok(startupIssues({
    ...source,
    html: source.html.replace("background-color: #04111e", "background-color: white"),
  }).some(x => x.includes("fallbacks")));
  assert.ok(startupIssues({
    ...source,
    entry: source.entry.replace("window.setTimeout(", "requestAnimationFrame("),
  }).some(x => x.includes("requestAnimationFrame")));
});
