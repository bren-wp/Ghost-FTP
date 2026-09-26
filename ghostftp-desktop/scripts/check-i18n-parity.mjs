import { readFileSync } from "node:fs";

const source = readFileSync(new URL("../src/lib/i18n.ts", import.meta.url), "utf8");

const localeType = source.match(/export type AppLocale\s*=\s*([^;]+);/s);
if (!localeType) throw new Error("Unable to locate AppLocale union in src/lib/i18n.ts");

const locales = [...localeType[1].matchAll(/"([^"]+)"/g)].map((m) => m[1]);
const expectedLocales = ["en","hr","de","fr","es","it","pt","nl","pl","sl","sr","bs","mk"];

if (JSON.stringify(locales) !== JSON.stringify(expectedLocales)) {
  throw new Error(`Unexpected advertised locales: ${locales.join(", ")}`);
}

function extractKeys(body) {
  return [...body.matchAll(/^\s*"([^"]+)":/gm)].map((m) => m[1]);
}

const hrMatch = source.match(/const HR:[^{]+\{([\s\S]*?)^\};/m);
if (!hrMatch) throw new Error("Unable to locate Croatian canonical dictionary");

const canonical = extractKeys(hrMatch[1]);
if (canonical.length === 0) throw new Error("Croatian canonical dictionary is empty");

const duplicateCanonical = canonical.filter((key, index) => canonical.indexOf(key) !== index);
if (duplicateCanonical.length) {
  throw new Error(`Duplicate Croatian keys: ${[...new Set(duplicateCanonical)].join(", ")}`);
}

const otherStart = source.indexOf("const OTHER:");
const referenceStart = source.indexOf("const REFERENCE_UI_TRANSLATIONS");
if (otherStart < 0 || referenceStart < 0 || referenceStart <= otherStart) {
  throw new Error("Unable to isolate OTHER locale dictionaries");
}
const otherBlock = source.slice(otherStart, referenceStart);

for (const locale of expectedLocales.filter((value) => value !== "en" && value !== "hr")) {
  const match = otherBlock.match(new RegExp(
    `^\\s{2}${locale}:\\s*\\{([\\s\\S]*?)^\\s{2}\\},?`,
    "m"
  ));
  if (!match) throw new Error(`Missing dictionary for locale: ${locale}`);

  const keys = extractKeys(match[1]);
  const keySet = new Set(keys);
  const missing = canonical.filter((key) => !keySet.has(key));
  const extra = keys.filter((key) => !canonical.includes(key));
  const duplicates = keys.filter((key, index) => keys.indexOf(key) !== index);

  if (missing.length || extra.length || duplicates.length) {
    const details = [
      missing.length ? `missing=${missing.join(" | ")}` : "",
      extra.length ? `extra=${extra.join(" | ")}` : "",
      duplicates.length ? `duplicates=${[...new Set(duplicates)].join(" | ")}` : "",
    ].filter(Boolean).join("; ");
    throw new Error(`${locale} dictionary mismatch: ${details}`);
  }

  if (keys.length !== canonical.length) {
    throw new Error(`${locale} key count ${keys.length} does not match canonical ${canonical.length}`);
  }
}

console.log(`Ghost FTP locale parity OK: ${canonical.length} keys across ${expectedLocales.length} advertised locales.`);
