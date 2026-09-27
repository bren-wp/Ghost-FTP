# Ghost FTP Desktop Update Security

## Trust model

Ghost FTP desktop updates are accepted only when the package signature validates against the public key embedded in the installed application.

The public key is safe to distribute. The matching private key is the release signing secret and must remain confidential.

## Private-key rules

- Store the private key only in an approved secret manager / GitHub Actions repository secret.
- Never commit it.
- Never put it in `updates/`, `website/`, release assets or CI logs.
- Never send it to the web server.
- Limit workflows that receive it to trusted `main` release builds.
- Pull-request builds must not receive it.

## Public artifacts

These are safe/required to publish:

- signed Windows NSIS Setup;
- Windows Setup `.sig`;
- signed Linux AppImage;
- Linux AppImage `.sig`;
- public update response containing signature text;
- public updater verification key already embedded in the application.

The updater response must contain the **signature file contents**, not a path or URL to the signature.

## Key loss

If the private signing key is lost, existing installed clients cannot verify packages signed only by a replacement key.

Treat key backup/escrow as production-critical.

## Key rotation

Do not simply replace the embedded public key and immediately sign with a new private key.

A safe rotation requires a transition release that existing clients can verify and that introduces the new verification trust before subsequent releases rely exclusively on the new key. Document and test the exact migration path before rotation.

## Compromised key

If compromise is suspected:

1. stop publishing new update responses;
2. restore the website to a previously known-good update response or remove it temporarily;
3. protect/revoke the compromised secret in CI;
4. investigate release/tag/account integrity;
5. prepare a reviewed key-rotation/recovery release;
6. communicate through official Ghost FTP channels.

Never “fix” a compromised release by retargeting an existing immutable version tag.
