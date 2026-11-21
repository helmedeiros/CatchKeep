import { useQuery } from "@tanstack/react-query";
import { unwrap } from "../api/errors";
import type { components } from "../api/schema.gen";
import { useApi } from "../api/useApi";
import { rulesKey } from "../keep/useKeep";

export type Plan = components["schemas"]["Plan"];
export type TargetKind = components["schemas"]["TargetKind"];

/** The suggested cleanup for a target; refreshed with the rules. */
export function usePlan(
  kind: TargetKind,
  value: number | undefined,
  enabled: boolean,
) {
  const api = useApi();
  const ready =
    enabled && (kind === "limit" || (value !== undefined && value >= 1));
  return useQuery({
    queryKey: [...rulesKey, "plan", kind, value],
    enabled: ready,
    placeholderData: (previous) => previous,
    queryFn: async () =>
      unwrap(
        await api.GET("/plan", {
          params: {
            query: kind === "limit" ? { kind } : { kind, value: value ?? 1 },
          },
        }),
      ),
  });
}
