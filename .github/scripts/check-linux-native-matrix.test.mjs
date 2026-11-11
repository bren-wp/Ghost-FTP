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

test("seven Linux QA captures open actual workspaces without a splash screen", () => {
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
      assert.ok(react.includes(`qaView === "${view}"`), `React did not route ${view}`);
    }
  }
  assert.ok(!react.includes("StartupSplash"), "The frontend must not import a splash component");
  assert.ok(!react.includes("startupVisible"), "No artificial startup delay should block Files");
  assert.ok(!rust.includes('"splash"'), "Native QA must not expose a retired splash view");
  assert.ok(!capture.includes("Linux-Splash-1290x852.png"), "No Linux splash capture");
  assert.ok(!build.includes("key = 'splash'"), "No Windows splash capture");
  assert.ok(!build.includes("for view in splash"), "No Linux splash workflow view");
  assert.ok(!capture.includes("WEBKIT_DISABLE_SANDBOX"));
  assert.ok(!capture.includes("export GDK_GL=disable"));
  assert.ok(!read("ghostftp-desktop/index.html").includes('lang="zh-CN"'), "Default language must not claim Chinese");
  assert.ok(read("ghostftp-desktop/index.html").includes('lang="en"'));
  assert.ok(!read("ghostftp-desktop/src/styles.css").includes(".ghost-startup-splash"), "No orphaned splash overlay CSS");
  assert.ok(!read("README.md").includes('src="docs/assets/screenshots/native-linux/ghostftp-linux-splash.png"'), "No obsolete splash in active README gallery");
  assert.ok(read(".github/scripts/promote-real-linux-readme.sh").includes("seven real native Ghost FTP screens"));
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
  assert.ok(sites.includes("const deletionTarget = confirmDeleteId"), "deletion must pin the originally confirmed profile");
  assert.ok(sites.includes("await deleteProfile(deletionTarget.id)"), "delete must await the pinned original profile, not a newly selected row");
  assert.ok(sites.includes('onConfirm={remove}'), "confirmation must use the guarded async remove callback");
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

test("native Linux capture recovers a genuine pre-mapped GTK window without weakening visual checks", () => {
  const capture = read(".github/scripts/capture-linux-premium.sh");
  const workflow = read(".github/workflows/ghostftp-build.yml");
  assert.ok(workflow.includes("-s '-screen 0 1290x852x24' dbus-run-session --"),
    "D-Bus must launch INSIDE Xvfb so desktop portals inherit DISPLAY");
  assert.ok(!workflow.includes("dbus-run-session -- xvfb-run"),
    "Launching D-Bus before DISPLAY causes post-merge native GUI failures");
  for (const guard of [
    'find_real_ghostftp_window() {',
    "xwininfo -root -tree",
    "xdotool search --name",
    'xprop -id "$candidate" WM_CLASS',
    '(( width >= 500 && height >= 350 ))',
    'xdotool windowmap "$window"',
    'xdotool windowraise "$window"',
    '"$validator" "$output/$qa_image"',
  ]) {
    assert.ok(capture.includes(guard), "Linux GUI capture regression: " + guard);
  }
  assert.ok(!capture.includes("WEBKIT_DISABLE_SANDBOX"), "Never disable WebKit process isolation");
  assert.ok(capture.includes('if [ "$valid_capture" -ne 1 ]'), "Blank screenshots must fail closed");
});
