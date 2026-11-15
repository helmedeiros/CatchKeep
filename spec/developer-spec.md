# CatchKeep — Pokémon GO Storage Planner

## Developer Specification · v1.0 · 15 November 2026

Source: [`brainstorming-handoff.md`](brainstorming-handoff.md) (brainstorming handoff, 15 Nov 2026). This document is the implementation contract. Where the handoff left something open, this spec either proposes a default (marked **[Proposed]**) or keeps it open (marked **[Open]**, see §16). Everything unmarked comes from the handoff and is **[Decided]**.

Keywords MUST / SHOULD / MAY follow RFC 2119. Requirement IDs (`V1-IMP-03`) are stable; reference them in commits, tests and issues.

---

## Contents

1. Product summary and scope
2. Glossary
3. Releases and exit criteria
4. Architecture
5. Data model
6. V1 functional requirements
7. Cleanup planner specification
8. API surface
9. Security and authentication
10. Error handling strategy
11. V2 · Screenshot import
12. V3 · Battle recommendations
13. Backup, restore and migrations
14. Non-functional requirements
15. Testing plan
16. Open decisions
17. Implementation sequence

---

## 1. Product summary and scope

A standalone family web app, hosted on a MacBook and used from iPhone Safari over home Wi-Fi, that recommends which Pokémon each trainer should keep or transfer to reach a storage target.

### Principles

- **The app never touches the game.** All transfers happen manually in Pokémon GO. The app records only what the trainer confirms happened.
- **Unknown is not absent.** Missing or unsupported data is `unknown`. If resolving an unknown could change a keep decision, the Pokémon is protected until the trainer confirms it.
- **Never auto-merge.** Possible duplicates are always shown to the trainer.
- **Every release stands alone** and carries profiles, rules, collections and history forward.
- **Conservative under uncertainty.** Ambiguity, invalid data, stale plans and failed refreshes can never justify a transfer.

### In scope

V1 CSV cleanup, V2 screenshot import, V3 battle recommendations (§3).

### Out of scope (all releases unless reopened)

- Access away from home (future release).
- Any integration that reads from or writes to Pokémon GO accounts or servers.
- Automating transfers, unfavoriting or tagging in the game.
- Non-English game UI for searches (V1) and screenshots (V2).
- Native iOS app. Safari is the only required client.
- Cloud processing of screenshots. OCR stays on the MacBook even after remote access is added, unless that decision is explicitly changed.

---

## 2. Glossary

| Term              | Meaning                                                                                                                                                  |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Trainer / profile | One family member's isolated space: collection, capacity, rules, teams, history.                                                                         |
| Home admin        | The person operating the MacBook host. Can issue PIN reset codes.                                                                                        |
| Record            | One Pokémon in the app, with a stable app ID.                                                                                                            |
| Observation       | One value for one field of one record, with source and timestamp.                                                                                        |
| Variant           | Distinct combination of species, form, shiny, costume, background, Dynamax/Gigantamax.                                                                   |
| Floor / minimum   | Number of copies a rule requires to be kept.                                                                                                             |
| Effective minimum | The floor actually applied to a group after defaults, overrides and overlap resolution.                                                                  |
| Protected         | Ineligible for transfer (manual protection, favorite species, team, unresolved unknown, pending duplicate review, accepted upgrade, recommended roster). |
| In-game Favorite  | The star in Pokémon GO. A separate attribute; it does **not** protect.                                                                                   |
| Occupied count    | Pokémon in storage: trainer-reported in-game count if supplied, otherwise imported count (labeled as estimate).                                          |
| Plan              | A computed cleanup recommendation bound to an input revision.                                                                                            |
| Stale plan        | A plan whose input revision no longer matches the profile's current revision.                                                                            |

---

## 3. Releases and exit criteria

| Release                     | Family workflow                                                                        | Exit criteria                                                                                   |
| --------------------------- | -------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| V1 · CSV cleanup            | Import, set keep rules, protect Pokémon and teams manually, plan and confirm transfers | A trainer can reach a feasible cleanup target and undo mistaken app confirmations               |
| V2 · Screenshot import      | Import batches, review uncertain readings, update existing records                     | English list, detail and appraisal screenshots work locally on the MacBook                      |
| V3 · Battle recommendations | Manage recommended rosters and upgrades                                                | Recommendations refresh from versioned battle data and use confirmed builds for roster coverage |

Each release MUST pass the full acceptance suite of all prior releases (§15.11) and MUST restore backups made by every prior release (§13).

---

## 4. Architecture

Two independent applications that share only a versioned HTTP contract:

- **Backend** — Go, hexagonal (ports and adapters). Owns all domain logic, persistence, auth, planning, imports and battle data.
- **Frontend** — a standalone web client. Holds no domain rules; renders what the API returns.

They have separate source trees, toolchains, dependencies, quality gates, CI jobs and release versions. Neither imports code from the other. The only shared artifact is the OpenAPI document `api/openapi.yaml`.

### 4.1 Topology

```text
 iPhone Safari (×N)
        │  HTTPS over home Wi-Fi (one URL, one certificate)
        ▼
 MacBook host
 ┌────────────────────────────────────────────────────────────┐
 │ catchkeep (Go binary)                                      │
 │   ├─ /api/v1/*      profile-scoped JSON API                │
 │   └─ /*             static files from a configured dir     │
 │                     (the frontend's build output)          │
 │   SQLite (WAL) · image store (V2) · OCR sidecar (V2)       │
 │   Admin console (loopback only)                            │
 └────────────────────────────────────────────────────────────┘
```

- The backend serves the frontend bundle from a directory set at runtime (`CATCHKEEP_WEB_DIR`). There is no build-time coupling: either side can be rebuilt and redeployed alone. Serving both from one origin keeps one URL and one TLS certificate on the iPhones and avoids CORS.
- In development the frontend runs on its own dev server and proxies `/api` to a backend, or to a mock server generated from `api/openapi.yaml`.
- Start manually by default; offer an opt-in "launch at Mac login" (**[Proposed]** a per-user `launchd` LaunchAgent installed/removed from the admin console).
- The host must be on and awake. The client MUST detect an unreachable host and say so plainly (§10.5).

### 4.2 API contract

- `api/openapi.yaml` (OpenAPI 3.1) is the source of truth for every route, request, response and error code (§8, §10.2).
- The API is versioned in the path (`/v1`). Breaking changes require `/v2`; additive changes do not.
- Backend: server interfaces and DTOs generated from the contract; handlers adapt DTOs to use cases.
- Frontend: typed client generated from the contract.
- CI checks that generated code is up to date on both sides and runs a breaking-change diff of the contract against `main`.
- A contract change lands in its own PR before either side implements it.

### 4.3 Backend stack **[Proposed]**

| Concern          | Choice                                                                                 | Rationale / tradeoff                                                                                                                                                                                    |
| ---------------- | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language         | Go (latest stable)                                                                     | Single static binary on the MacBook; strong standard library; easy concurrency for background jobs.                                                                                                     |
| HTTP             | `net/http` (method-pattern routing) + `oapi-codegen` strict server                     | No framework; contract-first.                                                                                                                                                                           |
| Database         | SQLite in WAL mode via `modernc.org/sqlite`                                            | Transactional, single file, trivial backup; pure Go driver.                                                                                                                                             |
| Queries          | `sqlc` over hand-written SQL                                                           | Type-safe queries; transactions stay explicit.                                                                                                                                                          |
| Migrations       | `goose`, SQL files embedded with `embed`                                               | Forward-only, numbered, tested (§13).                                                                                                                                                                   |
| Planner solver   | HiGHS MILP via its C API (cgo) behind a `Solver` port, plus a pure-Go greedy fallback  | Exact optimum in practice for 2,700 records; greedy gives a verified lower bound. Tradeoff: cgo complicates cross-compilation — acceptable since the only target is macOS. Must be benchmarked (§14.1). |
| Password hashing | Argon2id (`golang.org/x/crypto/argon2`)                                                | Standard for low-entropy secrets like PINs.                                                                                                                                                             |
| Logging          | `log/slog` JSON handler                                                                | Structured, standard library.                                                                                                                                                                           |
| OCR (V2)         | Apple Vision text recognition via a small Swift CLI sidecar behind an `OcrEngine` port | Local, strong English accuracy on macOS. Pending validation (§16).                                                                                                                                      |
| Local HTTPS      | Local CA (e.g. `mkcert`) + root profile installed on each iPhone, mDNS hostname        | Safari requires full-trust installation; validate on family iPhones (§16).                                                                                                                              |

