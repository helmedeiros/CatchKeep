import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import type { ReactElement } from "react";
import { ApiProvider } from "../api/ApiProvider";
import { createApi } from "../api/client";

export const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

export const healthyHost: typeof fetch = async () =>
  jsonResponse(200, { status: "ok", version: "test" });

/** Renders ui with a fresh query cache and an API client backed by fetchImpl. */
export function renderWithApi(
  ui: ReactElement,
  fetchImpl: typeof fetch = healthyHost,
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ApiProvider api={createApi({ fetch: fetchImpl })}>{ui}</ApiProvider>
    </QueryClientProvider>,
  );
}
