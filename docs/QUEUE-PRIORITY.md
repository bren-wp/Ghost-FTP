# Queue priority

Queue ordering and transfer lifecycle are maintained desktop capabilities.

## Transfer lifecycle

The queue distinguishes **queued**, **running**, **completed**, **failed** and **cancelled** work. Reordering never rewrites a running or terminal transfer into another lifecycle state.

## Rules

- priority/reordering applies only where the queue implementation allows it;
- queued work may be reordered without rewriting transfer identity;
- running or terminal work must not be silently converted back to queued work;
- UI buttons must reflect whether an action is currently valid;
- controls may move into **More** or compact layouts, but underlying behavior must remain real.

## Transfer Queue reference columns

- File
- Direction
- Progress
- Status
- Speed
- ETA

The visible status must be derived from real transfer state.

Windows and Linux expose maintained desktop queue capabilities. Android exposes mobile transfer controls appropriate to its implementation.

macOS queue parity is retired with the removed macOS application.

## Current implementation contract

Ghost FTP **0.0.8** includes queue priority/reordering. Desktop queue actions expose `MoveTransferTop(id)` and `MoveTransferBottom(id)` while preserving transfer identity and lifecycle safety. The connection binding map `jobConnections` is not rewritten by a reorder. Tree-transfer preparation must complete before `reservation.Commit()` so dependency preparation cannot be reordered behind queue publication.

Root `VERSION` is **0.0.8**. This contract is validated by exact-head tests/builds before release claims are made.

## Release boundary

Queue behavior belongs to the active Windows, Linux and Android product line. The current release shape is **13 platform artifacts / 16 public files** including browser helpers and release metadata.
