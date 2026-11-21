import { ivPercent } from "./pokemon";
import type { Pokemon } from "./useCollection";

export type Chip =
  | "shiny"
  | "lucky"
  | "shadow"
  | "purified"
  | "costume"
  | "protected"
  | "team"
  | "transfer"
  | "unconfirmed";

/** "plan" keeps the order the list came in, such as weakest first. */
export type Sort = "plan" | "cp" | "iv" | "recent" | "species";

export interface Filters {
  query: string;
  chips: Chip[];
  minIv: number;
  sort: Sort;
}

export const NO_FILTERS: Filters = {
  query: "",
  chips: [],
  minIv: 0,
  sort: "cp",
};

/** What some chips need beyond the Pokémon itself. */
export interface FilterContext {
  teamIds: Set<string>;
  transferIds: Set<string>;
}

const VARIANT = [
  "shiny",
  "costume",
  "background",
  "dynamax",
  "gigantamax",
] as const;

function hasChip(p: Pokemon, chip: Chip, ctx: FilterContext): boolean {
  switch (chip) {
    case "shiny":
    case "lucky":
    case "shadow":
    case "purified":
    case "costume":
      return p.attributes[chip] === "yes";
    case "protected":
      return p.protected;
    case "team":
      return ctx.teamIds.has(p.id);
    case "transfer":
      return ctx.transferIds.has(p.id);
    case "unconfirmed":
      return VARIANT.some((f) => p.attributes[f] === "unknown");
  }
}

function matches(p: Pokemon, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (q === "") return true;
  if (/^\d+$/.test(q)) return p.dexNumber === Number(q);
  return (
    p.species.toLowerCase().includes(q) || p.form.toLowerCase().includes(q)
  );
}

const byCp = (a: Pokemon, b: Pokemon) => (b.cp ?? -1) - (a.cp ?? -1);
const SORTS: Record<Sort, (a: Pokemon, b: Pokemon) => number> = {
  plan: () => 0,
  cp: byCp,
  iv: (a, b) => (ivPercent(b) ?? -1) - (ivPercent(a) ?? -1) || byCp(a, b),
  recent: (a, b) =>
    (b.catchDate ?? "").localeCompare(a.catchDate ?? "") || byCp(a, b),
  species: (a, b) => a.species.localeCompare(b.species) || byCp(a, b),
};

/** The Pokémon that match the search and every chosen chip, sorted. */
export function applyFilters(
  pokemon: Pokemon[],
  f: Filters,
  ctx: FilterContext,
): Pokemon[] {
  return pokemon
    .filter(
      (p) =>
        matches(p, f.query) &&
        f.chips.every((c) => hasChip(p, c, ctx)) &&
        (f.minIv === 0 || (ivPercent(p) ?? -1) >= f.minIv),
    )
    .sort(SORTS[f.sort]);
}

export function isFiltering(f: Filters): boolean {
  return f.query.trim() !== "" || f.chips.length > 0 || f.minIv > 0;
}
