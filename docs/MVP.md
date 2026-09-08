# Mistral MVP — Technical Product Contract

## Product thesis

Mistral is a browser MMORPG IDLE centered on grind, combat, gathering, crafting and cooperative dungeon progression. Characters have races but no classes; specialization should emerge mainly from equipment and progression choices rather than a permanent class selection.

Five races are part of the MVP contract: Human, Elf, Orc, Troll and Goblin. Race modifiers must remain small enough not to become implicit classes.

The core economic invariant is that monsters do not drop ready-to-use equipment or tools. They may drop materials, ingredients, currency, keys, reagents, rare components and recipes/blueprints. Equipment and tools must be crafted.

## Core loop

```text
Gather
  -> obtain resources
  -> craft equipment
  -> become stronger
  -> enter dungeon
  -> defeat monsters
  -> obtain rare materials / boss key
  -> defeat boss
  -> unlock higher dungeon tier
  -> unlock better resources / recipes / equipment
  -> repeat
```

## Data-Driven boundary

Data-Driven does not mean moving engine logic into JSON.

Code owns invariants and mechanics: dungeon behavior, combat rules, inventory semantics, crafting rules, expiry behavior and legal operations.

JSON owns concrete game content and parameters: races, items, monsters, loot tables, recipes, dungeons, gathering areas, mounts, crops, animals, decay and balance values.

```text
JSON -> schema/semantic validation -> typed definition -> domain engine
```

The engine must never scatter ad-hoc `map[string]any`/JSON key checks throughout gameplay code.

## Game Content Release

Content is versioned and content-addressed. The loader computes a SHA-256 digest over the JSON release, and a run pins the resulting `version@sha256:...` identity. A release is immutable once a run references it.

A dungeon run records at minimum:

- content release
- dungeon id
- dungeon tier
- seed
- character snapshot
- started_at

A balancing update produces a new content release. Existing runs continue resolving against the release with which they started.

## IDLE execution model

Mistral must not keep a sleeping process/goroutine per player activity. Progress is derived from elapsed time and immutable activity state.

For a dungeon tier with a 45 second encounter interval:

```text
elapsed = now - started_at
encounters_due = floor(elapsed / encounter_interval)
```

The deterministic dungeon engine derives the encounter sequence from the run seed and selected dungeon/tier definition. Gathering uses the same principle but derives RNG independently per cycle, so claiming one hundred cycles at once or in smaller batches produces the same aggregate result and cannot duplicate already claimed cycles.

Loot materialization is also deterministic: a defeated encounter is mapped to its monster loot table using a seed derived from the pinned run seed and encounter ordinal. Combat outcome itself is intentionally not implemented as a formula yet; core exposes a `CombatResolver` port so combat rules can be defined without contaminating dungeon, loot or progression mechanics with temporary balance assumptions.

## Dungeon model

Solo and group dungeons use the same dungeon engine. Party size is an input; separate `SoloDungeonEngine` and `GroupDungeonEngine` implementations should not exist.

The MVP party target is 2–4 players. Group scaling belongs in dungeon content parameters.

Boss keys are obtained from monster loot and make the tier boss available. Defeating the boss unlocks the next dungeon tier only when that next tier exists in the pinned content release.

The exact boss-key consumption rule is not yet defined by the product contract. The boss application service therefore receives a key-consumption policy rather than hard-coding consume-on-attempt, consume-on-victory or reusable-key behavior.

## Gathering and crafting

Gathering categories in the MVP contract:

- woodcutting
- mining
- fishing
- herbalism

Crafting stations/NPC specializations:

- blacksmith
- tailor
- carpenter
- cook
- alchemist

NPCs represent production stations; they must not create materials ex nihilo.

Crafting is an atomic inventory operation: all recipe requirements are validated on a working copy, then consumed and the output is materialized. A failed craft cannot partially consume ingredients.

## Inventory and perishability

Inventory supports batches/stacks with different expiry times. A single `item_id + quantity` aggregate is insufficient for perishable items.

Implemented stack contract:

```text
InventoryStack
- item_id
- quantity
- acquired_at
- expires_at
- metadata
```

Compatible stacks may merge, while different expiry or metadata batches remain separate. Consumption prioritizes the earliest-expiring compatible batch. Multi-item consumption is atomic.

Perishable item transformation itself (for example Fresh -> Spoiled) remains future work because decay definitions are not yet present in the active content release.

## Homestead and mounts

Homestead v0 is intentionally small: two farming plots and one animal slot are sufficient to prove the system.

Mounts affect the time between encounters rather than physical movement speed. Better mounts therefore increase encounters/hour and indirectly XP/resources/hour.

## Repository architecture

The target is a modular monolith with explicit composition roots:

```text
mistral-frontend -> mistral-api -> mistral-core <- mistral-workers
content -------------------------> typed content loader/core
```

Core modules are bounded contexts. The planned contexts are:

- identity
- character
- content
- inventory
- combat
- dungeon
- party
- gathering
- crafting
- mount
- housing

Within a mature context the intended dependency direction is:

```text
domain <- application <- composition/entrypoint
infra implements ports defined inward
presentation contains transport-facing contracts but no transport framework
```

HTTP and worker hosts remain outside the core.

## Current executable vertical slice

The integration suite now executes the current release through:

```text
create Human character
  -> start Iron Mine session
  -> resolve/claim deterministic mining cycles
  -> obtain Iron Ore + Coal
  -> smelt 3 Iron Ingots
  -> craft Iron Sword
  -> start Abandoned Mine Tier I
  -> materialize deterministic encounters
  -> resolve defeated-monster loot
  -> obtain Goblin King Key
  -> challenge Goblin King through injected CombatResolver
  -> materialize boss loot
```

Two deliberate content gaps are visible instead of being silently invented:

1. `oak_handle` and a pre-dungeon source for `leather_strip` are not yet defined, although both are required by the Iron Sword recipe. The integration fixture seeds only these missing prerequisites.
2. `Abandoned Mine` currently defines Tier I only. The progression engine and tests support unlocking Tier II when Tier II exists, but the active content release does not invent an unbalanced Tier II.

## MVP scope boundaries

Included by the product proposal: account/login, one character, five races, no classes, character level, combat, solo dungeon, 2–4 player group dungeon, dungeon tiers, boss keys, four gathering disciplines, five crafting stations, craft-only equipment, consumables, expiration, basic mounts, basic homestead/farming/animals and parties.

Explicitly excluded from the first MVP: PvP, guilds, marketplace, direct P2P trading, open world, raids and quests.
