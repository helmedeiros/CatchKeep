import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { jsonResponse, renderWithApi } from "../test/render";
import { HostStatus } from "./HostStatus";

describe("HostStatus", () => {
  it("shows that the host is reachable and which version it runs", async () => {
    const fetchImpl: typeof fetch = async (input) => {
      expect(new Request(input).url).toMatch(/\/api\/v1\/health$/);
      return jsonResponse(200, { status: "ok", version: "1.2.3" });
    };

    renderWithApi(<HostStatus />, fetchImpl);

    expect(await screen.findByText("Connected")).toBeInTheDocument();
    expect(screen.getByText("Version 1.2.3")).toBeInTheDocument();
  });

  it("tells the trainer the host is unreachable when the request fails", async () => {
    const fetchImpl: typeof fetch = async () => {
      throw new TypeError("Failed to fetch");
    };

    renderWithApi(<HostStatus />, fetchImpl);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "CatchKeep isn't reachable — is the computer running it on and awake?",
    );
  });

  it("treats an error response as unreachable", async () => {
    const fetchImpl: typeof fetch = async () =>
      jsonResponse(500, {
        error: { code: "INTERNAL", message: "boom", correlationId: "c1" },
      });

    renderWithApi(<HostStatus />, fetchImpl);

    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });

  it("explains how to open CatchKeep when it was opened by a name it does not answer to", async () => {
    const fetchImpl: typeof fetch = async () =>
      jsonResponse(421, {
        error: {
          code: "HOST_NOT_ALLOWED",
          message:
            "Open CatchKeep by the address shown on the computer running it.",
          correlationId: "x",
        },
      });

    renderWithApi(<HostStatus />, fetchImpl);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Open CatchKeep using the address shown on the computer running it, under “Use CatchKeep on other devices”.",
    );
  });
});