### 4.4 Frontend stack **[Proposed]**

| Concern           | Choice                                 | Rationale / tradeoff                                |
| ----------------- | -------------------------------------- | --------------------------------------------------- |
| Language          | TypeScript (strict)                    | Typed client generated from the contract.           |
| Framework / build | React + Vite                           | Static bundle; no server runtime needed.            |
| API client        | `openapi-typescript` + `openapi-fetch` | Types come from the contract; no hand-written DTOs. |
| UI libraries      | See §4.6                               |                                                     |

### 4.5 Backend hexagonal layout

The domain core has no I/O and imports nothing outside the standard library. Dependencies point inward: `adapters → app → domain`. Ports are interfaces defined by the consumer (domain or app), implemented by adapters.

```text
backend/
  cmd/catchkeep/         # main: config, wiring (composition root), server start
  internal/
    domain/              # pure: entities, variant keys, rules engine, planner model, roster model
      rules/
      planner/
      reconciliation/
      battle/            # V3
    app/                 # use cases; own transactions and revision checks; define driven ports
    ports/               # driven ports: repositories, Clock, IDGenerator, Solver, OcrEngine,
                         # BattleSource, Mailer, Hasher, UnitOfWork
    adapters/
      httpapi/           # driving: generated server, auth middleware, DTO ↔ use-case mapping
      sqlite/            # repositories, migrations, UnitOfWork
      csvimport/         # Poke Genie + generic mapped-CSV adapters
      ocr/               # V2 Vision sidecar
      battlesource/      # V3, one adapter per source
      solver/            # HiGHS (cgo) + greedy fallback
      mailer/            # SMTP (optional, OD-07)
      clock/, ids/       # system implementations
  go.mod
frontend/
  src/
  package.json
api/
  openapi.yaml
```

Rules:

- Every repository method takes a `TrainerID`. There is no unscoped query path for trainer data.
- Use cases are the only place transactions begin. One use case = one transaction (`UnitOfWork`).
- The planner is a pure function `Plan(input, solver, budget) (PlanResult, error)`.
- Import adapters produce `[]ImportCandidate` (normalized observations). They never write to the database.
- Layering is enforced in CI (`go-arch-lint` or `depguard` rules): `domain` imports only the standard library; `app` imports `domain` and `ports`; only `cmd` imports `adapters`.
- Every driven port has an in-memory fake in `internal/adapters/fake` used by use-case tests.

## 5. Data model

### 5.1 Conventions

- IDs: ULID strings, generated by the app. Never derived from source data.
- Timestamps: UTC ISO-8601 with milliseconds.
- Tri-state attributes use `true | false | null`, where `null` = unknown. Code MUST NOT coerce `null` to `false`.
- Every trainer-owned table has `trainer_id NOT NULL` and a composite index starting with it.
- Each trainer has a monotonically increasing `revision`. Every mutation that can affect recommendations increments it in the same transaction.

### 5.2 Entities

#### Trainer

| Field              | Type  | Notes                                                            |
| ------------------ | ----- | ---------------------------------------------------------------- |
| id                 | ULID  |                                                                  |
| display_name       | text  | Shown on every import/confirm screen.                            |
| email              | text? | For reset codes.                                                 |
| pin_hash           | text  | Argon2id.                                                        |
| storage_capacity   | int   | Current in-game capacity.                                        |
| reported_count     | int?  | Trainer-entered in-game storage count, with `reported_count_at`. |
| goals              | json  | Ordered subset of `collecting`, `trading`, `battling`.           |
| preferences        | json  | Advanced priorities (rarity, IV, battle usefulness weights).     |
| revision           | int   | §5.1.                                                            |
| setup_completed_at | ts?   |                                                                  |

#### PokemonRecord

| Field            | Type  | Notes                                                    |
| ---------------- | ----- | -------------------------------------------------------- |
| id               | ULID  | Stable for life, survives updates, refresh, undo.        |
| trainer_id       | ULID  |                                                          |
| status           | enum  | `active`, `transferred`, `removed_by_refresh`, `merged`. |
| manual_protected | bool  | Trainer-set.                                             |
| merged_into      | ULID? | Set when a duplicate review concludes "same".            |
| created_batch_id | ULID  |                                                          |

Current field values are **not** columns on the record. They are resolved from observations (below) via a view/materialized projection `record_current` refreshed in the same transaction.

#### FieldObservation

| Field         | Type  | Notes                                                                                                                                                                                                                               |
| ------------- | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id            | ULID  |                                                                                                                                                                                                                                     |
| record_id     | ULID  |                                                                                                                                                                                                                                     |
| field         | enum  | `species`, `form`, `dex_number`, `cp`, `hp`, `iv_atk`, `iv_def`, `iv_sta`, `fast_move`, `charged_move_1`, `charged_move_2`, `in_game_favorite`, `catch_date`, `shiny`, `costume`, `background`, `dynamax`, `gigantamax`, `pvp_*`, … |
| value         | json  | `null` allowed only for "observed unknown".                                                                                                                                                                                         |
| source        | enum  | `csv`, `screenshot`, `manual`.                                                                                                                                                                                                      |
| source_ref    | text  | Batch ID + row number or image ID + region.                                                                                                                                                                                         |
| observed_at   | ts    | When the source captured it (CSV export time or screenshot time if known; else import time).                                                                                                                                        |
| recorded_at   | ts    | When the app stored it.                                                                                                                                                                                                             |
| confidence    | real? | OCR only.                                                                                                                                                                                                                           |
| superseded_by | ULID? |                                                                                                                                                                                                                                     |

Resolution rule for the current value of a field: latest `manual` observation wins; otherwise the latest `observed_at` among non-superseded observations. Manual corrections MUST survive later imports; an import that conflicts with a manual value is recorded but does not override it, and the conflict is shown on the record.

**ImportBatch**: `id, trainer_id, kind (csv|screenshot), mode (incremental|full_refresh), status (staged|committed|discarded), source_filename, column_mapping json, stats json, created_at, committed_at`.

**ImportRow / StagedChange**: per staged item, `action (add|update|possible_match|missing_from_refresh|invalid)`, parsed observations, validation messages, resolution.

**DuplicateReview**: `id, trainer_id, record_ids[], reason, status (pending|same|different), resolved_at`. While pending, see §6.4 (V1-IMP-07).

**KeepRule**: `id, trainer_id, scope (species|variant|saved_search|record), selector json, minimum int, origin (generated|override), generated_by text, enabled`.

**SavedSearchGroup**: `id, trainer_id, name, predicate json` (e.g. `{"tag":"mythical"}`). Membership is evaluated against reference data, not free text.

**FavoriteSpecies**: `trainer_id, species, form?` — exempts all copies.

**Team**: `id, trainer_id, name, kind (manual|recommended), mode? (V3), member record_ids`.

**CleanupPlan**: `id, trainer_id, input_revision, target_kind (free_more|reach_free|percent), target_value, denominator, target_slots, max_feasible, max_is_lower_bound bool, recommended_count, shortfall, status (current|stale|superseded), created_at`.

**PlanItem**: `plan_id, record_id, rank, reasons json, search_string, search_collisions json, requires_individual_verification bool, is_in_game_favorite bool`.

**TransferEvent**: `id, trainer_id, kind (single|bulk|refresh_removal|undo), record_ids[], plan_id?, created_at, undone_by?`. Undo is itself an event; nothing is hard-deleted.

