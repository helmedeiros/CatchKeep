# Pokémon GO Storage Planner

## Developer handoff · 15 November 2026

A standalone family web app that recommends which Pokémon to keep or transfer to reach a storage target. Each release is independently usable and carries existing profiles, rules, collections and history forward. All transfers happen manually in Pokémon GO.

| Release                     | Family workflow                                                                        | Exit criteria                                                                                   |
| --------------------------- | -------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| V1 · CSV cleanup            | Import, set keep rules, protect Pokémon and teams manually, plan and confirm transfers | A trainer can reach a feasible cleanup target and undo mistaken app confirmations               |
| V2 · Screenshot import      | Import batches, review uncertain readings, update existing records                     | English list, detail and appraisal screenshots work locally on the MacBook                      |
| V3 · Battle recommendations | Manage recommended rosters and upgrades                                                | Recommendations refresh from versioned battle data and use confirmed builds for roster coverage |

## V1 · CSV cleanup

### Home deployment and profiles

- Run on the MacBook and serve one shared home Wi-Fi URL, usable in iPhone Safari. Start manually by default and offer launch at Mac login. The host must be on and awake. Access away from home is a future release.
- Give each trainer their own collection, storage capacity, rules, manual teams and history. Show the active profile before imports and confirmations. Require the destination trainer's private PIN when switching profiles.
- Let the home admin generate a one-time PIN reset code, either emailed to the registered address or delivered in person from the MacBook. The trainer chooses the new PIN. Hash PINs, rate-limit attempts, expire codes, and revoke sessions on reset. Validate encrypted local access on the family iPhones before shipping.
- Provide manual export and restore of profiles, collections, rules, team definitions, and transfer history. Extend the backup schema for later releases, including upgrade plans.

### Progressive setup

1. Enter storage capacity and choose any combination of Collecting, Trading, and Battling. If more than one is selected, ask the trainer to order them. Show capacity-based defaults and allow setup to finish.
2. Optionally refine priorities such as rarity, IVs, and battle usefulness. Allow fixed minimums for an individual species or a saved search group, such as all Mythical Pokémon. Show the effective minimum and the rule that produced it.
3. Optionally select favorite species to exempt all copies. Let trainers manually protect individual records and named teams from the first release. An in-game Favorite star is a separate attribute.

### Keep rules

- Default to protecting one copy of each distinct observed variant, including applicable species, form, shiny, costume, background and Dynamax/Gigantamax attributes. Allow explicit setup overrides. Default species and variant minimums scale with capacity. Trading increases default minimums on desirable trade variants such as shinies, legendaries, and rare costumes. Fixed overrides replace the relevant generated default, and the highest minimum wins when explicit overrides overlap. One Pokémon can satisfy multiple rules. If only two of a minimum of four exist, keep both.
- Provide editable capacity bands. A small bag may generally keep one ordinary copy. A 2,700-slot trainer should be able to keep two ordinary copies and four or more rare or battle-useful copies. Exact thresholds are a calibration item.
- Mark missing data as unknown. If resolving an unknown could change a keep decision, protect the Pokémon until the trainer confirms it. An unsupported export field cannot establish that a feature is absent.

### Import and reconciliation

- Support the supplied Poke Genie CSV and let trainers map unfamiliar CSV columns. Preview mapped fields, invalid values and unsupported data. The supplied header contains species, form, Pokédex number, CP, HP, IV components, moves, favorite status, catch date and PvP metrics. It has no dedicated shiny, costume, background or Max fields. Validate any source-specific encodings before relying on them.
- Assign a stable app ID to each Pokémon. Keep per-field source and observation timestamps, and preserve manual corrections. Import incrementally by default. If any records might represent the same catch, show them side by side and ask whether they are the same or different. Never auto-merge. Until resolved, do not count ambiguous records as independent copies to justify a transfer.
- Offer a full refresh that stages additions, possible matches and previously owned records missing from the new import. Confirm removals and resolve ambiguous matches before applying them. Compare the game's trainer-reported count to the active imported count, show the gap, and allow cleanup relative to the imported set.
- Let trainers mark actual transfers from the collection view and bulk-confirm transfers from a plan. Keep transfer history and support undo for individual, bulk and full-refresh removals. Undo restores app records. Recalculate recommendations after confirmation or restoration.

### Cleanup workflow

1. Accept a target in slots or percentage. Proposed UI distinction: “Free X more slots” is an incremental target, while “Reach X free slots” accounts for existing headroom. Define the percentage as a reduction in occupied storage and show the denominator. Use the reported in-game count for headroom when supplied and label imported-count estimates.
2. Compute the maximum feasible transfers under the current rule set. Evaluate the batch as a whole, including overlapping minimums, preserved variants, protected individual records and teams, and uncertainty. When current counts are already below a floor, the plan cannot reduce those counts further. Find a feasible set that preserves higher-priority candidates according to the trainer's ordered goals and advanced preferences. Use a stable tie-breaker. If planning times out, present any verified feasible count as a lower bound.
3. Recommend only enough transfers to reach the requested target. If the target exceeds the maximum, stop at that maximum and show the remaining shortfall. Compare cleanup only, capacity upgrade only, and cleanup plus upgrade. Example: 400 feasible transfers and a request for 100 additional slots produces 100 recommendations.
4. Show a checklist with species/form, CP, catch date and other identifiers when available, plus copyable English Pokémon GO searches. Flag searches that could match a kept or protected Pokémon, and require individual verification when identity is ambiguous. Show why each candidate is eligible.
5. Include in-game Favorites among eligible candidates and feasibility counts. Show their count and share of the plan. Flag that the trainer must unfavorite those Pokémon in Pokémon GO before transferring. The app records only what the trainer confirms happened in the game.
6. Recompute after imports, corrections, rules, upgrades, roster changes or confirmations. Flag stale plans and show changed recommendations before further action. A trainer can still reconcile an actual transfer through the collection view.

