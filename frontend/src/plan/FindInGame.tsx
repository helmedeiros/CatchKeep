import { toast } from "sonner";
import type { Pokemon } from "../collection/useCollection";
import { secondaryButton } from "../ui/styles";
import type { Plan } from "./usePlan";

type Search = Plan["searches"][number];

const label = (p: Pokemon | undefined) =>
  p ? `CP ${p.cp ?? "?"}${p.hp === undefined ? "" : `, HP ${p.hp}`}` : "?";

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Search copied. Paste it in the game.");
  } catch {
    toast.error("Couldn't copy; type the search instead.");
  }
}

/**
 * Searches to paste into the game's Pokémon list, one per species, with a
 * warning when a search would also find Pokémon to keep (V1-CLN-04).
 */
export function FindInGame({
  searches,
  byId,
}: {
  searches: Search[];
  byId: Map<string, Pokemon>;
}) {
  if (searches.length === 0) return null;
  return (
    <section
      aria-label="Find them in the game"
      className="flex flex-col gap-2 rounded-xl bg-soft p-3 text-sm"
    >
      <h5 className="font-semibold">Find them in the game</h5>
      <p className="text-muted">
        Copy a search, paste it into the search bar above your Pokémon list in
        the game, then select and transfer what it finds.
      </p>
      <ul className="flex flex-col gap-3">
        {searches.map((s) => (
          <li key={s.species}>
            <SpeciesSearch search={s} byId={byId} />
          </li>
        ))}
      </ul>
    </section>
  );
}

function SpeciesSearch({
  search: s,
  byId,
}: {
  search: Search;
  byId: Map<string, Pokemon>;
}) {
  const kept = s.collisions.map((id) => byId.get(id));
  return (
    <div role="group" aria-label={s.species} className="flex flex-col gap-1.5">
      <p className="font-medium">
        {s.species} · {s.pokemonIds.length} to transfer
      </p>
      {s.query === "" ? (
        <p className="text-muted">
          Find these by hand: the export has no Pokédex number for them.
        </p>
      ) : (
        <div className="flex items-center gap-2">
          <code className="min-w-0 flex-1 truncate rounded-full bg-surface px-3 py-2 font-mono">
            {s.query}
          </code>
          <button
            type="button"
            aria-label={`Copy the search for ${s.species}`}
            className={secondaryButton}
            onClick={() => void copy(s.query)}
          >
            Copy
          </button>
        </div>
      )}
      {kept.length > 0 && (
        <p className="text-warn">
          This search also finds {kept.length} {s.species} you keep (
          {kept.map(label).join("; ")}). Check CP and HP before transferring
          {s.items.length > 0 ? ", or use the narrower searches below." : "."}
        </p>
      )}
      {s.items.map((it) => {
        const p = byId.get(it.pokemonId);
        return (
          <div key={it.pokemonId} className="flex flex-col gap-1 pl-3">
            <p>
              {s.species} {label(p)}
              {it.collisions.length > 0 &&
                ": looks the same as one you keep, so check its IVs in the game before transferring"}
            </p>
            {it.query !== "" && (
              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 truncate rounded-full bg-surface px-3 py-1.5 font-mono">
                  {it.query}
                </code>
                <button
                  type="button"
                  aria-label={`Copy the search for ${s.species} ${label(p)}`}
                  className={secondaryButton}
                  onClick={() => void copy(it.query)}
                >
                  Copy
                </button>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
