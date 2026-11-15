import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import App from "./App";
import { renderWithApi } from "./test/render";

describe("App", () => {
  it("introduces CatchKeep as a Pokémon GO storage helper", () => {
    renderWithApi(<App />);

    expect(
      screen.getByRole("heading", { level: 1, name: "CatchKeep" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("A Pokémon GO storage helper."),
    ).toBeInTheDocument();
  });

  it("shows whether the host is reachable", async () => {
    renderWithApi(<App />);

    expect(await screen.findByText("Connected")).toBeInTheDocument();
  });
});
