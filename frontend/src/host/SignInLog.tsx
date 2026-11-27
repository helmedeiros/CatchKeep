import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import type { components } from "../api/schema.gen";
import { unwrap } from "../api/errors";
import { useApi } from "../api/useApi";
import { useHost } from "../session/useSession";
import { secondaryButton } from "../ui/styles";

type Entry = components["schemas"]["SignInEntry"];

const PAGE = 20;

const when = (at: string) =>
  new Date(at).toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });

function what(e: Entry): string {
  const name = e.profileName;
  switch (e.kind) {
    case "signed_in":
      return `${name ?? "A profile"} signed in`;
    case "wrong_pin":
      return name
        ? `Wrong PIN for ${name}`
        : "Wrong PIN for a profile that doesn't exist";
    case "paused":
      return name
        ? `Sign-in paused for ${name} after too many wrong PINs`
        : "Sign-in paused after too many wrong PINs";
    case "profile_created":
      return name ? `Profile ${name} created` : "A profile was created";
    case "pin_reset":
      return `PIN reset for ${name ?? "a profile"}`;
  }
}

const from = (device: string) =>
  device === "127.0.0.1" || device === "::1"
    ? "this computer"
    : `the device at ${device}`;

/**
 * On the computer running CatchKeep: who signed in lately, wrong PINs and
 * pauses, from which device (spec §13 sign-in log).
 */
export function SignInLog() {
  const api = useApi();
  const host = useHost();
  const admin = host.data?.admin ?? false;
  const [shown, setShown] = useState(PAGE);
  const log = useQuery({
    queryKey: ["sign-ins"],
    enabled: admin,
    queryFn: async () => unwrap(await api.GET("/sign-ins")).entries,
  });

  if (!admin || !log.data) return null;
  const entries = log.data;
  return (
    <details className="mt-3 border-t border-line pt-3 text-sm">
      <summary className="cursor-pointer font-semibold">
        Recent sign-ins
      </summary>
      {entries.length === 0 ? (
        <p className="mt-2 text-muted">No sign-ins yet.</p>
      ) : (
        <>
          <ul aria-label="Recent sign-ins" className="mt-2 flex flex-col gap-2">
            {entries.slice(0, shown).map((e, i) => (
              <li key={`${e.at}-${i}`} className="flex flex-col">
                <span
                  className={
                    e.kind === "wrong_pin" || e.kind === "paused"
                      ? "text-warn"
                      : undefined
                  }
                >
                  {what(e)}
                </span>
                <span className="text-xs text-muted">
                  {when(e.at)}, from {from(e.device)}
                </span>
              </li>
            ))}
          </ul>
          {entries.length > shown && (
            <button
              type="button"
              className={`${secondaryButton} mt-2 self-start`}
              onClick={() => setShown(shown + PAGE)}
            >
              Show more
            </button>
          )}
        </>
      )}
      <p className="mt-2 text-xs text-muted">
        Kept for 90 days, only on this computer, and never in backups.
      </p>
    </details>
  );
}