**UpgradePlan (V3)**: `id, trainer_id, record_id, role, target_build json, status (proposed|accepted|declined|canceled|confirmed), dataset_version_id, timestamps`.

**BattleDatasetVersion (V3)**: `id, source, fetched_at, validated_at, status (staged|active|rejected), checksum, provenance json, payload ref`.

**ScreenshotImage (V2)**: `id, trainer_id, batch_id, path, sha256, screen_kind (list|detail|appraisal|unknown), status (pending|processed|needs_review|purged), extracted json`.

**AuthSession / PinResetCode / AuditLog**: see §9.

### 5.3 Variant key

```text
variant_key = (species, form, shiny, costume, background, dynamax, gigantamax)
```

Each component may be unknown. A record with any unknown component has a _set_ of possible variant keys; §7.3 defines how that affects planning.

---

## 6. V1 functional requirements

### 6.1 Home deployment and profiles

| ID         | Requirement                                                                                          | Acceptance                                                                     |
| ---------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| V1-HOST-01 | Serve one HTTPS URL on the home network, usable in iPhone Safari.                                    | Loads on each family iPhone without certificate warnings after one-time setup. |
| V1-HOST-02 | Manual start by default; opt-in launch at Mac login.                                                 | Toggle in admin console installs/removes the LaunchAgent.                      |
| V1-PRF-01  | Each trainer has their own collection, capacity, rules, manual teams, history.                       | Cross-profile isolation test (§15.9).                                          |
| V1-PRF-02  | Show the active profile name before every import and confirmation.                                   | Present on staging, plan and confirm screens.                                  |
| V1-PRF-03  | Switching profile requires the destination trainer's PIN.                                            | No path switches profile without PIN.                                          |
| V1-PRF-04  | Admin generates a one-time PIN reset code, emailed or shown on the MacBook. Trainer chooses new PIN. | Code single use, expires, all sessions for that trainer revoked on reset.      |
| V1-BAK-01  | Manual export/restore of profiles, collections, rules, teams, transfer history.                      | §13.                                                                           |

### 6.2 Progressive setup

| ID        | Requirement                                                                                                                                                                                            |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| V1-SET-01 | Step 1 (required): capacity + any combination of Collecting, Trading, Battling. If >1 goal, trainer orders them. Show capacity-based defaults. Setup MAY finish here.                                  |
| V1-SET-02 | Step 2 (optional): refine priorities (rarity, IVs, battle usefulness). Fixed minimums per species or saved search group (e.g. all Mythical). Show the effective minimum and the rule that produced it. |
| V1-SET-03 | Step 3 (optional): favorite species exempt all copies. Manual protection of individual records and named teams. In-game Favorite is separate and does not protect.                                     |
| V1-SET-04 | Every setting is editable later and bumps the trainer revision.                                                                                                                                        |

### 6.3 Keep rules

| ID        | Requirement                                                                                                                                                                                                                                        |
| --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| V1-RUL-01 | Default: protect one copy of each distinct observed variant (§5.3). Setup may override explicitly.                                                                                                                                                 |
| V1-RUL-02 | Default species and variant minimums scale with capacity via editable capacity bands. Thresholds are a calibration item (§16). Reference points: small bag ≈ 1 ordinary copy; 2,700 slots ≈ 2 ordinary copies and ≥4 rare or battle-useful copies. |
| V1-RUL-03 | Trading goal raises default minimums on desirable trade variants (shiny, legendary, rare costume).                                                                                                                                                 |
| V1-RUL-04 | A fixed override replaces the generated default it targets. Where explicit overrides overlap on the same records, the highest minimum wins.                                                                                                        |
| V1-RUL-05 | One Pokémon can satisfy multiple rules simultaneously.                                                                                                                                                                                             |
| V1-RUL-06 | If fewer copies exist than the minimum, keep all of them (floor = min(minimum, available)).                                                                                                                                                        |
| V1-RUL-07 | Missing data is unknown. If resolving an unknown could change a keep decision, the record is protected until confirmed (§7.3).                                                                                                                     |
| V1-RUL-08 | An unsupported export field cannot establish absence of a feature.                                                                                                                                                                                 |
| V1-RUL-09 | The rule inspector shows, per group: generated default, overrides, effective minimum, the winning rule, current count, and which records satisfy it.                                                                                               |

**Effective minimum resolution [Proposed precise semantics]:**

1. Generate defaults per group from capacity band × goals.
2. For each group with an explicit override targeting it, replace the default with the override.
3. Rules are constraints, not a single number per record: each rule `r` with selector `S_r` and minimum `m_r` yields constraint _keep ≥ min(m_r, |S_r ∩ active|)_.
4. "Highest minimum wins" for overlapping overrides falls out of step 3 when selectors are identical; when selectors nest (Mew ⊂ Mythical), both constraints hold. The inspector reports the binding one. Acceptance scenario "Mew minimum 2 and Mythical minimum 4 → effective Mew floor 4 when available" means: when Mew is the only Mythical available in sufficient numbers, the Mythical-4 constraint forces up to 4 Mew to be kept; the inspector shows Mew's effective floor as 4 with the Mythical rule as source. **Confirm this reading with the product owner** (§16, OD-02) — the alternative reading (any species in a group inherits the group minimum individually) changes the model.

### 6.4 Import and reconciliation

| ID        | Requirement                                                                                                                                                                                                                                    |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| V1-IMP-01 | Support the supplied Poke Genie CSV out of the box via a built-in mapping.                                                                                                                                                                     |
| V1-IMP-02 | Generic CSV: trainer maps unfamiliar columns to fields. Mappings are saved per header signature.                                                                                                                                               |
| V1-IMP-03 | Preview before commit: mapped fields, invalid values (with row/column), unsupported data (columns ignored).                                                                                                                                    |
| V1-IMP-04 | The supplied header covers species, form, dex number, CP, HP, IVs, moves, favorite, catch date, PvP metrics. It has no shiny, costume, background or Max columns → those fields are `unknown` unless a validated encoding exists (§16, OD-03). |
| V1-IMP-05 | Every record gets a stable app ID. Per-field source and observation timestamps are kept. Manual corrections are preserved.                                                                                                                     |
| V1-IMP-06 | Incremental import is the default.                                                                                                                                                                                                             |
| V1-IMP-07 | Any records that might be the same catch are shown side by side; trainer answers same/different. Never auto-merge. Until resolved, the group counts as **at most one copy** toward floors and **no member is a transfer candidate**.           |
| V1-IMP-08 | Full refresh stages additions, possible matches and previously owned records missing from the new import. Removals and ambiguous matches MUST be resolved before apply.                                                                        |
| V1-IMP-09 | Show trainer-reported count vs active imported count and the gap. Allow cleanup relative to the imported set.                                                                                                                                  |
| V1-IMP-10 | Mark actual transfers from the collection view; bulk-confirm from a plan.                                                                                                                                                                      |
| V1-IMP-11 | Transfer history with undo for individual, bulk, and full-refresh removals. Undo restores app records only (it cannot restore in-game Pokémon — say so in UI).                                                                                 |
| V1-IMP-12 | Recalculate recommendations after any confirmation or restoration.                                                                                                                                                                             |

**Matching heuristic for "might be the same catch" [Proposed]:** candidates are any pair with same species+form and (a) identical catch date and IVs, or (b) identical IVs and CP within the range explainable by power-up/evolution, or (c) any source row hash collision. Tune with fixtures. False positives are cheap (one extra question); false negatives are dangerous (double-counted copies). Bias toward asking.

**Import pipeline:**

```text
upload → parse (adapter) → normalize → validate → match against active records
       → stage (ImportBatch=staged) → trainer review (preview, duplicates, removals)
       → commit (single transaction: observations, records, reviews, revision++) → replan
```

A staged batch is invalidated if the trainer revision changes before commit (another import landed); the trainer is asked to re-stage.

