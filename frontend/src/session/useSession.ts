import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, unwrap } from "../api/errors";
import { useApi } from "../api/useApi";

const meKey = ["me"] as const;
const profilesKey = ["profiles"] as const;

/** The signed-in profile, or null when signed out. */
export function useMe() {
  const api = useApi();
  return useQuery({
    queryKey: meKey,
    queryFn: async () => {
      const result = await api.GET("/me");
      if (result.response.status === 401) {
        return null;
      }
      return unwrap(result);
    },
  });
}

export function useProfiles() {
  const api = useApi();
  return useQuery({
    queryKey: profilesKey,
    queryFn: async () => unwrap(await api.GET("/profiles")),
  });
}

export function useCreateProfile() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { displayName: string; pin: string }) =>
      unwrap(await api.POST("/profiles", { body: input })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: profilesKey }),
  });
}

/** Whether this device is the Mac running CatchKeep, which manages profiles. */
export function useHost() {
  const api = useApi();
  return useQuery({
    queryKey: ["host"],
    staleTime: Infinity,
    queryFn: async () => unwrap(await api.GET("/host")),
  });
}

export function useResetPin() {
  const api = useApi();
  return useMutation({
    mutationFn: async ({
      profileId,
      pin,
    }: {
      profileId: string;
      pin: string;
    }) =>
      unwrap(
        await api.PUT("/profiles/{profileId}/pin", {
          params: { path: { profileId } },
          body: { pin },
        }),
      ),
  });
}

export function useSignIn() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { profileId: string; pin: string }) =>
      unwrap(await api.POST("/session", { body: input })),
    onSuccess: (me) => queryClient.setQueryData(meKey, me),
  });
}

export function useSignOut() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const result = await api.DELETE("/session");
      if (result.error && result.response.status !== 401) {
        throw new ApiError(result.error);
      }
    },
    onSuccess: () => queryClient.setQueryData(meKey, null),
  });
}

/** A message for the trainer explaining why signing in failed. */
export function signInErrorMessage(error: unknown): string {
  if (error instanceof ApiError && error.code === "PIN_INVALID") {
    return "Wrong PIN.";
  }
  if (error instanceof ApiError && error.code === "RATE_LIMITED") {
    const retry =
      typeof error.details?.["retryAfter"] === "string"
        ? new Date(error.details["retryAfter"])
        : null;
    return retry
      ? `Too many wrong PINs. Try again after ${retry.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}.`
      : "Too many wrong PINs. Try again later.";
  }
  return error instanceof Error ? error.message : "Something went wrong.";
}
