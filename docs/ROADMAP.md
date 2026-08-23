# Roadmap

The roadmap is capability-based rather than tied to speculative product-version numbers.

Current priorities are clearer shared-hosting diagnostics, secure Android Keystore-backed SFTP private-key handling, secure Unix credential handling for SFTP cases that cannot safely use command-line secrets, real platform signing when publisher credentials are available, further keyboard/accessibility improvements and broader server compatibility without weakening TLS, host-key or path protections.

Android debug and unsigned release APK generation is now part of the verified release pipeline. The remaining Android distribution milestone is a real externally managed production signing identity plus a signing-verification gate; the repository will not substitute a debug or fabricated key for that requirement.

A feature is complete only when its failure modes are understood, regression coverage exists where practical and every affected platform build gate remains green.
