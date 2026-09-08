# Mistral

Mistral is a browser-based idle MMORPG focused on persistent progression through gathering, crafting, combat, scalable dungeons and cooperative play.

The project is intentionally built as a server-authoritative modular monolith. Game rules live in typed Go domain engines; mutable game content lives in versioned JSON contracts that are validated before use.

## Bootstrap scope

This repository currently proves the two highest-risk architectural decisions from the MVP proposal:

1. Versioned, typed and cross-validated Data-Driven content.
2. Deterministic IDLE dungeon scheduling from immutable run input (`started_at`, `seed`, character snapshot, dungeon/tier and content release).

Combat formulas, persistence, authentication and the complete economy are deliberately not invented yet because the product proposal does not define their contracts sufficiently.

## Repository layout

```text
packages/
  mistral-core/       Game domain and application modules
  mistral-api/        HTTP composition root
  mistral-workers/    Async composition root (reserved; no permanent player loops)
  mistral-frontend/   Browser client
content/               Versioned game content
 tooling/
  content-validator/  Content build validation
  quality-gate/       Architecture dependency enforcement
```

## Run

```bash
go test ./...
go run ./tooling/content-validator -content ./content
go run ./tooling/quality-gate -root .
go run ./packages/mistral-api/cmd/mistral-api -content ./content
```

The API exposes:

- `GET /healthz`
- `GET /api/v1/content/release`

## Architectural invariants

- The browser is untrusted.
- `mistral-core` cannot depend on API, workers or frontend.
- Domain packages cannot import infrastructure, composition or presentation packages.
- Monster loot cannot directly contain equipment or tools.
- A dungeon run is pinned to one content release.
- IDLE progression is derived from time and immutable state; it is not implemented as one sleeping loop per player.
