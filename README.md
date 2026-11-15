# CatchKeep

A self-hosted web app that recommends which Pokémon GO Pokémon to keep or transfer to reach a storage target. All transfers happen manually in the game.

## Status

Specification phase. See [`spec/`](spec/):

- [`developer-spec.md`](spec/developer-spec.md) — implementation contract: architecture, data model, planner, API, security, error handling, testing plan, open decisions.

## Development

Requires Node.js 24 (`.nvmrc`) and Go (version pinned in `backend/go.mod`).

```sh
npm install   # also installs git hooks
npm run check # formatting, markdown lint, shellcheck, OpenAPI lint
make -C backend check # gofmt/goimports, go vet, golangci-lint, go test -race
```

### Workflow

Ship each change with `scripts/ship.sh <branch> "<type(scope): subject>" "<PR description>"` after staging it. It refuses to start unless `main` is current and green, runs the local gates, opens the PR, waits for CI, squash-merges exactly the commit CI tested, and waits for `main` to be green again.

- Small, focused commits following [Conventional Commits](https://www.conventionalcommits.org/).
- All changes land on `main` through pull requests; `main` is protected.
- Quality gates run locally (pre-commit and commit-msg hooks) and in CI on every PR. A commit must pass them before the next one starts.
