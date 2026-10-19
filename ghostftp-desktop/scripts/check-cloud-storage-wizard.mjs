#!/usr/bin/env node
/**
 * Exercise the actual CloudStorageConnect profile-builder logic instead of
 * duplicating it in the test. No network or OS credentials are accessed.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import assert from "node:assert/strict";
import ts from "typescript";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const src = fs.readFileSync(path.join(root, "src/components/CloudStorageConnect.tsx"), "utf8");
const compiled = ts.transpileModule(src, {
  fileName: "CloudStorageConnect.tsx",
  compilerOptions: {
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.ES2022,
    jsx: ts.JsxEmit.ReactJSX,
    esModuleInterop: true,
  },
});
assert.equal(compiled.diagnostics?.length ?? 0, 0, "TypeScript transpilation diagnostics");
const runtime = { exports: {} };
// Component dependency modules are not invoked: only the actual pure builder
// is exercised. The normal CI TypeScript build checks their types/imports.
new Function("require", "exports", "module", compiled.outputText)(
  () => ({}), runtime.exports, runtime,
);
const { buildStorageProfile } = runtime.exports;
assert.equal(typeof buildStorageProfile, "function");

const args = (provider, extra = {}) => ({
  provider, id: "test-connection-id", name: "", bucket: "my-bucket", region: "us-east-1",
  endpoint: "", username: "ACCESS_KEY_ID", secret: "TEST_SECRET_NOT_REAL", keyPath: "",
  ...extra,
});
const s3 = buildStorageProfile(args("s3"));
assert.equal(s3.protocol, "s3");
assert.equal(s3.bucket, "my-bucket");
assert.equal(s3.endpoint, undefined);
assert.deepEqual(s3.auth, { kind: "password", password: "TEST_SECRET_NOT_REAL" });
assert.equal(s3.group, "Cloud");

const b2 = buildStorageProfile(args("b2", { region: "us-west-004" }));
assert.equal(b2.protocol, "s3");
assert.equal(b2.endpoint, "https://s3.us-west-004.backblazeb2.com");
assert.equal(b2.host, "s3.us-west-004.backblazeb2.com");
assert.throws(() => buildStorageProfile(args("b2", { region: "us-west-004@evil.invalid" })));

const r2 = buildStorageProfile(args("r2", {
  endpoint: "https://example-account.r2.cloudflarestorage.com", region: "auto",
}));
assert.equal(r2.region, "auto");
assert.equal(r2.protocol, "s3");

const gcs = buildStorageProfile(args("gcs", {
  keyPath: "/protected/service-account.json", secret: "",
}));
assert.equal(gcs.protocol, "gcs");
assert.deepEqual(gcs.auth, { kind: "key", path: "/protected/service-account.json" });
assert.equal(gcs.host, "storage.googleapis.com");
assert.throws(() => buildStorageProfile(args("gcs", { keyPath: "" })));

const nc = buildStorageProfile(args("nextcloud", {
  endpoint: "https://cloud.example.com/nextcloud",
  username: "alice@example.com",
}));
assert.equal(nc.protocol, "webdav");
assert.equal(nc.endpoint, "https://cloud.example.com/nextcloud/remote.php/dav/files/alice%40example.com/");
assert.equal(nc.username, "alice@example.com");
assert.equal(buildStorageProfile(args("webdav", {
  endpoint: "https://cloud.example.com/dav/", username: "alice",
})).endpoint, "https://cloud.example.com/dav/");

for (const unsafe of [
  "http://example.com/dav", "https://user:password@example.com/dav",
  "https://example.com/?token=secret", "https://example.com/#secret",
]) {
  assert.throws(() => buildStorageProfile(args("webdav", { endpoint: unsafe })));
}
for (const invalid of ["", "../evil", "a", "three words", "my/bucket"]) {
  assert.throws(() => buildStorageProfile(args("s3", { bucket: invalid })));
}
assert.throws(() => buildStorageProfile(args("s3", { secret: "" })));
assert.throws(() => buildStorageProfile(args("s3", { endpoint: "http://localhost:9000" })));

assert.ok(src.includes('provider !== "mega" && provider !== "proton"'),
  "MEGA / Proton must stay informational until native adapters exist");
assert.ok(src.includes("ipc.probeCloudStorage(profile)"),
  "The Test Connection UI must use the actual native provider probe");
assert.ok(src.includes("await saveProfile(profile)"),
  "The wizard must save through the existing OS-keychain-protected profile path");
console.log("Cloud provider setup: S3/B2/R2/GCS/WebDAV/Nextcloud + unsafe endpoint guards passed");
