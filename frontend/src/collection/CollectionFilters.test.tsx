import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { goTo, importField } from "../test/places";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";

describe("searching and filtering the collection", () => {
  it("narrows the list, remembers it while moving around, and clears", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);
    await user.upload(
      await importField(user),
      new File(
        [
          [
            HEADER,
            "1,Pidgey,,16,,120,30",
            "2,Pidgey,,16,,300,40",
            "3,Rattata,,19,,90,20",
          ].join("\n"),
        ],
        "export.csv",
        { type: "text/csv" },
      ),
    );
    const preview = await screen.findByRole("region", {
      name: "Import preview",
    });
    await user.click(
      within(preview).getByRole("button", { name: "Add 3 Pokémon" }),
    );
    await screen.findAllByText(/to your collection\.$/);
    await goTo(user, "Collection");
    const rows = () =>
      within(screen.getByRole("list", { name: "Collection" }))
        .getAllByRole("listitem")
        .map((li) => li.textContent);

    await user.type(
      screen.getByRole("searchbox", { name: "Search Pokémon" }),
      "ratt",
    );
    expect(await screen.findByText("Showing 1 of 3")).toBeInTheDocument();
    expect(rows()).toEqual([expect.stringMatching(/^Rattata/)]);

    await goTo(user, "Home");
    await goTo(user, "Collection");
    expect(
      screen.getByRole("searchbox", { name: "Search Pokémon" }),
    ).toHaveValue("ratt");

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    await user.click(
      within(screen.getByRole("list", { name: "Collection" })).getByRole(
        "button",
        {
          name: "Protect Pidgey CP 120",
        },
      ),
    );
    await user.click(
      within(screen.getByRole("group", { name: "Filters" })).getByRole(
        "button",
        { name: "Protected" },
      ),
    );
    expect(await screen.findByText("Showing 1 of 3")).toBeInTheDocument();
    expect(rows()).toEqual([expect.stringMatching(/^Pidgey.*CP 120/)]);

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    await user.selectOptions(screen.getByLabelText("Sort by"), "species");
    expect(rows()).toEqual([
      expect.stringMatching(/^Pidgey.*CP 300/),
      expect.stringMatching(/^Pidgey.*CP 120/),
      expect.stringMatching(/^Rattata/),
    ]);
  });
});
