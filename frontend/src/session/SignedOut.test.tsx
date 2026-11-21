import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { goTo } from "../test/places";
import { renderWithApi } from "../test/render";

describe("signing in", () => {
  it("lets the first trainer create a profile and then sign in to it", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    renderWithApi(<App />, api.fetch);

    await user.type(await screen.findByLabelText("Trainer name"), "Ash");
    await user.type(screen.getByLabelText("New PIN"), "2468");
    await user.type(screen.getByLabelText("Repeat PIN"), "2468");
    await user.click(screen.getByRole("button", { name: "Create profile" }));

    await user.click(await screen.findByRole("button", { name: "Ash" }));
    await user.type(screen.getByLabelText("PIN for Ash"), "2468");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    expect(
      await screen.findByRole("heading", { name: "Hi, Ash" }),
    ).toBeInTheDocument();
  });

  it("refuses to create a profile when the PINs differ", async () => {
    const user = userEvent.setup();
    renderWithApi(<App />, createFakeApi().fetch);

    await user.type(await screen.findByLabelText("Trainer name"), "Ash");
    await user.type(screen.getByLabelText("New PIN"), "2468");
    await user.type(screen.getByLabelText("Repeat PIN"), "1357");
    await user.click(screen.getByRole("button", { name: "Create profile" }));

    expect(
      await screen.findByText("The PINs don't match."),
    ).toBeInTheDocument();
  });

  it("explains a wrong PIN and a lockout", async () => {
    const user = userEvent.setup();
    const api = createFakeApi({ lockAfter: 1 });
    api.addProfile("Ash", "2468");
    renderWithApi(<App />, api.fetch);

    await user.click(await screen.findByRole("button", { name: "Ash" }));
    await user.type(screen.getByLabelText("PIN for Ash"), "1111");
    await user.click(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Wrong PIN.");

    await user.clear(screen.getByLabelText("PIN for Ash"));
    await user.type(screen.getByLabelText("PIN for Ash"), "2468");
    await user.click(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Too many wrong PINs",
    );
  });

  it("signs out back to the profile picker", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.addProfile("Ash", "2468");
    renderWithApi(<App />, api.fetch);
    await user.click(await screen.findByRole("button", { name: "Ash" }));
    await user.type(screen.getByLabelText("PIN for Ash"), "2468");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    await goTo(user, "More");
    await user.click(await screen.findByRole("button", { name: "Sign out" }));

    expect(
      await screen.findByRole("button", { name: "Ash" }),
    ).toBeInTheDocument();
  });

  it("on another device, sends new trainers to the host instead of creating profiles", async () => {
    const api = createFakeApi({ admin: false });
    api.addProfile("Ash", "2468");
    renderWithApi(<App />, api.fetch);

    expect(
      await screen.findByRole("button", { name: "Ash" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "New profile" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByText(
        /add a trainer or reset a PIN on the computer running CatchKeep/,
      ),
    ).toBeInTheDocument();
  });

  it("on the host, resets a forgotten PIN", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.addProfile("Ash", "2468");
    renderWithApi(<App />, api.fetch);

    await user.click(
      await screen.findByRole("button", { name: "Reset a PIN" }),
    );
    const form = screen.getByRole("form", { name: "Reset a PIN" });
    await user.selectOptions(within(form).getByLabelText("Profile"), "Ash");
    await user.type(within(form).getByLabelText("New PIN"), "1357");
    await user.type(within(form).getByLabelText("Repeat PIN"), "1357");
    await user.click(within(form).getByRole("button", { name: "Set new PIN" }));
    expect(
      await screen.findByText(
        "Ash's PIN is changed and they're signed out everywhere.",
      ),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ash" }));
    await user.type(screen.getByLabelText("PIN for Ash"), "1357");
    await user.click(screen.getByRole("button", { name: "Sign in" }));
    expect(
      await screen.findByRole("heading", { name: "Hi, Ash" }),
    ).toBeInTheDocument();
  });
});
