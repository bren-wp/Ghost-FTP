# Ghost FTP 0.99.71 — Windows native security and performance acceptance

**Status:** Pending physical Windows 10/11 tests; automated CI success alone is not a substitute.

| Area | Required real-world test | Pass condition |
| --- | --- | --- |
| Official external links | Open each allowlisted support/release URL; plant a harmless fake `rundll32.exe` in a test directory and monitor the process tree | Browser opens; fake executable never runs; non-allowlisted URLs rejected |
| File association | Open a saved local editor document via Edit-in-Place with default association | Opens once without console flash or improper command lookup |
| Invalid shell target | Embedded NUL, empty target, broken default association | Safe error without UI hang or leaked path/credential |
| Deep link | Oversize URL, 65 parameters, percent-encoded newline and ESC, fragment, userinfo/port | No parsed event or unexpected network connection |
| SFTP terminal | Split UTF-8 packet boundaries, emoji, control sequences, invalid bytes, sustained input and abrupt disconnect | Correct characters/replacement handling; responsive UI; no unbounded memory growth |
| Performance | 100+ opens, 30-minute terminal output, long-lived FTP/FTPS/SFTP connections | No progressive resource leak or loss of user-requested session persistence |
| Crash/stress | Restart app, interleave file uploads/downloads with abort/pause/disconnect and forced network loss | Preserves original remote files, restores state and logs redacted errors |
| Install and upgrade | Win10/Win11, Portable/NSIS/MSI fresh install, update, uninstall, non-admin account, high DPI | Expected GUI appears, no white/black flash or lost saved profiles |

Use real screenshots and logs. Mark each scenario passed only after direct native evidence has been captured, not on the basis of source tests.
