# Ghost FTP v0.99.47 — safe destructive confirmation acceptance

**Author:** Brendigo — https://brendigo.com  
**Verified predecessor:** Public v0.99.46 on exact source SHA `82dd63d82cf6ba41cff3f998671730cf9261aa49`  
**Status:** Source change only pending exact-head CI and installed-app acceptance.

## Confirmed code defect and mitigation

The reusable desktop `ConfirmModal` awaits the Promise returned by `onConfirm`, disables its buttons and shows a working state until it settles. However, the Sites delete callback discarded its Promise with `void remove()`; Find Duplicates, Disk Usage and Snippets likewise closed their dialogs before their asynchronous deletes returned. This made errors harder to retry and allowed workspace changes while an operation was still pending.

Changed callback contracts:
- **Sites:** `onConfirm={remove}`; the confirmation pins its target to the original saved profile ID, independent of later row selection or profile reload. `remove` propagates any rejected credential/profile deletion to `ConfirmModal`, which surfaces the error and keeps the confirmation open.
- **Find Duplicates:** `onConfirm={deleteSelected}`; the parent Escape listener ignores Escape while confirming or deleting.
- **Disk Usage:** `onConfirm={() => removeNode(confirmDel)}`, no early confirmation reset.
- **Snippets:** `onConfirm={() => remove(deleting.id)}`, no early confirmation reset.
- `scripts/check-interaction-hardening.mjs` rejects reintroducing discarded destructive Promises, premature modal closure and unsafe parent Escape handling.

## Security and functional boundaries

The backend remains authoritative for deletion and errors. The duplicate/disk/snippet stores currently surface errors via their own toasts and settle their promises afterward; those dialogs stay pending until the store completes but may close after an error. Sites failures are retryable in-place. There is no claim that all destructive operations are transactional or immune to remote filesystem races.

## Required real-device QA

Test successful and denied deletion, connection loss, delayed backend deletion, double-clicked Confirm, Escape while pending, Cancel before starting deletion, selected profile reload and modal focus on Windows 10/11, Linux GTK/WebKit and macOS. Ensure no action occurs on cancel, no duplicate delete starts, error messages are redacted, and original objects survive failures.

No mock screenshots or 1:1 visual acceptance claimed. Complete the project's 59-reference Windows/Linux acceptance separately.
