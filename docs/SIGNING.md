# Code signing

Windows Verified Publisher status requires a real Authenticode certificate controlled by the legitimate publisher. macOS trust requires a real Developer ID identity and Apple notarization credentials.

Development and CI builds may be unsigned. The project must never claim a simulated publisher identity, forged signature or notarized status. Signing credentials belong in protected CI secrets and must not be committed to the repository.
