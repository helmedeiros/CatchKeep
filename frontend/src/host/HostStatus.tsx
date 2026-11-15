import { useQuery } from "@tanstack/react-query";
import { useApi } from "../api/useApi";

const HEALTH_REFRESH_MS = 30_000;

export function HostStatus() {
  const api = useApi();
  const health = useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      const { data, error } = await api.GET("/health");
      if (error || !data) {
        throw new Error("host reported an error");
      }
      return data;
    },
    refetchInterval: HEALTH_REFRESH_MS,
    refetchOnWindowFocus: true,
  });

  if (health.isPending) {
    return <p className="text-sm text-slate-500">Checking host…</p>;
  }

  if (health.isError) {
    return (
      <p
        role="alert"
        className="rounded-lg bg-amber-100 px-4 py-3 text-sm text-amber-900"
      >
        CatchKeep isn't reachable — is the host on and awake?
      </p>
    );
  }

  return (
    <div className="flex items-center gap-2 text-sm">
      <span aria-hidden="true" className="size-2 rounded-full bg-emerald-500" />
      <span className="font-medium text-emerald-700 dark:text-emerald-400">
        Connected
      </span>
      <span className="text-slate-500">Version {health.data.version}</span>
    </div>
  );
}
