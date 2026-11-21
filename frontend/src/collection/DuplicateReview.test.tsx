import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import App from "../App";
import { createFakeApi } from "../test/fakeApi";
import { renderWithApi } from "../test/render";
import { goTo, importField } from "../test/places";

const HEADER = "Index,Name,Form,Pokemon Number,Gender,CP,HP";
const csv = (rows: string[]) =>
  new File([[HEADER, ...rows].join("\n")], "poke_genie_export.csv", {
    type: "text/csv",
  });

async function importAndAdd(
  user: ReturnType<typeof userEvent.setup>,
  rows: string[],
) {
  await user.upload(await importField(user), csv(rows));
  const preview = await screen.findByRole("region", { name: "Import preview" });
  await user.click(
    within(preview).getByRole("button", { name: /^Add \d+ Pokémon$/ }),
  );
  await screen.findAllByText(/to your collection\.$/);
}

async function stage(user: ReturnType<typeof userEvent.setup>, rows: string[]) {
  await user.upload(await importField(user), csv(rows));
  return screen.findByRole("region", { name: "Import preview" });
}

describe("re-importing an updated export", () => {
  it("updates exact matches in bulk once the trainer confirms", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);
    await importAndAdd(user, [
      "1,Pidgey,,16,,120,30",
      "2,Mewtwo,,150,,4178,180",
    ]);

    const preview = await stage(user, [
      "1,Pidgey,,16,,300,50",
      "2,Mewtwo,,150,,4178,180",
      "3,Rattata,,19,,50,20",
    ]);
    expect(preview).toHaveTextContent("2 may already be in your collection");

    await user.click(
      within(preview).getByRole("button", {
        name: "Update 2 Pokémon you already have",
      }),
    );
    await user.click(
      await within(preview).findByRole("button", {
        name: "Add 1 and update 2 Pokémon",
      }),
    );

    expect(
      await screen.findByText("Added 1 and updated 2 Pokémon."),
    ).toBeInTheDocument();
    await goTo(user, "Collection");
    const list = await screen.findByRole("list", { name: "Collection" });
    expect(within(list).getAllByRole("listitem")).toHaveLength(3);
    expect(list).toHaveTextContent("CP 300");
  });

  it("shows each possible duplicate side by side and records the answer", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);
    await importAndAdd(user, ["1,Pidgey,,16,,120,30", "2,Pidgey,,16,,130,31"]);

    const preview = await stage(user, [
      "1,Pidgey,,16,,300,50",
      "2,Pidgey,,16,,310,51",
    ]);
    expect(
      within(preview).queryByRole("button", { name: /you already have/ }),
    ).not.toBeInTheDocument();
    await user.click(
      within(preview).getByRole("button", { name: "Review one by one" }),
    );

    const first = await screen.findByRole("group", {
      name: "Possible duplicate 1 of 2",
    });
    expect(
      within(first).getByText("In this file · line 2"),
    ).toBeInTheDocument();
    expect(within(first).getByText("CP 300")).toBeInTheDocument();
    expect(
      within(first).getAllByRole("button", { name: "Same Pokémon" }),
    ).toHaveLength(2);
    await user.click(
      within(first).getAllByRole("button", { name: "Same Pokémon" })[0]!,
    );

    const second = await screen.findByRole("group", {
      name: "Possible duplicate 1 of 1",
    });
    expect(
      within(second).getByText("In this file · line 3"),
    ).toBeInTheDocument();
    await user.click(
      within(second).getByRole("button", { name: "Different Pokémon" }),
    );

    await user.click(
      await within(preview).findByRole("button", {
        name: "Add 1 and update 1 Pokémon",
      }),
    );
    expect(
      await screen.findByText("Added 1 and updated 1 Pokémon."),
    ).toBeInTheDocument();
    await goTo(user, "Collection");
    const list = await screen.findByRole("list", { name: "Collection" });
    expect(within(list).getAllByRole("listitem")).toHaveLength(3);
  });

  it("can stop reviewing and keep the rest held back", async () => {
    const user = userEvent.setup();
    const api = createFakeApi();
    api.signIn("Ash");
    renderWithApi(<App />, api.fetch);
    await importAndAdd(user, ["1,Pidgey,,16,,120,30", "2,Pidgey,,16,,130,31"]);

    const preview = await stage(user, ["1,Pidgey,,16,,300,50"]);
    await user.click(
      within(preview).getByRole("button", { name: "Review one by one" }),
    );
    await user.click(
      await screen.findByRole("button", { name: "Stop reviewing" }),
    );

    expect(
      screen.queryByRole("group", { name: /Possible duplicate/ }),
    ).not.toBeInTheDocument();
    expect(preview).toHaveTextContent("1 may already be in your collection");
    expect(
      within(preview).getByRole("button", { name: "Nothing to add yet" }),
    ).toBeDisabled();
  });
});