### 6.5 Cleanup workflow

| ID        | Requirement                                                                                                                                                                                                                                                       |
| --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| V1-CLN-01 | Target in slots or percent. "Free X more slots" = incremental. "Reach X free slots" accounts for headroom. Percent = reduction of occupied storage; show the denominator. Use reported in-game count for headroom when supplied; label imported-count estimates.  |
| V1-CLN-02 | Compute maximum feasible transfers under all rules, protections and uncertainty, evaluated as a batch (§7). Below-floor groups cannot be reduced. Stable tie-breaker. On timeout, show a verified feasible count as a lower bound.                                |
| V1-CLN-03 | Recommend exactly `min(target, max)`. If target > max, show shortfall and compare: cleanup only, capacity upgrade only, cleanup + upgrade.                                                                                                                        |
| V1-CLN-04 | Checklist per candidate: species/form, CP, catch date, other identifiers; copyable English Pokémon GO search string; flag searches that could match kept/protected Pokémon; require individual verification when identity is ambiguous; show eligibility reasons. |
| V1-CLN-05 | In-game Favorites are eligible and counted. Show their count and share of the plan, and an "unfavorite in game first" flag on each.                                                                                                                               |
| V1-CLN-06 | Recompute after imports, corrections, rules, upgrades, roster changes, confirmations. Flag stale plans and show the diff before further action. Collection-view reconciliation stays available regardless.                                                        |

**Target arithmetic:**

```text
O = reported_count ?? active_imported_count        (label "estimate" if imported)
C = storage_capacity
H = max(0, C − O)
free_more X   → T = X
reach_free X  → T = max(0, X − H)
percent p     → T = ceil(p/100 × O)                 (display "p% of O")
recommended   = min(T, MaxFeasible)
shortfall     = max(0, T − MaxFeasible)
```

Upgrade comparison: capacity-upgrade-only = slots needed `T`; cleanup+upgrade = `MaxFeasible` transfers + `shortfall` slots. Upgrade increment size is a configurable constant (**[Open]**, verify current in-game value).

**Search strings [Proposed]:** generate from species, CP (`cp<N>`), and other supported search terms in English; then evaluate the string against the trainer's own active records using a local implementation of the supported search grammar subset. If any kept/protected record matches, flag the item and list colliding records. Only emit search terms the implementation can evaluate; never emit terms whose game semantics are unverified.

---

## 7. Cleanup planner specification

### 7.1 Inputs

`PlanInput = { trainerRevision, records (active, resolved fields), rules, protections, duplicateReviews, goals, preferences, target, rosters (V3), upgradePlans (V3), datasetVersion (V3) }`

The input is hashed; the plan stores `input_revision`. Plans are pure functions of input + solver version; same input ⇒ same output (determinism test, §15.7).

### 7.2 Hard protections (removed from candidate set)

A record is ineligible if any of:
manual protection · favorite species · member of a manual team · member of a current recommended roster (V3) · accepted/pending upgrade (V3) · member of a pending duplicate review · unresolved decision-relevant unknown (§7.3) · status ≠ active.

In-game Favorite is **not** a protection.

### 7.3 Unknown attributes

For each record `r` and each unknown tri-state attribute `a` (shiny, costume, background, dynamax, gigantamax, and any field a rule depends on):

- `r` is **decision-relevant unknown** if there exists a value of `a` under which `r` would belong to a rule group whose known-member count is below that group's floor (i.e. `r` might be needed to satisfy a floor).
- Decision-relevant unknown ⇒ protected, with reason "Confirm shiny status (etc.) to make eligible".
- Unknown records never count toward a floor for a value they _might_ have. They count only toward groups determined by their known fields.

Consequence to surface in UI and in testing: with a CSV lacking shiny/costume/background, many records may be protected pending confirmation. **[Proposed]** Provide bulk confirmation per species ("None of my 34 Pidgey are shiny, costumed or have a background") recorded as `manual` observations. This is the main V1 usability risk; validate with real family data early (§17, M2).

### 7.4 Optimization model

Decision variable `x_i ∈ {0,1}` per eligible record (1 = transfer). Ineligible records are fixed at 0.

Constraints, for each rule `r`:
`Σ_{i ∈ S_r} (1 − x_i) ≥ min(m_r, |S_r|)` (fixed records count as kept).
Pending duplicate groups count as one member of `S_r`.

**Phase A — maximum:** maximize `Σ x_i` → `MaxFeasible`.

**Phase B — selection:** with `N = min(T, MaxFeasible)`, add `Σ x_i = N`, minimize `Σ w_i x_i` where `w_i` is the keep-value of record `i` from ordered goals and preferences. Ordered goals become lexicographic weights (goal 1 dominates goal 2 dominates goal 3) **[Proposed]**: `w_i = Σ_k B^(K−k) · score_k(i)` with integer-scaled scores and `B` large enough to preserve lexicographic order.

**Tie-breaker:** after Phase B, among equal-cost solutions prefer transferring the record with the lower `(score, catch_date older first, app ID ascending)`. Implement as a tiny ε-term derived from a total order on app ID, or as a deterministic post-pass swap; must be stable across runs.

**Timeouts:** each phase has a budget (default 5 s Phase A, 5 s Phase B, configurable; final values from §14.1 benchmark). On timeout:

- Phase A returns the best incumbent; it is feasible by construction and is reported as `max_is_lower_bound = true` ("At least 312 transfers possible").
- If no incumbent, run the greedy fallback: iterate candidates by ascending keep-value, transfer if all constraints remain satisfied. Always feasible, reported as a lower bound.
- Every returned solution is re-verified against all constraints by an independent checker before being returned. A verification failure is a bug: return an error, never an unverified plan.

### 7.5 Output

`PlanResult = { maxFeasible, maxIsLowerBound, target, recommended[], shortfall, upgradeOptions, favoriteCount, favoriteShare, protectedSummary (counts by reason), staleOf? }`
Each recommended item carries `reasons` (e.g. "4 kept copies satisfy Pidgey ≥2; lowest IV of remaining") and `searchCollisions`.

### 7.6 Staleness

Any revision change marks current plans `stale`. Opening a stale plan computes the new plan and shows a diff (added, removed, unchanged). Bulk confirm against a stale plan is rejected (§10.3).

---

## 8. API surface

JSON over HTTPS, defined in `api/openapi.yaml` (§4.2). Paths below are relative to `/api/v1`. All trainer routes derive `trainerId` from the session; it never appears in the path or body. Mutations accept `If-Match: <revision>` and an `Idempotency-Key` header.

| Method    | Path                                  | Purpose                                                         |
| --------- | ------------------------------------- | --------------------------------------------------------------- |
| POST      | `/session`                            | Login: `{profileId, pin}`                                       |
| POST      | `/session/switch`                     | Switch: `{profileId, pin}` (revokes current session)            |
| DELETE    | `/session`                            | Logout                                                          |
| POST      | `/pin/reset`                          | `{profileId, code, newPin}`                                     |
| GET/PATCH | `/me`                                 | Profile, capacity, reported count, goals, preferences           |
| GET/PUT   | `/rules` · `/rules/inspect?group=`    | Rules and effective-minimum inspector                           |
| GET/PUT   | `/favorite-species`                   |                                                                 |
| CRUD      | `/teams`                              | Manual teams                                                    |
| GET       | `/records?filter=` · `/records/:id`   | Collection, record detail with observations                     |
| PATCH     | `/records/:id`                        | Manual corrections / protection                                 |
| POST      | `/records/confirm-attributes`         | Bulk attribute confirmation (§7.3)                              |
| POST      | `/imports`                            | Upload + stage (multipart)                                      |
| GET       | `/imports/:id`                        | Staged preview                                                  |
| PUT       | `/imports/:id/mapping`                | Column mapping                                                  |
| POST      | `/imports/:id/commit` · `/discard`    |                                                                 |
| GET/POST  | `/duplicate-reviews` · `/:id/resolve` |                                                                 |
| POST      | `/plans`                              | Compute plan for target                                         |
| GET       | `/plans/current`                      | Current plan (+ stale flag/diff)                                |
| POST      | `/transfers`                          | Confirm transfers `{recordIds, planId?}`                        |
| POST      | `/transfers/:eventId/undo`            |                                                                 |
| GET       | `/history`                            | Transfer/import/undo events                                     |
| GET       | `/backup` · POST `/restore`           | §13                                                             |
| —         | Admin (loopback only)                 | profiles, reset codes, launch-at-login, backups of all profiles |

