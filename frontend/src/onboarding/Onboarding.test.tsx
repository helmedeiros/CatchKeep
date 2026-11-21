import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { goTo } from "../test/places";
import { renderWithApi } from "../test/render";

describe("onboarding", () => {
  it("introduces CatchKeep once per profile and can be shown again", async () => {
    const user = userEvent.setup();
    const api = createFakeApi({ onboarded: false });
    api.signIn("Ash");
    const { unmount } = renderWithApi(<App />, api.fetch);

    const intro = await screen.findByRole("dialog", {
      name: "Keep the best, let the rest go",
    });
    expect(intro).toHaveTextContent("Step 1 of 4");
    await user.click(within(intro).getByRole("button", { name: "Next" }));
    expect(
      await screen.findByRole("dialog", { name: "Bring in your collection" }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Back" }));
    expect(
      await screen.findByRole("dialog", {
        name: "Keep the best, let the rest go",
      }),
    ).toBeInTheDocument();
    for (let i = 0; i < 3; i++)
      await user.click(screen.getByRole("button", { name: "Next" }));
    const last = await screen.findByRole("dialog", {
      name: "Use it on your phone",
    });
    expect(
      within(last).getByRole("img", { name: /^QR code for / }),
    ).toBeInTheDocument();
    await user.click(within(last).getByRole("button", { name: "Let's go" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    unmount();
    renderWithApi(<App />, api.fetch);
    await goTo(user, "More");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await user.click(
      screen.getByRole("button", { name: "Show the introduction again" }),
    );
    const again = await screen.findByRole("dialog", {
      name: "Keep the best, let the rest go",
    });
    await user.click(within(again).getByRole("button", { name: "Skip" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
