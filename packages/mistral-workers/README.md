# mistral-workers

Composition root reserved for asynchronous workloads that genuinely need execution outside the request path.

The IDLE model explicitly forbids using one long-running sleep loop/goroutine/job per player activity. Dungeon and gathering progress should normally be reconstructed from timestamps and immutable state. Workers will be introduced only for bounded asynchronous operations such as materialization, fan-out, cleanup or scheduled economy processes when those contracts exist.
