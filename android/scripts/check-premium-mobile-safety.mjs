import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const source=readFileSync("android/app/src/main/java/com/ghostftp/android/MainActivity.kt","utf8");
const style=readFileSync("android/app/src/main/java/com/ghostftp/android/PremiumReferenceStyle.kt","utf8");
const instrument=readFileSync("android/app/src/androidTest/java/com/ghostftp/android/PremiumVisualReferenceTest.kt","utf8");

for (const text of [
  "runCatching { (localBrowseDirectory ?: base).canonicalFile }",
  "Local storage is temporarily unavailable",
  "base == null || chosen == null",
  "setCompoundDrawables(icon, null, null, null)",
  "foreground = android.graphics.drawable.RippleDrawable(",
]) assert.ok(source.includes(text), "Android mobile filesystem/UI guard missing: "+text);

const navStart = source.indexOf("private fun workspaceNavItem(");
const navEnd = source.indexOf("private fun styleWorkspaceNavItem(", navStart);
assert.ok(navStart >= 0 && navEnd > navStart, "Real native Android navigation block required");
const navigation = source.slice(navStart, navEnd);
assert.ok(navigation.includes("setCompoundDrawables(icon, null, null, null)"),
  "Navigation icons must appear on the leading edge");
assert.ok(!navigation.includes("setCompoundDrawables(null, icon, null, null)"),
  "Tablet navigation cannot place icons on the trailing edge");
assert.ok(source.includes("!chosen.path.startsWith(base.path + File.separator)"),
  "Local upload must stay inside app-owned storage");
assert.ok(style.includes("val purple = Color.rgb(101, 75, 255)"),
  "Approved dark premium reference accent must remain stable");
assert.ok(instrument.includes("captureNineReferenceScreensAt480x960"),
  "Real-view reference capture must be kept for screenshot comparison");
// A canceled navigation prompt must never mutate activeWorkspace or hide the
// unsaved editor before the confirmation callback is explicitly accepted.
const workspaceStart = source.indexOf("private fun setWorkspace(");
const workspaceEnd = source.indexOf("private fun ", workspaceStart + 24);
assert.ok(workspaceStart >= 0 && workspaceEnd > workspaceStart,
  "Full-workspace navigation method must remain available");
const workspace = source.slice(workspaceStart, workspaceEnd);
const editorPromptAt = workspace.indexOf("EditorNavigationGuard.needsDiscardConfirmation(");
const changeAt = workspace.indexOf("activeWorkspace = workspace");
assert.ok(editorPromptAt >= 0 && changeAt > editorPromptAt,
  "Unsaved text must be confirmed before Android workspace state changes");
assert.ok(workspace.includes("confirmCloseRemoteEditor { setWorkspace(workspace, announce) }"),
  "Confirmed navigation must restore the originally requested destination");
console.log("Android navigation preserves unsaved editor draft until user consent.");
console.log("Android premium nav, guarded private file browser and real-view reference checks passed.");
