import { toast } from "sonner";
import { type Pokemon, useProtect } from "./useCollection";

/** Protected Pokémon are never suggested for transfer. */
export function ProtectButton({ pokemon: p }: { pokemon: Pokemon }) {
  const protect = useProtect();
  const name = `${p.species}${p.cp === undefined ? "" : ` CP ${p.cp}`}`;
  return (
    <button
      type="button"
      aria-pressed={p.protected}
      aria-label={`${p.protected ? "Unprotect" : "Protect"} ${name}`}
      disabled={protect.isPending}
      onClick={() =>
        protect.mutate(
          { id: p.id, protect: !p.protected },
          {
            onSuccess: () =>
              toast.success(
                p.protected
                  ? `${name} can be suggested for transfer again.`
                  : `${name} is protected and will always be kept.`,
              ),
            onError: () => toast.error(`Couldn't change ${name}.`),
          },
        )
      }
      className={`shrink-0 rounded-full px-2 py-1 text-xs font-semibold ring-1 disabled:opacity-50 ${
        p.protected
          ? "bg-accent text-accent-ink ring-accent hover:bg-accent/90"
          : "text-muted ring-line hover:bg-soft"
      }`}
    >
      {p.protected ? "Protected" : "Protect"}
    </button>
  );
}
