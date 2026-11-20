import { toast } from "sonner";
import { useHost } from "../session/useSession";
import { secondaryButton } from "../ui/styles";

/**
 * On the computer running CatchKeep, the web address phones, tablets and
 * other computers on the same network open, so nobody has to look it up.
 */
export function OtherDevices() {
  const host = useHost();
  if (!host.data?.admin) return null;
  const { addresses } = host.data;

  const copy = async (address: string) => {
    try {
      await navigator.clipboard.writeText(address);
      toast.success("Address copied.");
    } catch {
      toast.error("Couldn't copy; type the address instead.");
    }
  };

  return (
    <section
      aria-label="Use CatchKeep on other devices"
      className="mt-3 flex flex-col gap-2 border-t border-slate-100 pt-3 text-sm dark:border-slate-800"
    >
      <h2 className="font-semibold">Use CatchKeep on other devices</h2>
      {addresses.length > 0 ? (
        <>
          <p className="text-slate-600 dark:text-slate-400">
            On a phone, tablet or another computer connected to the same network
            (the same Wi-Fi), open this address in the web browser:
          </p>
          <ul className="flex flex-col gap-2">
            {addresses.map((a) => (
              <li key={a} className="flex items-center gap-2">
                <code className="min-w-0 flex-1 truncate rounded-lg bg-slate-100 px-3 py-2 font-mono text-base dark:bg-slate-800">
                  {a}
                </code>
                <button
                  type="button"
                  aria-label={`Copy ${a}`}
                  className={secondaryButton}
                  onClick={() => void copy(a)}
                >
                  Copy
                </button>
              </li>
            ))}
          </ul>
          <p className="text-xs text-slate-500">
            Keep this computer on and awake while others use it.
          </p>
        </>
      ) : (
        <p className="text-slate-600 dark:text-slate-400">
          Right now only this computer can open CatchKeep. To use it on phones
          and other devices too, start CatchKeep so it listens on your network
          (see "Run it on your network" in the guide).
        </p>
      )}
    </section>
  );
}
