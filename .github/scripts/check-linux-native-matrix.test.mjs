import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";

const root = resolve(fileURLToPath(new URL("../..", import.meta.url)));
const read = (path) => readFileSync(resolve(root, path), "utf8");
const map = {
  splash: "Splash",
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
  assert.match(build, /for view in splash main siteManager newConnection transferCenter settings sync about; do/);
  assert.match(build, /path: qa\/linux\/\*\*/);
  assert.ok(capture.includes("No user profiles"));
  for (const [view, label] of Object.entries(map)) {
    assert.ok(capture.includes(`${view}) qa_image="Linux-${label}-1290x852.png"`), view);
    assert.ok(build.includes(`${view}) label=${label} ;;`), view);
    assert.ok(rust.includes(`"${view}"`), `Tauri did not allow-list ${view}`);
    if (view !== "main" && view !== "splash") {
      assert.ok(react.includes(`qaView === "${view}"`), `React did not route ${view}`);
    }
  }
  assert.ok(react.includes('__GHOSTFTP_QA_VIEW__ === "splash"'));
  assert.ok(react.includes("<StartupSplash />"));
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

test("Linux README screenshots come only from checksummed native PR artifacts", () => {
  const build = read(".github/workflows/ghostftp-build.yml");
  const promotion = read(".github/scripts/promote-real-linux-readme.sh");
  assert.ok(build.includes("promote_real_linux_native_readme:"));
  assert.ok(build.includes("name: GhostFTP-Linux-x86_64-Preview-QA"));
  assert.ok(build.includes("needs: build"));
  assert.ok(promotion.includes('sha256sum "$source_path/$filename"'));
  assert.ok(promotion.includes("real-linux-native-captures"));
  assert.ok(promotion.includes("No design mockups") || promotion.includes("no design mockups"));
  assert.ok(!promotion.includes("screens/linux/"), "must not publish concept art as native screenshots");
});

test("Windows/Linux Quick Connect warns on FTP cleartext and resets passwords on protocol switch", () => {
  const chrome = read("ghostftp-desktop/src/components/PremiumFilesChrome.tsx");
  const connection = read("ghostftp-desktop/src/components/QuickConnectionDialog.tsx");
  assert.ok(chrome.includes('protocol === "ftp"'), "plain FTP must have a visible warning");
  assert.ok(chrome.includes("Plain FTP is unencrypted"), "plain FTP warning must be explicit");
  assert.ok(chrome.includes("if (value !== protocol) {"), "switching protocol must clear temporary secrets");
  assert.ok(chrome.includes('setPassword("");'), "Quick Connect must clear stale password");
  assert.ok(connection.includes("Unencrypted FTP"), "saved-profile editor must also warn on cleartext FTP");
  assert.ok(!connection.includes('{ftp ? "Recommended" : "Secure"}'), "plain FTP cannot be advertised Recommended");
});


test("Windows/Linux Sites shows real saved-profile cards with separate keyboard actions", () => {
  const sites = read("ghostftp-desktop/src/components/SiteManagerDialog.tsx");
  const styles = read("ghostftp-desktop/src/styles.css");
  assert.ok(sites.includes('className="ghost-site-row-main"'));
  assert.ok(sites.includes('className="ghost-site-row-favorite"'));
  assert.ok(sites.includes('aria-current={active ? "true" : undefined}'));
  assert.ok(sites.includes("await connect(selected.id)"));
  assert.ok(sites.includes("await deleteProfile(selected.id)"));
  assert.ok(sites.includes("profiles.filter("));
  assert.ok(styles.includes(".ghost-site-row.is-selected"));
  assert.ok(styles.includes(".ghost-site-row-main:focus-visible"));
});

test("Windows/Linux New Connection follows reference groups and retains secret-safe actions", () => {
  const dialog = read("ghostftp-desktop/src/components/QuickConnectionDialog.tsx");
  const css = read("ghostftp-desktop/src/styles.css");
  const required = [
    'aria-labelledby="ghost-new-connection-title"',
    'ghost-new-connection-server-grid',
    'ghost-new-connection-auth-grid',
    'ghost-new-connection-panel-heading',
    'await saveProfile(profile)',
    'await ipc.testEphemeralConnection(makeProfile())',
    'setPassword("")',
    'setKeyPassphrase("")',
    'Unencrypted FTP',
    'type="number"',
  ];
  for (const value of required) assert.ok(dialog.includes(value), "New Connection missing: " + value);
  assert.ok(css.includes(".ghost-new-connection-panel"));
  assert.ok(css.includes(".ghost-new-connection-actions .ghost-primary-button"));
  assert.ok(!dialog.includes("localStorage"));
  assert.ok(!dialog.includes("sessionStorage"));
});