V2 adds `/screenshots/*`; V3 adds `/battle/*`, `/upgrades/*`, `/rosters/*`.

Error envelope (all non-2xx):

```json
{
  "error": {
    "code": "STALE_REVISION",
    "message": "Your collection changed since this plan was made.",
    "details": { "currentRevision": 42 },
    "correlationId": "01J…"
  }
}
```

---

## 9. Security and authentication

| ID     | Requirement                                                                                                                                                                                               |
| ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| SEC-01 | PINs hashed with Argon2id; never logged or returned. PIN length **[Proposed]** 4–8 digits (family app; brute force bounded by SEC-02).                                                                    |
| SEC-02 | Rate limit PIN attempts per profile and per client IP. **[Proposed]** 5 failures → 1 min lockout, doubling to 1 h max; counter resets on success.                                                         |
| SEC-03 | Sessions: random 256-bit token, `HttpOnly; Secure; SameSite=Strict` cookie, bound to one profile, idle timeout **[Proposed]** 30 days on device, revocable.                                               |
| SEC-04 | Reset codes: random, single use, stored hashed, expire **[Proposed]** 15 min. Delivered by email (if SMTP configured) or shown on the MacBook admin console. Reset revokes all sessions for that trainer. |
| SEC-05 | Admin console binds to loopback only; no admin routes on the LAN interface.                                                                                                                               |
| SEC-06 | Profile isolation: trainerId always from session; every repository query scoped; cross-profile access returns 404 (not 403) to avoid enumeration.                                                         |
| SEC-07 | HTTPS only on the LAN interface; HTTP redirects. Validate on every family iPhone before shipping V1.                                                                                                      |
| SEC-08 | CSRF: SameSite=Strict + require `Content-Type: application/json` or multipart with a CSRF token for uploads.                                                                                              |
| SEC-09 | Upload limits: CSV ≤ 10 MB, screenshots ≤ 15 MB each, batch ≤ 200 images **[Proposed]**. Reject by content sniffing, not extension.                                                                       |
| SEC-10 | Audit log: logins, failed PINs, switches, resets, restores, bulk confirms, undos.                                                                                                                         |
| SEC-11 | Backups contain personal data and PIN hashes; UI labels them sensitive. **[Open]** whether to encrypt backups with a passphrase (OD-08).                                                                  |

---

## 10. Error handling strategy

### 10.1 Principles

1. **Fail safe for the collection.** Any error during planning or import results in _fewer_ recommended transfers, never more.
2. **Atomic or nothing.** Import commits, bulk confirmations, undos, full-refresh applies and restores are single transactions. Partial application is a bug.
3. **Say what happened and what to do next**, in trainer language. Technical detail goes to logs with a correlation ID shown in the UI.
4. **Idempotent mutations.** Flaky Wi-Fi retries must not double-confirm transfers.

### 10.2 Error taxonomy

| Code                          | HTTP | Cause                                                | Client behaviour                                               |
| ----------------------------- | ---- | ---------------------------------------------------- | -------------------------------------------------------------- |
| `VALIDATION_FAILED`           | 422  | Bad input; `details.fields[]` with paths             | Inline field errors                                            |
| `UNAUTHENTICATED`             | 401  | No/expired session                                   | Go to PIN screen, preserve intended action                     |
| `PIN_INVALID`                 | 401  | Wrong PIN                                            | Show remaining attempts                                        |
| `RATE_LIMITED`                | 429  | Lockout                                              | Show unlock time                                               |
| `NOT_FOUND`                   | 404  | Missing or other profile's resource                  | Generic not found                                              |
| `STALE_REVISION`              | 409  | `If-Match` ≠ current revision                        | Refetch; for plans show diff before allowing confirm           |
| `STAGED_BATCH_OUTDATED`       | 409  | Revision changed since staging                       | Offer re-stage                                                 |
| `UNRESOLVED_REVIEWS`          | 409  | Commit/full refresh with open duplicates or removals | Link to review queue                                           |
| `IDEMPOTENT_REPLAY`           | 200  | Same key replayed                                    | Return original result                                         |
| `UNSUPPORTED_FILE`            | 415  | Not a CSV/PNG/JPEG/HEIC                              | Explain accepted types                                         |
| `PAYLOAD_TOO_LARGE`           | 413  | Limits §9                                            | Explain limit                                                  |
| `PLANNER_VERIFICATION_FAILED` | 500  | Solver result violated a constraint                  | "Couldn't produce a safe plan" + correlation ID; no plan shown |
| `RESTORE_INCOMPATIBLE`        | 422  | Unknown/newer schema, checksum mismatch              | Nothing changed; explain                                       |
| `INTERNAL`                    | 500  | Anything else                                        | Generic message + correlation ID                               |

Not errors (normal results with explicit UI): infeasible target (shortfall), planner timeout (lower bound), unknown-attribute protections, missing reported count (estimate label).

### 10.3 Per-area handling

| Area             | Failure                                          | Handling                                                                                                                                                                                         |
| ---------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| CSV parse        | Encoding, delimiter, quoted newlines             | Detect UTF-8/UTF-8-BOM/UTF-16; on failure stage nothing, show first bad line.                                                                                                                    |
| CSV rows         | Invalid value (IV 16, CP "abc", unknown species) | Row staged as `invalid`, excluded from commit, listed in preview; valid rows still importable. Field-level invalid → that field `unknown`, not the row, when the row still identifies a Pokémon. |
| CSV columns      | Unmapped/unsupported                             | Listed as ignored; never inferred silently.                                                                                                                                                      |
| Reconciliation   | Possible duplicates                              | DuplicateReview; record excluded from candidates (V1-IMP-07).                                                                                                                                    |
| Full refresh     | Missing records                                  | Staged `missing_from_refresh`; commit blocked until each is confirmed removed or kept.                                                                                                           |
| Planner          | Timeout                                          | Lower bound (§7.4).                                                                                                                                                                              |
| Planner          | Verification failure                             | Hard error, logged with input hash for reproduction; previous non-stale plan stays viewable but flagged.                                                                                         |
| Transfers        | Record already transferred/removed               | Whole bulk confirm rejected with list; trainer may retry without those.                                                                                                                          |
| Transfers        | Stale plan                                       | `STALE_REVISION`; show diff. Collection-view single confirmation remains available.                                                                                                              |
| Undo             | Record since modified/re-imported                | Undo still restores status; conflicts flagged on the record for review. Undo of an undo not supported **[Proposed]** (re-confirm instead).                                                       |
| DB               | `SQLITE_BUSY`                                    | Busy timeout 5 s, then retry once, then `INTERNAL`.                                                                                                                                              |
| DB               | Corruption / disk full                           | Refuse writes, show admin banner, point to last backup. Integrity check at startup.                                                                                                              |
| Email            | SMTP failure                                     | Reset code remains valid; admin shown "email failed, deliver in person".                                                                                                                         |
| OCR (V2)         | Sidecar crash/timeout                            | Image → `needs_review` with reason; batch continues.                                                                                                                                             |
| Battle data (V3) | Fetch/validation failure                         | Keep last good dataset; show its age and the failure; never activate partial data.                                                                                                               |

### 10.4 Logging

Structured JSON logs to a rotating file under the app data dir. Every request has a correlation ID. Never log PINs, reset codes, session tokens, or full CSV contents.

### 10.5 Host unavailable

