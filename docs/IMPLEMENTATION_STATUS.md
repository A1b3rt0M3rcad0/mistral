# Mistral implementation status

This document tracks the engineering state of the MVP bootstrap without treating undefined product decisions as implemented game rules.

## Implemented

### Content runtime

- Strict JSON loading with unknown-field rejection.
- Duplicate-id rejection.
- Cross-reference validation.
- Content-addressed release identity (`version@sha256:<digest>`).
- Craft-only invariant preventing monster/gathering drops of ready equipment or tools.

### Character

- Race-backed character creation.
- Level initialized at 1.
- Explicit per-dungeon unlocked-tier progression.
- Idempotent progression advancement.

### Inventory

- Batch-aware stacks.
- Independent expiry metadata per stack.
- Metadata-aware stack compatibility.
- Earliest-expiry-first consumption.
- Atomic multi-item consumption.

### Gathering

- Elapsed-time session model.
- Deterministic RNG per cycle.
- Claim cursor (`claimed_cycles`) preventing duplicate rewards.
- Claim results independent of claim batching.
- Pure claim use case that materializes rewards atomically in an inventory clone.
- Persisted claim command that updates Gathering Session + Inventory in one transaction boundary.
- Command replay returns the stored original result and does not materialize resources again.

### Crafting

- Recipe/station validation.
- Multi-craft support.
- Atomic ingredient consumption/output materialization in the pure use case.
- Persisted crafting command with command-level idempotency and optimistic inventory versioning.
- Replayed craft commands return the stored result without consuming ingredients or creating output again.

### Dungeon

- One engine for solo/group party sizes.
- Deterministic encounter scheduling.
- Content-release pinning.
- Deterministic loot for defeated encounters.
- Boss-key gating.
- Boss progression that unlocks a next tier only if that tier is present in the content release.
- Persisted boss command spanning Character + Inventory + idempotency ledger in one transaction boundary.
- Replayed boss commands do not rerun combat, duplicate loot or advance progression again.

### Combat boundary

- `CombatResolver` application port.
- Disabled production resolver rather than a fabricated combat equation.
- Test resolver used only to prove orchestration.

### Persistence and replay safety

- Generic versioned persistence record (`Record[T]`).
- Explicit `ErrNotFound`, `ErrAlreadyExists` and optimistic `ErrConflict` semantics.
- Repository ports for Character, Inventory, Gathering Session and Dungeon Run.
- `Save(value, expectedVersion)` compare-and-swap contract.
- Thread-safe in-memory adapters used to validate repository behavior.
- Tests proving stale writes are rejected instead of silently overwriting newer state.
- `Transactor` application-facing port for atomic work spanning multiple repositories.
- PostgreSQL `Transactor` using `database/sql`, including commit, rollback and nested-transaction reuse tests.
- PostgreSQL JSONB aggregate store with relational identity and monotonic version columns.
- PostgreSQL repository adapters for Character, Inventory, Gathering Session and Dungeon Run.
- Initial up/down SQL migration for gameplay aggregate tables and idempotency ledger.
- Command idempotency ledger with `(scope, idempotency_key)` identity, request hashes, in-progress/completed states and stored replay responses.
- In-memory and PostgreSQL ledger adapters.
- Same key + different request hash is rejected instead of silently reusing a command identity.

The intended delivery semantic is not “exactly once”. The design supports replay-safe command processing under an at-least-once transport model when the idempotency ledger mutation and gameplay side effects execute inside the same database transaction.

Optimistic concurrency, transaction atomicity and idempotency remain separate mechanisms:

```text
optimistic version -> prevents lost updates
transaction         -> atomically commits/rolls back related writes
idempotency ledger  -> deduplicates retried commands and replays the original result
```

Aggregate state is deliberately stored as JSONB in the first PostgreSQL schema while identity, versions, idempotency keys and transaction semantics remain relational. This keeps the initial persistence schema tolerant of domain evolution without weakening concurrency guarantees.

### Architecture enforcement

- Core cannot depend on API, Workers or Frontend hosts.
- Domain cannot depend on application or outward layers.
- Application cannot depend on infra, composition, presentation or entrypoint.
- Infra cannot depend upward on composition, presentation or entrypoint.
- Quality-gate tests cover allowed and forbidden dependency directions.

## Proven executable path

The integration/core tests now prove both game flow and replay behavior:

```text
Human character
-> Iron Mine session
-> deterministic Iron Ore / Coal claim
-> replay-safe persisted gathering claim
-> 3 Iron Ingots
-> replay-safe persisted crafting
-> Iron Sword
-> Abandoned Mine Tier I
-> deterministic Goblin / Cave Spider encounters
-> deterministic loot
-> Goblin King Key
-> replay-safe persisted Goblin King challenge
-> boss loot / progression
```

## Product/content decisions still required

### Blocking full Tier-I -> Tier-II live progression

- Define `Abandoned Mine` Tier II content and its actual difficulty/reward changes.
- Define acquisition path for `oak_handle`.
- Define intended pre-dungeon acquisition path for `leather_strip`, or change the Iron Sword recipe/order of progression.

### Combat

- Character/combat attributes.
- Damage formula.
- Attack cadence/turn semantics.
- Hit/crit/evasion/block semantics, if any.
- Death/failure/recovery behavior.
- Equipment stat model.
- Group-combat targeting/scaling semantics.

### Boss keys

The current contract says a key makes the boss available but does not specify when the key is consumed. The service therefore injects `BossKeyPolicy`. Product must choose a rule such as consume-on-attempt, consume-on-victory or another explicit behavior before production exposure.

## Remaining engineering work

The persistence contracts and PostgreSQL adapters now exist. Remaining infrastructure work is narrower:

- Register/select a PostgreSQL `database/sql` driver in the host and own connection lifecycle outside `mistral-core`.
- Wire database configuration and repository composition into `mistral-api` / workers.
- Add a migration execution/deployment strategy.
- Add integration tests against a real PostgreSQL instance; current repository adapters compile and transaction semantics are tested with a minimal `database/sql` test driver, but no live PostgreSQL service is part of CI yet.
- Persist normal dungeon encounter reward/materialization checkpoints so retries cannot re-materialize already-applied non-boss encounters.
- Define account/authentication identity and ownership checks at the host boundary.
- Expose mutable HTTP commands only after PostgreSQL composition is active.

## Next engineering slice

1. Wire PostgreSQL into a host composition root while keeping driver selection outside Core.
2. Add live PostgreSQL integration tests and migration validation in CI.
3. Add dungeon encounter materialization/checkpoint persistence.
4. Expose authenticated, server-authoritative HTTP commands for gathering claim, crafting and boss challenge using idempotency keys.
5. Then continue with the still-undefined combat contract and live Tier II content.