## V2 · Screenshot import

- Accept English Pokémon GO storage-list, detail and appraisal screenshots in batches from iPhone Safari. Support screenshot-only onboarding and mixed CSV and screenshot imports. Recognize multiple Pokémon per list screenshot. Process locally on the MacBook.
- Store confidently extracted observations and queue only uncertain readings. Allow a manual correction or an additional screenshot. A view that cannot display a feature cannot confirm its absence. Keep the same unknown-field protection and duplicate confirmation as V1, including across overlapping screenshots and CSVs.
- Let trainers select an existing Pokémon and add an updated screenshot after evolution, power-up or a move change. Preview conflicting changes and keep the same record ID.
- Provide a cleanup action for processed images that preserves extracted data. Keep images needed for pending reviews available. Use the supplied PNGs as initial validation fixtures and add cases for every supported screen and protected feature. Validate which details each view can establish before shipping.

## V3 · Battle recommendations

Offer a single Battle usefulness setting, with optional expansion into specific modes. Recommend across the trainer's whole collection. Current recommended rosters are protected, and a shortfall in the cleanup target never breaks those teams.

| Mode                  | Default                                                                                                                                                                                                                       |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Raids and gym attacks | Aim for two raid teams of six per attacking type. One Pokémon can cover multiple types if the confirmed build serves both without switching moves. Distinct builds can justify keeping additional copies.                     |
| Gym defense           | Recommend three defenders across the collection. Allow adjustment in advanced settings.                                                                                                                                       |
| GO Battle League      | Per selected format, recommend three complete teams of three with complementary coverage. A Pokémon can appear on multiple teams using its confirmed build.                                                                   |
| Max Battles           | Separate opt-in. Scale roster size and default minimums with available capacity, allow adjustment, and prioritize Dynamax and Gigantamax builds that cover more matchups. Apply normal collectible protections when disabled. |

- Show each Pokémon's best build per relevant role, including evolution, power-ups and moveset, plus special requirements such as Elite TMs. Assess IVs and moves by role. Call out when the trainer's build already serves the role well.
- Show proposed, accepted, declined, canceled and confirmed upgrades. Accepting an upgrade protects that Pokémon until completed or canceled. Declining an upgrade dismisses that suggestion and uses the current confirmed build. Only count completed, trainer-confirmed builds toward roster slots. Show open slots and planned builds separately.
- Update recommended lineups when a stronger Pokémon is imported. If a displaced Pokémon leaves its last recommended team and has no other protection, it can become eligible immediately. Identify the lineup change and remind the trainer to update in-game tags or saved parties. Provide quick access to teams and defenders, suggested tags, and copyable searches.
- Automatically refresh official Pokémon GO rules and announced changes. Use established community rankings for rankings and movesets, starting with [GameInfo Top Attackers](https://pokemon.gameinfo.io/en/pokemon/list/top-attackers) as a raid source candidate. Show provenance and the last successful refresh. Validate source access, permitted reuse, data availability and the PvP and Max ranking sources before implementation. The [official GO Battle League leaderboard](https://pokemongo.com/en/leaderboard) ranks trainers. Validate datasets before activation, keep the last good snapshot on failure and show its age. Unsupported or invalid data cannot justify a transfer.

## Implementation boundaries

- Suggested architecture is a responsive web client, a profile-scoped API on the MacBook, and a transactional local database such as SQLite. Isolate auth, import adapters, cleanup planning, and battle-data ingestion. Keep screenshot processing local as remote access is added unless that decision changes explicitly.
- Core entities are Trainer, Pokémon Record, Field Observation, Import Batch, Duplicate Review, Keep Rule, Team, Cleanup Plan, Transfer Event, Upgrade Plan, and Battle Dataset Version. Version plans or inputs so concurrent edits do not silently produce stale results. Make bulk confirmations, import commits and restore operations transactional. Use migrations and test restoring backups from older versions.
- Open implementation decisions include exact capacity bands, what the Poké Genie exporter encodes without explicit columns, local OCR accuracy and which views expose each field, ranking-source integration and reuse terms, and local HTTPS setup on family iPhones. Benchmark the planner and batch importer on at least 2,700 records before choosing performance thresholds.

## Acceptance scenarios

| Scenario                                                           | Expected result                                                                                                 |
| ------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------- |
| Last shiny costume and several ordinary copies                     | The shiny costume stays by default. Any explicit override is visible.                                           |
| Mew minimum 2 and Mythical minimum 4                               | The effective Mew floor is 4 when available. A shiny can satisfy both the species and shiny floors.             |
| CSV lacks background and a screenshot may duplicate another record | The review queue presents the uncertainty and the plan does not rely on those records as safe duplicates.       |
| Target 100 additional slots, maximum 70                            | Recommend at most 70 and show the 30-slot gap and upgrade options.                                              |
| 18 of 100 candidates are starred                                   | Show 18% favorited and the unfavorite step on those candidates.                                                 |
| A full refresh omits previously imported Pokémon                   | Require confirmation before removing them. Undo restores the selected records and recalculates the plan.        |
| A raid upgrade has been accepted and is pending                    | Protect the Pokémon and leave the roster slot open. Completion fills the slot if the confirmed build qualifies. |
| A stronger imported attacker displaces a roster member             | Refresh the roster, explain the change, and flag eligible transfers and obsolete tags.                          |
| Trainers import concurrently on different iPhones                  | Each operation affects only the authenticated profile.                                                          |
| Battle data refresh fails validation                               | Keep the last successful dataset and show when it was refreshed.                                                |
