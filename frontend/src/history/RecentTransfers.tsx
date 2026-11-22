import type { components } from "../api/schema.gen";
import { useHistory, useUndoTransfer } from "../plan/useTransfers";
import { card } from "../ui/styles";

type TransferEvent = components["schemas"]["TransferEvent"];

const SHOWN = 10;
const LISTED = 3;

const when = (at: string) =>
  new Date(at).toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });

function names(e: TransferEvent): string {
  const listed = e.pokemon
    .slice(0, LISTED)
    .map((p) => `${p.species}${p.cp === undefined ? "" : ` CP ${p.cp}`}`)
    .join(", ");
  const more = e.pokemon.length - LISTED;
  return more > 0 ? `${listed} and ${more} more` : listed;
}

/**
 * What was marked as transferred lately, each with a way to put it back
 * in CatchKeep if it was marked by mistake.
 */
export function RecentTransfers() {
  const history = useHistory();
  const undo = useUndoTransfer();
  const transfers = (history.data ?? [])
    .filter((e) => e.kind !== "undo")
    .slice(0, SHOWN);
  return (
    <section
      aria-label="Recent transfers"
      className={`${card} flex flex-col gap-2`}
    >
      <h3 className="font-semibold">Recent transfers</h3>
      {history.isSuccess && transfers.length === 0 && (
        <p className="text-sm text-muted">Nothing marked as transferred yet.</p>
      )}
      {transfers.length > 0 && (
        <ul className="flex flex-col divide-y divide-line">
          {transfers.map((e) => (
            <li key={e.id} className="flex items-center gap-3 py-2 text-sm">
              <div className="min-w-0 flex-1">
                <p className="font-medium">{names(e)}</p>
                <p className="text-xs text-muted">{when(e.at)}</p>
              </div>
              {e.undoneBy ? (
                <span className="shrink-0 rounded-full bg-soft px-2 py-1 text-xs font-semibold text-muted">
                  Put back
                </span>
              ) : (
                <button
                  type="button"
                  aria-label={`Put back ${names(e)}`}
                  disabled={undo.isPending}
                  onClick={() => undo.mutate(e.id)}
                  className="shrink-0 rounded-full px-2 py-1 text-xs font-semibold text-accent ring-1 ring-line hover:bg-highlight disabled:opacity-50"
                >
                  Undo
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
