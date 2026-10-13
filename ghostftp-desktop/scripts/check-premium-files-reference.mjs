#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const source = (file) => fs.readFileSync(path.join(root, file), "utf8");
const chrome = source("ghostftp-desktop/src/components/PremiumFilesChrome.tsx");
const app = source("ghostftp-desktop/src/App.tsx");
const css = source("ghostftp-desktop/src/styles.css");
const savedSiteDialog = source("ghostftp-desktop/src/components/QuickConnectionDialog.tsx");
const desktopSidebar = source("ghostftp-desktop/src/components/ReferenceSiteSidebar.tsx");
const macOS = source("macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift");
const macOSFiles = source("macos/Sources/GhostFTPMacApp/Views/ContentView.swift");
const android = source("android/app/src/main/java/com/ghostftp/android/MainActivity.kt");
const androidSaved = source("android/app/src/main/java/com/ghostftp/android/SavedSiteProfiles.kt");
const connectionStore = source("ghostftp-desktop/src/stores/connectionsStore.ts");
const violations = [];
const requireText = (actual, text, error) => {
  if (!actual.includes(text)) violations.push(error);
};
for (const component of ["PremiumFilesChrome", "PremiumLiveTransferStrip"]) {
  requireText(app, `<${component} />`, `${component} is missing from real Files workspace`);
}
requireText(chrome, "await connectTemporary(profile)", "Quick Connect does not create a real ephemeral backend session");
requireText(chrome, 'type="password"', "Quick Connect must collect a masked transient password");
requireText(chrome, 'setPassword("")', "Quick Connect does not clear the transient credential on success");
if (chrome.includes("openNewConnection(")) violations.push("Quick Connect still launches the saved-profile editor");
if (chrome.includes("New connection</span>")) violations.push("Duplicate New connection button in Files chrome");
requireText(savedSiteDialog, "await saveProfile(profile)", "New connection does not save a reusable profile");
requireText(savedSiteDialog, "Save profile", "New connection editor lacks a visible Save profile action");
if (savedSiteDialog.includes("await connectTemporary(") || savedSiteDialog.includes("await connectProfile(")) {
  violations.push("New saved-site dialog must not start network sessions");
}
requireText(desktopSidebar, 'aria-label="New connection"', "Sidebar lacks the single New connection action");
requireText(macOS, "newSiteDraft = ConnectionProfile()", "macOS New connection does not start an unsaved draft");
if (macOS.includes("profiles.addProfile()")) violations.push("macOS New connection persists an empty site before Save");
requireText(macOSFiles, "session.connect(profile: profile, password: password)", "macOS temporary FTP Connect is not bound to its backend");
requireText(macOSFiles, "profiles.save(profile)", "macOS saved-site draft cannot be persisted on explicit Save");
requireText(android, 'primaryButton(getString(R.string.action_connect)) { openConnection() }', "Android direct Connect no longer invokes its transport");
requireText(android, 'SavedSiteProfiles.save(this@MainActivity, name, profile)', "Android lacks explicit Save new profile");
requireText(android, 'passwordInput.text.clear()', "Android saved-profile selection does not clear credentials");
if (androidSaved.includes('put("password"') || androidSaved.includes('put("token"')) {
  violations.push("Android saved-site metadata includes secrets");
}
requireText(chrome, 'type="submit"', "Quick connection cannot submit from keyboard");
requireText(chrome, '"Connecting…"', "Quick Connect must report the pending state");
requireText(chrome, '"Failed — check connection details"', "Quick Connect must report a failed attempt");
requireText(connectionStore, 'password: ""', "Temporary sessions retain a password in the UI store");
requireText(connectionStore, 'passphrase: undefined', "Temporary sessions retain a key passphrase in the UI store");
requireText(macOS, "newSiteDraft = nil\n                    selectedSiteID = id", "macOS saved-site Open must clear an abandoned draft");
requireText(android, "pendingSavedProfilePort = entry.protocol to entry.port", "Android saved-site loading has no custom-port guard");
requireText(android, "nextProtocol == loadedPort.first", "Android protocol spinner can overwrite a loaded port");
if (/ghost-premium-quickconnect\s*>\s*label:nth-child\(2\)\s*\{\s*display:\s*none/i.test(css)) {
  violations.push("Compact desktop layout hides the required Quick Connect username");
}
requireText(chrome, 'event.preventDefault()', "Quick connection submits an invalid native form");
requireText(chrome, 'PROTOCOL_DEFAULT_PORT[value]', "Protocol change does not update default port");
requireText(chrome, 'state.activeSessionId', "Connected status not bound to real session");
requireText(chrome, 'state.byId', "Transfer rows are not bound to live backend transfer store");
requireText(chrome, 'openDialog("transferCenter")', "Transfer strip does not open working transfer center");
requireText(chrome, 'No transfers in progress', "No honest empty transfer state");
for (const cssClass of [
  "ghost-premium-files-heading",
  "ghost-premium-quickconnect",
  "ghost-premium-connection-state",
  "ghost-premium-transfer-strip",
  "ghost-premium-transfer-progress",
]) {
  requireText(css, `.${cssClass}`, `Missing genuine Files reference styling: ${cssClass}`);
}
for (const dangerous of ['localStorage', 'sessionStorage', 'fakeTransfer', 'mockTransfer', 'example.com']) {
  if (chrome.includes(dangerous)) violations.push(`Credential/demo logic in inline quick-connect: ${dangerous}`);
}
if (violations.length > 0) {
  console.error("Premium Windows/Linux Files UI contract failed:");
  for (const violation of violations) console.error(" - " + violation);
  process.exit(1);
}
console.log("Real Windows/Linux Files header, immediate ephemeral Quick Connect, one saved-profile action and live transfer strip: contract passed. Pixel acceptance still pending.");
