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
- Atomic inventory materialization.

### Crafting

- Recipe/station validation.
- Multi-craft support.
- Atomic ingredient consumption/output materialization.

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

- Account/authentication contract.
- Character repository/persistence contract.
- Inventory persistence and transaction boundary.
- Gathering session persistence.
- Dungeon run persistence and materialization cursor.
- Concurrency/idempotency strategy for multi-request claims.

## Next engineering slice

The next high-leverage slice is persistence around the already-tested aggregates, not additional game systems. Recommended order:

1. Define repository ports and optimistic-concurrency/version semantics.
2. Add PostgreSQL adapters for character, inventory and activity/run state.
3. Add an application transaction boundary for claim/craft/reward operations.
4. Expose server-authoritative HTTP commands only after those writes are atomic and idempotent.
5. Then define the real combat contract and live Tier II content.
