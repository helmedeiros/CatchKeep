import { useState } from "react";
import { toast } from "sonner";
import type { Pokemon } from "../collection/useCollection";
import { Sprite } from "../ui/Sprite";
import { input, secondaryButton } from "../ui/styles";
import {
  type SpeciesRule,
  useSaveSpeciesRules,
  useSpeciesRules,
} from "./useKeep";

const describe = (r: SpeciesRule) =>
  r.favorite ? "favorite, keep every copy" : `keep at least ${r.minimum}`;

/** Sets or clears the trainer's own rule for one species. */
export function SpeciesRuleEditor({ species }: { species: string }) {
  const rules = useSpeciesRules(true);
  const save = useSaveSpeciesRules();
  const current = rules.data?.find((r) => r.species === species);
  const [minimum, setMinimum] = useState(String(current?.minimum || ""));
  const [favorite, setFavorite] = useState(current?.favorite ?? false);
  const count = Number(minimum);
  const valid = favorite || (Number.isInteger(count) && count >= 1);

  const replace = (rule: SpeciesRule | null) => {
    const others = (rules.data ?? []).filter((r) => r.species !== species);
    save.mutate(rule ? [...others, rule] : others, {
      onSuccess: () =>
        toast.success(
          rule
            ? `${species}: ${describe(rule)}.`
            : `${species} follows the default rules again.`,
        ),
      onError: () => toast.error(`Couldn't save the rule for ${species}.`),
    });
  };

  return (
    <form
      aria-label={`Keep rule for ${species}`}
      className="flex flex-wrap items-end gap-3 rounded-xl bg-slate-50 p-3 text-sm dark:bg-slate-950"
      onSubmit={(e) => {
        e.preventDefault();
        if (valid)
          replace({ species, minimum: favorite ? 0 : count, favorite });
      }}
    >
      <label className="flex flex-col gap-1">
        <span className="text-slate-600 dark:text-slate-400">
          Keep at least
        </span>
        <input
          type="number"
          inputMode="numeric"
          min={1}
          disabled={favorite}
          value={favorite ? "" : minimum}
          onChange={(e) => setMinimum(e.target.value)}
          className={`${input} w-24`}
        />
      </label>
      <label className="flex items-center gap-2 pb-2">
        <input
          type="checkbox"
          checked={favorite}
          onChange={(e) => setFavorite(e.target.checked)}
          className="size-4 accent-teal-700"
        />
        Favorite: keep every copy
      </label>
      <div className="flex gap-2">
        <button
          type="submit"
          disabled={!valid || save.isPending}
          className="rounded-lg bg-teal-700 px-3 py-2 font-semibold text-white hover:bg-teal-800 disabled:opacity-50"
        >
          Save rule
        </button>
        {current && (
          <button
            type="button"
            className={secondaryButton}
            disabled={save.isPending}
            onClick={() => replace(null)}
          >
            Use the default
          </button>
        )}
      </div>
      <p className="basis-full text-xs text-slate-500">
        Counts copies across all of {species}'s variants and replaces the
        default minimum.
      </p>
    </form>
  );
}

/** The trainer's species rules, each removable. */
export function SpeciesRuleList({ pokemon }: { pokemon: Pokemon[] }) {
  const rules = useSpeciesRules(true);
  const save = useSaveSpeciesRules();
  const list = rules.data ?? [];
  if (list.length === 0) return null;
  const dexOf = (species: string) =>
    pokemon.find((p) => p.species === species)?.dexNumber;
  return (
    <section aria-label="Your species rules" className="flex flex-col gap-1">
      <h4 className="text-sm font-semibold">Your species rules</h4>
      <ul className="flex flex-col divide-y divide-slate-100 text-sm dark:divide-slate-800">
        {list.map((r) => (
          <li key={r.species} className="flex items-center gap-3 py-1.5">
            <Sprite
              size="sm"
              species={r.species}
              dexNumber={dexOf(r.species)}
            />
            <span className="min-w-0 flex-1 truncate">
              <span className="font-medium">{r.species}</span> · {describe(r)}
            </span>
            <button
              type="button"
              className={secondaryButton}
              disabled={save.isPending}
              aria-label={`Remove the rule for ${r.species}`}
              onClick={() =>
                save.mutate(list.filter((x) => x.species !== r.species))
              }
            >
              Remove
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}
