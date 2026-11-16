import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, unwrap } from "../api/errors";
import type { components } from "../api/schema.gen";
import { useApi } from "../api/useApi";

export type Setup = components["schemas"]["Setup"];
export type Goal = Setup["goals"][number];
export type KeepSummary = components["schemas"]["KeepSummary"];

const setupKey = ["setup"] as const;
// Under "collection" so anything that changes the collection refreshes it.
const summaryKey = ["collection", "keep-summary"] as const;

/** The trainer's setup, or null until one is saved. */
export function useSetup() {
  const api = useApi();
  return useQuery({
    queryKey: setupKey,
    queryFn: async () => {
      try {
        return unwrap(await api.GET("/setup"));
      } catch (e) {
        if (e instanceof ApiError && e.code === "NOT_FOUND") return null;
        throw e;
      }
    },
  });
}

export function useSaveSetup() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: Setup) =>
      unwrap(await api.PUT("/setup", { body })),
    onSuccess: (saved) => {
      queryClient.setQueryData(setupKey, saved);
      return queryClient.invalidateQueries({ queryKey: summaryKey });
    },
  });
}

export function useKeepSummary(enabled: boolean) {
  const api = useApi();
  return useQuery({
    queryKey: summaryKey,
    enabled,
    queryFn: async () => unwrap(await api.GET("/keep-summary")),
  });
}

export type SpeciesRule = components["schemas"]["SpeciesRule"];
const speciesRulesKey = ["setup", "species-rules"] as const;

export function useSpeciesRules(enabled: boolean) {
  const api = useApi();
  return useQuery({
    queryKey: speciesRulesKey,
    enabled,
    queryFn: async () => unwrap(await api.GET("/species-rules")).rules,
  });
}

/** Replaces every species rule; the keep summary follows. */
export function useSaveSpeciesRules() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (rules: SpeciesRule[]) =>
      unwrap(await api.PUT("/species-rules", { body: { rules } })).rules,
    onSuccess: (saved) => {
      queryClient.setQueryData(speciesRulesKey, saved);
      return queryClient.invalidateQueries({ queryKey: summaryKey });
    },
  });
}
