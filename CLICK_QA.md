# Click-by-Click QA

## Executed in this environment

The Linux compatibility host was started in headless/local-server mode. Direct API checks passed for local directory listing, folder creation, rename, SHA-256 checksum, duplicate and delete. The compatibility runtime truth endpoint correctly reports that it is not the native protocol engine.

## Source-inspected controls

Main menu, toolbar actions, Site Manager, New Connection, Preferences, Transfer Center, File Properties and About/Updates exist as real component/control code in the native Tauri source. The native titlebar uses Tauri window APIs; the fallback Windows host exposes minimize/maximize/restore/close and drag actions through Win32.

## Not executable here

A complete pointer/keyboard click sweep of every native control, 50× modal open/close stress test, target-OS shell integrations and all 14 language UI passes require a built Tauri application on Windows/Linux. Those gates remain open rather than being marked passed from static source inspection.
