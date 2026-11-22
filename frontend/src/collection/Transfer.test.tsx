import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { goTo, importField } from "../test/places";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";

describe("marking one Pokémon as transferred from the collection", () => {
  it("takes it out of the list, with an undo", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);
    await user.upload(
      await importField(user),
      new File(
        [[HEADER, "1,Pidgey,,16,,120,30", "2,Rattata,,19,,90,20"].join("\n")],
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
    await screen.findAllByText(/to your collection\.$/);
    await goTo(user, "Collection");
    const list = await screen.findByRole("list", { name: "Collection" });

    await user.click(
      within(list).getByRole("button", {
        name: "Mark Rattata CP 90 as transferred",
      }),
    );
    expect(
      await screen.findByText("Marked 1 Pokémon as transferred."),
    ).toBeInTheDocument();
    expect(await screen.findByText("1 Pokémon")).toBeInTheDocument();
    expect(within(list).queryByText("Rattata")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Undo" }));
    expect(await within(list).findByText("Rattata")).toBeInTheDocument();
  });
});
