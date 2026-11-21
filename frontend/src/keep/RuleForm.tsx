import { useState } from "react";
import { input, secondaryButton } from "../ui/styles";

export type RuleValue = { minimum: number; favorite: boolean };

/** Sets a minimum or favorite for a species or category, or clears it. */
export function RuleForm({
  name,
  current,
  pending,
  favoriteLabel,
  hint,
  onSave,
}: {
  name: string;
  current: RuleValue | undefined;
  pending: boolean;
  favoriteLabel: string;
  hint: string;
  onSave: (rule: RuleValue | null) => void;
}) {
  const [minimum, setMinimum] = useState(String(current?.minimum || ""));
  const [favorite, setFavorite] = useState(current?.favorite ?? false);
  const count = Number(minimum);
  const valid = favorite || (Number.isInteger(count) && count >= 1);

  return (
    <form
      aria-label={`Keep rule for ${name}`}
      className="flex flex-wrap items-end gap-3 rounded-xl bg-soft p-3 text-sm"
      onSubmit={(e) => {
        e.preventDefault();
        if (valid) onSave({ minimum: favorite ? 0 : count, favorite });
      }}
    >
      <label className="flex flex-col gap-1">
        <span className="text-muted">Keep at least</span>
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
          className="size-4 accent-accent"
        />
        {favoriteLabel}
      </label>
      <div className="flex gap-2">
        <button
          type="submit"
          disabled={!valid || pending}
          className="rounded-full bg-accent px-3 py-2 font-semibold text-accent-ink hover:bg-accent/90 disabled:opacity-50"
        >
          Save rule
        </button>
        {current && (
          <button
            type="button"
            className={secondaryButton}
            disabled={pending}
            onClick={() => onSave(null)}
          >
            Use the default
          </button>
        )}
      </div>
      <p className="basis-full text-xs text-muted">{hint}</p>
    </form>
  );
}
