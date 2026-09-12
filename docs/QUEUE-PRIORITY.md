# Ghost FTP queue priority and reordering

Ghost FTP **0.0.5** includes queue priority/reordering as a maintained Windows/Linux capability. Reordering is deliberately limited to jobs whose current status is `queued`; it never rewrites transfer identity, connection ownership or the lifecycle state of running/terminal work.

## User contract

A single selected transfer whose status is `queued` can be moved through four explicit actions:

- **Top** — move to the first queued scheduler position;
- **Move up** — move ahead of the nearest earlier queued transfer;
- **Move down** — move behind the nearest later queued transfer;
- **Bottom** — move to the final queued scheduler position.

Running, completed, failed, cancelled and other non-queued jobs are not reorderable. At a queued boundary, Top/Up or Down/Bottom is an idempotent no-op rather than an error-producing state change.

## Scheduler invariants

`internal/transfer/queue_order.go` owns the scheduler mutation. The implementation builds the ordered list of slots currently occupied by `queued` jobs and rotates jobs only through those slots.

For example:

```text
before: Q1, RUNNING, Q2, DONE, Q3
action: Q3 -> Top
after:  Q3, RUNNING, Q1, DONE, Q2
```

The maintained fail-closed guarantees are:

- running/terminal jobs keep their exact history slots;
- the selected queued job keeps the same transfer ID;
- Top/Bottom preserves the relative order of all other queued jobs;
- `jobConnections` is not rewritten, so connection-generation/session ownership remains attached to transfer identity;
- reordering never creates, duplicates, retries, cancels or starts transfer work;
- one real reorder emits one complete `state` snapshot with the existing paused state;
- an edge no-op emits no redundant queue event;
- a closed manager, unknown transfer or non-queued transfer is rejected.

Reordering changes scheduler order only. It does not mutate transfer paths, credentials, remote-session identity, conflict policy or transfer state.

## Tree-transfer safety

Directory-tree transfers prepare structural dependencies before their file jobs become runnable. Upload planning ensures required remote directories before the concrete `reservation.Commit()` boundary. Download planning prepares the safe local directory structure before the same queue-commit boundary.

Queue priority operates only on the resulting queued file-transfer jobs. It cannot move a file ahead of an unexecuted directory-creation queue job because those directory preparations are not represented as reorderable transfer jobs in this scheduler.

## Engine API

The desktop frontends use four bounded engine calls:

```text
MoveTransferTop(id)
MoveTransferUp(id)
MoveTransferDown(id)
MoveTransferBottom(id)
```

Each delegates to the transfer manager's queued-only operation. No frontend edits the queue slice directly.

## Windows behavior

Windows renders four real owner-drawn controls beside the existing queue toolbar. Enabled state comes from the shared single-selection/queued-only policy. Commands route through the Engine API, and refresh restores selection by transfer ID after the row moves.

The controls use the maintained 24-language local catalog and introduce no network service, telemetry path or hidden persistence layer.

## Linux behavior

Linux renders Top, Up, Down and Bottom through the same shared policy. Mouse actions call the same four Engine operations, refresh from `Engine.Transfers()` and restore selection by transfer ID rather than a stale row index.

Running or otherwise non-queued selections expose no active priority action. Layout regression coverage keeps the priority controls clear of the existing queue toolbar/actions.

## Interaction with pause, retry and connection lifecycle

Queue priority does not call the transfer pump as a side effect. Existing pause/resume/cancel/retry behavior remains owned by the transfer manager and action-state layers.

A reorder also does not rewrite `jobConnections`. Existing generation/connection-identity protections remain authoritative, so changing visual scheduler order cannot make stale queued work belong to a later session.

## Regression coverage

The 0.0.5 contract is protected by:

- `internal/transfer/queue_order_test.go` — four-way ordering, non-queued slot preservation, connection binding, edge idempotence, rejection and complete state snapshots;
- `internal/desktop/queue_priority_test.go` — shared single-selection/queued-only policy and all 24 translations;
- `internal/desktop/queue_priority_linux_test.go` — Linux layout, queued-only state and ID-based selection restoration;
- `scripts/test_queue_priority_contract.py` — Engine/API/Windows/Linux wiring plus tree-transfer dependency ordering and current-release documentation binding.

Native Windows/Linux CI builds remain the compile/runtime gate for the platform frontends. Authentic Windows/Linux runtime evidence is required on the exact final release-prep head where the maintained queue UI changes are part of the candidate.

## 0.0.5 release boundary

Root `VERSION` is **0.0.5**. Queue priority is part of the 0.0.5 source/release contract, but this document does not authorize publication by itself.

Publication still requires exact-head tests/builds, Linux packaging/install gates, Android development APK validation, read-only authentic Windows/Linux/Android runtime evidence, review/merge, exact post-merge verification and the canonical `ghostftp-v0.0.5` publication/read-back/retention lifecycle.

Queue priority does not change the public platform allow-list or artifact count: Windows/Linux remain the 14-platform-artifact / 17-public-file release surface, and Android remains a separately validated development APK.

See [Roadmap](ROADMAP.md), [Testing](TESTING.md), [Architecture](ARCHITECTURE.md) and [Platform parity](PLATFORM-PARITY.md).
