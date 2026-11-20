import { useState } from "react";
import { toast } from "sonner";
import {
  CATEGORIES,
  CATEGORY_NAMES,
  CATEGORY_TITLES,
  type Category,
} from "./categories";
import { RuleForm, type RuleValue } from "./RuleForm";
import { secondaryButton } from "../ui/styles";
import { useCategoryRules, useSaveCategoryRules } from "./useKeep";

/**
 * One rule for every legendary, mythical or Ultra Beast species; each
 * species' own rule still wins (OD-02).
 */
export function CategoryRules() {
  const [open, setOpen] = useState(false);
  const rules = useCategoryRules(open);
  const save = useSaveCategoryRules();

  const replace = (category: Category, rule: RuleValue | null) => {
    const others = (rules.data ?? []).filter((r) => r.category !== category);
    const title = CATEGORY_TITLES[category];
    save.mutate(rule ? [...others, { category, ...rule }] : others, {
      onSuccess: () =>
        toast.success(
          rule
            ? `${title}: rule saved.`
            : `${title} follow the default rules again.`,
        ),
      onError: () => toast.error(`Couldn't save the rule for ${title}.`),
    });
  };

  return (
    <div className="flex flex-col gap-2">
      <button
        type="button"
        aria-expanded={open}
        className={`${secondaryButton} self-start`}
        onClick={() => setOpen((v) => !v)}
      >
        Rules for legendary, mythical and Ultra Beasts
      </button>
      {open &&
        rules.data &&
        CATEGORIES.map((c) => (
          <RuleForm
            key={c}
            name={CATEGORY_TITLES[c]}
            current={rules.data.find((r) => r.category === c)}
            pending={save.isPending}
            favoriteLabel="Favorite: keep every copy of each"
            hint={`Applies to each ${CATEGORY_NAMES[c]} species as if it were that species' own rule. A species with its own rule follows that instead.`}
            onSave={(rule) => replace(c, rule)}
          />
        ))}
    </div>
  );
}
