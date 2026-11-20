import { toast } from "sonner";
import type { Pokemon } from "../collection/useCollection";
import { Sprite } from "../ui/Sprite";
import { secondaryButton } from "../ui/styles";
import { RuleForm, type RuleValue } from "./RuleForm";
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

  const replace = (rule: RuleValue | null) => {
    const others = (rules.data ?? []).filter((r) => r.species !== species);
    const next = rule && { species, ...rule };
    save.mutate(next ? [...others, next] : others, {
      onSuccess: () =>
        toast.success(
          next
            ? `${species}: ${describe(next)}.`
            : `${species} follows the default rules again.`,
        ),
      onError: () => toast.error(`Couldn't save the rule for ${species}.`),
    });
  };

  return (
    <RuleForm
      name={species}
      current={current}
      pending={save.isPending}
      favoriteLabel="Favorite: keep every copy"
      hint={`Counts copies across all of ${species}'s variants and replaces the default minimum.`}
      onSave={replace}
    />
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
