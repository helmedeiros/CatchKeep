import { describe, expect, it } from "vitest";
import { applyFilters, NO_FILTERS } from "./filters";
import type { Pokemon } from "./useCollection";

const unknown = "unknown" as const;
function mon(
  id: string,
  species: string,
  cp: number,
  iv: number,
  extra: Partial<Pokemon> = {},
): Pokemon {
  return {
    id,
    species,
    form: "Normal",
    dexNumber: species === "Pidgey" ? 16 : 19,
    protected: false,
    cp,
    iv: { attack: iv, defense: iv, stamina: iv },
    catchDate: `2026-10-${String((cp % 28) + 1).padStart(2, "0")}`,
    attributes: {
      shiny: "no",
      lucky: unknown,
      shadow: unknown,
      purified: unknown,
      favorite: unknown,
      costume: "no",
      background: "no",
      dynamax: "no",
      gigantamax: "no",
    },
    ...extra,
  };
}

const list = [
  mon("P1", "Pidgey", 120, 15),
  mon("P2", "Pidgey", 300, 5, { protected: true }),
  mon("R1", "Rattata", 90, 10, {
    attributes: { ...mon("x", "Rattata", 1, 1).attributes, shiny: "yes" },
  }),
  mon("R2", "Rattata", 200, 12, {
    attributes: { ...mon("x", "Rattata", 1, 1).attributes, costume: unknown },
  }),
];
const ctx = { teamIds: new Set(["R2"]), transferIds: new Set(["P1"]) };
const ids = (f: Partial<typeof NO_FILTERS>) =>
  applyFilters(list, { ...NO_FILTERS, ...f }, ctx).map((p) => p.id);

describe("applyFilters", () => {
  it("searches species, form and Pokédex number", () => {
    expect(ids({ query: "pid" })).toEqual(["P2", "P1"]);
    expect(ids({ query: "19" })).toEqual(["R2", "R1"]);
    expect(ids({ query: "  RATT " })).toEqual(["R2", "R1"]);
  });

  it("keeps only Pokémon matching every chosen chip", () => {
    expect(ids({ chips: ["shiny"] })).toEqual(["R1"]);
    expect(ids({ chips: ["protected"] })).toEqual(["P2"]);
    expect(ids({ chips: ["team"] })).toEqual(["R2"]);
    expect(ids({ chips: ["transfer"] })).toEqual(["P1"]);
    expect(ids({ chips: ["unconfirmed"] })).toEqual(["R2"]);
    expect(ids({ chips: ["shiny", "protected"] })).toEqual([]);
  });

  it("filters by IVs and sorts", () => {
    expect(ids({ minIv: 80 })).toEqual(["R2", "P1"]);
    expect(ids({ sort: "iv" })).toEqual(["P1", "R2", "R1", "P2"]);
    expect(ids({ sort: "species" })).toEqual(["P2", "P1", "R2", "R1"]);
    expect(ids({})).toEqual(["P2", "R2", "P1", "R1"]);
  });
});
