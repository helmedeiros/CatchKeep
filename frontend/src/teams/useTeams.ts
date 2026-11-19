import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { unwrap } from "../api/errors";
import type { components } from "../api/schema.gen";
import { useApi } from "../api/useApi";

export type Team = components["schemas"]["Team"];
export type TeamInput = components["schemas"]["TeamInput"];

const teamsKey = ["teams"] as const;
const summaryKey = ["collection", "keep-summary"] as const;

export function useTeams() {
  const api = useApi();
  return useQuery({
    queryKey: teamsKey,
    queryFn: async () => unwrap(await api.GET("/teams")).teams,
  });
}

/** Team changes change protection, so the keep summary follows. */
function useTeamMutation<T, R>(fn: (input: T) => Promise<R>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: teamsKey }),
        queryClient.invalidateQueries({ queryKey: summaryKey }),
      ]),
  });
}

export function useCreateTeam() {
  const api = useApi();
  return useTeamMutation(async (body: TeamInput) =>
    unwrap(await api.POST("/teams", { body })),
  );
}

export function useUpdateTeam() {
  const api = useApi();
  return useTeamMutation(async ({ id, ...body }: TeamInput & { id: string }) =>
    unwrap(
      await api.PUT("/teams/{teamId}", {
        params: { path: { teamId: id } },
        body,
      }),
    ),
  );
}

export function useDeleteTeam() {
  const api = useApi();
  return useTeamMutation(async (id: string) =>
    unwrap(
      await api.DELETE("/teams/{teamId}", {
        params: { path: { teamId: id } },
      }),
    ),
  );
}
