import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";
const csv = (rows: string[]) =>
  new File([[HEADER, ...rows].join("\n")], "poke_genie_export.csv", {
    type: "text/csv",
  });

describe("importing a collection", () => {
  it("previews the export, adds it, and shows the collection strongest first", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);

    await user.upload(
      await screen.findByLabelText("Choose your Poke Genie export"),
      csv([
        "1,Pidgey,,16,,120,30",
        "2,Mewtwo,,150,,4178,180",
        "3,Charizard,,6,,1054,94,15,12,11,84.4,13.0,13.0,,,,x,x,x,,,1",
      ]),
    );

    const preview = await screen.findByRole("region", {
      name: "Import preview",
    });
    expect(within(preview).getByText("3 Pokémon found")).toBeInTheDocument();
    expect(
      within(preview).getByText(
        /shiny, costume, background, dynamax, gigantamax/i,
      ),
    ).toBeInTheDocument();

    await user.click(
      within(preview).getByRole("button", { name: "Add 3 Pokémon" }),
    );

    expect(
      await screen.findByText("Added 3 Pokémon to your collection."),
    ).toBeInTheDocument();
    const list = await screen.findByRole("list", { name: "Collection" });
    const items = within(list).getAllByRole("listitem");
    expect(items[0]).toHaveTextContent("Mewtwo");
    expect(items[0]).toHaveTextContent("CP 4178");
    expect(screen.getByText("3 Pokémon")).toBeInTheDocument();
    expect(within(list).getByText("Lucky")).toBeInTheDocument();
  });

  it("explains files that are not a Poke Genie export", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);

    await user.upload(
      await screen.findByLabelText("Choose your Poke Genie export"),
      new File(["name,cp\nPikachu,500"], "other.csv", { type: "text/csv" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "not a Poke Genie CSV export",
    );
  });

  it("lets the trainer cancel a staged import without changing anything", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);

    await user.upload(
      await screen.findByLabelText("Choose your Poke Genie export"),
      csv(["1,Pidgey,,16,,120,30"]),
    );
    await user.click(await screen.findByRole("button", { name: "Cancel" }));

    expect(
      screen.queryByRole("region", { name: "Import preview" }),
    ).not.toBeInTheDocument();
    expect(screen.getByText("No Pokémon yet.")).toBeInTheDocument();
  });
});
