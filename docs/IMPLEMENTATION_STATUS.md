# Mistral implementation status

This document tracks the engineering state of the MVP bootstrap without treating undefined product decisions as implemented game rules.

## Implemented

### Content runtime

- Strict JSON loading with unknown-field rejection.
- Duplicate-id rejection.
- Cross-reference validation.
- Content-addressed release identity (`version@sha256:<digest>`).
- Craft-only invariant preventing monster/gathering drops of ready equipment or tools.
- Data-driven decay definitions with reference validation and cycle rejection.

### Character

- Race-backed character creation.
- Level initialized at 1.
- Explicit per-dungeon unlocked-tier progression.
- Idempotent progression advancement.

### Identity / ownership boundary

- Authentication mechanism remains deliberately unspecified.
- Core now models only the durable ownership fact: `subject_id -> character_id`.
- A character can have at most one owner; one subject may own multiple characters.
- Ownership authorization returns forbidden for missing or mismatched ownership instead of exposing ownership existence.
- In-memory and PostgreSQL ownership repositories exist.
- PostgreSQL ownership rows reference `characters(id)` and are deleted with the character.
- Host gameplay composition includes the ownership repository.

### Inventory

- Batch-aware stacks.
- Independent expiry metadata per stack.
- Metadata-aware stack compatibility.
- Earliest-expiry-first consumption.
- Atomic multi-item consumption.
- Perishable batches transform through decay rules instead of disappearing.

### Gathering

- Elapsed-time session model.
- Deterministic RNG per cycle.
- Claim cursor (`claimed_cycles`) preventing duplicate rewards.
- Claim results independent of claim batching.
- Reward batches retain their actual production time.
- Offline claim does not freeze perishability; expired batches can already transform by claim time.
- Persisted claim command updates Gathering Session + Inventory in one transaction boundary.
- Command replay returns the stored original result and does not materialize resources again.

### Crafting

- Recipe/station validation.
- Multi-craft support.
- Inventory decay is resolved before ingredient consumption.
- Crafted output receives `expires_at` from the content decay contract when applicable.
- Atomic ingredient consumption/output materialization in the pure use case.
- Persisted crafting command with command-level idempotency and optimistic inventory versioning.
- Replayed craft commands return the stored result without consuming ingredients or creating output again.

### Dungeon

- One engine for solo/group party sizes.
- Deterministic encounter scheduling.
- Content-release pinning.
- Deterministic loot for defeated encounters.
- Normal encounter reward materialization is replay-safe by deterministic `(run_id, encounter_ordinal)` command identity.
- Encounter identity is derived server-side from the pinned run rather than trusting a client-supplied monster id.
- Boss-key gating.
- Inventory decay is resolved before boss-key validation/consumption.
- Boss and normal encounter loot receive perishable expiry metadata when configured.
- Boss progression unlocks a next tier only if that tier is present in the content release.
- Persisted boss command spans Character + Inventory + idempotency ledger in one transaction boundary.
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
- PostgreSQL driver selection remains outside Core and is registered by the API host.
- `mistral-migrate` applies ordered SQL migrations under a PostgreSQL advisory transaction lock.
- Live PostgreSQL 17 is part of backend CI and validates migration/repository behavior.
- Command idempotency ledger with `(scope, idempotency_key)` identity, request hashes, in-progress/completed states and stored replay responses.
- In-memory and PostgreSQL ledger adapters.
- Same key + different request hash is rejected instead of silently reusing a command identity.
- `/readyz` reports database readiness when persistence is configured.

The intended delivery semantic is not “exactly once”. The design supports replay-safe command processing under an at-least-once transport model when the idempotency ledger mutation and gameplay side effects execute inside the same database transaction.

Optimistic concurrency, transaction atomicity and idempotency remain separate mechanisms:

```text
optimistic version -> prevents lost updates
transaction         -> atomically commits/rolls back related writes
idempotency ledger  -> deduplicates retried commands and replays the original result
```

Aggregate state is deliberately stored as JSONB in the first PostgreSQL schema while identity, versions, ownership, idempotency keys and transaction semantics remain relational. This keeps the initial persistence schema tolerant of domain evolution without weakening concurrency guarantees.

### Architecture enforcement

- Core cannot depend on API, Workers or Frontend hosts.
- Domain cannot depend on application or outward layers.
- Application cannot depend on infra, composition, presentation or entrypoint.
- Infra cannot depend upward on composition, presentation or entrypoint.
- Quality-gate tests cover allowed and forbidden dependency directions.

## Proven executable path

The integration/core tests now prove game flow, replay behavior and live PostgreSQL persistence:

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
-> replay-safe encounter loot materialization
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

### Authentication

Ownership persistence is now defined, but authentication is intentionally still open. The host still needs a concrete mechanism that resolves an external request to an authenticated `subject_id` (for example session/JWT/OIDC/etc.). Core should not choose that mechanism.

## Remaining engineering work

- Add the API authentication/principal resolver port and ownership guard around mutable character-scoped routes.
- Add atomic persisted character registration so Character + Inventory + Ownership are created together.
- Expose server-authoritative gathering/crafting/dungeon commands only behind authenticated ownership checks and idempotency keys.
- Define the production combat contract before exposing real boss combat.
- Add live Tier II content only after difficulty/reward rules are specified.
- Add concrete perishable content when the actual food/farming/fishing balance is defined; current active release contains decay capability but does not fabricate balance data.

## Next engineering slice

1. Add the host authentication principal boundary without selecting a provider.
2. Implement persisted character registration with Character + Inventory + Ownership in one transaction.
3. Put mutable HTTP commands behind principal + ownership authorization.
4. Then continue with combat/equipment and Tier II once their game rules are defined.
