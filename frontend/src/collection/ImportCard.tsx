import { type ChangeEvent, useState } from "react";
import { toast } from "sonner";
import { alert, card, primaryButton, secondaryButton } from "../ui/styles";
import { ColumnMapper } from "./ColumnMapper";
import { DuplicateReview } from "./DuplicateReview";
import { MissingFromImport } from "./MissingFromImport";
import {
  type ImportPreview,
  type MatchAnswer,
  useAnswerMatches,
  useCommitImport,
  useMapColumns,
  useStageImport,
} from "./useCollection";

export function ImportCard({ hasPokemon }: { hasPokemon: boolean }) {
  const stage = useStageImport();
  const commit = useCommitImport();
  const answer = useAnswerMatches();
  const map = useMapColumns();
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [remapping, setRemapping] = useState(false);
  const [missing, setMissing] = useState<string[]>([]);

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
        setMissing(done.missing);
        toast.success(doneMessage(done));
      },
    });

  if (missing.length > 0) {
    return <MissingFromImport ids={missing} onDone={() => setMissing([])} />;
  }

  if (preview && (preview.needsMapping || remapping)) {
    return (
      <ColumnMapper
        key={preview.id}
        preview={preview}
        busy={map.isPending}
        error={map.isError ? map.error.message : null}
        onRead={(mapping) =>
          map.mutate(
            { importId: preview.id, mapping },
            {
              onSuccess: (p) => {
                setRemapping(false);
                setPreview(p);
              },
            },
          )
        }
        onCancel={() => {
          setRemapping(false);
          setPreview(null);
        }}
      />
    );
  }

  if (preview) {
    return (
      <Preview
        preview={preview}
        onAdd={add}
        onAnswer={(answers) =>
          answer.mutate(
            { importId: preview.id, answers },
            { onSuccess: setPreview },
          )
        }
        answerError={answer.isError ? answer.error.message : null}
        onCancel={() => setPreview(null)}
        onRemap={
          preview.source === "csv" ? () => setRemapping(true) : undefined
        }
        busy={commit.isPending || answer.isPending}
      />
    );
  }

  return (
    <section className={`${card} flex flex-col gap-3`}>
      <h3 className="font-semibold">
        {hasPokemon ? "Update from a new export" : "Bring in your collection"}
      </h3>
      {!hasPokemon && (
        <p className="text-sm text-muted">
          Export your Pokémon from Poke Genie as CSV, then choose the file here.
          A CSV from another app works too: you'll match its columns first.
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
        <p className="text-sm text-muted">Reading the export…</p>
      )}
      {stage.isError && (
        <p role="alert" className={alert}>
          {stage.error.message}
        </p>
      )}
    </section>
  );
}

function doneMessage({ newCount, updatedCount }: ImportPreview) {
  if (newCount + updatedCount === 0) return "Nothing new in this file.";
  if (updatedCount === 0)
    return `Added ${newCount} Pokémon to your collection.`;
  if (newCount === 0) return `Updated ${updatedCount} Pokémon.`;
  return `Added ${newCount} and updated ${updatedCount} Pokémon.`;
}

function commitLabel({ newCount, updatedCount }: ImportPreview) {
  if (newCount + updatedCount === 0) return "Continue";
  if (updatedCount === 0) return `Add ${newCount} Pokémon`;
  if (newCount === 0) return `Update ${updatedCount} Pokémon`;
  return `Add ${newCount} and update ${updatedCount} Pokémon`;
}

function Preview({
  preview,
  onAdd,
  onAnswer,
  answerError,
  onCancel,
  onRemap,
  busy,
}: {
  preview: ImportPreview;
  onAdd: (p: ImportPreview) => void;
  onAnswer: (answers: MatchAnswer[]) => void;
  answerError: string | null;
  onCancel: () => void;
  onRemap: (() => void) | undefined;
  busy: boolean;
}) {
  const [reviewing, setReviewing] = useState(false);
  const pending = preview.possibleMatches.filter((m) => m.answer === "pending");
  const exact = pending.filter((m) => m.exact && m.candidates.length === 1);
  const current = reviewing ? pending[0] : undefined;
  const nothingToDo =
    preview.newCount + preview.updatedCount === 0 &&
    preview.missing.length === 0;

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
        {preview.updatedCount > 0 && (
          <li>
            <strong>{preview.updatedCount}</strong> will update Pokémon you
            already have
          </li>
        )}
        {preview.possibleMatchCount > 0 && (
          <li>
            <strong>{preview.possibleMatchCount}</strong> may already be in your
            collection. They are held back until you say whether they're the
            same Pokémon, so nothing is counted twice.
          </li>
        )}
        {preview.missing.length > 0 && (
          <li>
            <strong>{preview.missing.length}</strong>{" "}
            {preview.missing.length === 1
              ? "Pokémon in CatchKeep isn't in this file."
              : "Pokémon in CatchKeep aren't in this file."}{" "}
            After this, you can mark{" "}
            {preview.missing.length === 1 ? "it" : "them"} as transferred.
          </li>
        )}
      </ul>
      {current ? (
        <DuplicateReview
          key={current.row}
          match={current}
          remaining={pending.length}
          busy={busy}
          onAnswer={(a) => onAnswer([a])}
          onStop={() => setReviewing(false)}
        />
      ) : (
        pending.length > 0 && (
          <div className="flex flex-col gap-2">
            {exact.length > 0 && (
              <>
                <p className="text-sm text-muted">
                  {exact.length} match exactly one Pokémon you already have:
                  same species, form, gender, IVs and catch date.
                </p>
                <button
                  type="button"
                  className={primaryButton}
                  disabled={busy}
                  onClick={() =>
                    onAnswer(
                      exact.map((m) => ({
                        row: m.row,
                        answer: "same",
                        sameAs: m.candidates[0]?.pokemon.id ?? "",
                      })),
                    )
                  }
                >
                  Update {exact.length} Pokémon you already have
                </button>
              </>
            )}
            <button
              type="button"
              className={secondaryButton}
              onClick={() => setReviewing(true)}
            >
              Review one by one
            </button>
          </div>
        )
      )}
      {answerError && (
        <p role="alert" className={alert}>
          {answerError}
        </p>
      )}
      {preview.invalidRows.length > 0 && (
        <details className="text-sm">
          <summary className="cursor-pointer">
            {preview.invalidRows.length === 1
              ? "1 note about the file"
              : `${preview.invalidRows.length} notes about the file`}
          </summary>
          <ul className="mt-2 flex flex-col gap-1 text-muted">
            {preview.invalidRows.map((r) => (
              <li key={`${r.row}-${r.message}`}>
                Line {r.row}: {r.message}
              </li>
            ))}
          </ul>
        </details>
      )}
      {preview.unknownFields.length > 0 && (
        <p className="rounded-xl bg-highlight px-3 py-2 text-sm text-ink">
          This export can't tell {preview.unknownFields.join(", ")}. CatchKeep
          treats them as unknown, never as "no".
        </p>
      )}
      <button
        type="button"
        className={primaryButton}
        disabled={busy || nothingToDo}
        onClick={() => onAdd(preview)}
      >
        {nothingToDo ? "Nothing to add yet" : commitLabel(preview)}
      </button>
      {onRemap && (
        <button type="button" className={secondaryButton} onClick={onRemap}>
          Change how columns are read
        </button>
      )}
      <button type="button" className={secondaryButton} onClick={onCancel}>
        Cancel
      </button>
    </section>
  );
}
