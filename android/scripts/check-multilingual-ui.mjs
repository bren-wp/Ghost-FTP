import fs from "node:fs";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..", "app", "src", "main", "res");
const locales = ["en","hr","cs","sk","hu","ro","bg","el","tr","uk","da","sv","no","de","fr","es","it","pt","nl","pl","sl","sr","bs","mk"];
const source = fs.readFileSync(path.join(root, "values", "strings_ui_extra.xml"), "utf8");
const extract = (xml) => [...xml.matchAll(/<string name="([^"]+)">([^<]*)<\/string>/g)].map(x => x[1]);
const expected = extract(source);
if (expected.length !== 10 || new Set(expected).size !== expected.length) {
  throw Error("Incorrect Android multilingual UI key catalog");
}
for (const locale of locales) {
  const folder = locale === "en" ? "values" : `values-${locale}`;
  const xml = fs.readFileSync(path.join(root, folder, "strings_ui_extra.xml"), "utf8");
  const keys = extract(xml);
  if (keys.length !== expected.length || keys.some((key, i) => key !== expected[i])) {
    throw Error(`Android language parity mismatch for ${locale}`);
  }
  if (xml.includes("TODO") || xml.includes("FIXME")) {
    throw Error(`Unfinished Android translation in ${locale}`);
  }
}
const activity = fs.readFileSync(path.resolve(import.meta.dirname, "..", "app", "src", "main", "java", "com", "ghostftp", "android", "MainActivity.kt"), "utf8");
for (const key of expected) {
  if (!activity.includes(`getString(R.string.${key})`)) {
    throw Error(`Android text not migrated to resources: ${key}`);
  }
}
console.log(`Android UI localization parity OK: ${expected.length} additional messages × ${locales.length} locales`);

// Accessibility copy is a separate resource catalog; every advertised
// locale must ship exactly the same 15 keys and nonempty translations.
const accessibilityPath = (locale) => path.join(root, locale === "en" ? "values" : `values-${locale}`, "strings_ui_accessibility.xml");
const accessibilitySource = fs.readFileSync(accessibilityPath("en"), "utf8");
const accessibilityExpected = extract(accessibilitySource);
if (accessibilityExpected.length !== 15 || new Set(accessibilityExpected).size !== 15) {
  throw Error("Incorrect Android accessibility translation catalog");
}
for (const locale of locales) {
  const xml = fs.readFileSync(accessibilityPath(locale), "utf8");
  const entries = [...xml.matchAll(/<string name="([^"]+)">([^<]*)<\/string>/g)];
  const keys = entries.map(x => x[1]);
  if (keys.length !== accessibilityExpected.length || keys.some((x, i) => x !== accessibilityExpected[i])) {
    throw Error(`Android accessibility key parity mismatch for ${locale}`);
  }
  if (entries.some(x => !x[2].trim() || x[2].includes("TODO") || x[2].includes("FIXME"))) {
    throw Error(`Missing or unfinished Android accessibility translation: ${locale}`);
  }
}
for (const key of accessibilityExpected) {
  if (!activity.includes(`getString(R.string.${key})`)) throw Error(`Android accessibility text not migrated: ${key}`);
}
console.log(`Android accessibility UI parity OK: ${accessibilityExpected.length} × ${locales.length} locales`);

// Splash, touch-target and protocol accessibility copy must be localized in
// every advertised language; require exact key order and placeholder parity.
const brandKeys = [
  "ui_header_back", "ui_header_add", "ui_splash_tagline",
  "ui_splash_opening", "ui_splash_initializing", "ui_protocol_server"
];
const brandPath = (locale) => path.join(root,
  locale === "en" ? "values" : `values-${locale}`, "strings_ui_brand.xml");
for (const locale of locales) {
  const xml = fs.readFileSync(brandPath(locale), "utf8");
  const entries = [...xml.matchAll(/<string name="([^"]+)">([^<]*)<\/string>/g)];
  if (entries.length !== brandKeys.length ||
      entries.some((entry, index) => entry[1] !== brandKeys[index] || !entry[2].trim())) {
    throw Error(`Ghost FTP branding/controls localization mismatch in ${locale}`);
  }
  if (entries.some(x => /TODO|FIXME/.test(x[2]))) {
    throw Error(`Incomplete branded native UI localization in ${locale}`);
  }
  for (const [index, entry] of entries.entries()) {
    const placeholders = [...entry[2].matchAll(/%\d+\$[sd]/g)].map(x => x[0]);
    const expectedParams = brandKeys[index] === "ui_protocol_server" ? ["%1$s"] : [];
    if (JSON.stringify(placeholders) !== JSON.stringify(expectedParams)) {
      throw Error(`Unsafe translation format placeholders: ${locale}/${brandKeys[index]}`);
    }
  }
}
const nativeStyle = fs.readFileSync(path.resolve(import.meta.dirname,
  "..", "app", "src", "main", "java", "com", "ghostftp", "android",
  "PremiumReferenceStyle.kt"), "utf8");
const nativeSplash = fs.readFileSync(path.resolve(import.meta.dirname,
  "..", "app", "src", "main", "java", "com", "ghostftp", "android",
  "SplashActivity.kt"), "utf8");
for (const key of brandKeys) {
  const file = key.startsWith("ui_splash_") ? nativeSplash : nativeStyle;
  if (!file.includes(`R.string.${key}`)) {
    throw Error(`Branded native UI still has an untranslated label: ${key}`);
  }
}
console.log(`Native Ghost FTP brand/splash accessibility parity OK: ${brandKeys.length} × ${locales.length} locales`);

// Local file search progress, completion and fail-closed states must remain
// translated in every advertised locale, with safe integer placeholders.
const searchKeys = [
  "ui_local_search_progress", "ui_local_search_unavailable",
  "ui_local_search_completed", "ui_local_search_limited"
];
for (const locale of locales) {
  const folder = locale === "en" ? "values" : `values-${locale}`;
  const xml = fs.readFileSync(path.join(root, folder, "strings_ui_search.xml"), "utf8");
  const items = [...xml.matchAll(/<string name="([^"]+)">([^<]*)<\/string>/g)];
  if (items.length !== searchKeys.length || items.some((item, i) =>
      item[1] !== searchKeys[i] || !item[2].trim() || /TODO|FIXME/.test(item[2]))) {
    throw Error(`Incomplete local file search translation for ${locale}`);
  }
  for (let i = 0; i < items.length; i++) {
    const placeholders = [...items[i][2].matchAll(/%\d+\$[sd]/g)].map(x=>x[0]);
    const expected = i < 2 ? [] : ["%1$d"];
    if (JSON.stringify(placeholders) !== JSON.stringify(expected)) {
      throw Error(`Invalid local search placeholder at ${locale}/${searchKeys[i]}`);
    }
  }
}
for (const key of searchKeys) {
  if (!activity.includes(`R.string.${key}`)) {
    throw Error(`Android local search did not reference translated resource ${key}`);
  }
}
if (!activity.includes("LocalScopedSearch.find(") ||
    activity.includes("val entries = directory.listFiles()?.sortedBy") ||
    activity.includes('phoneMenuButton.contentDescription == "Back to Sites"')) {
  throw Error("Android local search main-thread IO or locale-sensitive toolbar regression");
}
console.log(`Android local search: ${searchKeys.length} messages × ${locales.length} locales`);