Client checks `/health` on focus and before mutations. If unreachable: banner "CatchKeep isn't reachable — is the MacBook on and awake?"; mutations disabled; nothing queued offline **[Proposed]** (avoids confirming transfers against an unknown server state).

---

## 11. V2 · Screenshot import

| ID        | Requirement                                                                                                                                            |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| V2-SCR-01 | Batch upload of English storage-list, detail and appraisal screenshots from iPhone Safari.                                                             |
| V2-SCR-02 | Screenshot-only onboarding and mixed CSV + screenshot imports.                                                                                         |
| V2-SCR-03 | Multiple Pokémon per list screenshot.                                                                                                                  |
| V2-SCR-04 | All processing local on the MacBook.                                                                                                                   |
| V2-SCR-05 | Store confident observations; queue only uncertain readings for review. Trainer can correct manually or add another screenshot.                        |
| V2-SCR-06 | A view that cannot display a feature cannot confirm its absence (per-view capability matrix below).                                                    |
| V2-SCR-07 | Same unknown-field protection and duplicate confirmation as V1, including across overlapping screenshots and CSVs.                                     |
| V2-SCR-08 | Trainer can pick an existing record and add an updated screenshot (evolution, power-up, move change); preview conflicting changes; keep the record ID. |
| V2-SCR-09 | Cleanup action purges processed images while preserving extracted data; images tied to pending reviews are kept.                                       |
| V2-SCR-10 | Supplied PNGs are the initial fixtures; add cases for every supported screen and protected feature.                                                    |

**Pipeline:** upload → store (sha256 dedupe) → classify screen kind → OCR + layout parse per kind → per-field confidence → observations above threshold stored; below threshold → review queue → reconciliation as V1.

**View capability matrix [Open — must be filled by validation before V2 ships]:**

| Field              | List | Detail | Appraisal |
| ------------------ | ---- | ------ | --------- |
| species/name       | ?    | ?      | ?         |
| CP                 | ?    | ?      | ?         |
| IVs                | —    | —      | ?         |
| shiny              | ?    | ?      | ?         |
| costume / form     | ?    | ?      | ?         |
| background         | ?    | ?      | ?         |
| Dynamax/Gigantamax | ?    | ?      | ?         |
| moves              | —    | ?      | —         |
| in-game Favorite   | ?    | ?      | ?         |

A field may only be set to a definite value (including `false`) from a view whose matrix cell is validated "establishes". Otherwise it stays unknown.

**Confidence thresholds:** per field, calibrated on fixtures so that accepted readings meet a target precision **[Proposed]** ≥ 99.5% for fields that affect keep decisions; everything else goes to review.

---

## 12. V3 · Battle recommendations

| ID        | Requirement                                                                                                |
| --------- | ---------------------------------------------------------------------------------------------------------- |
| V3-BAT-01 | One "Battle usefulness" setting, optionally expanded per mode. Recommendations cover the whole collection. |
| V3-BAT-02 | Current recommended rosters are protected; a cleanup shortfall never breaks them.                          |
| V3-BAT-03 | Modes and defaults:                                                                                        |

| Mode                  | Default                                                                                                                                                                                      |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Raids and gym attacks | Two raid teams of six per attacking type. One Pokémon may cover multiple types if its confirmed build serves both without move changes. Distinct builds can justify extra copies.            |
| Gym defense           | Three defenders across the collection; adjustable.                                                                                                                                           |
| GO Battle League      | Per selected format, three complete teams of three with complementary coverage. A Pokémon may appear on multiple teams with its confirmed build.                                             |
| Max Battles           | Separate opt-in. Roster size and minimums scale with capacity, adjustable; prioritize Dynamax/Gigantamax builds covering more matchups. When disabled, normal collectible protections apply. |

| ID        | Requirement                                                                                                                                                                                                                  |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| V3-BLD-01 | Show each Pokémon's best build per relevant role (evolution, power-ups, moveset, special requirements such as Elite TMs). Assess IVs and moves by role. Call out when the current build already serves the role well.        |
| V3-UPG-01 | Upgrade lifecycle: `proposed → accepted → confirmed` or `→ canceled`; `proposed → declined`. Accepted protects the Pokémon until confirmed/canceled. Declined dismisses the suggestion and uses the current confirmed build. |
| V3-UPG-02 | Only trainer-confirmed builds count toward roster slots. Open slots and planned builds shown separately.                                                                                                                     |
| V3-RST-01 | Re-rank rosters when a stronger Pokémon is imported. A displaced Pokémon leaving its last recommended team with no other protection becomes eligible immediately. Explain the change; remind to update in-game tags/parties. |
| V3-RST-02 | Quick access to teams, defenders, suggested tags, copyable searches.                                                                                                                                                         |
| V3-DAT-01 | Automatic refresh of official rules/announcements and community rankings/movesets. Candidate raid source: GameInfo Top Attackers. The official GBL leaderboard ranks trainers, not Pokémon — not a ranking source.           |
| V3-DAT-02 | Before implementation validate source access, permitted reuse, data availability, PvP and Max ranking sources (§16, OD-06).                                                                                                  |
| V3-DAT-03 | Each dataset is versioned, validated before activation, shown with provenance and last successful refresh. On failure keep the last good snapshot and show its age.                                                          |
| V3-DAT-04 | Unsupported or invalid data cannot justify a transfer.                                                                                                                                                                       |

**Dataset validation [Proposed]:** schema check; referential check (every species/move exists in reference data); sanity bounds (no >20% of species disappearing between versions without manual approval); checksum. Activation is atomic: rosters recompute against the new version in one transaction, bumping every trainer's revision.

**Integration with planner:** recommended roster members and accepted upgrades enter §7.2 as hard protections. Roster computation runs before cleanup planning; planner never sees roster members as candidates.

---

## 13. Backup, restore and migrations

- **Format [Proposed]:** a `.catchkeep` zip containing `manifest.json` (`schemaVersion`, `appVersion`, `createdAt`, `profiles[]`, sha256 per file) and one JSON file per entity type. Scope: one profile or all (all only from admin console).
- **Contents:** trainers, records, observations, import batches (metadata, not raw files), duplicate reviews, rules, saved searches, favorite species, teams, transfer events. V2 adds screenshot metadata (images optional). V3 adds upgrade plans and the active dataset version reference.
- **Restore:** validate manifest and checksums → refuse newer `schemaVersion` → run in-memory migrations from backup version to current → preview what will be replaced → single transaction replace for the target profile(s) → bump revision → replan. Failure leaves the database untouched.
- **Migrations:** forward-only, numbered SQL files, each with a test. Automatic pre-migration database snapshot on app upgrade.
- **Compatibility guarantee:** every release restores backups from every earlier release. Golden backup fixtures are committed per release (§15.10).

---

## 14. Non-functional requirements

### 14.1 Performance

Benchmark before fixing thresholds (handoff requirement). Benchmark suite: synthetic collections of 2,700 and 5,400 records with realistic species distributions and rule sets.

| Operation                            | Provisional target **[Proposed, replace after benchmark]** |
| ------------------------------------ | ---------------------------------------------------------- |
| CSV stage, 2,700 rows                | < 3 s on the MacBook                                       |
| Commit, 2,700 rows                   | < 2 s                                                      |
| Plan (Phase A + B), 2,700 records    | < 5 s p95; timeout fallback otherwise                      |
| Collection list first page on iPhone | < 1 s on home Wi-Fi                                        |
| Screenshot OCR (V2)                  | < 3 s per image, processed in background                   |

### 14.2 Reliability

- SQLite `PRAGMA integrity_check` on startup; WAL; `synchronous=NORMAL` **[Proposed]**.
- Automatic local rolling backups (daily, keep 14) in addition to manual export **[Proposed]**.

### 14.3 Usability and accessibility

- Mobile-first layouts for iPhone screen sizes; tap targets ≥ 44 pt; supports Dynamic Type scaling in Safari; WCAG 2.2 AA contrast.
- Active profile name and color visible on every screen.
- Destructive or irreversible-in-game actions (bulk confirm, full refresh apply) require an explicit confirm step showing count and profile.

