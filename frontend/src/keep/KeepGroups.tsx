import { useMemo, useState } from "react";
import { PokemonRow } from "../collection/PokemonRow";
import type { Pokemon } from "../collection/useCollection";
import { Sprite } from "../ui/Sprite";
import { secondaryButton } from "../ui/styles";
import type { KeepSummary } from "./useKeep";

type Group = KeepSummary["groups"][number];

const TOP_GROUPS = 8;
const FLAGS = [
  "shiny",
  "lucky",
  "shadow",
  "purified",
  "costume",
  "background",
  "dynamax",
  "gigantamax",
] as const;

function variantKey(g: Group): string {
  return [g.species, g.form, ...FLAGS.map((f) => g[f])].join("|");
}

function variantLabels(g: Group): string[] {
  return [
    ...(g.form !== "Normal" ? [g.form] : []),
    ...FLAGS.filter((f) => g[f]),
    ...(g.rare ? ["rare"] : []),
  ];
}

/**
 * The variants with possible transfers, largest first. Each opens to show
 * its copies, the ones kept first.
 */
export function KeepGroups({
  groups,
  pokemon,
}: {
  groups: Group[];
  pokemon: Pokemon[];
}) {
  const [showAll, setShowAll] = useState(false);
  const byId = useMemo(() => new Map(pokemon.map((p) => [p.id, p])), [pokemon]);
  const withSurplus = groups.filter((g) => g.surplusIds.length > 0);
  const shown = showAll ? withSurplus : withSurplus.slice(0, TOP_GROUPS);
  if (shown.length === 0) return null;
  return (
    <>
      <ul
        aria-label="Possible transfers by species"
        className="flex flex-col divide-y divide-slate-100 dark:divide-slate-800"
      >
        {shown.map((g) => (
          <GroupRow key={variantKey(g)} group={g} byId={byId} />
        ))}
      </ul>
      {withSurplus.length > TOP_GROUPS && (
        <button
          type="button"
          className={`${secondaryButton} self-start`}
          onClick={() => setShowAll((v) => !v)}
        >
          {showAll ? "Show fewer" : `Show all ${withSurplus.length}`}
        </button>
      )}
    </>
  );
}

function GroupRow({
  group: g,
  byId,
}: {
  group: Group;
  byId: Map<string, Pokemon>;
}) {
  const [open, setOpen] = useState(false);
  const first = byId.get(g.keptIds[0] ?? g.surplusIds[0] ?? "");
  const name = [g.species, ...variantLabels(g)].join(" · ");
  return (
    <li className="py-1.5">
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-3 rounded-xl p-1 text-left hover:bg-slate-50 dark:hover:bg-slate-800"
      >
        <Sprite
          size="sm"
          species={g.species}
          dexNumber={first?.dexNumber}
          shiny={g.shiny}
        />
        <span className="min-w-0 flex-1 truncate font-medium">{name}</span>
        <span className="shrink-0 text-sm tabular-nums text-slate-500">
          {g.count} · keep {g.keep} · {g.surplusIds.length} extra
        </span>
      </button>
      {open && (
        <ul
          aria-label={`${name} copies`}
          className="mt-1 flex flex-col divide-y divide-slate-100 pl-2 dark:divide-slate-800"
        >
          {[...g.keptIds, ...g.surplusIds].map((id, i) => {
            const p = byId.get(id);
            return (
              p && (
                <li key={id} className="py-2">
                  <PokemonRow pokemon={p}>
                    {i < g.keep ? (
                      <Tag tone="keep">Keep</Tag>
                    ) : (
                      <Tag tone="go">Possible transfer</Tag>
                    )}
                  </PokemonRow>
                </li>
              )
            );
          })}
        </ul>
      )}
    </li>
  );
}

function Tag({ tone, children }: { tone: "keep" | "go"; children: string }) {
  const colors =
    tone === "keep"
      ? "bg-teal-100 text-teal-900 dark:bg-teal-900 dark:text-teal-100"
      : "bg-rose-100 text-rose-900 dark:bg-rose-900 dark:text-rose-100";
  return (
    <span className={`rounded-full px-2 py-0.5 font-medium ${colors}`}>
      {children}
    </span>
  );
}
