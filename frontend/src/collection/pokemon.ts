import type { Pokemon } from "./useCollection";

export function ivPercent(p: Pokemon): number | null {
  return p.iv
    ? Math.round(((p.iv.attack + p.iv.defense + p.iv.stamina) / 45) * 100)
    : null;
}
