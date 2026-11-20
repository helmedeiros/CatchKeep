import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { jsonResponse, renderWithApi } from "../test/render";
import { OtherDevices } from "./OtherDevices";

const host =
  (body: { admin: boolean; addresses: string[] }): typeof fetch =>
  async () =>
    jsonResponse(200, body);

describe("OtherDevices", () => {
  it("shows the computer running CatchKeep the address to open on other devices", async () => {
    renderWithApi(
      <OtherDevices />,
      host({ admin: true, addresses: ["http://192.168.1.20:8080"] }),
    );

    const region = await screen.findByRole("region", {
      name: "Use CatchKeep on other devices",
    });
    expect(region).toHaveTextContent("http://192.168.1.20:8080");
    expect(region).toHaveTextContent(/same network/);
    expect(
      screen.getByRole("button", { name: "Copy http://192.168.1.20:8080" }),
    ).toBeInTheDocument();
  });

  it("explains when only this computer can open CatchKeep", async () => {
    renderWithApi(<OtherDevices />, host({ admin: true, addresses: [] }));

    expect(
      await screen.findByText(
        /Right now only this computer can open CatchKeep/,
      ),
    ).toBeInTheDocument();
  });

  it("shows nothing on other devices", async () => {
    const { container } = renderWithApi(
      <OtherDevices />,
      host({ admin: false, addresses: [] }),
    );
    await new Promise((r) => setTimeout(r, 50));
    expect(container).toBeEmptyDOMElement();
  });
});
