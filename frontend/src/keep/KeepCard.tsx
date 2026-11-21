import { type ReactNode, useState } from "react";
import {
  alert,
  card,
  input,
  primaryButton,
  secondaryButton,
} from "../ui/styles";
import type { Pokemon } from "../collection/useCollection";
import { CategoryRules } from "./CategoryRules";
import { KeepGroups } from "./KeepGroups";
import { SpeciesRuleList } from "./SpeciesRules";
import {
  type CapacityBand,
  type Goal,
  type KeepSummary,
  type Setup,
  useCapacityBands,
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

/**
 * Setup step 1 (storage and goals) and what the keep rules do with it. After
 * the first setup it points to the optional steps: species rules, favorites,
 * protection and teams (spec V1-SET-02, V1-SET-03).
 */
export function KeepCard({ pokemon }: { pokemon: Pokemon[] }) {
  const setup = useSetup();
  const [editing, setEditing] = useState(false);
  const [justSetUp, setJustSetUp] = useState(false);

  if (setup.isPending) return null;
  if (setup.isError) {
    return <p className={alert}>Couldn't load your setup.</p>;
  }
  if (!setup.data || editing) {
    return (
      <SetupForm
        initial={setup.data}
        onDone={() => {
          setJustSetUp(!setup.data);
          setEditing(false);
        }}
        onCancel={setup.data ? () => setEditing(false) : undefined}
      />
    );
  }
  return (
    <Summary
      setup={setup.data}
      pokemon={pokemon}
      onChange={() => {
        setJustSetUp(false);
        setEditing(true);
      }}
      nextSteps={
        justSetUp && <NextSteps onDismiss={() => setJustSetUp(false)} />
      }
    />
  );
}

function NextSteps({ onDismiss }: { onDismiss: () => void }) {
  return (
    <section
      aria-label="Optional next steps"
      className="flex flex-col gap-2 rounded-xl bg-teal-50 p-3 text-sm dark:bg-teal-950"
    >
      <p>
        You're set. If you like, fine-tune it: tap a species below to set your
        own rule or make it a favorite, protect single Pokémon in your
        collection, or build a team. Everything can change later.
      </p>
      <button
        type="button"
        className={`${secondaryButton} self-start`}
        onClick={onDismiss}
      >
        Got it
      </button>
    </section>
  );
}

function defaultsText(
  bands: CapacityBand[] | undefined,
  capacity: number,
  trading: boolean,
): string | null {
  const band = bands?.find((b) => capacity <= b.upTo);
  if (!band) return null;
  const extra = trading
    ? ", plus 1 more of rare, shiny, costume and background variants for trading"
    : "";
  return `With ${capacity} slots the defaults keep the best ${band.ordinary} of each variant and ${band.rare} of legendary, mythical and Ultra Beast ones${extra}.`;
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
  const bands = useCapacityBands();
  const [slots, setSlots] = useState(initial ? String(initial.capacity) : "");
  const [goals, setGoals] = useState<Goal[]>(initial?.goals ?? []);
  const [inGame, setInGame] = useState(
    initial?.inGameCount ? String(initial.inGameCount) : "",
  );
  const [limitText, setLimitText] = useState(String(initial?.limit ?? 80));
  const capacity = Number(slots);
  const inGameCount = Number(inGame);
  const limit = Number(limitText);
  const capacityOk =
    Number.isInteger(capacity) && capacity >= 1 && capacity <= 20000;
  const problem = !capacityOk
    ? "Enter how many Pokémon your storage holds (1 to 20,000)."
    : goals.length === 0
      ? "Choose at least one goal."
      : inGame !== "" &&
          !(
            Number.isInteger(inGameCount) &&
            inGameCount >= 0 &&
            inGameCount <= 20000
          )
        ? "The number of Pokémon in the game must be 0 to 20,000."
        : !(Number.isInteger(limit) && limit >= 50 && limit <= 100)
          ? "The storage limit must be 50% to 100%."
          : null;

  const move = (from: number, to: number) =>
    setGoals((cur) => {
      const next = [...cur];
      const [g] = next.splice(from, 1);
      if (g) next.splice(to, 0, g);
      return next;
    });

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
          What do you keep Pokémon for? Pick any; order them below if more than
          one.
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
        <div className="flex flex-col gap-1.5 text-sm">
          <p className="text-slate-600 dark:text-slate-400">
            Priority: {priority(goals)}
          </p>
          <ol aria-label="Goal priority" className="flex flex-col gap-1">
            {goals.map((g, i) => (
              <li key={g} className="flex items-center gap-2">
                <span className="w-5 tabular-nums text-slate-500">
                  {i + 1}.
                </span>
                <span className="flex-1">{label(g)}</span>
                <button
                  type="button"
                  aria-label={`Move ${label(g)} up`}
                  className={secondaryButton}
                  disabled={i === 0}
                  onClick={() => move(i, i - 1)}
                >
                  ↑
                </button>
                <button
                  type="button"
                  aria-label={`Move ${label(g)} down`}
                  className={secondaryButton}
                  disabled={i === goals.length - 1}
                  onClick={() => move(i, i + 1)}
                >
                  ↓
                </button>
              </li>
            ))}
          </ol>
        </div>
      )}
      {capacityOk && (
        <p className="text-sm text-slate-600 dark:text-slate-400">
          {defaultsText(bands.data, capacity, goals.includes("trading"))}
        </p>
      )}
      <label className="flex flex-col gap-1 text-sm">
        Pokémon in the game now (optional)
        <input
          type="number"
          inputMode="numeric"
          min={0}
          max={20000}
          className={input}
          value={inGame}
          onChange={(e) => setInGame(e.target.value)}
        />
        <span className="text-xs text-slate-500">
          The game shows it at the top of your Pokémon list, like 2500/4000.
          Without it, CatchKeep counts what you imported.
        </span>
      </label>
      <label className="flex flex-col gap-1 text-sm">
        Keep storage under (%)
        <input
          type="number"
          inputMode="numeric"
          min={50}
          max={100}
          className={input}
          value={limitText}
          onChange={(e) => setLimitText(e.target.value)}
        />
        <span className="text-xs text-slate-500">
          Room left for new catches, like a Community Day. 80% is a good start.
        </span>
      </label>
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
        onClick={() =>
          save.mutate(
            {
              capacity,
              goals,
              limit,
              ...(inGame !== "" && inGameCount > 0 && { inGameCount }),
            },
            { onSuccess: onDone },
          )
        }
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
  nextSteps,
}: {
  setup: Setup;
  pokemon: Pokemon[];
  onChange: () => void;
  nextSteps: ReactNode;
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
      {nextSteps}
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
      {s.protected > 0 && (
        <p className="text-sm text-slate-600 dark:text-slate-400">
          {s.protected} protected by you, always kept.
        </p>
      )}
      {s.needsConfirmation > 0 && (
        <p className="text-sm text-slate-600 dark:text-slate-400">
          Pokémon with unconfirmed features are always kept. Confirm them above
          to see what you can let go.
        </p>
      )}
      <SpeciesRuleList pokemon={pokemon} />
      <CategoryRules />
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
