# CatchKeep

A Pokémon GO storage helper.

## What you can do today

- Give every trainer in the house their own PIN-protected profile, so collections never get mixed up.

## Status

Early development. The plan is in [`spec/developer-spec.md`](spec/developer-spec.md).

## Development

Requires Node.js 24 (`.nvmrc`) and Go (version pinned in `backend/go.mod`).

```sh
npm install   # also installs git hooks
npm run check # formatting, markdown lint, shellcheck, OpenAPI lint
make -C backend check # gofmt/goimports, go vet, golangci-lint, go test -race
npm --prefix frontend install && npm --prefix frontend run check # tsc, oxlint, vitest
scripts/smoke.sh backend/bin/catchkeep frontend/dist # after building both: real binary + bundle
```

### Workflow

Ship each change with `scripts/ship.sh <branch> "<type(scope): subject>" "<PR description>"` after staging it. It refuses to start unless `main` is current and green, runs the local gates, opens the PR, waits for CI, squash-merges exactly the commit CI tested, and waits for `main` to be green again.

- Small, focused commits following [Conventional Commits](https://www.conventionalcommits.org/).
- All changes land on `main` through pull requests; `main` is protected.
- Quality gates run locally (pre-commit and commit-msg hooks) and in CI on every PR. A commit must pass them before the next one starts.
