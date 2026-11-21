import { Dialog } from "@base-ui/react/dialog";
import { type ReactNode, useState } from "react";
import { QrCode } from "../host/QrCode";
import { useHost } from "../session/useSession";
import { Keepy } from "../ui/Keepy";
import { button } from "../ui/styles";

interface Step {
  title: string;
  text: string;
  extra?: ReactNode;
}

/**
 * A short, friendly introduction shown once per profile on this device
 * (ck-pwk.6). Skippable at every step and reachable again from More.
 */
export function Onboarding({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const host = useHost();
  const address = host.data?.admin ? host.data.addresses[0] : undefined;
  const [at, setAt] = useState(0);
  const steps: Step[] = [
    {
      title: "Keep the best, let the rest go",
      text: "CatchKeep keeps the best copies of every Pokémon by rules you set once, and shows what you can safely transfer.",
    },
    {
      title: "Bring in your collection",
      text: "Export your Pokémon from Poke Genie, or any spreadsheet saved as CSV, and choose the file. You always see a preview before anything is added.",
    },
    {
      title: "Ready for Community Day",
      text: "Tell CatchKeep how full you like your storage. Before an event it suggests exactly what to transfer, with searches to paste in the game.",
    },
    {
      title: "Use it on your phone",
      text: address
        ? "Your Pokémon stay on this computer. Point your phone's camera at the code to open CatchKeep there."
        : "Your Pokémon stay on the computer running CatchKeep. Open it on any device on the same Wi-Fi with the address shown on that computer.",
      ...(address && { extra: <QrCode text={address} size={140} /> }),
    },
  ];
  const step = steps[at] ?? steps[0];
  const last = at === steps.length - 1;
  const close = () => {
    setAt(0);
    onClose();
  };

  return (
    <Dialog.Root open={open} onOpenChange={(next) => !next && close()}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-ink/40" />
        <Dialog.Popup className="fixed inset-0 z-50 flex flex-col justify-between gap-6 bg-ground p-6 sm:inset-auto sm:top-1/2 sm:left-1/2 sm:w-[28rem] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[28px] sm:shadow-card sm:ring-1 sm:ring-line">
          <div className="flex justify-end">
            <Dialog.Close className={button({ intent: "secondary" })}>
              Skip
            </Dialog.Close>
          </div>
          <div className="flex flex-1 flex-col items-center justify-center gap-5 text-center">
            <div className="grid size-44 place-items-center rounded-full bg-highlight">
              {step?.extra ?? <Keepy size={132} />}
            </div>
            <Dialog.Title className="text-3xl font-extrabold tracking-tight">
              {step?.title}
            </Dialog.Title>
            <Dialog.Description className="max-w-sm text-lg text-muted">
              {step?.text}
            </Dialog.Description>
          </div>
          <div className="flex flex-col gap-4">
            <p className="sr-only" aria-live="polite">
              Step {at + 1} of {steps.length}
            </p>
            <div className="flex justify-center gap-1.5" aria-hidden="true">
              {steps.map((s, i) => (
                <span
                  key={s.title}
                  className={`h-2 rounded-full ${i === at ? "w-6 bg-accent" : "w-2 bg-line"}`}
                />
              ))}
            </div>
            <div className="flex gap-2">
              {at > 0 && (
                <button
                  type="button"
                  className={button({ intent: "outline" })}
                  onClick={() => setAt((i) => i - 1)}
                >
                  Back
                </button>
              )}
              <button
                type="button"
                className={button({ intent: "primary" })}
                onClick={() => (last ? close() : setAt((i) => i + 1))}
              >
                {last ? "Let's go" : "Next"}
              </button>
            </div>
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
