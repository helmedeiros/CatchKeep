import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { goTo, importField } from "../test/places";
import { renderWithApi } from "../test/render";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";
const PIDGEY = "1,Pidgey,,16,,120,30";
const RATTATA = "2,Rattata,,19,,90,20";
const ZUBAT = "3,Zubat,,41,,30,15";

async function importRows(
  user: ReturnType<typeof userEvent.setup>,
  rows: string[],
) {
  await user.upload(
    await importField(user),
    new File([[HEADER, ...rows].join("\n")], "export.csv", {
      type: "text/csv",
    }),
  );
  return screen.findByRole("region", { name: "Import preview" });
}

async function reimportWithoutZubat() {
  const user = userEvent.setup();
  const api = createFakeApi();
  api.signIn("Ash");
  renderWithApi(<App />, api.fetch);
  const first = await importRows(user, [PIDGEY, RATTATA, ZUBAT]);
  await user.click(
    within(first).getByRole("button", { name: "Add 3 Pokémon" }),
  );
  await screen.findAllByText(/to your collection\.$/);

  const again = await importRows(user, [PIDGEY, RATTATA]);
  expect(again).toHaveTextContent("1 Pokémon in CatchKeep isn't in this file.");
  await user.click(
    within(again).getByRole("button", {
      name: "Update 2 Pokémon you already have",
    }),
  );
  await user.click(
    await within(again).findByRole("button", { name: "Update 2 Pokémon" }),
  );
  const missing = await screen.findByRole("region", {
    name: "Not in this file",
  });
  return { user, missing };
}

describe("Pokémon missing from a new export", () => {
  it("offers to mark them as transferred", async () => {
    const { user, missing } = await reimportWithoutZubat();
    expect(missing).toHaveTextContent("Zubat CP 30");

    await user.click(
      within(missing).getByRole("button", { name: "Mark 1 as transferred" }),
    );
    expect(
      await screen.findByText("Marked 1 Pokémon as transferred."),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("region", { name: "Not in this file" }),
    ).not.toBeInTheDocument();
    await goTo(user, "Collection");
    expect(await screen.findByText("2 Pokémon")).toBeInTheDocument();
  });

  it("keeps them when the file had only some Pokémon", async () => {
    const { user, missing } = await reimportWithoutZubat();
    await user.click(within(missing).getByRole("button", { name: "Keep it" }));
    expect(
      screen.queryByRole("region", { name: "Not in this file" }),
    ).not.toBeInTheDocument();
    await goTo(user, "Collection");
    expect(await screen.findByText("3 Pokémon")).toBeInTheDocument();
  });
});
