# Ghost FTP queue priority and reordering

Ghost FTP **0.0.6** maintains queue priority/reordering for desktop transfers. Reordering is limited to jobs whose current status is `queued`; it never changes transfer identity, connection ownership or lifecycle state of running/terminal work.

## User contract

A selected queued transfer can move Top, Up, Down or Bottom. Running, completed, failed, cancelled and other non-queued jobs are not reorderable. Boundary moves are idempotent.

## Scheduler invariants

`internal/transfer/queue_order.go` owns queue mutation. Running/terminal history slots are preserved, the selected transfer ID remains stable, other queued jobs retain relative order, and `jobConnections` is not rewritten.

Reordering never creates, retries, cancels, duplicates or starts work. A real reorder emits one state snapshot; a no-op does not emit redundant state.

## Shared engine

All maintained native desktop frontends use the **same typed `internal/api.Engine` operations**:

```text
MoveTransferTop(id)
MoveTransferUp(id)
MoveTransferDown(id)
MoveTransferBottom(id)
```

No frontend implements a second scheduler.

## Windows and Linux

Windows and Linux expose real queue controls under the same queued-only policy and preserve selection by stable transfer ID rather than stale row index.

## macOS development behavior

The native AppKit Transfer Queue uses the **same typed `internal/api.Engine` operations** and **does not create a Mac-only scheduler or protocol stack**. macOS remains a separately validated native development/source frontend; a source build is not Developer ID/notarization evidence.

## Regression coverage

Maintained tests cover four-way ordering, non-queued slot preservation, connection binding, edge idempotence, state snapshots, Engine wiring and native frontend behavior.

## 0.0.6 release boundary

The published Ghost FTP 0.0.6 release is **13 platform artifacts / 16 public files**. Queue priority remains part of the maintained product contract while Android SFTP stays hidden until strict host-key verification exists and macOS remains outside the public release.

See [Testing](TESTING.md), [Architecture](ARCHITECTURE.md), [Platform parity](PLATFORM-PARITY.md) and [`../macos/PARITY.md`](../macos/PARITY.md).
