import { useMemo, useState } from "react";
import { toast } from "sonner";
import { alert, card, secondaryButton } from "../ui/styles";
import {
  type ConfirmableFeature,
  type Pokemon,
  useConfirmAbsent,
} from "./useCollection";

const FEATURES: Array<[ConfirmableFeature, string]> = [
  ["shiny", "Shiny"],
  ["costume", "Costume"],
  ["background", "Background"],
  ["dynamax", "Dynamax"],
  ["gigantamax", "Gigantamax"],
];
const INITIAL_GROUPS = 12;

interface Group {
  species: string;
  form: string;
  unknown: number;
}

/**
 * Lets the trainer confirm, per species, features a CSV export can't tell
 * (spec §7.3). Only species with a chosen feature still unknown are listed.
 */
export function ConfirmFeatures({ pokemon }: { pokemon: Pokemon[] }) {
  const confirm = useConfirmAbsent();
  const [chosen, setChosen] = useState<ConfirmableFeature[]>(
    FEATURES.map(([f]) => f),
  );
  const [showAll, setShowAll] = useState(false);
  const groups = useMemo(
    () => unknownGroups(pokemon, chosen),
    [pokemon, chosen],
  );
  const visible = showAll ? groups : groups.slice(0, INITIAL_GROUPS);

  const toggle = (f: ConfirmableFeature) =>
    setChosen((cur) =>
      cur.includes(f)
        ? cur.filter((x) => x !== f)
        : FEATURES.map(([x]) => x).filter((x) => x === f || cur.includes(x)),
    );

  const submit = (g: Group) =>
    confirm.mutate(
      { species: g.species, form: g.form, features: chosen },
      {
        onSuccess: ({ updated }) =>
          toast.success(
            `Confirmed ${updated} ${g.species}: no ${describe(chosen)}.`,
          ),
      },
    );

  if (pokemon.length === 0) return null;
  return (
    <section
      aria-label="Confirm what the export can't tell"
      className={`${card} flex flex-col gap-3`}
    >
      <h3 className="font-semibold">Confirm what the export can't tell</h3>
      <p className="text-sm text-slate-600 dark:text-slate-400">
        Until you confirm them, CatchKeep treats these as unknown and keeps
        those Pokémon safe. Check a species in the game, then confirm that none
        of them have:
      </p>
      <div className="flex flex-wrap gap-x-4 gap-y-2 text-sm">
        {FEATURES.map(([f, label]) => (
          <label key={f} className="flex items-center gap-1.5">
            <input
              type="checkbox"
              checked={chosen.includes(f)}
              onChange={() => toggle(f)}
              className="size-4 accent-teal-700"
            />
            {label}
          </label>
        ))}
      </div>
      {chosen.length === 0 ? (
        <p className="text-sm text-slate-500">
          Choose at least one feature to confirm.
        </p>
      ) : groups.length === 0 ? (
        <p className="text-sm text-slate-500">
          Nothing left to confirm for these features.
        </p>
      ) : (
        <ul className="flex flex-col divide-y divide-slate-100 dark:divide-slate-800">
          {visible.map((g) => {
            const name =
              g.form === "Normal" ? g.species : `${g.species} · ${g.form}`;
            return (
              <li key={`${g.species}|${g.form}`}>
                <div
                  role="group"
                  aria-label={name}
                  className="flex items-center justify-between gap-3 py-2"
                >
                  <p className="min-w-0 truncate">
                    <span className="font-medium">{name}</span>{" "}
                    <span className="text-sm text-slate-500">
                      {g.unknown} Pokémon
                    </span>
                  </p>
                  <button
                    type="button"
                    className="shrink-0 rounded-lg bg-teal-700 px-3 py-1.5 text-sm font-semibold text-white hover:bg-teal-800 disabled:opacity-50"
                    disabled={confirm.isPending}
                    onClick={() => submit(g)}
                  >
                    Confirm
                  </button>
                </div>
              </li>
            );
          })}
        </ul>
      )}
      {groups.length > INITIAL_GROUPS && (
        <button
          type="button"
          className={secondaryButton}
          onClick={() => setShowAll((v) => !v)}
        >
          {showAll ? "Show fewer" : `Show all ${groups.length} species`}
        </button>
      )}
      {confirm.isError && (
        <p role="alert" className={alert}>
          {confirm.error.message}
        </p>
      )}
    </section>
  );
}

function unknownGroups(pokemon: Pokemon[], chosen: ConfirmableFeature[]) {
  const byKey = new Map<string, Group>();
  for (const p of pokemon) {
    if (!chosen.some((f) => p.attributes[f] === "unknown")) continue;
    const key = `${p.species}|${p.form}`;
    const g = byKey.get(key) ?? {
      species: p.species,
      form: p.form,
      unknown: 0,
    };
    g.unknown += 1;
    byKey.set(key, g);
  }
  return [...byKey.values()].sort(
    (a, b) => b.unknown - a.unknown || a.species.localeCompare(b.species),
  );
}

function describe(features: ConfirmableFeature[]) {
  const words = features.map((f) =>
    f === "dynamax" || f === "gigantamax"
      ? f[0]!.toUpperCase() + f.slice(1)
      : f,
  );
  return words.length > 1
    ? `${words.slice(0, -1).join(", ")} or ${words.at(-1)}`
    : (words[0] ?? "");
}
