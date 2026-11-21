import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { jsonResponse, renderWithApi } from "../test/render";
import { StartAtLogin } from "./StartAtLogin";

function host(state: { supported: boolean; enabled: boolean }): typeof fetch {
  return async (input) => {
    const req = new Request(input);
    if (req.url.endsWith("/host"))
      return jsonResponse(200, { admin: true, addresses: [] });
    if (req.method === "PUT") {
      state.enabled = ((await req.json()) as { enabled: boolean }).enabled;
    }
    return jsonResponse(200, state);
  };
}

describe("StartAtLogin", () => {
  it("turns starting CatchKeep at login on and off", async () => {
    const user = userEvent.setup();
    const state = { supported: true, enabled: false };
    renderWithApi(<StartAtLogin />, host(state));

    const box = await screen.findByRole("checkbox", {
      name: "Start CatchKeep when I log in to this computer",
    });
    expect(box).not.toBeChecked();
    await user.click(box);
    await expect.poll(() => state.enabled).toBe(true);
    expect(box).toBeChecked();
    expect(screen.getByText(/next time you log in/)).toBeInTheDocument();
  });

  it("stays hidden where it is not available", async () => {
    const { container } = renderWithApi(
      <StartAtLogin />,
      host({ supported: false, enabled: false }),
    );
    await new Promise((r) => setTimeout(r, 50));
    expect(container).toBeEmptyDOMElement();
  });
});
