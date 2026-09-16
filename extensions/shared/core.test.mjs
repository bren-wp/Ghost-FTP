import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const source = fs.readFileSync(new URL('./core.js', import.meta.url), 'utf8');
const context = vm.createContext({ URL, URLSearchParams });
context.globalThis = context;
vm.runInContext(source, context);

const api = context.GhostFTPConnection;

const parsed = api.parseConnectionTarget('sftp://alice:secret@example.com:2222/home/alice?token=private#fragment');
assert.equal(parsed.ok, true);
assert.equal(parsed.passwordDetected, true);
assert.equal(parsed.safeTarget, 'sftp://example.com:2222/home/alice');

const launch = api.buildDesktopLaunchURL(parsed);
assert.equal(launch, 'ghostftp://connect?protocol=sftp&host=example.com&port=2222&username=alice&path=%2Fhome%2Falice');
assert.equal(launch.includes('secret'), false);
assert.equal(launch.includes('token'), false);
assert.equal(launch.includes('fragment'), false);

for (const unsafe of [
  'https://example.com',
  'sftp://example.com/%0Aetc',
  'sftp://example.com\n',
  ''
]) {
  assert.equal(api.parseConnectionTarget(unsafe).ok, false, unsafe);
}

console.log('Ghost FTP extension core tests passed');
