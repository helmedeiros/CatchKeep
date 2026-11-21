import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { renderWithApi } from "../test/render";
import { goTo, importField } from "../test/places";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";

describe("marking one Pokémon's features", () => {
  it("records what the game shows for that Pokémon only", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);
    await user.upload(
      await importField(user),
      new File(
        [
          [HEADER, "1,Dialga,,483,,2759,150", "2,Dialga,,483,,2249,140"].join(
            "\n",
          ),
        ],
        "export.csv",
        { type: "text/csv" },
      ),
    );
    const preview = await screen.findByRole("region", {
      name: "Import preview",
    });
    await user.click(
      within(preview).getByRole("button", { name: "Add 2 Pokémon" }),
    );
    await goTo(user, "Collection");
    const list = await screen.findByRole("list", { name: "Collection" });
    const [strong, weak] = within(list).getAllByRole("listitem");
    if (!strong || !weak) throw new Error("two rows expected");
    const details = within(strong).getByRole("button", {
      name: "Details for Dialga CP 2759",
    });

    await user.click(details);
    const features = within(strong).getByRole("group", {
      name: "Special features of Dialga CP 2759",
    });
    const feature = (name: string) =>
      within(features).getByRole("group", { name });
    const answer = (name: string, value: "Yes" | "No") =>
      within(feature(name)).getByRole("button", { name: value });
    expect(feature("Shadow")).toHaveTextContent("Not set");

    await user.click(answer("Shadow", "Yes"));
    await expect
      .poll(() => answer("Shadow", "Yes").getAttribute("aria-pressed"))
      .toBe("true");
    await user.click(answer("Purified", "Yes"));
    await expect
      .poll(() => answer("Purified", "Yes").getAttribute("aria-pressed"))
      .toBe("true");
    expect(answer("Shadow", "No")).toHaveAttribute("aria-pressed", "true");

    await user.click(details);
    expect(within(strong).getByText("Purified")).toBeInTheDocument();
    expect(within(strong).queryByText("Shadow")).not.toBeInTheDocument();
    expect(within(weak).queryByText("Purified")).not.toBeInTheDocument();
  });
});
