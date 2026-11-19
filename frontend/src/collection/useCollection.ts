import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { unwrap } from "../api/errors";
import type { components } from "../api/schema.gen";
import { useApi } from "../api/useApi";

export type Pokemon = components["schemas"]["Pokemon"];
export type ImportPreview = components["schemas"]["ImportPreview"];

const collectionKey = ["collection"] as const;

export function useCollection() {
  const api = useApi();
  return useQuery({
    queryKey: collectionKey,
    queryFn: async () => unwrap(await api.GET("/collection")),
  });
}

export function useStageImport() {
  const api = useApi();
  return useMutation({
    mutationFn: async (csv: string) =>
      unwrap(
        await api.POST("/imports", {
          body: csv,
          bodySerializer: (b: string) => b,
          headers: { "Content-Type": "text/csv" },
        }),
      ),
  });
}

export function useCommitImport() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (importId: string) =>
      unwrap(
        await api.POST("/imports/{importId}/commit", {
          params: { path: { importId } },
        }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: collectionKey }),
  });
}

export type PossibleMatch = components["schemas"]["PossibleMatch"];
export type MatchAnswer = components["schemas"]["MatchAnswer"];

export function useAnswerMatches() {
  const api = useApi();
  return useMutation({
    mutationFn: async ({
      importId,
      answers,
    }: {
      importId: string;
      answers: MatchAnswer[];
    }) =>
      unwrap(
        await api.POST("/imports/{importId}/answers", {
          params: { path: { importId } },
          body: { answers },
        }),
      ),
  });
}

export type ConfirmableFeature = components["schemas"]["ConfirmableFeature"];

export function useConfirmAbsent() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components["schemas"]["AbsenceConfirmation"]) =>
      unwrap(await api.POST("/collection/confirmations", { body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: collectionKey }),
  });
}

/** Protects or unprotects one Pokémon; the collection and keep summary follow. */
export function useProtect() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, protect }: { id: string; protect: boolean }) =>
      unwrap(
        await api.PUT("/collection/{pokemonId}/protection", {
          params: { path: { pokemonId: id } },
          body: { protected: protect },
        }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: collectionKey }),
  });
}
