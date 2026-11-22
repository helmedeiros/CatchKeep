import { useConfirmTransfers } from "../plan/useTransfers";
import { alert, card, primaryButton, secondaryButton } from "../ui/styles";
import { useCollection } from "./useCollection";

const LISTED = 30;

/**
 * After adding a new export: the Pokémon in CatchKeep the file doesn't
 * have. After a full export they were probably transferred; after one with
 * only some Pokémon, the trainer keeps them.
 */
export function MissingFromImport({
  ids,
  onDone,
}: {
  ids: string[];
  onDone: () => void;
}) {
  const collection = useCollection();
  const confirm = useConfirmTransfers();
  const all = collection.data?.pokemon ?? [];
  const missing = all.filter((p) => ids.includes(p.id));
  const most = all.length > 0 && missing.length > all.length / 2;
  const one = missing.length === 1;
  return (
    <section
      aria-label="Not in this file"
      className={`${card} flex flex-col gap-3`}
    >
      <h3 className="font-semibold">
        {one
          ? "1 Pokémon in CatchKeep isn't in this file"
          : `${missing.length} Pokémon in CatchKeep aren't in this file`}
      </h3>
      <p className="text-sm text-muted">
        If you exported everything from Poke Genie,{" "}
        {one ? "it was" : "they were"} probably transferred since your last
        import. If the file had only some of your Pokémon, keep{" "}
        {one ? "it" : "them"}.
      </p>
      {most && (
        <p className={alert}>
          That's most of your collection, so this file may hold only part of it.
        </p>
      )}
      <ul className="flex flex-wrap gap-1.5 text-xs">
        {missing.slice(0, LISTED).map((p) => (
          <li
            key={p.id}
            className="rounded-full bg-soft px-2 py-1 font-medium text-muted"
          >
            {p.species} CP {p.cp ?? "?"}
          </li>
        ))}
        {missing.length > LISTED && (
          <li className="px-2 py-1 text-muted">
            and {missing.length - LISTED} more
          </li>
        )}
      </ul>
      <button
        type="button"
        className={primaryButton}
        disabled={confirm.isPending || missing.length === 0}
        onClick={() =>
          confirm.mutate(
            { pokemonIds: missing.map((p) => p.id) },
            { onSuccess: onDone },
          )
        }
      >
        Mark {missing.length} as transferred
      </button>
      <button type="button" className={secondaryButton} onClick={onDone}>
        Keep {one ? "it" : "them"}
      </button>
    </section>
  );
}
