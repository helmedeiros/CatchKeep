import { useConfirmTransfers } from "../plan/useTransfers";
import type { Pokemon } from "./useCollection";

/**
 * Marks one Pokémon as transferred in the game, outside any plan; the
 * toast that follows can undo it.
 */
export function TransferButton({ pokemon: p }: { pokemon: Pokemon }) {
  const confirm = useConfirmTransfers();
  return (
    <button
      type="button"
      aria-label={`Mark ${p.species} CP ${p.cp ?? "?"} as transferred`}
      disabled={confirm.isPending}
      onClick={() => confirm.mutate({ pokemonIds: [p.id] })}
      className="shrink-0 rounded-full px-2 py-1 text-xs font-semibold text-muted ring-1 ring-line hover:bg-soft disabled:opacity-50"
    >
      Mark transferred
    </button>
  );
}
