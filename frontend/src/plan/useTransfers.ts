import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { unwrap } from "../api/errors";
import { useApi } from "../api/useApi";

/** Everything a transfer changes: the collection, rules, plans and history. */
const changedKeys = [["collection"], ["history"]] as const;

/**
 * A request id for the server to count a confirmation once. Plain HTTP on a
 * network is not a secure context, so crypto.randomUUID may be missing.
 */
function requestId(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

const pokemon = (n: number) => `${n} Pokémon`;

/**
 * Marks Pokémon as transferred in the game, all or none, and offers to undo
 * it. With a plan revision the server refuses when the plan is out of date.
 */
export function useConfirmTransfers() {
  const api = useApi();
  const queryClient = useQueryClient();
  const refresh = () =>
    Promise.all(
      changedKeys.map((queryKey) =>
        queryClient.invalidateQueries({ queryKey }),
      ),
    );
  const undo = useMutation({
    mutationFn: async (eventId: string) =>
      unwrap(
        await api.POST("/transfers/{eventId}/undo", {
          params: { path: { eventId } },
        }),
      ),
    onSuccess: (e) => {
      toast.success(`Put ${pokemon(e.pokemon.length)} back in CatchKeep.`);
      return refresh();
    },
    onError: (e) => toast.error(e.message),
  });
  return useMutation({
    mutationFn: async (input: {
      pokemonIds: string[];
      planRevision?: number;
    }) =>
      unwrap(
        await api.POST("/transfers", {
          body: { ...input, requestId: requestId() },
        }),
      ),
    onSuccess: (e) => {
      toast.success(`Marked ${pokemon(e.pokemon.length)} as transferred.`, {
        action: { label: "Undo", onClick: () => undo.mutate(e.id) },
      });
      return refresh();
    },
    onError: (e, input) => {
      if (input.planRevision === undefined) toast.error(e.message);
      return refresh();
    },
  });
}
