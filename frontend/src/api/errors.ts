import type { components } from "./schema.gen";

export type ApiErrorBody = components["schemas"]["Error"];

/** An error response from the API, carrying the contract error envelope. */
export class ApiError extends Error {
  readonly code: string;
  readonly details: Record<string, unknown> | undefined;

  constructor(body: ApiErrorBody) {
    super(body.error.message);
    this.code = body.error.code;
    this.details = body.error.details;
  }
}

/** Throws ApiError if error is set, otherwise returns data. */
export function unwrap<T>(result: { data?: T; error?: ApiErrorBody }): T {
  if (result.error) {
    throw new ApiError(result.error);
  }
  return result.data as T;
}
