#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (p) => fs.readFileSync(path.join(root, p), "utf8");
const tokens = JSON.parse(read("docs/branding/premium-ui-tokens.json"));
const screenContract = JSON.parse(read("docs/qa/premium-75-screen-contract.json"));
const css = read("ghostftp-desktop/src/styles.css");
const android = read("android/app/src/main/java/com/ghostftp/android/MainActivity.kt");
const responsive = read("android/app/src/main/java/com/ghostftp/android/AndroidResponsiveLayout.kt");
const mac = read("macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift");
const macCloud = read("macos/Sources/GhostFTPMacApp/Views/CloudWorkspace.swift");
const errors = [];

function requireSource(source, needle, description) {
  if (!source.includes(needle)) errors.push(description);
}
function rgb(hex) {
  const value = hex.replace("#", "");
  return [0, 2, 4].map((i) => Number.parseInt(value.slice(i, i + 2), 16));
}
const colorAssertions = [
  ["background", "--bg", "background"],
  ["panel", "--bg-panel", "panel"],
  ["raised", "--bg-subtle", "panelStrong"],
  ["accent", "--accent", "accent"],
  ["success", "--success", "success"],
  ["danger", "--danger", "danger"],
];
for (const [semantic, cssName, androidName] of colorAssertions) {
  const [r, g, b] = rgb(tokens.colors[semantic]);
  requireSource(css, `${cssName}: ${r} ${g} ${b};`, `desktop ${semantic} token drift`);
  if (semantic === "accent") {
    // The October 16 mobile reference intentionally uses violet while the
    // Windows/Linux/macOS brand palette remains the original blue. Check the
    // Android token and actual runtime value, not a stale shared RGB literal.
    requireSource(android, "val accent: Int = PremiumReferenceStyle.purple", "Android reference accent token drift");
  } else {
    requireSource(android, `val ${androidName}: Int = Color.rgb(${r}, ${g}, ${b})`, `Android ${semantic} token drift`);
  }
  const macName = semantic === "panel" ? "sidebar" : semantic === "raised" ? "raised" : semantic;
  requireSource(mac, `static let ${macName} = Color(red: ${r}/255, green: ${g}/255, blue: ${b}/255)`, `macOS ${semantic} token drift`);
}
for (const semantic of ["navy", "slate", "cyan"]) {
  requireSource(css, `--ghost-premium-${semantic}: ${tokens.colors[semantic]};`, `desktop ${semantic} token drift`);
}
requireSource(android, "val accentSurface: Int = PremiumReferenceStyle.selected", "Android active surface differs from user-approved violet selection token");
const androidReference = read("android/app/src/main/java/com/ghostftp/android/PremiumReferenceStyle.kt");
requireSource(androidReference, "val purple = Color.rgb(101, 75, 255)", "actual Android violet accent is not #654BFF");
requireSource(androidReference, "val selected = Color.rgb(32, 40, 88)", "Android reference selected-row color drift");
if (tokens.android?.accent !== "#654BFF") errors.push("Android mobile-only reference accent changed");
requireSource(mac, "static let slate = Color(red: 19/255, green: 45/255, blue: 82/255)", "macOS active surface differs from slate token");
requireSource(css, "width: 214px !important", "214px desktop sidebar reference not enforced");
requireSource(css, "min-height: 43px", "43px desktop navigation reference not enforced");
// A fixed 280dp drawer clips screens as narrow as 240–320dp; keep the
// approved maximum, but enforce the real bounded responsive implementation.
requireSource(android, "AndroidResponsiveLayout.drawerWidthDp(resources.configuration.screenWidthDp)",
  "reference mobile drawer must use the width-safe policy");
requireSource(responsive, "(widthDp - 56).coerceIn(192, 280)",
  "drawer must leave room to dismiss on narrow phones and cap at 280dp");
requireSource(responsive, ".coerceAtMost(widthDp.coerceAtLeast(1))",
  "drawer must never extend beyond device width");
requireSource(android, "private fun buildPremiumPhoneChrome(): View", "full-width reference identity bar absent on Android");
requireSource(android, "ViewGroup.LayoutParams.MATCH_PARENT, dp(52)", "52dp mobile reference top chrome geometry missing");
requireSource(android, "resources.configuration.screenWidthDp < 600", "full-width phone/overlay breakpoint missing");
requireSource(android, "val iconRail = false", "mobile reference drawer labels hidden by legacy compact rail");
requireSource(android, "icon?.setBounds(0, 0, dp(20), dp(20))", "Android 20dp reference icon dimensions not implemented");
requireSource(android, "setCompoundDrawables(null, icon, null, null)", "Android interactive vector navigation icons missing");
for (const name of ["files","sites","transfers","sync","cloud","settings","about"]) {
  const rel = `android/app/src/main/res/drawable/ic_premium_nav_${name}.xml`;
  const icon = read(rel);
  if (!icon.includes('android:viewportWidth="24"') ||
      !icon.includes('android:viewportHeight="24"') ||
      !icon.includes('android:strokeColor="#38ABFF"') ||
      !icon.includes("android:pathData=")) {
    errors.push(`invalid branded icon geometry or tint: ${rel}`);
  }
  requireSource(android, `R.drawable.ic_premium_nav_${name}`, `Android ${name} workspace does not use approved navigation icon`);
}
requireSource(mac, ".listRowBackground(", "macOS workspace selection brand surface absent");
requireSource(macCloud, 'Button(L10n.text("Cancel transfer"))', "macOS cloud cancellation control missing");
requireSource(macCloud, ".buttonStyle(.borderedProminent)", "macOS premium cloud primary actions absent");
for (const [platform, count] of Object.entries(tokens.referenceCounts)) {
  if (screenContract.platforms?.[platform]?.length !== count) {
    errors.push(`${platform} reference registry mismatch`);
  }
  if (screenContract.platforms?.[platform]?.some((entry) => entry.visual_acceptance !== "pending")) {
    errors.push(`${platform}: no 1:1 pixel acceptance can be claimed without captures`);
  }
}
if (tokens.macos.referenceScreenCount !== 0 || screenContract.macos?.referenceCount !== 0) {
  errors.push("a macOS reference screenshot must not be invented");
}
if (errors.length) {
  console.error("Premium cross-platform design contract failed:");
  for (const error of errors) console.error(" - " + error);
  process.exit(1);
}
console.log("Premium design tokens, Android navigation assets and macOS controls verified; screenshot pixel parity remains pending.");
