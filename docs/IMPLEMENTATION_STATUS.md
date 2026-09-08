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
- Atomic inventory materialization in the pure use case.

### Crafting

- Recipe/station validation.
- Multi-craft support.
- Atomic ingredient consumption/output materialization in the pure use case.

### Dungeon

- One engine for solo/group party sizes.
- Deterministic encounter scheduling.
- Content-release pinning.
- Deterministic loot for defeated encounters.
- Boss-key gating.
- Boss progression that unlocks a next tier only if that tier is present in the content release.

### Combat boundary

- `CombatResolver` application port.
- Disabled production resolver rather than a fabricated combat equation.
- Test resolver used only to prove orchestration.

### Persistence boundary

- Generic versioned persistence record (`Record[T]`).
- Explicit `ErrNotFound`, `ErrAlreadyExists` and optimistic `ErrConflict` semantics.
- Repository ports for Character, Inventory, Gathering Session and Dungeon Run.
- `Save(value, expectedVersion)` contract for compare-and-swap style writes.
- Thread-safe in-memory adapters used to validate the repository contract before introducing a database.
- Tests proving stale writes are rejected instead of silently overwriting newer state.
- `Transactor` application-facing port for atomic work spanning multiple repositories.
- Explicit separation between transaction atomicity, optimistic concurrency and command-level idempotency.

This is intentionally not yet a database implementation. The purpose of the current layer is to freeze persistence and transaction semantics so a PostgreSQL adapter has precise contracts to implement.

### Architecture enforcement

- Core cannot depend on API, Workers or Frontend hosts.
- Domain cannot depend on application or outward layers.
- Application cannot depend on infra, composition, presentation or entrypoint.
- Infra cannot depend upward on composition, presentation or entrypoint.
- Quality-gate tests cover allowed and forbidden dependency directions.

## Proven executable path

The integration suite executes:

```text
Human character
-> Iron Mine session
-> deterministic Iron Ore / Coal claim
-> 3 Iron Ingots
-> Iron Sword
-> Abandoned Mine Tier I
-> deterministic Goblin / Cave Spider encounters
-> deterministic loot
-> Goblin King Key
-> Goblin King challenge
-> boss loot
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

### Persistence and identity

Repository, optimistic-concurrency and transaction-port semantics are now defined. Remaining work is:

- Account/authentication contract.
- Concrete PostgreSQL implementation of the `Transactor` and versioned repository ports.
- Database migrations/schema.
- Idempotency keys or equivalent command-deduplication semantics at the host boundary.
- Persisted gathering claim/crafting/reward commands using one transaction.
- Dungeon materialization cursor/checkpoint semantics once defeated-encounter persistence is exposed.

## Next engineering slice

The next high-leverage slice is the concrete transactional persistence implementation around the already-tested aggregates:

1. Specify command-level idempotency semantics independently from optimistic concurrency.
2. Add PostgreSQL schema/migrations and adapters for Character, Inventory, Gathering Session and Dungeon Run.
3. Implement the PostgreSQL `Transactor` so multi-repository writes share one database transaction.
4. Implement atomic persisted gathering-claim/crafting/reward commands.
5. Expose server-authoritative mutable HTTP commands only after those writes are transactional and replay-safe.
6. Then define the real combat contract and live Tier II content.
