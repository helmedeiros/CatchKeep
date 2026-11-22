import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { goTo, importField } from "../test/places";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";

describe("recent transfers", () => {
  it("lists what was marked and can put it back", async () => {
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

    await goTo(user, "More");
    const recent = await screen.findByRole("region", {
      name: "Recent transfers",
    });
    expect(
      await within(recent).findByText("Nothing marked as transferred yet."),
    ).toBeInTheDocument();

    await goTo(user, "Collection");
    await user.click(
      await screen.findByRole("button", {
        name: "Mark Rattata CP 90 as transferred",
      }),
    );
    await screen.findByText("Marked 1 Pokémon as transferred.");

    await goTo(user, "More");
    const list = await within(
      screen.getByRole("region", { name: "Recent transfers" }),
    ).findByRole("list");
    expect(list).toHaveTextContent("Rattata CP 90");
    await user.click(
      within(list).getByRole("button", { name: "Put back Rattata CP 90" }),
    );
    expect(await within(list).findByText("Put back")).toBeInTheDocument();
    await goTo(user, "Collection");
    expect(
      await screen.findByRole("button", {
        name: "Mark Rattata CP 90 as transferred",
      }),
    ).toBeInTheDocument();
  });
});
