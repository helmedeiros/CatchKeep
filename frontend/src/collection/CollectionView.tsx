import { forwardRef, type HTMLAttributes } from "react";
import { Virtuoso } from "react-virtuoso";
import { card } from "../ui/styles";
import { PokemonItem } from "./PokemonItem";
import type { Pokemon } from "./useCollection";

const INITIAL_ROWS = 30;

export function CollectionView({ pokemon }: { pokemon: Pokemon[] }) {
  if (pokemon.length === 0) {
    return <p className="px-1 text-sm text-slate-500">No Pokémon yet.</p>;
  }
  return (
    <section className={`${card} flex flex-col gap-2 p-0`}>
      <h3 className="px-4 pt-4 font-semibold">{pokemon.length} Pokémon</h3>
      <Virtuoso
        useWindowScroll
        data={pokemon}
        initialItemCount={Math.min(INITIAL_ROWS, pokemon.length)}
        components={{ List, Item }}
        computeItemKey={(_, p) => p.id}
        itemContent={(_, p) => <PokemonItem pokemon={p} />}
      />
    </section>
  );
}

const List = forwardRef<HTMLUListElement, HTMLAttributes<HTMLUListElement>>(
  (props, ref) => (
    <ul
      {...props}
      ref={ref}
      aria-label="Collection"
      className="divide-y divide-slate-100 dark:divide-slate-800"
    />
  ),
);
List.displayName = "CollectionList";

function Item(props: HTMLAttributes<HTMLLIElement>) {
  return <li {...props} className="px-4 py-2.5" />;
}
