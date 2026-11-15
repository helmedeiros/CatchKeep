import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";

async function withCollection(rows: string[]) {
  const user = userEvent.setup();
  const api = createFakeApi();
  api.signIn("Ash");
  renderWithApi(<App />, api.fetch);
  await user.upload(
    await screen.findByLabelText("Choose your Poke Genie export"),
    new File([[HEADER, ...rows].join("\n")], "export.csv", {
      type: "text/csv",
    }),
  );
  const preview = await screen.findByRole("region", { name: "Import preview" });
  await user.click(
    within(preview).getByRole("button", { name: /^Add \d+ Pokémon$/ }),
  );
  await screen.findByRole("list", { name: "Collection" });
  return user;
}

describe("confirming what the export can't tell", () => {
  it("confirms in bulk that none of a species has the chosen features", async () => {
    const user = await withCollection([
      "1,Pidgey,,16,,120,30",
      "2,Pidgey,,16,,130,31",
      "3,Rattata,,19,,50,20",
    ]);

    const panel = screen.getByRole("region", {
      name: "Confirm what the export can't tell",
    });
    await user.click(within(panel).getByRole("checkbox", { name: "Dynamax" }));
    const pidgey = within(panel).getByRole("group", { name: "Pidgey" });
    expect(pidgey).toHaveTextContent("2 Pokémon");

    await user.click(within(pidgey).getByRole("button", { name: "Confirm" }));

    expect(
      await screen.findByText(
        "Confirmed 2 Pidgey: no shiny, costume, background or Gigantamax.",
      ),
    ).toBeInTheDocument();
    expect(
      within(panel).queryByRole("group", { name: "Pidgey" }),
    ).not.toBeInTheDocument();
    expect(
      within(panel).getByRole("group", { name: "Rattata" }),
    ).toBeInTheDocument();

    await user.click(within(panel).getByRole("checkbox", { name: "Dynamax" }));
    expect(
      await within(panel).findByRole("group", { name: "Pidgey" }),
    ).toBeInTheDocument();
  });

  it("needs at least one feature chosen", async () => {
    const user = await withCollection(["1,Pidgey,,16,,120,30"]);
    const panel = screen.getByRole("region", {
      name: "Confirm what the export can't tell",
    });

    for (const name of [
      "Shiny",
      "Costume",
      "Background",
      "Dynamax",
      "Gigantamax",
    ]) {
      await user.click(within(panel).getByRole("checkbox", { name }));
    }

    expect(
      within(panel).getByText("Choose at least one feature to confirm."),
    ).toBeInTheDocument();
    expect(
      within(panel).queryByRole("button", { name: "Confirm" }),
    ).not.toBeInTheDocument();
  });
});
