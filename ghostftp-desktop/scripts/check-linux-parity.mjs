import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const read = (file) => fs.readFileSync(path.join(root, file), 'utf8');

const checks = [];
const expect = (label, ok) => checks.push({ label, ok: Boolean(ok) });
const contains = (file, text) => read(file).includes(text);
const matches = (file, pattern) => pattern.test(read(file));

expect('package version is RC22', contains('package.json', '"version": "2.1.1-rc.22"'));
expect('release metadata is RC22', contains('src/lib/release.ts', 'PRODUCT_VERSION = "2.1.1-rc.22"'));
expect('release build stamp is RC22', contains('src/lib/release.ts', 'PRODUCT_BUILD = "2026.09.24.22"'));
expect('Linux Tauri package version is RC22', contains('src-tauri/tauri.conf.json', '"version": "2.1.1-rc.22"'));
expect('Linux Rust package version is RC22', contains('src-tauri/Cargo.toml', 'version = "2.1.1-rc.22"'));
expect('Linux runtime helper remains versioned', matches('../tools/ghostftp-runtime/main.go', /const version = "2\.1\.1-rc\.(21|22)"/));
expect('Linux update channel is RC22', contains('../updates/channels/preview.template.json', '"version": "2.1.1-rc.22"'));
expect('Linux latest update manifest is RC22', contains('../updates/latest.template.json', '"version": "2.1.1-rc.22"'));
expect('Linux latest update build is RC22', contains('../updates/latest.template.json', '"build": "2026.09.24.22"'));
expect('native build includes AppImage', matches('../.github/workflows/ghostftp-build.yml', /--bundles deb,rpm,appimage/));
expect('native build uploads Linux desktop artifact', matches('../.github/workflows/ghostftp-build.yml', /GhostFTP-Linux-x86_64-RC(21|22)/));
expect('release normalizes Linux binary', contains('../.github/workflows/ghostftp-rc22-release.yml', 'GhostFTP-Linux-x86_64-v2.1.1-RC22'));
expect('release normalizes Linux AppImage', contains('../.github/workflows/ghostftp-rc22-release.yml', 'GhostFTP-Linux-x86_64-v2.1.1-RC22.AppImage'));
expect('release normalizes Linux deb', contains('../.github/workflows/ghostftp-rc22-release.yml', 'GhostFTP-Linux-amd64-v2.1.1-RC22.deb'));
expect('release normalizes Linux rpm', contains('../.github/workflows/ghostftp-rc22-release.yml', 'GhostFTP-Linux-x86_64-v2.1.1-RC22.rpm'));
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