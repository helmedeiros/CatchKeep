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

/**
 * Retries a failed request up to three times only when the server could
 * not be reached; an answer from the server, like "set up first", will not
 * change by asking again.
 */
export function shouldRetry(failures: number, error: unknown): boolean {
  return !(error instanceof ApiError) && failures < 3;
}
