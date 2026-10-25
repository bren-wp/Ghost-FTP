#!/usr/bin/env node
// A source consistency gate for the reviewed Ghost FTP brand on all OSes.
// Source-token parity must never be reported as measured 1:1 screenshot parity.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (relative) => fs.readFileSync(path.join(root, relative), "utf8");
const tokens = JSON.parse(read("docs/branding/premium-ui-tokens.json"));
const style = read("android/app/src/main/java/com/ghostftp/android/PremiumReferenceStyle.kt");
const splash = read("android/app/src/main/java/com/ghostftp/android/SplashActivity.kt");
const main = read("android/app/src/main/java/com/ghostftp/android/MainActivity.kt");
const original = read("android/app/src/main/res/drawable/ic_ghost_premium_splash.xml");
const shape = (xml) => [...xml.matchAll(/android:pathData="(M24,155[^"]+)"/g)].map(m => m[1]);
const canonical = shape(original);
if (canonical.length !== 2 || canonical[0] !== canonical[1]) {
  throw Error("Ghost FTP reviewed splash icon paths are inconsistent");
}
for (const relative of [
  "android/app/src/main/res/drawable/ic_ghost_ftp.xml",
  "android/app/src/main/res/drawable/ic_ghost_ftp_foreground.xml"
]) {
  const xml = read(relative);
  const current = shape(xml);
  if (current.length !== 2 || current.some(p => p !== canonical[0]) ||
      !xml.includes('android:fillColor="#38ABFF"') ||
      !xml.includes('android:viewportWidth="192"')) {
    throw Error("Android launcher/splash/app symbol geometry drift: " + relative);
  }
}
const launcher = read("android/app/src/main/res/mipmap-anydpi-v26/ic_ghost_ftp_launcher.xml");
const background = read("android/app/src/main/res/drawable/ic_ghost_ftp_launcher_background.xml");
const manifest = read("android/app/src/main/AndroidManifest.xml");
if (!launcher.includes("@drawable/ic_ghost_ftp_foreground") ||
    !launcher.includes("@drawable/ic_ghost_ftp_launcher_background") ||
    !background.includes(tokens.colors.background) ||
    !manifest.includes('android:icon="@mipmap/ic_ghost_ftp_launcher"')) {
  throw Error("Android adaptive launch icon does not match approved brand");
}
for (const api of ["v31", "v35"]) {
  if (!read("android/app/src/main/res/values-" + api + "/styles.xml")
      .includes("@drawable/ic_ghost_ftp")) {
    throw Error("Wrong Ghost FTP platform splash on Android " + api);
  }
}
for (const [key, rgb] of [
  ["canvas", [7, 14, 26]], ["panel", [13, 25, 43]],
  ["raised", [17, 31, 52]], ["purple", [56, 171, 255]],
  ["purpleLight", [126, 222, 255]], ["text", [234, 246, 255]]
]) {
  if (!style.includes("val " + key + " = Color.rgb(" + rgb.join(", ") + ")")) {
    throw Error("Ghost FTP Android UI color drift: " + key);
  }
}
if (tokens.android.referenceWidthPx !== 480 ||
    tokens.android.referenceHeightPx !== 960 ||
    tokens.android.phoneTopChromeHeightDp !== 52 ||
    tokens.android.drawerWidthDp !== 280 ||
    tokens.android.overlayBelowDp !== 600) {
  throw Error("Android reference geometry differs from user's 16-screen design manifest");
}
for (const required of [
  "setPadding(dp(8), dp(2), dp(10), dp(2))",
  "addView(phoneMenuButton, LinearLayout.LayoutParams(dp(48), dp(48)))",
  "addView(phoneFilesViewButton, LinearLayout.LayoutParams(dp(48), dp(48)))",
  "addView(phoneQuickButton, LinearLayout.LayoutParams(dp(48), dp(48)))",
  "minimumHeight = dp(48)"
]) {
  if (!main.includes(required)) {
    throw Error("Real Android phone touch target regression: " + required);
  }
}
for (const required of [
  "minimumHeight = dp(context, 48)",
  "LinearLayout.LayoutParams(dp(context, 48), dp(context, 48))",
  "contentDescription = context.getString(R.string.ui_header_back)",
  "contentDescription = context.getString(R.string.ui_header_add)",
  "contentDescription = context.getString(R.string.ui_protocol_server, protocol.label)"
]) {
  if (!style.includes(required)) {
    throw Error("Ghost FTP native accessible button drift: " + required);
  }
}
for (const key of ["ui_splash_tagline", "ui_splash_opening", "ui_splash_initializing"]) {
  if (!splash.includes("getString(R.string." + key + ")")) {
    throw Error("Android splash label is not localized: " + key);
  }
}
if (!main.includes("setImageResource(R.drawable.ic_ghost_ftp)")) {
  throw Error("Android toolbar no longer uses the approved Ghost FTP symbol");
}
const desktop = read("ghostftp-desktop/src/styles.css");
const mac = read("macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift");
if (!desktop.includes("width: 214px !important") ||
    !desktop.includes("min-height: 43px") ||
    !mac.includes(".frame(minWidth: 840, minHeight: 560)") ||
    tokens.referenceCounts.windows !== 30 ||
    tokens.referenceCounts.linux !== 29 ||
    tokens.referenceCounts.android !== 16 ||
    tokens.macos.referenceScreenCount !== 0) {
  throw Error("Cross-platform navigation geometry or reference inventory drift");
}
console.log("Cross-platform Ghost FTP symbol, palette, navigation and 48dp tap targets: PASS.");
console.log("75 reference concepts remain pending real screen-by-screen pixel acceptance.");
