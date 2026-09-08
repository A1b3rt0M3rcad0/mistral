# Mistral

Mistral is a browser-based IDLE MMORPG built around persistent progression through gathering, crafting, combat, scalable dungeons and cooperative play.

The project is server-authoritative and Data-Driven: game engines live in code while mutable game content is declared through versioned, strictly validated contracts.

## Current bootstrap

The repository is a Go modular monolith with explicit host boundaries:

```text
mistral-frontend -> mistral-api -> mistral-core <- mistral-workers
content -------------------------> typed content loader/core
```

Implemented foundations include:

- content-addressed Game Content Releases (`version@sha256:...`);
- typed/validated races, items, recipes, loot tables, monsters, gathering areas and dungeons;
- craft-only invariant for equipment/tools;
- deterministic IDLE dungeon encounter scheduling;
- deterministic, claim-idempotent gathering;
- batch-aware inventory with per-stack expiry metadata;
- atomic multi-item consumption and crafting;
- deterministic monster/boss loot materialization;
- boss-gated dungeon progression;
- a `CombatResolver` port with no invented production combat formula;
- architecture quality gate and GitHub Actions CI.

The integration suite executes the current content release through character creation, Iron Mine gathering, smelting, Iron Sword crafting, Abandoned Mine encounters, boss-key loot and a Goblin King challenge using an explicit test combat resolver.

## Known content gaps

The active content release intentionally exposes rather than hides incomplete game design:

- `oak_handle` and a pre-dungeon acquisition path for `leather_strip` are not yet defined even though the Iron Sword recipe requires them;
- `Abandoned Mine` currently defines Tier I only, so no unbalanced Tier II content is fabricated;
- combat equations and the boss-key consumption rule remain product decisions behind explicit contracts.

See [`docs/MVP.md`](docs/MVP.md) for the technical product contract.

## Validation

```bash
go test ./...
go vet ./...
go run ./tooling/content-validator -content ./content
go run ./tooling/quality-gate -root .
```

Frontend validation is enforced in CI with Node 22 and the TypeScript/Vite production build.
