# CatchKeep

A family web app, hosted on a MacBook and used from iPhone Safari on home Wi-Fi, that recommends which Pokémon GO Pokémon to keep or transfer to reach a storage target. All transfers happen manually in the game.

## Status

Specification phase. See [`spec/`](spec/):

- [`brainstorming-handoff.md`](spec/brainstorming-handoff.md) — product handoff from brainstorming.
- [`developer-spec.md`](spec/developer-spec.md) — implementation contract: architecture, data model, planner, API, security, error handling, testing plan, open decisions.

## Development

Requires Node.js 24 (`.nvmrc`).

```sh
npm install   # also installs git hooks
npm run check # formatting + markdown lint
```

### Workflow

- Small, focused commits following [Conventional Commits](https://www.conventionalcommits.org/).
- All changes land on `main` through pull requests; `main` is protected.
- Quality gates run locally (pre-commit and commit-msg hooks) and in CI on every PR. A commit must pass them before the next one starts.
