import createClient from "openapi-fetch";
import type { paths } from "./schema.gen";

export type Api = ReturnType<typeof createClient<paths>>;

export interface ApiOptions {
  fetch?: typeof fetch;
}

/** Creates a typed client for the CatchKeep API served under /api/v1. */
export function createApi(options: ApiOptions = {}): Api {
  return createClient<paths>({
    baseUrl: `${globalThis.location.origin}/api/v1`,
    ...(options.fetch ? { fetch: options.fetch } : {}),
  });
}
