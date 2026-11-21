import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { unwrap } from "../api/errors";
import { useApi } from "../api/useApi";
import { useHost } from "../session/useSession";

const key = ["autostart"] as const;

/** On the computer running CatchKeep: start it automatically at login. */
export function StartAtLogin() {
  const api = useApi();
  const queryClient = useQueryClient();
  const host = useHost();
  const admin = host.data?.admin ?? false;
  const state = useQuery({
    queryKey: key,
    enabled: admin,
    queryFn: async () => unwrap(await api.GET("/autostart")),
  });
  const change = useMutation({
    mutationFn: async (enabled: boolean) =>
      unwrap(await api.PUT("/autostart", { body: { enabled } })),
    onSuccess: (saved) => {
      queryClient.setQueryData(key, saved);
      toast.success(
        saved.enabled
          ? "CatchKeep will start the next time you log in."
          : "CatchKeep won't start on its own any more.",
      );
    },
    onError: () => toast.error("Couldn't change that setting."),
  });

  if (!admin || !state.data?.supported) return null;
  return (
    <div className="mt-3 flex flex-col gap-1 border-t border-line pt-3 text-sm">
      <label className="flex items-center gap-2 font-medium">
        <input
          type="checkbox"
          className="size-4 accent-accent"
          checked={state.data.enabled}
          disabled={change.isPending}
          onChange={(e) => change.mutate(e.target.checked)}
        />
        Start CatchKeep when I log in to this computer
      </label>
      <p className="text-xs text-muted">
        Takes effect the next time you log in, with the same settings as now.
      </p>
    </div>
  );
}
