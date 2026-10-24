// Regression-only inventory. A parity check is not linguistic certification.
// Budgets are the unresolved legacy sources; never increase them or claim
// that fallback English means a language has been completely translated.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read = (file) => fs.readFileSync(path.join(root, file), "utf8");
const android = read("android/app/src/main/java/com/ghostftp/android/MainActivity.kt");
const uiLiteral = /\b(?:text\s*=|contentDescription\s*=|showMessage\(|formLabel\(|sectionTitle\(|sectionDescription\(|toolbarButton\(|primaryButton\(|secondaryButton\()\s*"((?:[^"\\]|\\.)*)"/g;
const remainingAndroid = [...android.matchAll(uiLiteral)]
  .map((match) => match[1]).filter((value) => /[a-z]/i.test(value) && !value.includes("$"));
const views = [
  "macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift",
  "macos/Sources/GhostFTPMacApp/Views/ContentView.swift",
  "macos/Sources/GhostFTPMacApp/Views/CloudWorkspace.swift",
];
const native = views.map(read).join("\n");
const macDirect = [...native.matchAll(/\b(?:Text|Label|Button|Section|TextField|SecureField|Toggle|Picker)\(\s*"((?:[^"\\]|\\.)*)"/g)]
  .filter((match) => !match[1].includes("\\(") && /[a-z]/i.test(match[1]));
const catalog = JSON.parse(read("macos/Sources/GhostFTPMacApp/Resources/ghostftp-translations.json"));
const referenced = [...new Set([...native.matchAll(/L10n\.text\("((?:[^"\\]|\\.)*)"\)/g)].map(x=>x[1]))];
const nonEnglish = catalog.languages.filter(x => x !== "en");
const missing = referenced.filter(key => nonEnglish.some(lang => !catalog.strings[lang]?.[key]));
const counts = {
  android_inline_ui_literals: remainingAndroid.length,
  macos_inline_view_literals: macDirect.length,
  macos_localized_view_keys: referenced.length,
  macos_missing_locale_keys: missing.length,
};
if (counts.android_inline_ui_literals > 123 || counts.macos_inline_view_literals > 0 ||
    counts.macos_missing_locale_keys > 67) {
  console.error("Localization regression:", JSON.stringify(counts));
  console.error("Missing macOS keys:", JSON.stringify(missing));
  process.exit(1);
}
console.log("Ghost FTP localization debt inventory:", JSON.stringify(counts));
console.log("Untranslated user-visible text remains open, not certified as 100% localized.");
