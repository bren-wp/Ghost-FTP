import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const read = (file) => fs.readFileSync(path.join(root, file), 'utf8');

const checks = [];
const expect = (label, ok) => checks.push({ label, ok: Boolean(ok) });
const contains = (file, text) => read(file).includes(text);
const matches = (file, pattern) => pattern.test(read(file));

const rc23NativeWorkflow = '../.github/workflows/ghostftp-native-rc23.yml';

expect('package version is RC23', contains('package.json', '"version": "2.1.1-rc.23"'));
expect('release metadata is RC23', contains('src/lib/release.ts', 'PRODUCT_VERSION = "2.1.1-rc.23"'));
expect('release build stamp is RC23', contains('src/lib/release.ts', 'PRODUCT_BUILD = "2026.09.25.23"'));
expect('Linux Tauri package version is RC23', contains('src-tauri/tauri.conf.json', '"version": "2.1.1-rc.23"'));
expect('Linux Rust package version is RC23', contains('src-tauri/Cargo.toml', 'version = "2.1.1-rc.23"'));
expect('Linux runtime helper remains versioned', matches('../tools/ghostftp-runtime/main.go', /const version = "2\.1\.1-rc\.(22|23)"/));
expect('Linux update channel is RC23', contains('../updates/channels/preview.template.json', '"version": "2.1.1-rc.23"'));
expect('Linux latest update manifest is RC23', contains('../updates/latest.template.json', '"version": "2.1.1-rc.23"'));
expect('Linux latest update build is RC23', contains('../updates/latest.template.json', '"build": "2026.09.25.23"'));
expect('RC23 native build includes AppImage', matches(rc23NativeWorkflow, /--bundles deb,rpm,appimage/));
expect('RC23 native build uploads Linux desktop artifact', contains(rc23NativeWorkflow, 'GhostFTP-Linux-x86_64-RC23'));
expect('RC23 release normalizes Linux binary', contains('../.github/workflows/ghostftp-rc23-release.yml', 'GhostFTP-Linux-x86_64-v2.1.1-RC23'));
expect('RC23 release normalizes Linux AppImage', contains('../.github/workflows/ghostftp-rc23-release.yml', 'GhostFTP-Linux-x86_64-v2.1.1-RC23.AppImage'));
expect('RC23 release normalizes Linux deb', contains('../.github/workflows/ghostftp-rc23-release.yml', 'GhostFTP-Linux-amd64-v2.1.1-RC23.deb'));
expect('RC23 release normalizes Linux rpm', contains('../.github/workflows/ghostftp-rc23-release.yml', 'GhostFTP-Linux-x86_64-v2.1.1-RC23.rpm'));
expect('single native entrypoint is shared across platforms', contains('src-tauri/src/lib.rs', 'tauri::WebviewWindowBuilder::new(app, "main", tauri::WebviewUrl::default())'));
expect('single main window contract remains shared', contains('src-tauri/src/lib.rs', '.inner_size(1290.0, 852.0)'));
expect('minimum size contract remains shared', contains('src-tauri/src/lib.rs', '.min_inner_size(480.0, 600.0)'));
expect('window decoration contract remains shared', contains('src-tauri/src/lib.rs', '.decorations(false)'));

const failures = checks.filter((check) => !check.ok);
if (failures.length) {
  console.error('Linux parity contract failed:');
  for (const failure of failures) console.error(`- ${failure.label}`);
  process.exit(1);
}

console.log('Linux parity contract OK');
