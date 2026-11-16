import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Sprite } from "./Sprite";
import { spriteUrl } from "./sprites";

describe("Sprite", () => {
  it("loads the render for the Pokédex number, shiny when shiny", () => {
    expect(spriteUrl(150)).toMatch(/\/other\/home\/150\.png$/);
    expect(spriteUrl(150, true)).toMatch(/\/other\/home\/shiny\/150\.png$/);
  });

  it("falls back to the initial when there is no picture", () => {
    const { container } = render(<Sprite species="Mewtwo" dexNumber={150} />);
    const img = container.querySelector("img");
    expect(img).toHaveAttribute("src", spriteUrl(150));
    fireEvent.error(img as HTMLImageElement);
    expect(screen.getByText("M")).toBeInTheDocument();
  });
});
