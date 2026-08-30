# Roadmap

The roadmap is capability-based rather than tied to speculative product-version numbers.

The 1.3.0 mobile parity work established the practical native file-manager baseline on both mobile clients: filtering/search, deterministic directory-first sorting, direct path navigation, multi-file upload, compact mobile action surfaces and restoration of non-secret connection metadata without persistent passwords/passphrases.

The 1.4.0 transfer-control work added observable byte-level upload/download progress on Android and iOS plus a fail-safe batch stop boundary. Multi-file uploads can stop after the active file completes; ByFTP intentionally does not tear down an FTP/FTPS data/control channel mid-command merely to provide an aggressive cancel button.

The 1.5.0 shared-hosting diagnostics work makes successful connections easier to understand without adding scanners or hidden probes. Desktop, Android and iOS derive non-secret diagnostics from their existing initial remote listing, recognize common web-root directories, distinguish secure transport from plain FTP and keep detected paths advisory only. Security audits block credential-bearing diagnostic fields, independent diagnostic network activity, automatic web-root navigation and persistence of derived diagnostic state.

The 1.6.0 maintenance work makes the entire tracked repository part of the release contract. Cross-platform path portability, generated-artifact exclusion, UTF-8/text hygiene, merge-conflict detection and explicit current-release markers are validated before production packaging, reducing filesystem-specific and repository-state regressions that feature-level tests cannot see.

Current priorities are secure Android Keystore-backed SFTP private-key handling, secure iOS SFTP and explicit-FTPS support only when those transports can preserve the same fail-closed trust model, additional accessibility/keyboard improvements on tablets, secure Unix credential handling for SFTP cases that cannot safely use command-line secrets, real platform signing when publisher credentials are available, and broader server compatibility without weakening TLS, host-key or path protections.

True mid-file cancellation remains a future capability only if it can be implemented with protocol-aware abort/cleanup semantics, deterministic partial-file handling and regression coverage. A UI control must not imply that an active remote write was safely cancelled when the protocol or server state cannot prove that guarantee.

Shared-hosting diagnostics may later grow richer server capability summaries only when those facts can be derived from normal protocol exchanges or explicit user-requested checks. ByFTP will not add background port scanning, trust bypasses, arbitrary PASV destinations or secret-bearing diagnostic uploads to achieve a more detailed status screen.

Android debug and unsigned release APK generation is part of the verified release pipeline. The remaining Android distribution milestone is a real externally managed production signing identity plus a signing-verification gate; the repository will not substitute a debug or fabricated key for that requirement.

The native iOS application and verified unsigned arm64 IPA/app artifacts remain the iOS release baseline. Future iOS transport work must not claim SFTP or explicit FTPS until a reviewed implementation has host/certificate verification, credential handling, path safety, lifecycle tests and a real iPhoneOS CI gate. Production iOS distribution additionally requires a legitimate Apple signing identity and provisioning profile managed outside the repository.

A feature is complete only when its failure modes are understood, regression coverage exists where practical and every affected platform build gate remains green.
