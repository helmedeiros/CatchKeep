# CatchKeep

A Pokémon GO storage helper.

## What you can do today

- Give each trainer their own PIN-protected profile, so collections never get mixed up.
- Bring in your collection from a Poke Genie CSV export: see a preview first, then browse it strongest first. Anything the export can't tell (shiny, costume, background, Dynamax) stays "unknown" instead of being assumed absent.
- Keep it current by importing a newer export: Pokémon you already have are updated (new CP after a power-up, for example) instead of counted twice. CatchKeep never merges on its own. Exact matches are confirmed in one tap, and anything less certain is shown side by side for you to decide.
- Fill in what the export can't tell, one species at a time: "None of my 34 Pidgey are shiny, costumed or have a background." Only unknowns change; anything already known is kept.
- Tell CatchKeep how much storage you have and what you keep Pokémon for (collecting, trading, battling). It shows how many Pokémon your keep rules hold on to and how many are possible transfers, species by species. The best copies of each variant are kept, with more of legendary, mythical and Ultra Beast species. Lucky, shadow and purified copies count separately, so they are never traded away in place of ordinary ones, and anything not yet confirmed is never counted as a transfer. Tap a species to see each copy with its picture, marked keep or possible transfer, and set your own rule: keep at least N, or mark it a favorite to keep every copy. Protect any single Pokémon and it is always kept, even over a stronger copy. Group Pokémon into named teams (a raid team, your buddies) and every member is always kept.

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
