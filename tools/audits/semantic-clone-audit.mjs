#!/usr/bin/env node
// Advisory structural clone inventory. Never delete or rewrite code from this report.
import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";
import { walk } from "./full-dead-code-audit.mjs";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const WORD = /^[A-Za-z_$][\w$]*$/;
const KEYWORDS = new Set("if else for while do switch case default match loop break continue return throw try catch finally await async yield let const var mut fn func function def class struct enum trait impl interface type pub public private protected static export import from use mod crate self Self super this true false null nil None Some Ok Err in is as where unsafe move ref new delete sizeof typeof instanceof with except raise pass and or not lambda defer go select range package".split(" "));

// Lightweight lexical normalization, not an AST or proof of equivalence.
// Keep literal values, control words, calls and member names. Other identifiers
// may differ in renamed clones. Templates/raw strings are treated as opaque.
export function tokenize(source, language) {
  const pattern = /\/\/[^\n]*|\/\*[\s\S]*?\*\/|(?:br|r)#+\"[\s\S]*?\"#+|(?:\"(?:\\[\s\S]|[^\"\\])*\"|'(?:\\[\s\S]|[^'\\\n])*'|\x60(?:\\[\s\S]|[^\x60\\])*\x60)|#[^\n]*|[A-Za-z_$][\w$]*|\d+(?:\.\d+)?|::|=>|->|==|!=|<=|>=|&&|\|\||\?\?|\?\.|\+\+|--|[^\s]/g;
  const tokens = [];
  let line = 1, previous = 0;
  for (const match of source.matchAll(pattern)) {
    const raw = match[0];
    line += (source.slice(previous, match.index).match(/\n/g) || []).length;
    const startLine = line;
    line += (raw.match(/\n/g) || []).length;
    previous = match.index + raw.length;
    if (raw.startsWith("//") || raw.startsWith("/*")) continue;
    if (raw.startsWith("#") && ["py", "php"].includes(language)) continue;
    // Keep Rust attributes and other # constructs as opaque tokens rather
    // than pretending their contents have the same behavior.
    tokens.push({ raw, line: startLine });
  }
  return tokens.map((token, index) => {
    const raw = token.raw;
    const before = tokens[index - 1]?.raw;
    const after = tokens[index + 1]?.raw;
    const member = [".", "?.", "::", "->"].includes(before);
    const normalized = WORD.test(raw) && !KEYWORDS.has(raw) && !member && after !== "(" && after !== "::"
      ? "@id" : raw;
    return { value: normalized, line: token.line };
  });
}

function tokenHash(value) {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i++) hash = Math.imul(hash ^ value.charCodeAt(i), 16777619);
  return hash >>> 0;
}

// Winnowing chooses local minima from rolling hashes, so an inserted prefix
// does not change all sample positions. Confirm matches with SHA-256 over
// the complete normalized window; rolling-hash collisions are not evidence.
function anchors(tokens, width, span) {
  if (tokens.length < width) return [];
  const prime = 16777619;
  let power = 1, hash = 0;
  const values = tokens.map(token => tokenHash(token.value));
  for (let j = 0; j < width; j++) {
    if (j) power = Math.imul(power, prime);
    hash = (Math.imul(hash, prime) + values[j]) >>> 0;
  }
  const hashes = [hash];
  for (let j = width; j < values.length; j++) {
    hash = (Math.imul((hash - Math.imul(values[j - width], power)) | 0, prime) + values[j]) >>> 0;
    hashes.push(hash);
  }
  const result = [];
  let last = -1;
  const count = Math.max(1, hashes.length - span + 1);
  for (let start = 0; start < count; start++) {
    let best = start;
    for (let j = start + 1; j < Math.min(start + span, hashes.length); j++) {
      if (hashes[j] <= hashes[best]) best = j;
    }
    if (best !== last) { result.push(best); last = best; }
  }
  return result;
}

export function scanEntries(entries, { width = 80, span = 12, maxGroups = 500 } = {}) {
  const groups = new Map();
  let tokenCount = 0, anchorCount = 0;
  for (const entry of entries) {
    const tokens = tokenize(entry.source, entry.language);
    tokenCount += tokens.length;
    for (const start of anchors(tokens, width, span)) {
      const window = tokens.slice(start, start + width);
      const endLine = window.at(-1).line;
      if (endLine - window[0].line < 3) continue;
      const key = entry.language + ":" + createHash("sha256")
        .update(JSON.stringify(window.map(token => token.value))).digest("hex");
      anchorCount++;
      const location = { file: entry.file, startLine: window[0].line, endLine };
      let group = groups.get(key);
      if (!group) { group = { count: 0, locations: [] }; groups.set(key, group); }
      // Collapse overlapping matches in the same file, and bound location
      // storage for very common patterns. Reports do not expose source text.
      if (group.locations.some(old => old.file === location.file &&
        old.startLine <= location.endLine && location.startLine <= old.endLine)) continue;
      group.count++;
      if (group.locations.length < 24) group.locations.push(location);
    }
  }
  const candidates = [...groups.entries()].filter(([, group]) => group.count > 1)
    .sort(([a, x], [b, y]) => y.count - x.count || a.localeCompare(b))
    .map(([fingerprint, group]) => ({ fingerprint, occurrences: group.count, locations: group.locations }));
  return {
    method: "Advisory normalized token-window clone inventory (Type 1/2 candidates)",
    files: entries.length, tokenCount, anchorCount, width, span,
    candidateGroups: candidates.length, reportLimit: maxGroups,
    candidates: candidates.slice(0, maxGroups),
    limitations: [
      "Lexical heuristic; not semantic equivalence, AST, reachability or proof that code is safe to remove.",
      "Short clones, modified control flow, templates, nested comments, language-specific syntax and cross-language clones may be missed.",
      "Common scaffolding can produce false positives; overlapping windows may describe the same clone.",
      "Review security checks, OS-specific behavior, error handling and runtime tests before consolidation.",
    ],
  };
}

export function audit(directory = ROOT) {
  const languageOf = file => {
    const ext = path.extname(file);
    return ({ ".rs": "rs", ".ts": "ts", ".tsx": "ts", ".js": "js", ".mjs": "js", ".go": "go", ".php": "php", ".py": "py" })[ext];
  };
  const skipped = [];
  const entries = [];
  for (const file of walk(directory).sort()) {
    const relative = path.relative(directory, file).replaceAll("\\", "/");
    const language = languageOf(relative);
    if (!language || /(?:^|\/)(?:vendor|gen|generated|payload|fixtures|node_modules|\.venv)(?:\/|$)/.test(relative)) continue;
    if (fs.statSync(file).size > 4 * 1024 * 1024) { skipped.push(relative); continue; }
    entries.push({ file: relative, language, source: fs.readFileSync(file, "utf8") });
  }
  const result = scanEntries(entries);
  return { ...result, skippedOversizeFiles: skipped };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const report = audit();
  if (!report.files) throw new Error("No source files scanned");
  console.log(JSON.stringify(report, null, 2));
}
