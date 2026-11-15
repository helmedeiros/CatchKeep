import { forwardRef, type HTMLAttributes } from "react";
import { Virtuoso } from "react-virtuoso";
import { card } from "../ui/styles";
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
        itemContent={(_, p) => <Row pokemon={p} />}
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

function Row({ pokemon: p }: { pokemon: Pokemon }) {
  const ivPercent = p.iv
    ? Math.round(((p.iv.attack + p.iv.defense + p.iv.stamina) / 45) * 100)
    : null;
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="min-w-0">
        <p className="truncate font-medium">
          {p.species}
          {p.form !== "Normal" && (
            <span className="text-slate-500"> · {p.form}</span>
          )}
        </p>
        <p className="text-xs text-slate-500">
          {p.iv
            ? `${p.iv.attack}/${p.iv.defense}/${p.iv.stamina} · ${ivPercent}%`
            : "IVs unknown"}
          {p.catchDate && ` · caught ${p.catchDate}`}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        {p.attributes.lucky === "yes" && <Badge>Lucky</Badge>}
        {p.attributes.shadow === "yes" && <Badge>Shadow</Badge>}
        {p.attributes.favorite === "yes" && <Badge>★</Badge>}
        <span className="w-20 text-right text-sm font-semibold tabular-nums">
          {p.cp === undefined ? "CP ?" : `CP ${p.cp}`}
        </span>
      </div>
    </div>
  );
}

function Badge({ children }: { children: string }) {
  return (
    <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-900 dark:bg-amber-900 dark:text-amber-100">
      {children}
    </span>
  );
}
