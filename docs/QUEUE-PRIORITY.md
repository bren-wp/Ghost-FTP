# Ghost FTP queue priority and reordering

Ghost FTP **0.0.6** includes queue priority/reordering as a maintained Windows/Linux desktop capability and as part of the active native macOS development frontend. Reordering is deliberately limited to jobs whose current status is `queued`; it never rewrites transfer identity, connection ownership or lifecycle state of running/terminal work.

## User contract

A single selected queued transfer can be moved through four explicit actions:

- **Top** — first queued scheduler position;
- **Move up** — ahead of the nearest earlier queued transfer;
- **Move down** — behind the nearest later queued transfer;
- **Bottom** — final queued scheduler position.

Running, completed, failed, cancelled and other non-queued jobs are not reorderable. Boundary moves are idempotent no-ops.

## Scheduler invariants

`internal/transfer/queue_order.go` owns scheduler mutation and rotates jobs only through slots already occupied by queued transfers.

```text
before: Q1, RUNNING, Q2, DONE, Q3
action: Q3 -> Top
after:  Q3, RUNNING, Q1, DONE, Q2
```

Maintained guarantees:

- running/terminal jobs keep their history slots;
- selected queued job keeps the same transfer ID;
- Top/Bottom preserves relative order of other queued jobs;
- `jobConnections` is not rewritten, preserving connection/session ownership;
- reordering never creates, retries, cancels, duplicates or starts work;
- one real reorder emits one complete state snapshot;
- an edge no-op emits no redundant queue event;
- closed manager, unknown transfer or non-queued transfer is rejected.

Directory-tree structural preparation occurs before `reservation.Commit()` makes file jobs reorderable, so priority controls cannot move a file ahead of an unexecuted directory-creation queue dependency.

## Engine API

All maintained native desktop frontends use the **same typed `internal/api.Engine` operations**:

```text
MoveTransferTop(id)
MoveTransferUp(id)
MoveTransferDown(id)
MoveTransferBottom(id)
```

No frontend edits the queue slice directly or implements a second scheduler.

## Windows behavior

Windows renders four real controls beside the transfer queue. Enabled state comes from the shared single-selection/queued-only policy. Commands route through the Engine API and refresh restores selection by stable transfer ID.

## Linux behavior

Linux exposes Top, Up, Down and Bottom under the same policy. Actions use the same Engine methods and restore selection by transfer ID rather than a stale row index.

## macOS development behavior

The native AppKit Transfer Queue renders the authoritative shared manager snapshot and calls the **same typed `internal/api.Engine` operations** for Move Top / Move Up / Move Down / Move Bottom. It preserves selection by stable transfer ID, enables reordering only for one queued job and **does not create a Mac-only scheduler or protocol stack**.

This is source/development parity. macOS remains a separately validated native development/source frontend and a successful development build is not Developer ID/notarization evidence.

## Interaction with pause, retry and connection lifecycle

Queue priority does not call the transfer pump as a side effect and does not rewrite `jobConnections`. Existing pause/resume/cancel/retry behavior and generation/connection-identity protections remain authoritative, so visual scheduler ordering cannot attach stale queued work to a later session.

## Regression coverage

The 0.0.6 queue contract is protected by:

- `internal/transfer/queue_order_test.go` for four-way ordering, non-queued slot preservation, connection binding, edge idempotence and state snapshots;
- `internal/desktop/queue_priority_test.go` for shared queued-only policy and localization;
- `internal/desktop/queue_priority_linux_test.go` for Linux layout and ID-based selection restoration;
- `scripts/test_queue_priority_contract.py` for Engine/API/Windows/Linux wiring and tree-transfer dependency ordering;
- macOS parity tests/workflow for AppKit source parity.

Authentic Windows/Linux/Android runtime evidence remains the immutable 15-image evidence bundle; macOS development validation is separate and is not silently counted as public notarization evidence.

## 0.0.6 release boundary

Root `VERSION` is **0.0.6**. Queue priority is part of the maintained desktop/source contract; publication still requires exact-head tests/builds, exact post-merge main verification and canonical `ghostftp-v0.0.6` publication/readback/retention.

The current public release is **18 platform artifacts / 21 public files**: Windows/Linux desktop packages, a production-signed Android APK, Chrome/Edge/Firefox helper ZIPs and release metadata. Android SFTP remains hidden until strict maintained host-key verification exists; browser packages have no supported desktop launch/handoff; macOS remains a separately validated native development/source frontend.

See [Roadmap](ROADMAP.md), [Testing](TESTING.md), [Architecture](ARCHITECTURE.md), [Platform parity](PLATFORM-PARITY.md) and [`../macos/PARITY.md`](../macos/PARITY.md).
