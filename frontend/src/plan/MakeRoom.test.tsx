import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";

async function ready(rows: string[]) {
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
  const confirm = await screen.findByRole("region", {
    name: "Confirm what the export can't tell",
  });
  await user.click(within(confirm).getByRole("button", { name: /^Confirm/ }));
  const yes = within(confirm).queryByRole("button", {
    name: "Yes, I checked them all",
  });
  if (yes) await user.click(yes);
  const setup = screen.getByRole("region", { name: "Your storage and goals" });
  await user.type(within(setup).getByLabelText("Storage slots"), "10");
  await user.click(within(setup).getByRole("checkbox", { name: "Collecting" }));
  return { user, setup };
}

const pidgeys = [
  "1,Pidgey,,16,,120,30",
  "2,Pidgey,,16,,130,31",
  "3,Pidgey,,16,,140,32",
  "4,Pidgey,,16,,90,29",
];

describe("making room", () => {
  it("suggests the least valuable transfers to get under the limit", async () => {
    const { user, setup } = await ready(pidgeys);
    await user.type(
      within(setup).getByLabelText(/Pokémon in the game now/),
      "9",
    );
    await user.clear(within(setup).getByLabelText(/Keep storage under/));
    await user.type(within(setup).getByLabelText(/Keep storage under/), "70");
    await user.click(within(setup).getByRole("button", { name: "Save" }));

    const room = await screen.findByRole("region", { name: "Make room" });
    expect(room).toHaveTextContent("9 of 10 slots used");
    expect(room).toHaveTextContent("your limit is 70% (7 slots)");
    expect(
      await within(room).findByRole("heading", { name: "Transfer 2 Pokémon" }),
    ).toBeInTheDocument();
    const list = within(room).getByRole("list", {
      name: "Suggested transfers",
    });
    const items = within(list).getAllByRole("listitem");
    expect(items.map((i) => i.textContent)).toEqual([
      expect.stringMatching(/CP 90.*1 better Pidgey kept/),
      expect.stringMatching(/CP 120.*1 better Pidgey kept/),
    ]);
  });

  it("frees more slots before an event and says when the rules allow fewer", async () => {
    const { user, setup } = await ready(pidgeys);
    await user.click(within(setup).getByRole("button", { name: "Save" }));

    const room = await screen.findByRole("region", { name: "Make room" });
    expect(room).toHaveTextContent("estimate");
    await user.click(
      within(room).getByRole("radio", { name: /Free more slots/ }),
    );
    await user.type(within(room).getByLabelText("Slots to free"), "5");

    expect(
      await within(room).findByRole("heading", { name: "Transfer 3 Pokémon" }),
    ).toBeInTheDocument();
    expect(room).toHaveTextContent(
      "Your keep rules allow 3, so that's 2 short of your target.",
    );
  });

  it("gives a search to find them in the game and warns when it would catch one to keep", async () => {
    const { user, setup } = await ready([
      "1,Pidgey,,16,,120,30",
      "2,Pidgey,,16,,140,32",
      "3,Pidgey,,16,,140,31",
      "4,Pidgey,,16,,150,33",
    ]);
    await user.type(
      within(setup).getByLabelText(/Pokémon in the game now/),
      "10",
    );
    await user.click(within(setup).getByRole("button", { name: "Save" }));

    const room = await screen.findByRole("region", { name: "Make room" });
    const find = await within(room).findByRole("region", {
      name: "Find them in the game",
    });
    const pidgey = within(find).getByRole("group", { name: "Pidgey" });
    expect(pidgey).toHaveTextContent("16&cp120,cp140");
    expect(
      within(pidgey).getByRole("button", {
        name: "Copy the search for Pidgey",
      }),
    ).toBeInTheDocument();
    expect(pidgey).toHaveTextContent(
      "This search also finds 1 Pidgey you keep (CP 140, HP 31).",
    );
  });
});
