import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { jsonResponse, renderWithApi } from "../test/render";
import { SignInLog } from "./SignInLog";

const entries = [
  {
    kind: "paused",
    device: "192.168.1.30",
    at: "2026-11-27T09:05:00Z",
  },
  {
    kind: "wrong_pin",
    profileId: "01JD2Z8Q4M8V7Y6X5W4T3S2R1P",
    profileName: "Ash",
    device: "192.168.1.30",
    at: "2026-11-27T09:04:00Z",
  },
  {
    kind: "signed_in",
    profileId: "01JD2Z8Q4M8V7Y6X5W4T3S2R1P",
    profileName: "Ash",
    device: "127.0.0.1",
    at: "2026-11-27T09:00:00Z",
  },
];

function host(admin: boolean): typeof fetch {
  return async (input) => {
    const req = new Request(input);
    if (req.url.endsWith("/host"))
      return jsonResponse(200, { admin, addresses: [] });
    if (req.url.endsWith("/sign-ins")) return jsonResponse(200, { entries });
    return jsonResponse(404, {});
  };
}

describe("SignInLog", () => {
  it("lists recent sign-ins, wrong PINs and pauses with the device", async () => {
    const user = userEvent.setup();
    renderWithApi(<SignInLog />, host(true));

    await user.click(await screen.findByText("Recent sign-ins"));
    const items = within(
      screen.getByRole("list", { name: "Recent sign-ins" }),
    ).getAllByRole("listitem");
    expect(items.map((li) => li.textContent)).toEqual([
      expect.stringMatching(
        /^Sign-in paused after too many wrong PINs.*from the device at 192\.168\.1\.30$/,
      ),
      expect.stringMatching(
        /^Wrong PIN for Ash.*from the device at 192\.168\.1\.30$/,
      ),
      expect.stringMatching(/^Ash signed in.*from this computer$/),
    ]);
    expect(screen.getByText(/never in backups/)).toBeInTheDocument();
  });

  it("stays hidden on other devices", async () => {
    const { container } = renderWithApi(<SignInLog />, host(false));
    await new Promise((r) => setTimeout(r, 50));
    expect(container).toBeEmptyDOMElement();
  });
});
