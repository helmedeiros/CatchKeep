import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";

async function withCollection() {
  const user = userEvent.setup();
  const api = createFakeApi();
  api.signIn("Ash");
  renderWithApi(<App />, api.fetch);
  await user.upload(
    await screen.findByLabelText("Choose your Poke Genie export"),
    new File(
      [[HEADER, "1,Mewtwo,,150,,4000,180", "2,Pidgey,,16,,120,30"].join("\n")],
      "export.csv",
      { type: "text/csv" },
    ),
  );
  const preview = await screen.findByRole("region", { name: "Import preview" });
  await user.click(
    within(preview).getByRole("button", { name: "Add 2 Pokémon" }),
  );
  return {
    user,
    teams: await screen.findByRole("region", { name: "Your teams" }),
  };
}

describe("teams", () => {
  it("creates a team, adds Pokémon by searching, and deletes it", async () => {
    const { user, teams } = await withCollection();

    await user.type(within(teams).getByLabelText("New team name"), "Raid");
    await user.click(
      within(teams).getByRole("button", { name: "Create team" }),
    );

    const editor = await within(teams).findByRole("form", {
      name: "Edit Raid",
    });
    await user.type(within(editor).getByLabelText("Add Pokémon"), "mew");
    const matches = within(editor).getByRole("list", { name: "Matches" });
    expect(within(matches).getAllByRole("listitem")).toHaveLength(1);
    await user.click(
      within(matches).getByRole("button", { name: "Add Mewtwo CP 4000" }),
    );
    expect(
      within(within(editor).getByRole("list", { name: "Members" })).getByText(
        "Mewtwo",
      ),
    ).toBeInTheDocument();
    await user.click(within(editor).getByRole("button", { name: "Save team" }));

    const raid = await within(teams).findByRole("group", { name: "Raid" });
    expect(raid).toHaveTextContent("1 Pokémon");

    await user.click(within(raid).getByRole("button", { name: "Delete" }));
    await user.click(within(raid).getByRole("button", { name: "Yes, delete" }));
    await waitFor(() =>
      expect(
        within(teams).queryByRole("group", { name: "Raid" }),
      ).not.toBeInTheDocument(),
    );
  });

  it("explains a name another team already uses", async () => {
    const { user, teams } = await withCollection();
    for (const name of ["Raid", "raid"]) {
      await user.type(within(teams).getByLabelText("New team name"), name);
      await user.click(
        within(teams).getByRole("button", { name: "Create team" }),
      );
      if (name === "Raid") {
        await user.click(
          await within(teams).findByRole("button", { name: "Cancel" }),
        );
      }
    }
    expect(
      (await screen.findAllByText("Another team already has that name."))
        .length,
    ).toBeGreaterThan(0);
  });
});
