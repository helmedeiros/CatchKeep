import { screen, waitFor, within } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";

type Place = "Home" | "Collection" | "Make room" | "Rules" | "More";

/** Goes to a place through the main menu, as a trainer would. */
export async function goTo(user: UserEvent, place: Place) {
  const nav = await screen.findByRole("navigation", { name: "Main" });
  await user.click(within(nav).getByRole("link", { name: place }));
}

const PLACE_OF: Record<string, Place> = {
  "Confirm what the export can't tell": "Home",
  "Your storage and goals": "Rules",
  "Your keep rules": "Rules",
  "Your teams": "Rules",
  "Make room": "Make room",
};

/**
 * The region with this name, going to the place that holds it when it is
 * not on screen. Regions found before a move are gone after it.
 */
export async function inPlace(user: UserEvent, name: string) {
  await placeOpen();
  const here = screen.queryByRole("region", { name });
  if (here) return here;
  const place = PLACE_OF[name];
  if (place) await goTo(user, place);
  return screen.findByRole("region", { name });
}

/** The import field: on Home before the first import, in More after it. */
export async function importField(user: UserEvent) {
  await placeOpen();
  if (!screen.queryByLabelText("Choose your Poke Genie export")) {
    await goTo(user, "More");
  }
  return screen.findByLabelText("Choose your Poke Genie export");
}

/** Waits until the signed-in app shows a place. */
async function placeOpen() {
  await waitFor(() => {
    const heading = screen.queryByRole("heading", {
      level: 2,
      name: /^(Hi, .*|Collection|Make room|Rules|More)$/,
    });
    if (!heading) throw new Error("no place on screen yet");
  });
}
