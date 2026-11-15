import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import App from "./App";

describe("App", () => {
  it("introduces CatchKeep as a Pokémon GO storage helper", () => {
    render(<App />);

    expect(
      screen.getByRole("heading", { level: 1, name: "CatchKeep" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("A Pokémon GO storage helper."),
    ).toBeInTheDocument();
  });
});
