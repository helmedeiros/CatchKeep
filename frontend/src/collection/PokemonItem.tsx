import { type ReactNode, useState } from "react";
import { FeatureMarks } from "./FeatureMarks";
import { PokemonRow } from "./PokemonRow";
import { ProtectButton } from "./ProtectButton";
import type { Pokemon } from "./useCollection";

/** A Pokémon row with protection and its special features one tap away. */
export function PokemonItem({
  pokemon: p,
  children,
}: {
  pokemon: Pokemon;
  children?: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const label = `${p.species} CP ${p.cp ?? "?"}`;
  return (
    <div className="flex flex-col gap-2">
      <PokemonRow
        pokemon={p}
        action={
          <div className="flex flex-col items-end gap-1">
            <ProtectButton pokemon={p} />
            <button
              type="button"
              aria-expanded={open}
              aria-label={`Details for ${label}`}
              onClick={() => setOpen((v) => !v)}
              className="rounded-lg px-2 py-1 text-xs font-medium text-teal-800 hover:bg-teal-50 dark:text-teal-300 dark:hover:bg-slate-800"
            >
              Details
            </button>
          </div>
        }
      >
        {children}
      </PokemonRow>
      {open && <FeatureMarks pokemon={p} label={label} />}
    </div>
  );
}
