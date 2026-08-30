# Roadmap

The roadmap is capability-based rather than tied to speculative product-version numbers.

The 1.3.0 mobile parity work establishes the practical native file-manager baseline on both mobile clients: filtering/search, deterministic directory-first sorting, direct path navigation, multi-file upload, compact mobile action surfaces and restoration of non-secret connection metadata without persistent passwords/passphrases.

Current priorities are now clearer shared-hosting diagnostics, secure Android Keystore-backed SFTP private-key handling, secure iOS SFTP and explicit-FTPS support only when those transports can preserve the same fail-closed trust model, richer transfer progress/cancellation for mobile batch operations, additional accessibility/keyboard improvements on tablets, secure Unix credential handling for SFTP cases that cannot safely use command-line secrets, real platform signing when publisher credentials are available, and broader server compatibility without weakening TLS, host-key or path protections.

Android debug and unsigned release APK generation is part of the verified release pipeline. The remaining Android distribution milestone is a real externally managed production signing identity plus a signing-verification gate; the repository will not substitute a debug or fabricated key for that requirement.

The native iOS application and verified unsigned arm64 IPA/app artifacts remain the iOS release baseline. Future iOS transport work must not claim SFTP or explicit FTPS until a reviewed implementation has host/certificate verification, credential handling, path safety, lifecycle tests and a real iPhoneOS CI gate. Production iOS distribution additionally requires a legitimate Apple signing identity and provisioning profile managed outside the repository.

A feature is complete only when its failure modes are understood, regression coverage exists where practical and every affected platform build gate remains green.