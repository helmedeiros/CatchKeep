import { IvRing } from "../ui/IvRing";
import { Sprite } from "../ui/Sprite";
import { ivPercent } from "./pokemon";
import type { Pokemon } from "./useCollection";

/** One Pokémon the way trainers see it in their scanner: picture first. */
export function PokemonRow({
  pokemon: p,
  children,
}: {
  pokemon: Pokemon;
  children?: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-3">
      <Sprite
        species={p.species}
        dexNumber={p.dexNumber}
        shiny={p.attributes.shiny === "yes"}
      />
      <div className="min-w-0 flex-1">
        <p className="truncate">
          <span className="font-semibold">{p.species}</span>
          {p.form !== "Normal" && (
            <span className="text-slate-500"> · {p.form}</span>
          )}
          <span className="text-sm text-slate-500">
            {" "}
            · {p.cp === undefined ? "CP ?" : `CP ${p.cp}`}
            {p.hp !== undefined && ` / HP ${p.hp}`}
          </span>
        </p>
        <p className="text-sm text-slate-600 tabular-nums dark:text-slate-400">
          {p.iv
            ? `IV ${p.iv.attack}-${p.iv.defense}-${p.iv.stamina}`
            : "IVs unknown"}
          {p.level !== undefined && ` / LVL ${p.level.toFixed(1)}`}
        </p>
        <p className="flex flex-wrap items-center gap-1.5 text-xs text-slate-500">
          {p.attributes.shiny === "yes" && <Badge>Shiny</Badge>}
          {p.attributes.lucky === "yes" && <Badge>Lucky</Badge>}
          {p.attributes.shadow === "yes" && <Badge>Shadow</Badge>}
          {p.attributes.purified === "yes" && <Badge>Purified</Badge>}
          {p.attributes.favorite === "yes" && <Badge>★</Badge>}
          {p.catchDate && <span>Caught {p.catchDate}</span>}
          {children}
        </p>
      </div>
      <IvRing percent={ivPercent(p)} />
    </div>
  );
}

function Badge({ children }: { children: string }) {
  return (
    <span className="rounded-full bg-amber-100 px-2 py-0.5 font-medium text-amber-900 dark:bg-amber-900 dark:text-amber-100">
      {children}
    </span>
  );
}
