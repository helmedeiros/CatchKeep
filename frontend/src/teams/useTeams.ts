import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { unwrap } from "../api/errors";
import type { components } from "../api/schema.gen";
import { useApi } from "../api/useApi";
import { rulesKey } from "../keep/useKeep";

export type Team = components["schemas"]["Team"];
export type TeamInput = components["schemas"]["TeamInput"];

const teamsKey = ["teams"] as const;

export function useTeams() {
  const api = useApi();
  return useQuery({
    queryKey: teamsKey,
    queryFn: async () => unwrap(await api.GET("/teams")).teams,
  });
}

/**
 * Team changes change protection, so the keep rules follow. The team list
 * refreshes before the caller continues; the keep rules refresh in the
 * background so a slow summary never holds up the editor.
 */
function useTeamMutation<T, R>(fn: (input: T) => Promise<R>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: rulesKey });
      return queryClient.invalidateQueries({ queryKey: teamsKey });
    },
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
