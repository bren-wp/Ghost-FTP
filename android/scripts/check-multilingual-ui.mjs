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
