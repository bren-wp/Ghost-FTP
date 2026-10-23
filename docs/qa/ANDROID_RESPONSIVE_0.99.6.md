# Android 0.99.6 responsive and 16-screen design acceptance

This is a QA plan, **not a certification of 1:1 matching**. Source references are the two original user-supplied composite Android UI images (16 screens). Screens are rendered from actual app view hierarchies and must never inject imaginary FTP profiles, provider accounts, speeds or hostnames.

## Device and font matrix

| Viewport px at 160 dpi | Context | Required evidence |
| --- | --- | --- |
| 320×640 | Small phone | One readable Files pane, drawer closes safely, five action targets visible |
| 360×800 | Standard compact phone | Navigation and Settings never crop text or confirmations |
| 393×852, 1.4× font | Large text | Responsive Settings and action labels remain tappable |
| 480×960 | Approved reference | Live noncredential screenshots for Splash, Files, Sites, Add Site, Connect, Transfers, Details, Cloud, Bookmarks, Search, Sync, Settings, About |
| 720×1280 | Tablet | Permanent docked navigation; Files dual pane when room permits |
| 960×540 | Landscape | Phone-type navigation determined by smallest width, no clipped toolbar |

CI performs real click-through and writes per-viewport PNG hashes. PNGs are QA *inputs*, not screenshot similarity test results.

## Reference screen capability matrix

| # | Reference | Native candidate | 1:1 certified |
| --- | --- | --- | --- |
| 1 | Splash | Branded vector and startup handoff | No |
| 2 | Sidebar | Responsive live navigation | No |
| 3 | Sites | Real saved-site metadata only | No |
| 4 | Add/Edit Site | General/Advanced/Sync tabs and Cancel/Save | No |
| 5 | Connect | Live saved-profile details; session password required | No |
| 6 | Dual-pane Files | Actual server and app-private local data; compact single pane | No |
| 7 | Transfer Queue | Actual session operations; persistent multi-transfer queue missing | No |
| 8 | Transfer Details | Actual session state; synthetic ETA prohibited | No |
| 9 | Cloud | Real provider icons, signed URL transfers; OAuth account browser missing | No |
| 10 | Bookmarks | Real stored remote bookmarks | No |
| 11 | Remote Search | Real scoped search; advanced filters incomplete | No |
| 12 | Sync / Compare | Actual bounded compare; destructive mirror not implemented | No |
| 13 | Remote Text Editor | Real bounded UTF-8 editor and guarded save | No |
| 14 | Batch Operations | Validated numbered batch rename; attributes/time options missing | No |
| 15 | Settings | Adaptive two/one-column layout; some settings informational | No |
| 16 | About | Branded actual metadata and links | No |

Acceptance must verify no accidental persistence of FTP passwords, no runtime crashes, no clipped actionable controls and the exact source-code SHA of CI screenshots before calling any view visually equivalent. Reference data is fictional; application data must remain genuine.
