import { type ReactNode, useState } from "react";
import { FeatureMarks } from "./FeatureMarks";
import { PokemonRow } from "./PokemonRow";
import { ProtectButton } from "./ProtectButton";
import type { Pokemon } from "./useCollection";

/** A Pokémon row with protection and its special features one tap away. */
export function PokemonItem({
  pokemon: p,
  children,
  actions,
}: {
  pokemon: Pokemon;
  children?: ReactNode;
  /** More actions, before protection. */
  actions?: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const label = `${p.species} CP ${p.cp ?? "?"}`;
  return (
    <div className="flex flex-col gap-1.5">
      <PokemonRow pokemon={p}>{children}</PokemonRow>
      <div className="flex flex-wrap items-center gap-2 pl-17">
        {actions}
        <ProtectButton pokemon={p} />
        <button
          type="button"
          aria-expanded={open}
          aria-label={`Details for ${label}`}
          onClick={() => setOpen((v) => !v)}
          className="rounded-full px-2 py-1 text-xs font-semibold text-accent ring-1 ring-line hover:bg-highlight"
        >
          {open ? "Hide details" : "Details"}
        </button>
      </div>
      {open && <FeatureMarks pokemon={p} label={label} />}
    </div>
  );
}
