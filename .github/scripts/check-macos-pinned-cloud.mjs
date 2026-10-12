#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const file = (relative) => fs.readFileSync(path.join(root, relative), "utf8");
const transfer = file("macos/Sources/GhostFTPMacApp/Services/SignedCloudTransfer.swift");
const pinned = file("macos/Sources/GhostFTPMacApp/Services/PinnedCloudTransport.swift");
const tests = file("macos/Tests/GhostFTPMacAppTests/PinnedCloudTransportTests.swift");
const errors = [];
const requireToken = (value, pattern, why) => {
  if (!value.includes(pattern)) errors.push(why);
};
requireToken(transfer, "PinnedCloudTransport.transfer(", "Mac signed PUT/GET does not use public-peer transport");
if (transfer.includes("URLSession(configuration:")) errors.push("Unpinned URLSession still used for signed requests");
for (const [pattern, why] of [
  ["getaddrinfo(", "Missing DNS lookup"],
  ["guard allowed(bytes)", "Mixed private/public DNS responses not rejected"],
  ['arguments += ["--resolve"', "Curl must bind to vetted numeric peers"],
  ['"-q", "--silent"', "Curl config files must be disabled"],
  ['"--proxy", "", "--noproxy", "*"', "Proxy bypass protection missing"],
  ['"--proto", "=https"', "HTTPS-only transport policy missing"],
  ['"--max-redirs", "0"', "Redirect rejection missing"],
  ['"--config", "-"', "Signed URL must arrive through stdin config"],
  ["process.standardInput = input", "Curl stdin is not isolated"],
  ["try quoted(link.url.absoluteString)", "Signed URL not escaped as single config value"],
  ["SignedCloudTransfer.verifySHA256(", "Provider SHA-256 check lost"],
  ["manager.replaceItemAt(", "Existing local file protection lost"],
  ["withTaskCancellationHandler", "Actual live transport cancellation lost"],
]) requireToken(pinned, pattern, why);
for (const forbidden of ['"--insecure"', '"--location"', '"--location-trusted"', "ProcessInfo.processInfo.arguments", "print(link.url)", "print(config)"]) {
  if (pinned.includes(forbidden)) errors.push(`Forbidden unsafe transport option: ${forbidden}`);
}
for (const t of ["testOnlyGlobalPublicIPv4CanBecomePeerCandidates", "testOnlyReviewedGlobalUnicastIPv6CanBecomePeerCandidates", "testCurlConfigEscapesBearerURLAndRejectsNewlines", "testRejectsDuplicateProviderHeadersAndMissingResponseBlock"]) {
  requireToken(tests, t, `Missing macOS security regression: ${t}`);
}
if (errors.length) {
  for (const error of errors) console.error("macOS signed cloud: " + error);
  process.exit(1);
}
console.log("macOS signed cloud transport peer-pinning, TLS and token handling source contracts passed. Live provider E2E still required.");
