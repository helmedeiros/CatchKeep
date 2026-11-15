import { type ChangeEvent, useState } from "react";
import { toast } from "sonner";
import { alert, card, primaryButton, secondaryButton } from "../ui/styles";
import {
  type ImportPreview,
  useCommitImport,
  useStageImport,
} from "./useCollection";

export function ImportCard({ hasPokemon }: { hasPokemon: boolean }) {
  const stage = useStageImport();
  const commit = useCommitImport();
  const [preview, setPreview] = useState<ImportPreview | null>(null);

  const choose = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    stage.mutate(await file.text(), { onSuccess: setPreview });
  };

  const add = (p: ImportPreview) =>
    commit.mutate(p.id, {
      onSuccess: (done) => {
        setPreview(null);
        toast.success(`Added ${done.newCount} Pokémon to your collection.`);
      },
    });

  if (preview) {
    return (
      <Preview
        preview={preview}
        onAdd={add}
        onCancel={() => setPreview(null)}
        busy={commit.isPending}
      />
    );
  }

  return (
    <section className={`${card} flex flex-col gap-3`}>
      <h3 className="font-semibold">
        {hasPokemon ? "Update from a new export" : "Bring in your collection"}
      </h3>
      {!hasPokemon && (
        <p className="text-sm text-slate-600 dark:text-slate-400">
          Export your Pokémon from Poke Genie as CSV, then choose the file here.
          You'll see a preview before anything is added.
        </p>
      )}
      <label className={`${primaryButton} cursor-pointer text-center`}>
        Choose your Poke Genie export
        <input
          type="file"
          accept=".csv,text/csv"
          className="sr-only"
          onChange={choose}
        />
      </label>
      {stage.isPending && (
        <p className="text-sm text-slate-500">Reading the export…</p>
      )}
      {stage.isError && (
        <p role="alert" className={alert}>
          {stage.error.message}
        </p>
      )}
    </section>
  );
}

function Preview({
  preview,
  onAdd,
  onCancel,
  busy,
}: {
  preview: ImportPreview;
  onAdd: (p: ImportPreview) => void;
  onCancel: () => void;
  busy: boolean;
}) {
  return (
    <section
      aria-label="Import preview"
      className={`${card} flex flex-col gap-3`}
    >
      <h3 className="font-semibold">{preview.totalRows} Pokémon found</h3>
      <ul className="flex flex-col gap-1 text-sm">
        <li>
          <strong>{preview.newCount}</strong> new
        </li>
        {preview.possibleMatchCount > 0 && (
          <li>
            <strong>{preview.possibleMatchCount}</strong> may already be in your
            collection. They are held back so nothing is counted twice.
          </li>
        )}
      </ul>
      {preview.invalidRows.length > 0 && (
        <details className="text-sm">
          <summary className="cursor-pointer">
            {preview.invalidRows.length} notes about the file
          </summary>
          <ul className="mt-2 flex flex-col gap-1 text-slate-600 dark:text-slate-400">
            {preview.invalidRows.map((r) => (
              <li key={`${r.row}-${r.message}`}>
                Line {r.row}: {r.message}
              </li>
            ))}
          </ul>
        </details>
      )}
      {preview.unknownFields.length > 0 && (
        <p className="rounded-lg bg-sky-50 px-3 py-2 text-sm text-sky-900 dark:bg-sky-950 dark:text-sky-100">
          This export can't tell {preview.unknownFields.join(", ")}. CatchKeep
          treats them as unknown, never as "no".
        </p>
      )}
      <button
        type="button"
        className={primaryButton}
        disabled={busy || preview.newCount === 0}
        onClick={() => onAdd(preview)}
      >
        Add {preview.newCount} Pokémon
      </button>
      <button type="button" className={secondaryButton} onClick={onCancel}>
        Cancel
      </button>
    </section>
  );
}
