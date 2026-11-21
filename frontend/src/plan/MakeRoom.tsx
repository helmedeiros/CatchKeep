import { useMemo, useState } from "react";
import { PokemonItem } from "../collection/PokemonItem";
import type { Pokemon } from "../collection/useCollection";
import { useSetup } from "../keep/useKeep";
import { alert, card, input, secondaryButton } from "../ui/styles";
import { FilterBar } from "../collection/FilterBar";
import { applyFilters, NO_FILTERS } from "../collection/filters";
import { useFilters } from "../collection/useFilters";
import { FindInGame } from "./FindInGame";
import { StorageBar } from "./StorageBar";
import { type Plan, type TargetKind, usePlan } from "./usePlan";

const SHOWN = 30;
const WEAKEST_FIRST = { ...NO_FILTERS, sort: "plan" as const };
const NO_CONTEXT = {
  teamIds: new Set<string>(),
  transferIds: new Set<string>(),
};

const TARGETS: Array<{ kind: TargetKind; label: string; field?: string }> = [
  { kind: "limit", label: "Get under my limit" },
  {
    kind: "free_more",
    label: "Free more slots, like before a Community Day",
    field: "Slots to free",
  },
  {
    kind: "reach_free",
    label: "Have a number of slots free",
    field: "Free slots wanted",
  },
  {
    kind: "percent",
    label: "Free a share of my storage",
    field: "Percent to free",
  },
];

/**
 * How full storage is and what to transfer to make the room asked for
 * (spec §7). Nothing changes until the trainer transfers in the game.
 */
export function MakeRoom({ pokemon }: { pokemon: Pokemon[] }) {
  const setup = useSetup();
  const [kind, setKind] = useState<TargetKind>("limit");
  const [amount, setAmount] = useState("");
  const value = amount === "" ? undefined : Number(amount);
  const plan = usePlan(kind, value, Boolean(setup.data));
  if (!setup.data) return null;

  return (
    <section aria-label="Make room" className={`${card} flex flex-col gap-3`}>
      {plan.data && <StorageBar use={plan.data.use} />}
      <fieldset className="flex flex-col gap-1.5 text-sm">
        <legend className="mb-1">How much room do you want?</legend>
        {TARGETS.map((t) => (
          <label key={t.kind} className="flex items-center gap-2">
            <input
              type="radio"
              name="target"
              className="size-4 accent-accent"
              checked={kind === t.kind}
              onChange={() => {
                setKind(t.kind);
                setAmount("");
              }}
            />
            {t.label}
          </label>
        ))}
      </fieldset>
      {TARGETS.map(
        (t) =>
          t.kind === kind &&
          t.field && (
            <label key={t.kind} className="flex flex-col gap-1 text-sm">
              {t.field}
              <input
                type="number"
                inputMode="numeric"
                min={1}
                max={kind === "percent" ? 100 : 20000}
                className={input}
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
              />
            </label>
          ),
      )}
      {plan.isError && (
        <p role="alert" className={alert}>
          {plan.error.message}
        </p>
      )}
      {plan.data && <Suggestions plan={plan.data} pokemon={pokemon} />}
    </section>
  );
}

function Suggestions({ plan, pokemon }: { plan: Plan; pokemon: Pokemon[] }) {
  const [showAll, setShowAll] = useState(false);
  const { filters } = useFilters("make-room", WEAKEST_FIRST);
  const byId = useMemo(() => new Map(pokemon.map((p) => [p.id, p])), [pokemon]);
  if (plan.wanted === 0) {
    return (
      <p className="rounded-xl bg-highlight px-3 py-2 text-sm">
        {plan.kind === "limit"
          ? "You're under your limit. Nothing to transfer."
          : "You already have that much room. Nothing to transfer."}
      </p>
    );
  }
  const items = new Map(plan.transfers.map((t) => [t.pokemonId, t]));
  const planned = plan.transfers.flatMap((t) => {
    const p = byId.get(t.pokemonId);
    return p ? [p] : [];
  });
  const filtered = applyFilters(planned, filters, NO_CONTEXT);
  const shown = showAll ? filtered : filtered.slice(0, SHOWN);
  return (
    <div className="flex flex-col gap-2">
      <h4 className="text-lg font-semibold">
        Transfer {plan.transfers.length} Pokémon
      </h4>
      {plan.shortfall > 0 && (
        <p className="text-sm text-warn">
          Your keep rules allow {plan.maxFeasible}, so that's {plan.shortfall}{" "}
          short of your target. Lower a species rule or protect fewer Pokémon to
          free more.
        </p>
      )}
      {plan.favorites > 0 && (
        <p className="text-sm text-muted">
          {plan.favorites} of these are starred in the game: unstar them first.
        </p>
      )}
      <FindInGame searches={plan.searches} byId={byId} />
      <p className="text-xs text-muted">
        Weakest first: lowest IVs, then CP, then oldest catch. Every one keeps
        your rules: better copies of each stay.
      </p>
      <FilterBar
        list="make-room"
        chips={["shiny", "lucky", "shadow", "purified", "costume"]}
        shown={filtered.length}
        total={plan.transfers.length}
        defaults={WEAKEST_FIRST}
      />
      <ul
        aria-label="Suggested transfers"
        className="flex flex-col divide-y divide-line"
      >
        {shown.map((p) => (
          <li key={p.id} className="py-2">
            <PokemonItem pokemon={p}>
              <span className="rounded-full bg-soft px-2 py-0.5 font-medium text-muted">
                {items.get(p.id)?.keptInGroup} better {p.species} kept
              </span>
            </PokemonItem>
          </li>
        ))}
      </ul>
      {filtered.length > SHOWN && (
        <button
          type="button"
          className={`${secondaryButton} self-start`}
          onClick={() => setShowAll((v) => !v)}
        >
          {showAll ? "Show fewer" : `Show all ${filtered.length}`}
        </button>
      )}
    </div>
  );
}
