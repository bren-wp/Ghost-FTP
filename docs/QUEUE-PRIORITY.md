# Ghost FTP queue priority and reordering

Queue priority is implemented in the **post-0.0.3 source line** and is targeted for the next public release after the normal exact-head, post-merge and release-prep gates pass. It is not retroactively part of the already published Ghost FTP 0.0.3 release.

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

This gives the following fail-closed guarantees:

- a running or terminal job keeps its exact list/history slot;
- the selected queued job keeps the same transfer ID;
- Top/Bottom preserves the relative order of every other queued job rather than swapping arbitrary endpoints;
- `jobConnections` is not rewritten by reordering, so connection-generation/session ownership stays bound to transfer identity;
- no new transfer is created, duplicated, retried, cancelled or started by a reorder action;
- one successful reorder emits one complete `state` snapshot with the existing paused state;
- an edge no-op emits no redundant queue event;
- a closed manager, unknown transfer or non-queued transfer is rejected.

Reordering is therefore a scheduler-order operation only. It does not mutate transfer payload paths, credentials, remote-session identity, conflict policy or transfer state.

## Tree-transfer safety

Directory-tree transfers prepare structural dependencies before their file jobs become runnable. Upload tree planning ensures required remote directories before the concrete `reservation.Commit()` call crosses the `BatchReservation.Commit()` boundary. Download tree planning prepares the safe local directory structure before the same queue commit boundary.

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

Windows renders four real owner-drawn controls beside the existing queue toolbar. Their enabled state is derived from the shared queued-only policy and connection-busy state. Commands route through the engine API, and refresh restores selection by transfer ID after the row changes position.

The controls are localized through the same 24-language local catalog used by the existing Up/Down queue controls. They do not add a network service, telemetry path or hidden persistence layer.

## Linux behavior

Linux renders Top, Up, Down and Bottom controls using the same shared policy. Mouse actions call the same four engine operations, refresh from `Engine.Transfers()`, and restore the selected transfer by ID rather than stale row index.

Running or otherwise non-queued selections expose no active priority action.

## Interaction with pause, retry and connection lifecycle

Queue priority does not call the transfer pump as a side effect. Existing pause/resume/cancel/retry behavior remains owned by the transfer manager and action-state layers.

A reorder also does not rewrite `jobConnections`. If the active server connection changes, the existing generation/connection-identity protections remain authoritative; changing visual scheduler order cannot make stale queued work belong to a new session.

## Regression coverage

The maintained contract is protected by:

- `internal/transfer/queue_order_test.go` — stable four-way queued ordering, running/terminal slot preservation, connection-binding preservation, edge idempotence, rejection behavior and state snapshots;
- `internal/desktop/queue_priority_test.go` — shared single-selection/queued-only policy and all 24 translations;
- `internal/desktop/queue_priority_linux_test.go` — Linux layout, queued-only state and ID-based selection restoration;
- `scripts/test_queue_priority_contract.py` — engine/API/Windows/Linux wiring plus tree-transfer dependency ordering.

Native Windows and Linux CI builds remain the compile/runtime gate for the platform-specific frontends. Because the maintained Windows UI changes, authentic Windows screenshot evidence is also required on the exact final PR head before merge.

## Release boundary

Root `VERSION` remains **0.0.3** during this feature work. The already published `ghostftp-v0.0.3` release must not be rewritten. Queue priority becomes a public release capability only after a later reviewed release-prep change advances the version and the complete publication/read-back/retention lifecycle succeeds.

See [Roadmap](ROADMAP.md), [Testing](TESTING.md), [Architecture](ARCHITECTURE.md) and [Platform parity](PLATFORM-PARITY.md).