### 14.4 Localization

UI English for V1. Search strings and screenshot parsing are English-only; keep strings externalized for later.

---

## 15. Testing plan

### 15.1 Test-driven development

All production code is written test first.

1. **Red** — write the smallest failing test that expresses the next behaviour. Run it and see it fail for the expected reason.
2. **Green** — write the simplest code that passes.
3. **Refactor** — clean up with all tests green. No behaviour change.

Rules:

- Work **outside in** for each feature slice: start from an acceptance or API-level test, drive the use case with fakes of its driven ports, then drive the domain with unit tests. Adapters are driven by their own integration tests.
- A commit contains a behaviour change **and** the tests that drove it. Pure refactors go in their own commit with no test changes other than renames.
- A bug fix starts with a test that reproduces the bug.
- No production code without a test that failed first. Code review checks that test and code arrived together.
- Tests are named for behaviour (`TestPlan_NeverRecommendsProtectedRecord`), not for methods.
- Every test is deterministic: injected `Clock` and `IDGenerator`, seeded randomness, no sleeps, no network.
- Fakes (in-memory port implementations) are preferred over mocks. Mocks are only for verifying interactions that have no observable state (e.g. "mailer was called").

### 15.2 Test pyramid

Each side has its own balanced pyramid. Most tests are fast and isolated; few are slow and broad. Proportions are guidelines for review, not targets to game.

```text
                 ▲  manual (per release, every family iPhone)
                ▲▲▲  smoke (every build and every host start)
              ▲▲▲▲▲▲▲  end-to-end (full stack, few, critical journeys)
          ▲▲▲▲▲▲▲▲▲▲▲▲▲  integration / contract / component
   ▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲▲  unit (domain, use cases, pure frontend logic)
```

#### Backend (Go)

| Layer               | Share | Scope                                                                                                                                                                                  | Tooling                                    | Speed budget       |
| ------------------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ | ------------------ |
| Domain unit         | ~60%  | `internal/domain`: variant keys, tri-state, observation resolution, rules engine, target arithmetic, planner model and checker, search strings                                         | `go test`, table tests, `rapid` properties | whole layer < 10 s |
| Use case            | ~20%  | `internal/app` with in-memory fakes for every driven port: transactions, revision checks, protections, staleness, idempotency                                                          | `go test` + `internal/adapters/fake`       | < 10 s             |
| Adapter integration | ~15%  | SQLite repositories and migrations (temp DB file), CSV adapters (fixture files), HiGHS solver, OCR sidecar (V2), battle sources (recorded responses, V3), HTTP handlers via `httptest` | `go test -tags=integration`                | < 60 s             |
| Contract            | ~5%   | Every handler response validates against `api/openapi.yaml`; every documented status code is exercised                                                                                 | `kin-openapi` validator in `httptest`      | < 30 s             |

#### Frontend (TypeScript)

| Layer                 | Share | Scope                                                                                                                                                    | Tooling                                                 | Speed budget |
| --------------------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- | ------------ |
| Unit                  | ~50%  | Pure functions, formatters, client-side state stores, view-model mapping from API DTOs                                                                   | Vitest                                                  | < 10 s       |
| Component             | ~35%  | Components and screens rendered with the API mocked from the contract: PIN entry, import preview, duplicate review, rule inspector, plan checklist, undo | Vitest + Testing Library + contract-based request mocks | < 60 s       |
| Flow (mocked backend) | ~15%  | Multi-screen journeys in WebKit with iPhone viewports against the contract mock server: setup, import, plan, confirm, undo                               | Playwright                                              | < 3 min      |

Frontend tests never contain domain rules. If a test needs to assert _why_ a Pokémon is protected, that belongs in a backend test; the frontend asserts that the reason returned by the API is shown.

#### End-to-end (full stack)

Few, slow, and only for critical journeys: the acceptance scenarios in §15.11. They run the real backend binary with a seeded SQLite database and the built frontend bundle, in Playwright WebKit with iPhone profiles. Budget < 10 min.

### 15.3 Smoke tests

Smoke tests answer "is this build alive and safe to use?" in under a minute. They never mutate trainer data.

| When                                          | What runs                                                                            | Checks                                                                                                                                                                              |
| --------------------------------------------- | ------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CI, after building both sides                 | Backend binary started against an empty temp database with the built frontend dir    | `/api/v1/health` 200; migrations applied; static index served; one login with a seeded smoke profile; one read-only API call                                                        |
| Every host start                              | Built-in startup self-check, then `catchkeep smoke` on demand from the admin console | DB `integrity_check`; schema version current; web dir present; TLS certificate valid and not expiring within 30 days; OCR sidecar responds (V2); active battle dataset present (V3) |
| After installing a new release on the MacBook | `catchkeep smoke` + open the app on one iPhone                                       | Same as above + manual "page loads, PIN screen appears"                                                                                                                             |

A failed startup self-check keeps the API read-only and shows an admin banner (§10.3).

### 15.4 Manual tests

Run per release on **every** family iPhone, and record results in `docs/manual-test-log.md` (date, device, iOS version, release, pass/fail, notes).

- HTTPS trust, no certificate warnings.
- Login, profile switch with PIN, wrong-PIN lockout message.
- CSV upload from the Files app; preview readable on the smallest family iPhone.
- Copy a search string and paste it into Pokémon GO; it finds the intended Pokémon.
- Bulk confirm and undo with the real collection of one trainer (after a backup).
- Host asleep: banner shown, no mutation possible.
- V2: screenshot batch upload from Photos; review queue usable one-handed.
- V3: roster view and copyable tags.

Plus a 30-minute **exploratory session** per release with a written charter (e.g. "try to make the planner recommend something that should be kept").

### 15.5 Where each gate runs

| Gate                                              | Pre-commit (local) | PR CI     | Release            |
| ------------------------------------------------- | ------------------ | --------- | ------------------ |
| Format, lint, type-check (side touched)           | ✓                  | ✓         | ✓                  |
| Unit + use case / unit + component (side touched) | ✓                  | ✓         | ✓                  |
| Integration + contract (backend)                  |                    | ✓         | ✓                  |
| Flow tests (frontend, mocked backend)             |                    | ✓         | ✓                  |
| Full-stack E2E                                    |                    | ✓         | ✓                  |
| Smoke                                             |                    | ✓         | ✓ (on the MacBook) |
| Benchmarks (§15.12)                               |                    | on demand | ✓                  |
| Manual + exploratory                              |                    |           | ✓                  |

Coverage floors **[Proposed]**: backend domain ≥ 90% statements, backend overall ≥ 80%, frontend ≥ 80%. A PR may not lower coverage of the package it touches.

### 15.6 Unit test focus

- **Variant key** and tri-state handling: `null` never coerced to `false`.
- **Observation resolution:** manual beats import; latest observed_at wins; supersession.
- **Rules engine:** default generation per capacity band × goals; override replacement; nested selectors; floor = min(m, available); inspector output.
- **Target arithmetic:** all three target kinds, reported vs imported denominator, `H` clamp at 0, rounding.
- **Search string generator** and local search evaluator; collision detection.
- **CSV adapter:** Poke Genie header, BOM, quoted fields, invalid values, unknown columns, mapping by header signature.
- **Auth:** hashing, rate limiter schedule, reset code expiry/single-use, session revocation.

### 15.7 Planner tests

- **Brute-force oracle:** for random instances ≤ 16 eligible records, Phase A optimum equals exhaustive search; Phase B cost equals exhaustive minimum.
- **Property-based invariants** (rapid, thousands of random instances):
  - Every returned plan satisfies all floor constraints (independent checker).
  - No protected record is ever recommended.
  - `recommended = min(T, maxFeasible)`; `shortfall` correct.
  - Determinism: same input ⇒ identical ordered output across runs and process restarts.
  - Monotonicity: adding a rule or protection never increases `maxFeasible`; raising capacity band minimums never increases it.
  - Unknowns: setting an unknown attribute to the rare value never makes the previous plan violate a floor (i.e. protection of decision-relevant unknowns is sufficient).
  - Pending duplicate groups count at most once and are never candidates.
