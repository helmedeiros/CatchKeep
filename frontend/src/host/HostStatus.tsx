import { useQuery } from "@tanstack/react-query";
import { useApi } from "../api/useApi";

const HEALTH_REFRESH_MS = 30_000;

/** The page was opened by a name CatchKeep does not answer to. */
class WrongAddress extends Error {}

export function HostStatus() {
  const api = useApi();
  const health = useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      const { data, error } = await api.GET("/health");
      if (error?.error.code === "HOST_NOT_ALLOWED") {
        throw new WrongAddress();
      }
      if (error || !data) {
        throw new Error("host reported an error");
      }
      return data;
    },
    refetchInterval: HEALTH_REFRESH_MS,
    refetchOnWindowFocus: true,
  });

  if (health.isPending) {
    return <p className="text-sm text-muted">Checking host…</p>;
  }

  if (health.error instanceof WrongAddress) {
    return (
      <p
        role="alert"
        className="rounded-xl bg-warn/15 px-4 py-3 text-sm text-warn"
      >
        Open CatchKeep using the address shown on the computer running it, under
        “Use CatchKeep on other devices”.
      </p>
    );
  }

  if (health.isError) {
    return (
      <p
        role="alert"
        className="rounded-xl bg-warn/15 px-4 py-3 text-sm text-warn"
      >
        CatchKeep isn't reachable — is the computer running it on and awake?
      </p>
    );
  }

  return (
    <div className="flex items-center gap-2 text-sm">
      <span aria-hidden="true" className="size-2 rounded-full bg-good" />
      <span className="font-medium text-good">Connected</span>
      <span className="text-muted">Version {health.data.version}</span>
    </div>
  );
}
