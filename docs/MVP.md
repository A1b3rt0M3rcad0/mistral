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

The deterministic engine then derives the encounter sequence from the run seed and the selected dungeon/tier definition.

This bootstrap intentionally stops at deterministic encounter scheduling. Victory/defeat, damage, XP and loot materialization remain undefined until combat and progression contracts are specified.

## Dungeon model

Solo and group dungeons use the same dungeon engine. Party size is an input; separate `SoloDungeonEngine` and `GroupDungeonEngine` implementations should not exist.

The MVP party target is 2–4 players. Group scaling belongs in dungeon content parameters.

Boss keys are obtained from monster loot and make the tier boss available. Defeating the boss unlocks the next dungeon tier.

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

## Inventory and perishability

Inventory must support batches/stacks with different expiry times. A single `item_id + quantity` aggregate is insufficient for perishable items.

Conceptually:

```text
InventoryStack
- item_id
- quantity
- created_at
- expires_at
- metadata
```

Perishable items transform rather than simply disappear (for example Fresh -> Spoiled), allowing spoiled outputs to feed other recipes later.

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

## MVP scope boundaries

Included by the product proposal: account/login, one character, five races, no classes, character level, combat, solo dungeon, 2–4 player group dungeon, dungeon tiers, boss keys, four gathering disciplines, five crafting stations, craft-only equipment, consumables, expiration, basic mounts, basic homestead/farming/animals and parties.

Explicitly excluded from the first MVP: PvP, guilds, marketplace, direct P2P trading, open world, raids and quests.
