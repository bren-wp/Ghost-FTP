# Queue priority

Queue ordering and transfer lifecycle are maintained desktop capabilities.

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
