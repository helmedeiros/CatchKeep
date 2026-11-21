import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { importField, inPlace } from "../test/places";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";
const rows = [
  "1,Pidgey,,16,,120,30",
  "2,Pidgey,,16,,130,31",
  "3,Pidgey,,16,,140,32",
];

async function nextStep() {
  return screen.findByRole("region", { name: "Next step" });
}

describe("Home", () => {
  it("leads from the first import to making room, one step at a time", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);

    expect(await nextStep()).toHaveTextContent("Bring in your collection");
    await user.upload(
      await importField(user),
      new File([[HEADER, ...rows].join("\n")], "export.csv", {
        type: "text/csv",
      }),
    );
    const preview = await screen.findByRole("region", {
      name: "Import preview",
    });
    await user.click(
      within(preview).getByRole("button", { name: "Add 3 Pokémon" }),
    );

    expect(
      await within(await nextStep()).findByText(
        /3 Pokémon wait for you to confirm/,
      ),
    ).toBeInTheDocument();
    const confirm = await inPlace(user, "Confirm what the export can't tell");
    await user.click(within(confirm).getByRole("button", { name: "Confirm" }));

    const setUp = within(await nextStep()).findByRole("link", {
      name: "Set up storage",
    });
    await user.click(await setUp);
    const setup = await screen.findByRole("region", {
      name: "Your storage and goals",
    });
    await user.type(within(setup).getByLabelText("Storage slots"), "4");
    await user.type(
      within(setup).getByLabelText(/Pokémon in the game now/),
      "4",
    );
    await user.click(
      within(setup).getByRole("checkbox", { name: "Collecting" }),
    );
    await user.click(within(setup).getByRole("button", { name: "Save" }));

    await user.click(
      within(screen.getByRole("navigation", { name: "Main" })).getByRole(
        "link",
        { name: "Home" },
      ),
    );
    const step = await nextStep();
    expect(
      await within(step).findByText(/over your limit/),
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("region", { name: "Storage" }),
    ).toHaveTextContent("4 of 4 slots used");
    await user.click(within(step).getByRole("link", { name: "Make room" }));
    expect(
      await screen.findByRole("heading", { level: 2, name: "Make room" }),
    ).toBeInTheDocument();
  });
});
