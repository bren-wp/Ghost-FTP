import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";

const root = resolve(fileURLToPath(new URL("../..", import.meta.url)));
const read = (path) => readFileSync(resolve(root, path), "utf8");
const map = {
  main: "Files",
  siteManager: "Sites",
  newConnection: "New-Connection",
  transferCenter: "Transfers",
  settings: "Settings",
  sync: "Sync",
  about: "About",
};

test("seven Linux QA captures are mapped to real Tauri and React views", () => {
  const capture = read(".github/scripts/capture-linux-premium.sh");
  const build = read(".github/workflows/ghostftp-build.yml");
  const rust = read("ghostftp-desktop/src-tauri/src/lib.rs");
  const react = read("ghostftp-desktop/src/App.tsx");
  assert.match(build, /for view in main siteManager newConnection transferCenter settings sync about; do/);
  assert.match(build, /path: qa\/linux\/\*\*/);
  assert.ok(capture.includes("No user profiles"));
  for (const [view, label] of Object.entries(map)) {
    assert.ok(capture.includes(`${view}) qa_image="Linux-${label}-1290x852.png"`), view);
    assert.ok(build.includes(`${view}) label=${label} ;;`), view);
    assert.ok(rust.includes(`"${view}"`), `Tauri did not allow-list ${view}`);
    if (view !== "main") {
      assert.ok(react.includes(`qaView === "${view}"`) || react.includes(`qaView === "newConnection"`), `React did not route ${view}`);
    }
  }
  assert.ok(!capture.includes("WEBKIT_DISABLE_SANDBOX"));
  assert.ok(!capture.includes("export GDK_GL=disable"));
});

test("GitHub release notes disclose Linux distro package families", () => {
  const publisher = read(".github/scripts/publish-release.sh");
  const doc = read("docs/distribution/LINUX_PACKAGE_MATRIX.md");
  assert.ok(publisher.includes('LINUX_PACKAGE_MATRIX="docs/distribution/LINUX_PACKAGE_MATRIX.md"'));
  assert.ok(publisher.includes('--rawfile notes "$RELEASE_NOTES_WITH_LINUX"'));
  for (const format of [".deb", ".rpm", ".AppImage", ".tar.gz"]) {
    assert.ok(doc.includes(format), `missing distribution mapping: ${format}`);
  }
  for (const family of ["Debian", "Ubuntu", "Fedora", "openSUSE", "RHEL", "x86_64"]) {
    assert.ok(doc.includes(family), `missing Linux compatibility qualification: ${family}`);
  }
});

test("shared Windows/Linux connection editor clears secrets on auth and protocol switches", () => {
  const dialog = read("ghostftp-desktop/src/components/QuickConnectionDialog.tsx");
  assert.ok(dialog.includes("const switchAuth = (nextUseKey: boolean) => {"));
  assert.ok(dialog.includes("onClick={()=>switchAuth(true)}"));
  assert.ok(dialog.includes("onClick={()=>switchAuth(false)}"));
  assert.ok(dialog.includes("if(p!==protocol){clearSecrets();}"), "switching to plain FTP must not reuse another protocol password");
  assert.ok(dialog.includes('autoComplete="off"'), "password inputs must disable browser autocomplete");
  assert.ok(dialog.includes("setKeyPassphrase(\"\")"));
  assert.ok(dialog.includes("setPassword(\"\")"));
});
