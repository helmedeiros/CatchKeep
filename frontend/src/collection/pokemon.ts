import type { Pokemon } from "./useCollection";

export function ivPercent(p: Pokemon): number | null {
  return p.iv
    ? Math.round(((p.iv.attack + p.iv.defense + p.iv.stamina) / 45) * 100)
    : null;
}

const VARIANT_FEATURES = [
  "shiny",
  "costume",
  "background",
  "dynamax",
  "gigantamax",
] as const;

/** Pokémon with a variant feature the export could not tell. */
export function countUnconfirmed(pokemon: Pokemon[]): number {
  return pokemon.filter((p) =>
    VARIANT_FEATURES.some((f) => p.attributes[f] === "unknown"),
  ).length;
}
