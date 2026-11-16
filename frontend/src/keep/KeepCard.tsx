import { useState } from "react";
import {
  alert,
  card,
  input,
  primaryButton,
  secondaryButton,
} from "../ui/styles";
import type { Pokemon } from "../collection/useCollection";
import { KeepGroups } from "./KeepGroups";
import { SpeciesRuleList } from "./SpeciesRules";
import {
  type Goal,
  type KeepSummary,
  type Setup,
  useKeepSummary,
  useSaveSetup,
  useSetup,
} from "./useKeep";

const GOALS: Array<[Goal, string]> = [
  ["collecting", "Collecting"],
  ["trading", "Trading"],
  ["battling", "Battling"],
];
const label = (g: Goal) => GOALS.find(([x]) => x === g)?.[1] ?? g;
const priority = (goals: Goal[]) => goals.map(label).join(", then ");

/** Setup step 1 (storage and goals) and what the keep rules do with it. */
export function KeepCard({ pokemon }: { pokemon: Pokemon[] }) {
  const setup = useSetup();
  const [editing, setEditing] = useState(false);

  if (setup.isPending) return null;
  if (setup.isError) {
    return <p className={alert}>Couldn't load your setup.</p>;
  }
  if (!setup.data || editing) {
    return (
      <SetupForm
        initial={setup.data}
        onDone={() => setEditing(false)}
        onCancel={setup.data ? () => setEditing(false) : undefined}
      />
    );
  }
  return (
    <Summary
      setup={setup.data}
      pokemon={pokemon}
      onChange={() => setEditing(true)}
    />
  );
}

function SetupForm({
  initial,
  onDone,
  onCancel,
}: {
  initial: Setup | null;
  onDone: () => void;
  onCancel: (() => void) | undefined;
}) {
  const save = useSaveSetup();
  const [slots, setSlots] = useState(initial ? String(initial.capacity) : "");
  const [goals, setGoals] = useState<Goal[]>(initial?.goals ?? []);
  const capacity = Number(slots);
  const capacityOk =
    Number.isInteger(capacity) && capacity >= 1 && capacity <= 20000;
  const problem = !capacityOk
    ? "Enter how many Pokémon your storage holds (1 to 20,000)."
    : goals.length === 0
      ? "Choose at least one goal."
      : null;

  const toggle = (g: Goal) =>
    setGoals((cur) =>
      cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g],
    );

  return (
    <section
      aria-label="Your storage and goals"
      className={`${card} flex flex-col gap-3`}
    >
      <h3 className="font-semibold">Your storage and goals</h3>
      <label className="flex flex-col gap-1 text-sm">
        Storage slots
        <input
          type="number"
          inputMode="numeric"
          min={1}
          max={20000}
          className={input}
          value={slots}
          onChange={(e) => setSlots(e.target.value)}
        />
      </label>
      <fieldset className="flex flex-col gap-1.5 text-sm">
        <legend className="mb-1">
          What do you keep Pokémon for? Tick the most important first.
        </legend>
        {GOALS.map(([g, name]) => (
          <label key={g} className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={goals.includes(g)}
              onChange={() => toggle(g)}
              className="size-4 accent-teal-700"
            />
            {name}
          </label>
        ))}
      </fieldset>
      {goals.length > 1 && (
        <p className="text-sm text-slate-600 dark:text-slate-400">
          Priority: {priority(goals)}
        </p>
      )}
      {problem && slots !== "" && (
        <p className="text-sm text-slate-500">{problem}</p>
      )}
      {save.isError && (
        <p role="alert" className={alert}>
          {save.error.message}
        </p>
      )}
      <button
        type="button"
        className={primaryButton}
        disabled={problem !== null || save.isPending}
        onClick={() => save.mutate({ capacity, goals }, { onSuccess: onDone })}
      >
        Save
      </button>
      {onCancel && (
        <button type="button" className={secondaryButton} onClick={onCancel}>
          Cancel
        </button>
      )}
    </section>
  );
}

function Summary({
  setup,
  pokemon,
  onChange,
}: {
  setup: Setup;
  pokemon: Pokemon[];
  onChange: () => void;
}) {
  const summary = useKeepSummary(true);
  return (
    <section
      aria-label="Your keep rules"
      className={`${card} flex flex-col gap-3`}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="font-semibold">Your keep rules</h3>
          <p className="text-sm text-slate-500">
            {setup.capacity} slots · {priority(setup.goals)}
          </p>
        </div>
        <button type="button" className={secondaryButton} onClick={onChange}>
          Change
        </button>
      </div>
      {summary.isError && (
        <p className={alert}>Couldn't work out your keep rules.</p>
      )}
      {summary.data && <Numbers summary={summary.data} pokemon={pokemon} />}
    </section>
  );
}

function Numbers({
  summary: s,
  pokemon,
}: {
  summary: KeepSummary;
  pokemon: Pokemon[];
}) {
  return (
    <>
      <p className="text-sm text-slate-600 dark:text-slate-400">
        Keeping the best {s.minimum} of each variant and {s.rareMinimum} of
        legendary, mythical and Ultra Beast ones. Shiny, lucky, shadow and other
        variants each count on their own.
      </p>
      <div className="grid grid-cols-3 gap-2 text-center">
        <Stat value={s.kept} label="kept" />
        <Stat value={s.surplus} label="possible transfers" />
        <Stat value={s.needsConfirmation} label="waiting for confirmation" />
      </div>
      {s.needsConfirmation > 0 && (
        <p className="text-sm text-slate-600 dark:text-slate-400">
          Pokémon with unconfirmed features are always kept. Confirm them above
          to see what you can let go.
        </p>
      )}
      <SpeciesRuleList pokemon={pokemon} />
      <KeepGroups groups={s.groups} pokemon={pokemon} />
    </>
  );
}

function Stat({ value, label }: { value: number; label: string }) {
  return (
    <div className="rounded-xl bg-slate-50 px-2 py-3 dark:bg-slate-950">
      <p className="text-xl font-semibold tabular-nums">{value}</p>
      <p className="text-xs text-slate-500">{label}</p>
    </div>
  );
}
