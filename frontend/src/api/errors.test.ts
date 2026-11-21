import { describe, expect, it } from "vitest";
import { ApiError, shouldRetry } from "./errors";

describe("shouldRetry", () => {
  it("retries network failures a few times but never the server's answers", () => {
    const refused = new ApiError({
      error: {
        code: "SETUP_REQUIRED",
        message: "Set up first.",
        correlationId: "x",
      },
    });
    expect(shouldRetry(0, refused)).toBe(false);
    expect(shouldRetry(0, new TypeError("Failed to fetch"))).toBe(true);
    expect(shouldRetry(3, new TypeError("Failed to fetch"))).toBe(false);
  });
});
