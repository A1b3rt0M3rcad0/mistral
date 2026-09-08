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
- Public race-creation catalog derived from the active release and returned in deterministic id order together with `release_id`.

### Character

- Race-backed character creation.
- Level initialized at 1.
- Explicit per-dungeon unlocked-tier progression.
- Idempotent progression advancement.
- Public character registration no longer accepts a client-selected internal id.
- The API host generates opaque 128-bit character ids with `crypto/rand`.
- Character-id generation happens only after the idempotency claim is acquired; replay returns the original stored id and does not call the generator again.
- Authenticated subjects can rediscover their generated character ids through ownership-backed character listing.

### Identity / ownership boundary

- Authentication mechanism/provider remains deliberately unspecified.
- The HTTP host exposes a `PrincipalResolver` port that resolves a request into a `subject_id`.
- No gameplay payload is allowed to choose or override `subject_id`.
- Core models the durable ownership fact `subject_id -> character_id`.
- A character can have at most one owner; one subject may own multiple characters.
- Ownership repositories support both `ByCharacter` authorization lookup and `BySubject` owned-character discovery.
- PostgreSQL `BySubject` uses the existing `character_ownerships(subject_id)` index and returns deterministic character-id order.
- Ownership authorization returns forbidden for missing or mismatched ownership instead of exposing ownership existence.
- In-memory and PostgreSQL ownership repositories exist.
- PostgreSQL ownership rows reference `characters(id)` and are deleted with the character.
- Character-scoped HTTP reads and mutations execute the principal + ownership guard before reading or mutating authoritative state.
- Character-list discovery derives its subject exclusively from the trusted principal; the client cannot request another subject's list.
- The default `mistral-api` composition currently does not install a concrete `PrincipalResolver`, so protected routes fail closed with `401` until an authentication adapter is selected.

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
- Gathering idempotency keys are scoped per session, so unrelated sessions can reuse the same external key without collision.
- Tests explicitly distinguish same-session key reuse conflicts from legitimate cross-session key reuse.

### Crafting

- Recipe/station validation.
- Multi-craft support.
- Inventory decay is resolved before ingredient consumption.
- Crafted output receives `expires_at` from the content decay contract when applicable.
- Atomic ingredient consumption/output materialization in the pure use case.
- Persisted crafting command with command-level idempotency and optimistic inventory versioning.
- Replayed craft commands return the stored result without consuming ingredients or creating output again.
- Craft idempotency keys are scoped per character.

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
- Boss idempotency keys are scoped per character.
- Dungeon reward/boss HTTP mutation endpoints remain intentionally unexposed until authoritative combat/outcome rules are complete.

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
- PostgreSQL runner supports both point queries and multi-row queries while preserving transaction-bound execution.
- PostgreSQL JSONB aggregate store with relational identity and monotonic version columns.
- PostgreSQL repository adapters for Character, Inventory, Gathering Session and Dungeon Run.
- PostgreSQL driver selection remains outside Core and is registered by the API host.
- `mistral-migrate` applies ordered SQL migrations under a PostgreSQL advisory transaction lock.
- Live PostgreSQL 17 is part of backend CI and validates migration/repository behavior, including reverse ownership lookup.
- Command idempotency ledger with `(scope, idempotency_key)` identity, request hashes, in-progress/completed states and stored replay responses.
- In-memory and PostgreSQL ledger adapters.
- Same key + different request hash within the same resource scope is rejected instead of silently reusing a command identity.
- Resource-scoped idempotency avoids global collisions between unrelated players/sessions.
- Atomic persisted character registration creates Character + Inventory + Ownership + idempotency result in one transaction.
- `/readyz` reports database readiness when persistence is configured.

The intended delivery semantic is not “exactly once”. The design supports replay-safe command processing under an at-least-once transport model when the idempotency ledger mutation and gameplay side effects execute inside the same database transaction.

Optimistic concurrency, transaction atomicity and idempotency remain separate mechanisms:

```text
optimistic version -> prevents lost updates
transaction         -> atomically commits/rolls back related writes
idempotency ledger  -> deduplicates retried commands and replays the original result
```

Aggregate state is deliberately stored as JSONB in the first PostgreSQL schema while identity, versions, ownership, idempotency keys and transaction semantics remain relational. This keeps the initial persistence schema tolerant of domain evolution without weakening concurrency guarantees.

### HTTP host surface

Current routes:

```text
GET  /healthz
GET  /readyz
GET  /api/v1/content/release
GET  /api/v1/content/races
GET  /api/v1/characters
GET  /api/v1/characters/{characterID}
GET  /api/v1/characters/{characterID}/inventory
POST /api/v1/characters
POST /api/v1/characters/{characterID}/crafts
```

`GET /api/v1/content/races` is a public creation catalog tied to the active immutable content release. It exposes only race definitions required for creation and does not expose loot, dungeon or recipe tables.

`GET /api/v1/characters` resolves the authenticated `subject_id`, looks up only that subject's ownership rows and then loads those authoritative character aggregates. This closes the discovery loop created by server-generated character ids without allowing arbitrary subject lookup.

Character registration and crafting are server-authoritative and replay-safe. `POST /api/v1/characters` accepts only the race choice; `subject_id` comes from the authenticated principal and `character_id` is generated by the server. Crafting derives the character from the URL after ownership authorization and requires `Idempotency-Key`.

The individual character and inventory read routes are ownership-protected and never read authoritative character/inventory state before authorization succeeds.

### Architecture enforcement

- Core cannot depend on API, Workers or Frontend hosts.
- Domain cannot depend on application or outward layers.
- Application cannot depend on infra, composition, presentation or entrypoint.
- Infra cannot depend upward on composition, presentation or entrypoint.
- Quality-gate tests cover allowed and forbidden dependency directions.

## Proven executable path

The integration/core tests now prove game flow, replay behavior, ownership and live PostgreSQL persistence:

```text
authenticated subject boundary
-> transactional Human character + Inventory + Ownership registration
-> server-assigned character id
-> ownership-backed character rediscovery
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

The principal boundary and ownership authorization are implemented, but a concrete authentication adapter is intentionally still open. The host must eventually resolve an external request to a trusted `subject_id` through a selected mechanism (session/JWT/OIDC/upstream identity/etc.). Core must remain independent of that choice.

### Gathering session concurrency

The core can model independent gathering sessions, but the product contract does not yet define whether a character may have multiple simultaneous active gathering activities. The API therefore does not expose a public “start gathering” mutation yet; exposing it without a concurrency rule would allow the HTTP surface to invent an economy rule.

## Remaining engineering work

- Select and implement a concrete authentication adapter at the API-host boundary; protected routes already fail closed without it.
- Define the active-gathering concurrency/session policy before exposing gathering-session creation.
- Add the authenticated gathering-claim HTTP command once session creation/lifecycle is authoritative.
- Add an authoritative persisted dungeon encounter outcome/checkpoint before exposing encounter reward materialization over HTTP.
- Define the production combat contract and boss-key consumption rule before exposing real boss combat.
- Add live Tier II content only after difficulty/reward rules are specified.
- Add concrete perishable content when the actual food/farming/fishing balance is defined; current active release contains decay capability but does not fabricate balance data.

## Next engineering slice

1. Keep the current protected HTTP surface fail-closed and choose the authentication adapter separately from Core.
2. Define gathering session exclusivity/concurrency, then expose start + claim as one coherent server-authoritative flow.
3. Introduce a persisted authoritative dungeon encounter outcome/checkpoint before any reward HTTP endpoint.
4. Then continue with combat/equipment and Tier II once their game rules are defined.
