import { forwardRef, type HTMLAttributes, useMemo } from "react";
import { useKeepSummary, useSetup } from "../keep/useKeep";
import { useTeams } from "../teams/useTeams";
import { Virtuoso } from "react-virtuoso";
import { card } from "../ui/styles";
import { FilterBar } from "./FilterBar";
import { applyFilters } from "./filters";
import { PokemonItem } from "./PokemonItem";
import { TransferButton } from "./TransferButton";
import { useFilters } from "./useFilters";
import type { Pokemon } from "./useCollection";

const INITIAL_ROWS = 30;

export function CollectionView({ pokemon }: { pokemon: Pokemon[] }) {
  const { filters } = useFilters("collection");
  const setup = useSetup();
  const summary = useKeepSummary(Boolean(setup.data));
  const teams = useTeams();
  const shown = useMemo(
    () =>
      applyFilters(pokemon, filters, {
        teamIds: new Set(teams.data?.flatMap((t) => t.memberIds) ?? []),
        transferIds: new Set(
          summary.data?.groups.flatMap((g) => g.surplusIds) ?? [],
        ),
      }),
    [pokemon, filters, teams.data, summary.data],
  );
  if (pokemon.length === 0) {
    return <p className="px-1 text-sm text-muted">No Pokémon yet.</p>;
  }
  return (
    <section className={`${card} flex flex-col gap-2 p-0`}>
      <div className="px-4 pt-4">
        <FilterBar
          list="collection"
          chips={[
            ...(summary.data ? (["transfer"] as const) : []),
            "shiny",
            "lucky",
            "shadow",
            "purified",
            "costume",
            "protected",
            "team",
            "unconfirmed",
          ]}
          shown={shown.length}
          total={pokemon.length}
        />
      </div>
      {shown.length === 0 ? (
        <p className="px-4 pb-4 text-sm text-muted">No Pokémon match.</p>
      ) : (
        <Virtuoso
          useWindowScroll
          data={shown}
          initialItemCount={Math.min(INITIAL_ROWS, shown.length)}
          components={{ List, Item }}
          computeItemKey={(_, p) => p.id}
          itemContent={(_, p) => (
            <PokemonItem pokemon={p} actions={<TransferButton pokemon={p} />} />
          )}
        />
      )}
    </section>
  );
}

const List = forwardRef<HTMLUListElement, HTMLAttributes<HTMLUListElement>>(
  (props, ref) => (
    <ul
      {...props}
      ref={ref}
      aria-label="Collection"
      className="divide-y divide-line"
    />
  ),
);
List.displayName = "CollectionList";

function Item(props: HTMLAttributes<HTMLLIElement>) {
  return <li {...props} className="px-4 py-2.5" />;
}
