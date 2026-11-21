#!/usr/bin/env node
// Ghost FTP by Brendigo
// This contract checks a non-sensitive, user-invoked support action.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
const file = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../src/components/AboutDialog.tsx");
const source = fs.readFileSync(file, "utf8");
for (const required of [
  "Copy version details",
  "aria-label=\"Copy Ghost FTP version and build details\"",
  "navigator.clipboard.writeText(supportDetails)",
  "setCopyFeedback(\"Version details copied\")",
  'role="status" aria-live="polite"',
  "PRODUCT_BUILD", "PRODUCT_VERSION_DISPLAY", "PRODUCT_RELEASE_DATE",
]) {
  if (!source.includes(required)) throw new Error("Missing About support-info contract: " + required);
}
const begin = source.indexOf("const supportDetails = [");
const end = source.indexOf("].join(\"\\n\");", begin);
if (begin < 0 || end < begin) throw new Error("Missing bounded support details");
const payload = source.slice(begin, end);
if (/password|passphrase|token|privatekey|credential|profile\.host|connection/i.test(payload)) {
  throw new Error("Support clipboard must never include credentials, private paths or host identities");
}
console.log("About support version-copy UI and sensitive-data boundary verified.");