- **Timeout path:** forced tiny budget ⇒ result feasible and flagged `maxIsLowerBound`; greedy fallback feasible.
- **Verification failure path:** inject a faulty solver ⇒ `PLANNER_VERIFICATION_FAILED`, no plan returned.

### 15.8 Integration tests (API + real SQLite)

- Import stage → preview → commit; staged batch outdated by concurrent commit.
- Full refresh with missing records: commit blocked until resolved; undo restores and replans.
- Bulk confirm idempotency (same key twice ⇒ one event).
- Stale plan confirm ⇒ 409 with diff.
- Transaction atomicity: inject failure mid-commit/bulk-confirm/restore ⇒ database unchanged.
- Manual correction survives re-import.

### 15.9 Security tests

- Cross-profile isolation: for every trainer route, a session for A cannot read/modify B (expect 404). Generated from the route table so new routes are covered automatically.
- Static check: every repository method requires `trainerId`.
- Rate limiting and lockout progression; reset revokes sessions; admin routes unreachable from non-loopback.
- Concurrent imports from two sessions on different profiles touch only their own data.

### 15.10 Migration and backup tests

- Each migration: apply on fixture DB of previous version, assert schema and data.
- Golden backups per release restore into current version; round-trip export → restore ⇒ identical data.
- Corrupted/newer/checksum-mismatch backups rejected with no changes.

### 15.11 Acceptance tests (full-stack E2E)

| #   | Scenario                                                           | Expected                                                                                      | Req                  |
| --- | ------------------------------------------------------------------ | --------------------------------------------------------------------------------------------- | -------------------- |
| A1  | Last shiny costume and several ordinary copies                     | Shiny costume stays by default; any explicit override visible                                 | V1-RUL-01, -09       |
| A2  | Mew minimum 2 and Mythical minimum 4                               | Effective Mew floor 4 when available; a shiny satisfies both species and shiny floors         | V1-RUL-04, -05       |
| A3  | CSV lacks background and a screenshot may duplicate another record | Review queue presents the uncertainty; plan does not rely on those records as safe duplicates | V1-IMP-07, V2-SCR-07 |
| A4  | Target 100 additional slots, maximum 70                            | Recommend at most 70; show 30-slot gap and upgrade options                                    | V1-CLN-03            |
| A5  | 18 of 100 candidates are starred                                   | Show 18% favorited and the unfavorite step on those candidates                                | V1-CLN-05            |
| A6  | Full refresh omits previously imported Pokémon                     | Confirmation required before removal; undo restores and recalculates                          | V1-IMP-08, -11       |
| A7  | Raid upgrade accepted and pending                                  | Pokémon protected; roster slot open; completion fills slot if confirmed build qualifies       | V3-UPG-01, -02       |
| A8  | Stronger imported attacker displaces a roster member               | Roster refreshed, change explained, eligible transfers and obsolete tags flagged              | V3-RST-01            |
| A9  | Trainers import concurrently on different iPhones                  | Each operation affects only the authenticated profile                                         | V1-PRF-01, SEC-06    |
| A10 | Battle data refresh fails validation                               | Last successful dataset kept; refresh time shown                                              | V3-DAT-03            |
| A11 | 400 feasible, request 100 additional slots                         | Exactly 100 recommendations                                                                   | V1-CLN-03            |
| A12 | Two of a minimum of four exist                                     | Both kept                                                                                     | V1-RUL-06            |
| A13 | Mistaken bulk confirmation                                         | Undo restores all records and replans (V1 exit criterion)                                     | V1-IMP-11            |

A1–A6, A9, A11–A13 gate V1; A3 (screenshot half) gates V2; A7, A8, A10 gate V3.

### 15.12 Performance

Benchmarks (§14.1) run in CI on demand and before each release; results recorded in `docs/benchmarks.md`.

### 15.13 OCR validation (V2)

Fixture corpus: supplied PNGs + new captures for every screen kind × protected feature (shiny, costume, background, Dynamax, Gigantamax, Favorite). Report per-field precision/recall; fill the capability matrix (§11) from results; thresholds set to hit the precision target. Regression run on every OCR change.

---

## 16. Open decisions

| ID    | Decision                                                                                                        | Needed by            | Default until decided                                                 |
| ----- | --------------------------------------------------------------------------------------------------------------- | -------------------- | --------------------------------------------------------------------- |
| OD-01 | Exact capacity bands and default minimums                                                                       | V1 rules engine (M3) | Editable placeholder bands matching the reference points in V1-RUL-02 |
| OD-02 | Semantics of nested group vs species minimum (A2 reading, §6.3)                                                 | V1 rules engine (M3) | Constraint model in §6.3                                              |
| OD-03 | What the Poke Genie export encodes without explicit columns (shiny/costume/background/Max hidden in name/form?) | V1 import (M2)       | Treat as unknown; need the supplied sample CSV in repo as fixture     |
| OD-04 | Local HTTPS setup on family iPhones (CA install, hostname, mDNS reliability)                                    | V1 host (M1)         | mkcert + `.local` hostname                                            |
| OD-05 | Local OCR accuracy and per-view field capability                                                                | V2                   | Apple Vision; matrix all "unknown"                                    |
| OD-06 | Ranking sources (raid, PvP, Max), access and reuse terms                                                        | V3 start             | GameInfo Top Attackers as raid candidate only                         |
| OD-07 | SMTP for reset emails (which account/provider)                                                                  | V1 auth (M1)         | In-person delivery only                                               |
| OD-08 | Encrypt backups with passphrase?                                                                                | V1 backup (M6)       | Unencrypted, labeled sensitive                                        |
| OD-09 | Planner performance thresholds and timeouts                                                                     | After M4 benchmark   | §14.1 provisional values                                              |
| OD-10 | Storage upgrade increment for upgrade comparison                                                                | V1 cleanup (M4)      | Configurable constant                                                 |
| OD-11 | Bulk attribute confirmation UX (§7.3) acceptable to the family?                                                 | M2 usability check   | Per-species bulk confirm                                              |

---

## 17. Implementation sequence (V1)

| Milestone | Scope                                                                                                                                                                                               | Done when                                                                                            |
| --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| M0        | Contract skeleton (`api/openapi.yaml`); Go backend skeleton (hexagonal layout, layering check, SQLite + migrations, health endpoint); frontend skeleton (Vite + generated client); separate CI jobs | Both CI jobs green independently; frontend shows health from a mock and from the real backend        |
| M1        | Host + HTTPS on LAN, profiles, PIN auth, sessions, switching, admin console, reset codes                                                                                                            | Security tests §15.9 pass; device check on each iPhone (OD-04)                                       |
| M2        | CSV import: Poke Genie adapter, mapping, staging, preview, observations, duplicate review, incremental commit                                                                                       | Import of the real family CSVs; measure how many records end up protected by unknowns (OD-03, OD-11) |
| M3        | Setup wizard, rules engine, capacity bands, overrides, favorite species, manual protection/teams, rule inspector                                                                                    | A1, A2, A12 pass                                                                                     |
| M4        | Planner (Phase A/B, tie-break, timeout, verifier), target arithmetic, upgrade comparison, benchmark                                                                                                 | §15.7 passes; A4, A5, A11 pass; OD-09 resolved                                                       |
| M5        | Checklist, search strings + collision check, transfer confirm (single/bulk), history, undo, staleness + diff, full refresh                                                                          | A6, A9, A13 pass                                                                                     |
| M6        | Backup/restore, launch at login, rolling backups, polish, manual device test pass                                                                                                                   | V1 exit criteria met                                                                                 |

V2 and V3 follow the same pattern: resolve their open decisions (OD-05, OD-06) first, then build against §11 and §12.
