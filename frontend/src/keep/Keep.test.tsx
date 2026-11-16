import { screen, waitFor, within } from "@testing-library/react";
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
  const preview = await screen.findByRole("region", {
    name: "Import preview",
  });
  await user.click(
    within(preview).getByRole("button", { name: /^Add \d+ Pokémon$/ }),
  );
  await screen.findByRole("list", { name: "Collection" });
  return user;
}

async function saveSetup(
  user: ReturnType<typeof userEvent.setup>,
  slots: string,
  goals: string[],
) {
  const setup = await screen.findByRole("region", {
    name: "Your storage and goals",
  });
  const input = within(setup).getByLabelText("Storage slots");
  await user.clear(input);
  await user.type(input, slots);
  for (const g of goals) {
    await user.click(within(setup).getByRole("checkbox", { name: g }));
  }
  return setup;
}

const rows = [
  "1,Pidgey,,16,,120,30",
  "2,Pidgey,,16,,130,31",
  "3,Pidgey,,16,,140,32",
  "4,Rattata,,19,,50,20",
];

describe("keep rules", () => {
  it("asks for storage and goals, then shows what the rules keep", async () => {
    const user = await withCollection(rows);

    const setup = await saveSetup(user, "350", ["Trading", "Collecting"]);
    expect(setup).toHaveTextContent("Priority: Trading, then Collecting");
    await user.click(within(setup).getByRole("button", { name: "Save" }));

    const keep = await screen.findByRole("region", {
      name: "Your keep rules",
    });
    expect(keep).toHaveTextContent("350 slots · Trading, then Collecting");
    expect(keep).toHaveTextContent(/4\s*waiting for confirmation/);
    expect(keep).toHaveTextContent(/0\s*possible transfers/);

    const confirm = screen.getByRole("region", {
      name: "Confirm what the export can't tell",
    });
    await user.click(
      within(confirm).getByRole("button", { name: "Confirm all 2 species" }),
    );
    await user.click(
      within(confirm).getByRole("button", { name: "Yes, I checked them all" }),
    );

    await waitFor(() =>
      expect(keep).toHaveTextContent(/2\s*possible transfers/),
    );
    expect(keep).toHaveTextContent(
      "Keeping the best 1 of each variant and 2 of legendary",
    );
    const groups = within(keep).getByRole("list", {
      name: "Possible transfers by species",
    });
    const pidgey = within(groups).getByRole("button", { name: /^Pidgey/ });
    expect(pidgey).toHaveTextContent("3 · keep 1 · 2 extra");
    expect(pidgey.querySelector("img")).toHaveAttribute(
      "src",
      expect.stringMatching(/\/16\.png$/),
    );

    await user.click(pidgey);
    const copies = within(
      within(keep).getByRole("list", { name: "Pidgey copies" }),
    ).getAllByRole("listitem");
    expect(copies.map((c) => c.textContent)).toEqual([
      expect.stringMatching(/CP 140.*Keep/),
      expect.stringMatching(/CP 130.*Possible transfer/),
      expect.stringMatching(/CP 120.*Possible transfer/),
    ]);
  });

  it("keeps more copies of rare species", async () => {
    const user = await withCollection([
      "1,Mewtwo,,150,,4000,180",
      "2,Mewtwo,,150,,4100,181",
      "3,Mewtwo,,150,,4200,182",
    ]);
    const setup = await saveSetup(user, "350", ["Collecting"]);
    await user.click(within(setup).getByRole("button", { name: "Save" }));
    const confirm = await screen.findByRole("region", {
      name: "Confirm what the export can't tell",
    });
    await user.click(within(confirm).getByRole("button", { name: "Confirm" }));

    const keep = screen.getByRole("region", { name: "Your keep rules" });
    const mewtwo = await within(keep).findByRole("button", {
      name: /Mewtwo · rare/,
    });
    expect(mewtwo).toHaveTextContent("3 · keep 2 · 1 extra");
  });

  it("lets the trainer keep more of a species, then go back to the default", async () => {
    const user = await withCollection(rows);
    const setup = await saveSetup(user, "350", ["Collecting"]);
    await user.click(within(setup).getByRole("button", { name: "Save" }));
    const confirm = await screen.findByRole("region", {
      name: "Confirm what the export can't tell",
    });
    await user.click(
      within(confirm).getByRole("button", { name: "Confirm all 2 species" }),
    );
    await user.click(
      within(confirm).getByRole("button", { name: "Yes, I checked them all" }),
    );
    const keep = screen.getByRole("region", { name: "Your keep rules" });
    await user.click(
      await within(keep).findByRole("button", { name: /^Pidgey/ }),
    );

    const editor = within(keep).getByRole("form", {
      name: "Keep rule for Pidgey",
    });
    await user.type(within(editor).getByLabelText("Keep at least"), "2");
    await user.click(within(editor).getByRole("button", { name: "Save rule" }));

    await waitFor(() =>
      expect(keep).toHaveTextContent(/1\s*possible transfers/),
    );
    const mine = within(keep).getByRole("region", {
      name: "Your species rules",
    });
    expect(mine).toHaveTextContent("Pidgey · keep at least 2");
    expect(
      within(keep).getByRole("button", { name: /^Pidgey/ }),
    ).toHaveTextContent("keep 2 (your rule) · 1 extra");

    await user.click(
      within(mine).getByRole("button", { name: "Remove the rule for Pidgey" }),
    );
    await waitFor(() =>
      expect(keep).toHaveTextContent(/2\s*possible transfers/),
    );
    expect(
      within(keep).queryByRole("region", { name: "Your species rules" }),
    ).not.toBeInTheDocument();
  });

  it("keeps every copy of a favorite species", async () => {
    const user = await withCollection(rows);
    const setup = await saveSetup(user, "350", ["Collecting"]);
    await user.click(within(setup).getByRole("button", { name: "Save" }));
    const confirm = await screen.findByRole("region", {
      name: "Confirm what the export can't tell",
    });
    await user.click(
      within(confirm).getByRole("button", { name: "Confirm all 2 species" }),
    );
    await user.click(
      within(confirm).getByRole("button", { name: "Yes, I checked them all" }),
    );
    const keep = screen.getByRole("region", { name: "Your keep rules" });
    await user.click(
      await within(keep).findByRole("button", { name: /^Pidgey/ }),
    );
    const editor = within(keep).getByRole("form", {
      name: "Keep rule for Pidgey",
    });
    await user.click(
      within(editor).getByRole("checkbox", {
        name: "Favorite: keep every copy",
      }),
    );
    await user.click(within(editor).getByRole("button", { name: "Save rule" }));

    await waitFor(() =>
      expect(keep).toHaveTextContent(/0\s*possible transfers/),
    );
    expect(
      within(keep).getByRole("region", { name: "Your species rules" }),
    ).toHaveTextContent("Pidgey · favorite, keep every copy");
  });

  it("lets the trainer change the setup later", async () => {
    const user = await withCollection(rows);
    const setup = await saveSetup(user, "350", ["Collecting"]);
    await user.click(within(setup).getByRole("button", { name: "Save" }));

    const keep = await screen.findByRole("region", {
      name: "Your keep rules",
    });
    await user.click(within(keep).getByRole("button", { name: "Change" }));
    const edit = await screen.findByRole("region", {
      name: "Your storage and goals",
    });
    expect(within(edit).getByLabelText("Storage slots")).toHaveValue(350);
    await saveSetup(user, "2700", []);
    await user.click(within(edit).getByRole("button", { name: "Save" }));

    expect(
      await screen.findByText("2700 slots · Collecting"),
    ).toBeInTheDocument();
  });

  it("explains invalid setups before saving", async () => {
    const user = await withCollection(rows);
    const setup = await saveSetup(user, "350", []);

    expect(within(setup).getByRole("button", { name: "Save" })).toBeDisabled();
    expect(setup).toHaveTextContent("Choose at least one goal.");
  });
});
