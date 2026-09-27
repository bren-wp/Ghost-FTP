import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const read = (file) => fs.readFileSync(path.join(root, file), 'utf8');
const versionConfig = JSON.parse(read('../version.json'));
const version = versionConfig.version;
const build = versionConfig.build;

const checks = [];
const expect = (label, ok) => checks.push({ label, ok: Boolean(ok) });
const contains = (file, text) => read(file).includes(text);
const matches = (file, pattern) => pattern.test(read(file));

const nativeWorkflow = '../.github/workflows/ghostftp-build.yml';
const releaseWorkflow = '../.github/workflows/ghostftp-release.yml';

expect('package version matches central version', contains('package.json', `"version": "${version}"`));
expect('release metadata matches central version', contains('src/lib/release.ts', `PRODUCT_VERSION = "${version}"`));
expect('release build stamp matches central build', contains('src/lib/release.ts', `PRODUCT_BUILD = "${build}"`));
expect('Linux Tauri package version matches central version', contains('src-tauri/tauri.conf.json', `"version": "${version}"`));
expect('Linux Rust package version matches central version', contains('src-tauri/Cargo.toml', `version = "${version}"`));
expect('fallback runtime helper matches central version', contains('../tools/ghostftp-runtime/main.go', `const version = "${version}"`));
expect('Linux latest update service template matches central version', contains('../updates/latest.template.json', `"version": "${version}"`));
expect(
  'Linux latest update service points at the canonical versioned AppImage',
  contains('../updates/latest.template.json', `GhostFTP-Linux-x86_64-v${version}.AppImage`)
);
expect('native build includes AppImage', matches(nativeWorkflow, /--bundles deb,rpm,appimage/));
expect('native build uploads Linux preview artifact', contains(nativeWorkflow, 'GhostFTP-Linux-x86_64-Preview'));
expect('canonical release normalizes Linux binary', contains(releaseWorkflow, 'GhostFTP-Linux-x86_64-v$VERSION'));
expect('canonical release normalizes Linux AppImage', contains(releaseWorkflow, 'GhostFTP-Linux-x86_64-v$VERSION.AppImage'));
expect('canonical release normalizes Linux deb', contains(releaseWorkflow, 'GhostFTP-Linux-amd64-v$VERSION.deb'));
expect('canonical release normalizes Linux rpm', contains(releaseWorkflow, 'GhostFTP-Linux-x86_64-v$VERSION.rpm'));
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

console.log(`Linux parity contract OK for Ghost FTP ${version}`);
