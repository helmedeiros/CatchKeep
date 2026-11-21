import { useMemo, useState } from "react";
import { toast } from "sonner";
import { Sprite } from "../ui/Sprite";
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
  dexNumber: number | undefined;
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
  const [confirmingAll, setConfirmingAll] = useState(false);
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

  const submitAll = () =>
    confirm.mutate(
      { features: chosen },
      {
        onSuccess: ({ updated }) => {
          setConfirmingAll(false);
          toast.success(
            `Confirmed ${updated} Pokémon: no ${describe(chosen)}.`,
          );
        },
      },
    );
  const total = groups.reduce((n, g) => n + g.unknown, 0);

  if (pokemon.length === 0) return null;
  return (
    <section
      aria-label="Confirm what the export can't tell"
      className={`${card} flex flex-col gap-3`}
    >
      <h3 className="font-semibold">Confirm what the export can't tell</h3>
      <p className="text-sm text-muted">
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
              className="size-4 accent-accent"
            />
            {label}
          </label>
        ))}
      </div>
      {chosen.length === 0 ? (
        <p className="text-sm text-muted">
          Choose at least one feature to confirm.
        </p>
      ) : groups.length === 0 ? (
        <p className="text-sm text-muted">
          Nothing left to confirm for these features.
        </p>
      ) : (
        <ul className="flex flex-col divide-y divide-line">
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
                  <Sprite
                    size="sm"
                    species={g.species}
                    dexNumber={g.dexNumber}
                  />
                  <p className="min-w-0 flex-1 truncate">
                    <span className="font-medium">{name}</span>{" "}
                    <span className="text-sm text-muted">
                      {g.unknown} Pokémon
                    </span>
                  </p>
                  <button
                    type="button"
                    className="shrink-0 rounded-full bg-accent px-3 py-1.5 text-sm font-semibold text-accent-ink hover:bg-accent/90 disabled:opacity-50"
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
      {chosen.length > 0 && groups.length > 1 && !confirmingAll && (
        <button
          type="button"
          className={secondaryButton}
          onClick={() => setConfirmingAll(true)}
        >
          Confirm all {groups.length} species
        </button>
      )}
      {confirmingAll && chosen.length > 0 && groups.length > 0 && (
        <div className="flex flex-col gap-2 rounded-xl bg-warn/10 p-3 text-sm text-warn">
          <p>
            None of your {total} Pokémon in these {groups.length} species will
            be counted as {describe(chosen)}. Only do this if you've checked
            them in the game.
          </p>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className="rounded-full bg-accent px-3 py-1.5 font-semibold text-accent-ink hover:bg-accent/90 disabled:opacity-50"
              disabled={confirm.isPending}
              onClick={submitAll}
            >
              Yes, I checked them all
            </button>
            <button
              type="button"
              className={secondaryButton}
              onClick={() => setConfirmingAll(false)}
            >
              Not yet
            </button>
          </div>
        </div>
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
      dexNumber: p.dexNumber,
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
