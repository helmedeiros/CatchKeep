import type { ReactNode } from "react";
import { navigate } from "../app/router";
import { countUnconfirmed } from "../collection/pokemon";
import type { Pokemon } from "../collection/useCollection";
import { useSetup } from "../keep/useKeep";
import { StorageBar } from "../plan/StorageBar";
import { usePlan } from "../plan/usePlan";
import { Keepy } from "../ui/Keepy";
import { button, card } from "../ui/styles";

/**
 * The one most useful thing to do next, from the first import to making
 * room (ck-pwk.4), and how full storage is once it is set up.
 */
export function NextStep({ pokemon }: { pokemon: Pokemon[] }) {
  const setup = useSetup();
  const plan = usePlan("limit", undefined, Boolean(setup.data));
  const unconfirmed = countUnconfirmed(pokemon);

  let step: { title: string; text: string; action?: ReactNode };
  if (pokemon.length === 0) {
    step = {
      title: "Bring in your collection",
      text: "Export your Pokémon from Poke Genie, or any CSV, and choose the file below.",
    };
  } else if (unconfirmed > 0) {
    step = {
      title: "Confirm what the export can't tell",
      text: `${unconfirmed} Pokémon wait for you to confirm whether they are shiny, costumed and so on. Check them in the game, then confirm below.`,
    };
  } else if (setup.isSuccess && !setup.data) {
    step = {
      title: "Set up your storage",
      text: "Tell CatchKeep how many slots you have and what you keep Pokémon for.",
      action: <Go to="/rules">Set up storage</Go>,
    };
  } else if (plan.data && plan.data.wanted > 0) {
    step = {
      title: "Make room",
      text: `You're over your limit: transfer ${plan.data.wanted} to get back under it.`,
      action: <Go to="/make-room">Make room</Go>,
    };
  } else {
    step = {
      title: "All set",
      text: plan.data
        ? `${Math.max(0, plan.data.use.limit - plan.data.use.occupied)} slots left before your limit. Getting ready for an event? Make room ahead of it.`
        : "Your collection is in order.",
      action: <Go to="/make-room">Make room</Go>,
    };
  }

  return (
    <>
      <section
        aria-label="Next step"
        className="relative flex items-end gap-2 overflow-hidden rounded-[20px] bg-highlight p-5 shadow-card ring-1 ring-line"
      >
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <h3 className="text-2xl font-extrabold tracking-tight">
            {step.title}
          </h3>
          <p className="text-muted">{step.text}</p>
          {step.action && <div className="mt-1">{step.action}</div>}
        </div>
        <Keepy />
      </section>
      {plan.data && (
        <section aria-label="Storage" className={card}>
          <StorageBar use={plan.data.use} />
        </section>
      )}
    </>
  );
}

function Go({ to, children }: { to: string; children: string }) {
  return (
    <a
      href={to}
      onClick={(e) => {
        e.preventDefault();
        navigate(to);
      }}
      className={button({ intent: "primary", fit: true })}
    >
      {children} <span aria-hidden="true">→</span>
    </a>
  );
}
