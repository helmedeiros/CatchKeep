import { useState } from "react";
import {
  alert,
  card,
  input,
  primaryButton,
  secondaryButton,
} from "../ui/styles";
import { FIELD_LABELS, FIELDS, type MappableField } from "./fields";
import type { ColumnMapEntry, ImportPreview } from "./useCollection";

function problem(chosen: Map<string, MappableField>): string | null {
  const fields = [...chosen.values()];
  if (!fields.includes("species")) return "Pick the column with the species.";
  const twice = fields.find((f, i) => fields.indexOf(f) !== i);
  return twice ? `${FIELD_LABELS[twice]} is chosen for two columns.` : null;
}

/**
 * Matches the columns of a file CatchKeep does not recognize to fields
 * (spec V1-IMP-02). Columns left on "Skip" are ignored.
 */
export function ColumnMapper({
  preview,
  busy,
  error,
  onRead,
  onCancel,
}: {
  preview: ImportPreview;
  busy: boolean;
  error: string | null;
  onRead: (mapping: ColumnMapEntry[]) => void;
  onCancel: () => void;
}) {
  const [chosen, setChosen] = useState(
    () => new Map(preview.mapping.map((m) => [m.column, m.field])),
  );
  const issue = problem(chosen);

  const choose = (column: string, value: string) =>
    setChosen((cur) => {
      const next = new Map(cur);
      if (value === "") next.delete(column);
      else next.set(column, value as MappableField);
      return next;
    });

  return (
    <form
      aria-label="Match your columns"
      className={`${card} flex flex-col gap-3`}
      onSubmit={(e) => {
        e.preventDefault();
        if (!issue)
          onRead([...chosen].map(([column, field]) => ({ column, field })));
      }}
    >
      <h3 className="font-semibold">Match your columns</h3>
      <p className="text-sm text-muted">
        This isn't a Poke Genie export, so say what each column holds. Only the
        species is required; skipped columns are ignored. CatchKeep remembers
        this for your next file with the same columns.
      </p>
      <ul className="flex flex-col divide-y divide-line">
        {preview.columns.map((c) => (
          <li key={c.name} className="flex flex-col gap-1.5 py-2 text-sm">
            <div className="min-w-0">
              <p className="truncate font-medium">{c.name}</p>
              {c.samples.length > 0 && (
                <p className="truncate text-xs text-muted">
                  e.g. {c.samples.join(", ")}
                </p>
              )}
            </div>
            <select
              aria-label={`What "${c.name}" holds`}
              className={input}
              value={chosen.get(c.name) ?? ""}
              onChange={(e) => choose(c.name, e.target.value)}
            >
              <option value="">Skip this column</option>
              {FIELDS.map((f) => (
                <option key={f} value={f}>
                  {FIELD_LABELS[f]}
                </option>
              ))}
            </select>
          </li>
        ))}
      </ul>
      {issue && <p className="text-sm text-muted">{issue}</p>}
      {error && (
        <p role="alert" className={alert}>
          {error}
        </p>
      )}
      <button
        type="submit"
        className={primaryButton}
        disabled={issue !== null || busy}
      >
        Read the file
      </button>
      <button type="button" className={secondaryButton} onClick={onCancel}>
        Cancel
      </button>
    </form>
  );
}
