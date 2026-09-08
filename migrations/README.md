# Database migrations

The first persistence schema deliberately stores aggregate state as JSONB while keeping identity, optimistic versioning, command idempotency and transaction boundaries relational.

Each aggregate table has an `id`, monotonic `version`, JSONB `state` and `updated_at`. Repository updates use `WHERE id = ? AND version = ?` semantics and increment `version` atomically.

`idempotency_commands` is not a substitute for optimistic concurrency. It deduplicates host commands by `(scope, idempotency_key)`, binds the key to a request hash and stores the completed response so retries can replay the original result without repeating game mutations.

The PostgreSQL adapters use `database/sql` and intentionally do not register a driver in `mistral-core`. Driver selection and connection lifecycle belong to the host composition root.
