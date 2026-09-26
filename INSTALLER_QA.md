# Installer QA

Source and build checks completed here: the Windows setup fallback compiles as a PE32+ x64 GUI executable; its wizard contains six real pages; future sidebar steps are disabled; Next cannot leave the licence page until the EULA is accepted; the complete commercial licence is embedded; a custom installation folder is accepted and validated; upgrade replacement keeps a temporary rollback copy; Apps & Features publisher metadata names Brendigo; uninstall uses `GhostFTP.exe --uninstall` and no separate uninstaller binary is created.

Windows runtime QA is still required for Welcome/Back/Next, EULA scrolling/acceptance, folder picker, invalid/unwritable folder, shortcuts, existing install, upgrade, reinstall, cancel, rollback, Apps & Features registration, launch-after-install and removal. The black-titlebar acceptance test must be captured on Windows 10/11 before FINAL.
