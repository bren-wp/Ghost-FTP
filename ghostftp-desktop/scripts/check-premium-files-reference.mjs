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
// Allow the explicit Files-header New connection CTA. Only the Quick
// Connect submission path is forbidden from opening the profile editor.
const quickConnectHandler = chrome.split("const connectNow = async () => {")[1]?.split("\n  return (")[0] ?? "";
if (quickConnectHandler.includes("openNewConnection(")) {
  violations.push("Quick Connect still launches the saved-profile editor");
}
// The wide Windows/Linux reference places the only *visible* New connection
// action in the Files heading, rather than in the left navigation rail.
// Compact layouts retain the sidebar button for discoverability.
requireText(chrome, 'onClick={() => openNewConnection()}', "Files reference heading lacks a correctly typed new-profile action");
requireText(chrome, 'aria-label="Create a new saved connection"', "Files reference CTA has no accessible label");
requireText(css, '.ghost-primary-sidebar > .ghost-sidebar-new { display: none; }', "Desktop layout has duplicate visible New connection buttons");
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
requireText(chrome, 'type="submit"', "Quick connection cannot submit from keyboard");
requireText(chrome, '"Connecting…"', "Quick Connect must report the pending state");
requireText(chrome, '"Failed — check connection details"', "Quick Connect must report a failed attempt");
requireText(connectionStore, 'password: ""', "Temporary sessions retain a password in the UI store");
requireText(connectionStore, 'passphrase: undefined', "Temporary sessions retain a key passphrase in the UI store");
requireText(macOS, "newSiteDraft = nil\n                    selectedSiteID = id", "macOS saved-site Open must clear an abandoned draft");
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

// The native health badge is driven by the real session probe. A broken
// connection may not be represented by the green secure-transport icon.
const statusBar = source("ghostftp-desktop/src/components/ReferenceStatusBar.tsx");
requireText(chrome, 'activeSession?.health === "unhealthy"', "Files status ignores failing native connection probes");
requireText(chrome, 'ghost-connection-health-warning-icon', "Failed session still uses a green connection badge");
requireText(chrome, 'activeSession?.health === "healthy"', "Secure badge is not conditional on known health");
requireText(chrome, 'finally {\n      // A failed handshake', "Quick Connect does not clear passwords after failed connection attempts");
requireText(css, ".ghost-connection-health-warning-icon", "Missing prominent failed session styling");
const clock = statusBar.split("function StatusClock() {")[1]?.split("function formatSpeed(")[0] ?? "";
requireText(clock, 'document.visibilityState === "visible"', "Hidden Windows/Linux window still updates the status clock every second");
requireText(clock, 'window.clearInterval(interval)', "Clock interval cannot be stopped while hidden");


// New connection geometry follows the Windows/Linux 1290x852 workspace,
// without removing the working (opt-in) saved-site name field.
requireText(savedSiteDialog, "ghost-new-connection-overlay", "Desktop reference modal lacks a scoped overlay");
requireText(savedSiteDialog, 'Site name (optional)', "Saved-site naming was removed instead of moved to Advanced");
requireText(savedSiteDialog, 'advanced && <div className="grid grid-cols-2 gap-3 border-t border-border p-4">', "Advanced editor controls are no longer accessible");
const detailsStart = savedSiteDialog.indexOf('aria-labelledby="ghost-new-server-heading"');
const detailsEnd = savedSiteDialog.indexOf('aria-labelledby="ghost-new-auth-heading"');
if (detailsStart < 0 || detailsEnd < detailsStart ||
    savedSiteDialog.slice(detailsStart, detailsEnd).includes("Site name (optional)")) {
  violations.push("Optional site naming still shifts the real initial New Connection reference cards");
}
requireText(css, "padding-left: 230px !important", "Modal remains centered over Linux/Windows navigation rail");
requireText(css, "transform: translateY(51px)", "Modal lacks workspace-aligned reference Y offset");


// The native modal's warning icon must match the actual transport: plaintext
// FTP has no TLS/SSH peer confidentiality, regardless of successful login.
requireText(savedSiteDialog, 'protocol === "ftp"\n              ? <ShieldAlert', "Plain FTP editor incorrectly shows a green verified-transport icon");
requireText(css, ".ghost-ftp-warning-icon", "Plain FTP warning has no dedicated amber icon style");
const healthSection = statusBar.split("  useEffect(() => {")[1]?.split("  const healthLabel =")[0] ?? "";
requireText(healthSection, "if (healthInterval !== undefined) window.clearInterval(healthInterval)", "Native session health probe continues polling in hidden windows");
requireText(healthSection, 'document.addEventListener("visibilitychange", syncVisibility)', "Health polling fails to resume on visibility changes");

if (violations.length > 0) {
  console.error("Premium Windows/Linux Files UI contract failed:");
  for (const violation of violations) console.error(" - " + violation);
  process.exit(1);
}
console.log("Real Windows/Linux Files reference CTA, immediate ephemeral Quick Connect, responsive single visible saved-profile action and factual transfer strip: contract passed. Pixel acceptance still pending.");
