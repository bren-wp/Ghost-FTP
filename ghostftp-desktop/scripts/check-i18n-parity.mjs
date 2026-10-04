import { readFileSync } from "node:fs";

const source = readFileSync(new URL("../src/lib/i18n.ts", import.meta.url), "utf8");
const completionSource = readFileSync(
  new URL("../src/lib/i18n-reference-completion.ts", import.meta.url),
  "utf8"
);

const localeRegistry = source.match(/export const APP_LOCALES\s*=\s*\[([^\]]+)\]\s*as const;/s);
if (!localeRegistry) throw new Error("Unable to locate APP_LOCALES in src/lib/i18n.ts");

const locales = [...localeRegistry[1].matchAll(/"([^"]+)"/g)].map((m) => m[1]);
const expectedLocales = locales;
if (new Set(locales).size !== locales.length || locales.length < 2 || locales[0] !== "en") {
  throw new Error(`Invalid advertised locale registry: ${locales.join(", ")}`);
}
if (locales.includes("sq")) {
  throw new Error("Albanian must not be advertised by Ghost FTP");
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

const referenceMatch = source.match(
  /const REFERENCE_UI_TRANSLATIONS:[^{]+\{([\s\S]*?)^\};/m
);
if (!referenceMatch) throw new Error("Unable to locate reference UI translations");
const referenceKeys = extractKeys(referenceMatch[1]);
if (referenceKeys.length === 0) throw new Error("Reference UI translation set is empty");

const completionLocales = ["cs", "sk", "hu", "ro", "bg", "el", "tr", "uk", "da", "sv", "no"];
for (const locale of completionLocales) {
  if (!expectedLocales.includes(locale)) {
    throw new Error(`Completion dictionary exists for unadvertised locale: ${locale}`);
  }
  const match = completionSource.match(new RegExp(
    `^  ${locale}: \\{([\\s\\S]*?)^  \\},?`,
    "m"
  ));
  if (!match) throw new Error(`Missing reference completion dictionary for ${locale}`);
  const keys = extractKeys(match[1]);
  const missing = referenceKeys.filter((key) => !keys.includes(key));
  const extraKeys = keys.filter((key) => !referenceKeys.includes(key));
  const duplicates = keys.filter((key, index) => keys.indexOf(key) !== index);
  if (missing.length || extraKeys.length || duplicates.length || keys.length !== referenceKeys.length) {
    throw new Error(
      `${locale} reference UI mismatch: ` +
      [
        missing.length ? `missing=${missing.join(" | ")}` : "",
        extraKeys.length ? `extra=${extraKeys.join(" | ")}` : "",
        duplicates.length ? `duplicates=${[...new Set(duplicates)].join(" | ")}` : "",
        `count=${keys.length}/${referenceKeys.length}`,
      ].filter(Boolean).join("; ")
    );
  }
}

console.log(
  `Ghost FTP locale parity OK: ${canonical.length} core keys and ${referenceKeys.length} reference UI keys across ${expectedLocales.length} advertised locales.`
);
